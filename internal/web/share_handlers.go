package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mcbill1/birthdaymemo/internal/audit"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/models"
)

// randomShareToken 生成 32B crypto/rand hex（与 EmailConfirmation 同级强度）
func randomShareToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// ---- Owner 端：分享链接管理（需认证；写操作挂载在 BlockGuestWrites 组，guest 读列表允许） ----

type createShareRequest struct {
	Name      string `json:"name"`
	ScopeMode string `json:"scope_mode"` // all | tags
	TagIDs    []uint `json:"tag_ids"`
	ExpiresAt *time.Time `json:"expires_at"`
	Slug      string `json:"slug"` // 可选自定义短址，空则使用随机 token
}

type updateShareRequest struct {
	Slug *string `json:"slug"` // 指向空/空白字符串表示清除；nil 表示不改
	Name *string `json:"name"` // nil 表示不改
	ScopeMode *string `json:"scope_mode"` // all | tags；nil 表示不改
	TagIDs *[]uint `json:"tag_ids"` // nil 表示不改；非 nil 时按归属校验后覆盖
}

type shareLinkResponse struct {
	ID        uint       `json:"id"`
	Name      string     `json:"name"`
	ScopeMode string     `json:"scope_mode"`
	TagIDs    []uint     `json:"tag_ids"`
	ExpiresAt *time.Time `json:"expires_at"`
	Revoked   bool       `json:"revoked"`
	Slug      string     `json:"slug"`
	URL       string     `json:"url"`
	CreatedAt time.Time  `json:"created_at"`
}

func (s *Server) shareBaseURL(r *http.Request) string {
	if s.cfg.ExternalURL != "" {
		return strings.TrimSuffix(s.cfg.ExternalURL, "/")
	}
	scheme := "http"
	if IsHTTPS(r) {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func shareToResponse(s *Server, r *http.Request, link models.ShareLink) shareLinkResponse {
	var ids []uint
	_ = json.Unmarshal([]byte(link.TagIDs), &ids)
	if ids == nil {
		ids = []uint{}
	}
	// 有自定义 slug 时展示短址，否则回退到随机 token
	key := link.Token
	if link.Slug != "" {
		key = link.Slug
	}
	return shareLinkResponse{
		ID: link.ID, Name: link.Name, ScopeMode: link.ScopeMode, TagIDs: ids,
		ExpiresAt: link.ExpiresAt, Revoked: link.Revoked, Slug: link.Slug,
		URL:       s.shareBaseURL(r) + "/#/s/" + key,
		CreatedAt: link.CreatedAt,
	}
}

// resolveShareSlug 校验并规范化用户提交的 slug，空输入返回 ""（表示不设置）。
// 唯一性检查由调用方负责（全局唯一，更新时排除自身）。
func resolveShareSlug(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", true
	}
	return models.NormalizeShareSlug(raw)
}

// slugTaken 检查 slug 是否已被其他链接占用（excludeID 为 0 表示新建时检查）
func (s *Server) slugTaken(slug string, excludeID uint) bool {
	if slug == "" {
		return false
	}
	var count int64
	q := s.db.Model(&models.ShareLink{}).Where("slug = ?", slug)
	if excludeID != 0 {
		q = q.Where("id <> ?", excludeID)
	}
	q.Count(&count)
	return count > 0
}

func (s *Server) createShareLink(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	var req createShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	mode := strings.TrimSpace(req.ScopeMode)
	if mode == "" {
		mode = models.ShareScopeAll
	}
	if mode != models.ShareScopeAll && mode != models.ShareScopeTags {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	// tag 归属校验：必须全属于自己
	uniq := dedupeIDs(req.TagIDs)
	if mode == models.ShareScopeTags {
		if len(uniq) == 0 {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
			return
		}
		var count int64
		s.db.Model(&models.Tag{}).Where("id IN ? AND user_id = ?", uniq, uid).Count(&count)
		if count != int64(len(uniq)) {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.notFound"))
			return
		}
	} else {
		uniq = []uint{}
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now()) {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "validation.invalidDay"))
		return
	}
	rawIDs, _ := json.Marshal(uniq)
	name := strings.TrimSpace(req.Name)
	if len([]rune(name)) > 50 {
		name = string([]rune(name)[:50])
	}
	slug, ok := resolveShareSlug(req.Slug)
	if !ok {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "validation.shareSlugInvalid"))
		return
	}
	if s.slugTaken(slug, 0) {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "validation.shareSlugTaken"))
		return
	}
	link := models.ShareLink{
		Token: randomShareToken(), Slug: slug, OwnerUserID: uid, Name: name,
		ScopeMode: mode, TagIDs: string(rawIDs), ExpiresAt: req.ExpiresAt,
	}
	if err := s.db.Create(&link).Error; err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	audit.Log(s.db, uid, userFromContext(r).Username, "create_share_link", name, ipFromContext(r))
	OK(w, shareToResponse(s, r, link))
}

