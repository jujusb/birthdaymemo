package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mcbill1/birthdaymemo/internal/audit"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/models"
	"gorm.io/gorm"
)

// birthdayRequest 生日请求
type birthdayRequest struct {
	Name       string `json:"name"`
	Gender     string `json:"gender"`
	BirthYear  int    `json:"birth_year"`
	BirthMonth int    `json:"birth_month"`
	BirthDay   int    `json:"birth_day"`
	TagIDs     []uint `json:"tag_ids"` // 多选标签 ID 列表
}

// validGender 校验性别值
func validGender(g string) string {
	switch g {
	case models.GenderMale, models.GenderFemale, models.GenderNone:
		return g
	}
	return models.GenderNone
}

// validateBirthDate 校验出生日期合法性，返回错误信息
func validateBirthDate(year, month, day int, lang string) string {
	if month < 1 || month > 12 {
		return i18n.T(lang, "validation.invalidMonth")
	}
	if day < 1 || day > 31 {
		return i18n.T(lang, "validation.invalidDay")
	}
	// 校验该月天数
	t := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	if year == 0 {
		// 年份未知，按平年校验
		t = time.Date(2001, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	}
	if day > t.Day() {
		return i18n.T(lang, "validation.invalidDayForMonth")
	}
	if year != 0 && (year < 1900 || year > time.Now().Year()) {
		return i18n.T(lang, "validation.invalidYear")
	}
	return ""
}

// birthdayWithTag 生日附带多标签响应
type birthdayWithTag struct {
	models.Birthday
	Tags          []models.Tag `json:"tags"`
	Shared        bool         `json:"shared"`
	CanEdit       bool         `json:"can_edit"`
	OwnerUsername string       `json:"owner_username,omitempty"`
}

func (s *Server) listBirthdays(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	tagIDsStr := r.URL.Query().Get("tag_ids")
	var filterIDs []uint
	if tagIDsStr != "" {
		parts := strings.Split(tagIDsStr, ",")
		for _, p := range parts {
			if id, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && id > 0 {
				filterIDs = append(filterIDs, uint(id))
			}
		}
	}
	bds, err := visibleBirthdays(s.db, uid)
	if err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	// 多标签筛选：AND 逻辑 - 仅返回同时拥有所有所选标签的生日
	if len(filterIDs) > 0 {
		need := make(map[uint]struct{}, len(filterIDs))
		for _, id := range filterIDs {
			need[id] = struct{}{}
		}
		tagsByBid := s.birthdaysTagsMap(uid, bds)
		filtered := bds[:0]
		for _, b := range bds {
			have := make(map[uint]struct{}, len(tagsByBid[b.ID]))
			for _, t := range tagsByBid[b.ID] {
				have[t.ID] = struct{}{}
			}
			ok := true
			for id := range need {
				if _, has := have[id]; !has {
					ok = false
					break
				}
			}
			if ok {
				filtered = append(filtered, b)
			}
		}
		bds = filtered
	}
	// 排序：按月日
	sortBirthdays(bds)

	// 附带多标签信息
	tagsByBid := s.birthdaysTagsMap(uid, bds)
	ownerNames := s.birthdayOwnerNames(bds)
	out := make([]birthdayWithTag, 0, len(bds))
	for _, b := range bds {
		tags := tagsByBid[b.ID]
		if tags == nil {
			tags = []models.Tag{}
		}
		shared := b.UserID != uid
		out = append(out, birthdayWithTag{
			Birthday:      b,
			Tags:          tags,
			Shared:        shared,
			CanEdit:       !shared || canEditBirthday(s.db, uid, b),
			OwnerUsername: ownerNames[b.UserID],
		})
	}
	OK(w, out)
}

// sortBirthdays 按月日排序
func sortBirthdays(bds []models.Birthday) {
	for i := 1; i < len(bds); i++ {
		for j := i; j > 0; j-- {
			a, c := bds[j-1], bds[j]
			if a.BirthMonth > c.BirthMonth || (a.BirthMonth == c.BirthMonth && a.BirthDay > c.BirthDay) {
				bds[j-1], bds[j] = bds[j], bds[j-1]
			} else {
				break
			}
		}
	}
}

// birthdayOwnerNames 批量查询 owner 用户名
func (s *Server) birthdayOwnerNames(bds []models.Birthday) map[uint]string {
	out := map[uint]string{}
	if len(bds) == 0 {
		return out
	}
	set := map[uint]struct{}{}
	for _, b := range bds {
		set[b.UserID] = struct{}{}
	}
	ids := make([]uint, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	var users []models.User
	s.db.Where("id IN ?", ids).Find(&users)
	for _, u := range users {
		out[u.ID] = u.Username
	}
	return out
}

// birthdaysTagsMap 返回一组生日的标签 map[birthdayID][]Tag
// 注意：可见性已由调用方（visibleBirthdays/canView）保证，此处不按 user 过滤，
// 以便共享生日能带出 owner 的标签信息
func (s *Server) birthdaysTagsMap(uid uint, bds []models.Birthday) map[uint][]models.Tag {
	out := make(map[uint][]models.Tag, len(bds))
	if len(bds) == 0 {
		return out
	}
	bids := make([]uint, 0, len(bds))
	for _, b := range bds {
		bids = append(bids, b.ID)
	}
	// 查询关联表
	var bts []models.BirthdayTag
	s.db.Where("birthday_id IN ?", bids).Find(&bts)
	if len(bts) == 0 {
		return out
	}
	// 收集所有 tagID 并查 Tag
	tagIDSet := make(map[uint]struct{})
	for _, bt := range bts {
		tagIDSet[bt.TagID] = struct{}{}
	}
	tagIDs := make([]uint, 0, len(tagIDSet))
	for id := range tagIDSet {
		tagIDs = append(tagIDs, id)
	}
	var tags []models.Tag
	s.db.Where("id IN ?", tagIDs).Find(&tags)
	tagMap := make(map[uint]models.Tag, len(tags))
	for _, t := range tags {
		tagMap[t.ID] = t
	}
	for _, bt := range bts {
		if t, ok := tagMap[bt.TagID]; ok {
			out[bt.BirthdayID] = append(out[bt.BirthdayID], t)
		}
	}
	return out
}

// replaceBirthdayTags 替换生日的所有标签关联（先删后插）
func (s *Server) replaceBirthdayTags(birthdayID uint, tagIDs []uint) error {
	// 删除旧关联
	if err := s.db.Where("birthday_id = ?", birthdayID).Delete(&models.BirthdayTag{}).Error; err != nil {
		return err
	}
	// 去重
	seen := make(map[uint]struct{}, len(tagIDs))
	for _, tid := range tagIDs {
		if tid == 0 {
			continue
		}
		if _, ok := seen[tid]; ok {
			continue
		}
		seen[tid] = struct{}{}
		if err := s.db.Create(&models.BirthdayTag{BirthdayID: birthdayID, TagID: tid}).Error; err != nil {
			return err
		}
	}
	return nil
}

// validateTagIDs 校验所有 tagID 都属于该用户，返回有效列表
func (s *Server) validateTagIDs(uid uint, tagIDs []uint) ([]uint, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}
	var count int64
	s.db.Model(&models.Tag{}).Where("id IN ? AND user_id = ?", tagIDs, uid).Count(&count)
	if count != int64(len(tagIDs)) {
		return nil, gorm.ErrRecordNotFound
	}
	return tagIDs, nil
}

func (s *Server) createBirthday(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	var req birthdayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "birthday.name"))
		return
	}
	if msg := validateBirthDate(req.BirthYear, req.BirthMonth, req.BirthDay, s.cfg.Language); msg != "" {
		Fail(w, CodeBadRequest, msg)
		return
	}
	// 解析实际 owner：含被授予 edit 的他人 tag → 创建进对方日历（委托创建）
	ownerID, err := resolveCreateOwner(s.db, uid, req.TagIDs)
	if err != nil {
		Fail(w, CodeForbidden, i18n.T(s.cfg.Language, "error.forbidden"))
		return
	}
	// 共享创建必须携带至少一个 tag（否则新记录对 grantor 不可见）
	if ownerID != uid && len(req.TagIDs) == 0 {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	var tagIDs []uint
	if ownerID == uid {
		tagIDs, err = s.validateTagIDs(uid, req.TagIDs)
	} else {
		// 共享创建：resolver 已校验每个 tag 合法（自有或被授予 edit），此处仅去重
		seen := map[uint]struct{}{}
		for _, tid := range req.TagIDs {
			if tid == 0 {
				continue
			}
			if _, ok := seen[tid]; !ok {
				seen[tid] = struct{}{}
				tagIDs = append(tagIDs, tid)
			}
		}
	}
	if err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	bd := models.Birthday{
		UserID:     ownerID,
		Name:       name,
		Gender:     validGender(req.Gender),
		BirthYear:  req.BirthYear,
		BirthMonth: req.BirthMonth,
		BirthDay:   req.BirthDay,
	}
	if err := s.db.Create(&bd).Error; err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	if err := s.replaceBirthdayTags(bd.ID, tagIDs); err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	audit.Log(s.db, uid, userFromContext(r).Username, "create_birthday", name, ipFromContext(r))
	OK(w, bd)
}