func (s *Server) listShareLinks(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	var links []models.ShareLink
	s.db.Where("owner_user_id = ?", uid).Order("id desc").Find(&links)
	out := make([]shareLinkResponse, 0, len(links))
	for _, l := range links {
		out = append(out, shareToResponse(s, r, l))
	}
	OK(w, out)
}

func (s *Server) deleteShareLink(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	id := chi.URLParam(r, "id")
	var link models.ShareLink
	if err := s.db.Where("id = ? AND owner_user_id = ?", id, uid).First(&link).Error; err != nil {
		// admin 可删任意
		var u models.User
		s.db.First(&u, uid)
		if u.Role != models.RoleAdmin || s.db.First(&link, id).Error != nil {
			Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
			return
		}
	}
	s.db.Delete(&link)
	audit.Log(s.db, uid, userFromContext(r).Username, "revoke_share_link", link.Name, ipFromContext(r))
	OK(w, nil)
}

func (s *Server) rotateShareLink(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	id := chi.URLParam(r, "id")
	var link models.ShareLink
	if err := s.db.Where("id = ? AND owner_user_id = ?", id, uid).First(&link).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	link.Token = randomShareToken()
	link.Revoked = false
	s.db.Save(&link)
	audit.Log(s.db, uid, userFromContext(r).Username, "rotate_share_link", link.Name, ipFromContext(r))
	OK(w, shareToResponse(s, r, link))
}

func (s *Server) updateShareLink(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	id := chi.URLParam(r, "id")
	var link models.ShareLink
	if err := s.db.Where("id = ? AND owner_user_id = ?", id, uid).First(&link).Error; err != nil {
		// admin 可改任意
		var u models.User
		s.db.First(&u, uid)
		if u.Role != models.RoleAdmin || s.db.First(&link, id).Error != nil {
			Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
			return
		}
	}
	var req updateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	if req.Slug != nil {
		slug, ok := resolveShareSlug(*req.Slug)
		if !ok {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "validation.shareSlugInvalid"))
			return
		}
		if s.slugTaken(slug, link.ID) {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "validation.shareSlugTaken"))
			return
		}
		link.Slug = slug
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if len([]rune(name)) > 50 {
			name = string([]rune(name)[:50])
		}
		link.Name = name
	}
	if req.ScopeMode != nil || req.TagIDs != nil {
		mode := link.ScopeMode
		if req.ScopeMode != nil {
			mode = strings.TrimSpace(*req.ScopeMode)
			if mode != models.ShareScopeAll && mode != models.ShareScopeTags {
				Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
				return
			}
		}
		var uniq []uint
		if req.TagIDs != nil {
			uniq = dedupeIDs(*req.TagIDs)
		} else {
			_ = json.Unmarshal([]byte(link.TagIDs), &uniq)
			if uniq == nil {
				uniq = []uint{}
			}
		}
		if mode == models.ShareScopeTags {
			if len(uniq) == 0 {
				Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
				return
			}
			var count int64
			s.db.Model(&models.Tag{}).Where("id IN ? AND user_id = ?", uniq, link.OwnerUserID).Count(&count)
			if count != int64(len(uniq)) {
				Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.notFound"))
				return
			}
		} else {
			uniq = []uint{}
		}
		rawIDs, _ := json.Marshal(uniq)
		if rawIDs == nil {
			rawIDs = []byte("[]")
		}
		link.ScopeMode = mode
		link.TagIDs = string(rawIDs)
	}
	s.db.Save(&link)
	audit.Log(s.db, uid, userFromContext(r).Username, "update_share_link", link.Name, ipFromContext(r))
	OK(w, shareToResponse(s, r, link))
}