func (s *Server) updateBirthday(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	id := chi.URLParam(r, "id")
	var bd models.Birthday
	if err := s.db.Where("id = ?", id).First(&bd).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	if !canViewBirthday(s.db, uid, bd) {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	if !canEditBirthday(s.db, uid, bd) {
		Fail(w, CodeForbidden, i18n.T(s.cfg.Language, "error.forbidden"))
		return
	}
	var req birthdayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "birthday.name"))
		return
	}
	if msg := validateBirthDate(req.BirthYear, req.BirthMonth, req.BirthDay, s.cfg.Language); msg != "" {
		Fail(w, CodeBadRequest, msg)
		return
	}
	var tagIDs []uint
	var u models.User
	s.db.First(&u, uid)
	isOwnerOrAdmin := uid == bd.UserID || u.Role == models.RoleAdmin
	if isOwnerOrAdmin {
		// 本人：tag 须属于生日 owner；admin 跨 owner 编辑时同样约束到 owner 域
		var count int64
		scopeUID := bd.UserID
		if u.Role == models.RoleAdmin && uid != bd.UserID {
			// admin：允许使用 owner 的 tag 或自己的 tag
			var tags []models.Tag
			s.db.Where("id IN ?", req.TagIDs).Find(&tags)
			if len(tags) != len(req.TagIDs) {
				Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.notFound"))
				return
			}
			for _, t := range tags {
				if t.UserID != scopeUID && t.UserID != uid {
					Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.notFound"))
					return
				}
			}
			tagIDs = req.TagIDs
		} else {
			s.db.Model(&models.Tag{}).Where("id IN ? AND user_id = ?", req.TagIDs, scopeUID).Count(&count)
			if len(req.TagIDs) > 0 && count != int64(len(req.TagIDs)) {
				Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.notFound"))
				return
			}
			tagIDs = req.TagIDs
		}
	} else {
		// 被委托人：结果集必须保留 ≥1 授予-edit tag，且仅含自有/授予 tag
		if err := validateUpdateTagSet(s.db, uid, bd, req.TagIDs); err != nil {
			Fail(w, CodeForbidden, i18n.T(s.cfg.Language, "error.forbidden"))
			return
		}
		tagIDs = req.TagIDs
	}
	bd.Name = name
	bd.Gender = validGender(req.Gender)
	bd.BirthYear = req.BirthYear
	bd.BirthMonth = req.BirthMonth
	bd.BirthDay = req.BirthDay
	if err := s.db.Save(&bd).Error; err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	if err := s.replaceBirthdayTags(bd.ID, tagIDs); err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	OK(w, bd)
}

func (s *Server) deleteBirthday(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	id := chi.URLParam(r, "id")
	var bd models.Birthday
	if err := s.db.Where("id = ?", id).First(&bd).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	// 删除仅限本人/admin：被委托人（即使可编辑）也不可删除
	var u models.User
	s.db.First(&u, uid)
	if uid != bd.UserID && u.Role != models.RoleAdmin {
		if canViewBirthday(s.db, uid, bd) {
			Fail(w, CodeForbidden, i18n.T(s.cfg.Language, "error.forbidden"))
		} else {
			Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		}
		return
	}
	// 删除关联（BirthdayTag 不级联，手动删）
	s.db.Where("birthday_id = ?", bd.ID).Delete(&models.BirthdayTag{})
	s.db.Delete(&bd)
	audit.Log(s.db, uid, userFromContext(r).Username, "delete_birthday", bd.Name, ipFromContext(r))
	OK(w, nil)
}