func dedupeIDs(ids []uint) []uint {
	seen := map[uint]struct{}{}
	out := []uint{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

// ---- 公开端：只读分享（无需认证，仅 GET） ----

type publicShareData struct {
	Name      string       `json:"name"`
	ExpiresAt *time.Time   `json:"expires_at"`
	Tags      []models.Tag `json:"tags"`
	Birthdays []publicBirthday `json:"birthdays"`
}

type publicBirthday struct {
	ID          uint         `json:"id"`
	Name        string       `json:"name"`
	Gender      string       `json:"gender"`
	Color       string       `json:"color"`
	BirthYear   int          `json:"birth_year"`
	BirthMonth  int          `json:"birth_month"`
	BirthDay    int          `json:"birth_day"`
	Age         int          `json:"age"`          // 当前年龄，-1=未知年份
	UpcomingAge int          `json:"upcoming_age"` // 下一个生日后的年龄，-1=未知年份
	DaysUntil   int          `json:"days_until"`
	Tags        []models.Tag `json:"tags"`
}

// loadShareLink 按 token 或自定义 slug 查找：不存在/撤销/过期统一返回 nil（防枚举）
// token 固定 64 位、slug 最长 32 位，二者无交集，不会歧义。
func (s *Server) loadShareLink(key string) *models.ShareLink {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	var link models.ShareLink
	if err := s.db.Where("token = ?", key).First(&link).Error; err != nil {
		if err := s.db.Where("slug = ? AND slug <> ''", key).First(&link).Error; err != nil {
			return nil
		}
	}
	if link.Revoked || link.Expired() {
		return nil
	}
	return &link
}

// shareScopeBirthdays 按分享范围取 owner 的生日
func (s *Server) shareScopeBirthdays(link *models.ShareLink) ([]models.Birthday, []models.Tag) {
	var bds []models.Birthday
	s.db.Where("user_id = ?", link.OwnerUserID).Find(&bds)
	var tags []models.Tag
	s.db.Where("user_id = ?", link.OwnerUserID).Find(&tags)
	if link.ScopeMode == models.ShareScopeAll {
		return bds, tags
	}
	var want []uint
	_ = json.Unmarshal([]byte(link.TagIDs), &want)
	if len(want) == 0 {
		return []models.Birthday{}, []models.Tag{}
	}
	wantSet := map[uint]struct{}{}
	for _, id := range want {
		wantSet[id] = struct{}{}
	}
	// 仅保留仍存在的 tag
	keptTags := tags[:0]
	for _, t := range tags {
		if _, ok := wantSet[t.ID]; ok {
			keptTags = append(keptTags, t)
		}
	}
	tags = keptTags
	if len(tags) == 0 {
		return []models.Birthday{}, []models.Tag{}
	}
	var bts []models.BirthdayTag
	s.db.Where("tag_id IN ?", want).Find(&bts)
	bidSet := map[uint]struct{}{}
	for _, bt := range bts {
		bidSet[bt.BirthdayID] = struct{}{}
	}
	out := bds[:0]
	for _, b := range bds {
		if b.UserID != link.OwnerUserID {
			continue
		}
		if _, ok := bidSet[b.ID]; ok {
			out = append(out, b)
		}
	}
	return out, tags
}

func (s *Server) handlePublicShare(w http.ResponseWriter, r *http.Request) {
	link := s.loadShareLink(chi.URLParam(r, "token"))
	if link == nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	bds, tags := s.shareScopeBirthdays(link)
	tagsByBid := s.birthdaysTagsMap(link.OwnerUserID, bds)
	now := time.Now()
	out := make([]publicBirthday, 0, len(bds))
	for _, b := range bds {
		tgs := tagsByBid[b.ID]
		if tgs == nil {
			tgs = []models.Tag{}
		}
		next := nextBirthday(b.BirthMonth, b.BirthDay, now)
		days := int(next.Sub(now).Hours() / 24)
		if days < 0 {
			days = 0
		}
		out = append(out, publicBirthday{
			ID: b.ID, Name: b.Name, Gender: b.Gender, Color: models.GenderColor(b.Gender),
			BirthYear: b.BirthYear, BirthMonth: b.BirthMonth, BirthDay: b.BirthDay,
			Age: b.Age(now), UpcomingAge: b.UpcomingAge(now),
			DaysUntil: days, Tags: tgs,
		})
	}
	OK(w, publicShareData{Name: link.Name, ExpiresAt: link.ExpiresAt, Tags: tags, Birthdays: out})
}

func (s *Server) handlePublicShareCalendar(w http.ResponseWriter, r *http.Request) {
	link := s.loadShareLink(chi.URLParam(r, "token"))
	if link == nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	view := r.URL.Query().Get("view")
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if year == 0 {
		year = time.Now().Year()
	}
	bds, _ := s.shareScopeBirthdays(link)
	tagsByBid := s.birthdaysTagsMap(link.OwnerUserID, bds)
	now := time.Now()
	if view == "year" {
		months := make([]yearMonthData, 12)
		for i := range months {
			months[i] = yearMonthData{Month: i + 1, Genders: map[string]int{
				models.GenderMale: 0, models.GenderFemale: 0, models.GenderNone: 0,
			}}
		}
		for _, b := range bds {
			idx := b.BirthMonth - 1
			if idx < 0 || idx > 11 {
				continue
			}
			months[idx].Count++
			months[idx].Genders[b.Gender]++
		}
		OK(w, yearViewData{Year: year, Months: months})
		return
	}
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	if month < 1 || month > 12 {
		month = int(now.Month())
	}
	cells := make([]birthdayInCell, 0)
	for _, b := range bds {
		if b.BirthMonth != month {
			continue
		}
		tgs := tagsByBid[b.ID]
		if tgs == nil {
			tgs = []models.Tag{}
		}
		cells = append(cells, birthdayInCell{
			ID: b.ID, Name: b.Name, Gender: b.Gender, Color: models.GenderColor(b.Gender),
			Day: b.BirthDay, Age: b.Age(now), UpcomingAge: b.UpcomingAge(now), Tags: tgs,
		})
	}
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	daysInMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local).Day()
	OK(w, monthViewData{Year: year, Month: month, FirstWeekday: int(first.Weekday()), DaysInMonth: daysInMonth, Birthdays: cells})
}
