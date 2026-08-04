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
	Tags []models.Tag `json:"tags"`
}

func (s *Server) listBirthdays(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	tagIDsStr := r.URL.Query().Get("tag_ids")
	q := s.db.Where("user_id = ?", uid)
	// 多标签筛选：AND 逻辑 - 仅返回同时拥有所有所选标签的生日
	if tagIDsStr != "" {
		parts := strings.Split(tagIDsStr, ",")
		var ids []uint
		for _, p := range parts {
			if id, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && id > 0 {
				ids = append(ids, uint(id))
			}
		}
		if len(ids) > 0 {
			q = q.Where("id IN (SELECT birthday_id FROM birthday_tags WHERE tag_id IN ? GROUP BY birthday_id HAVING COUNT(DISTINCT tag_id) = ?)", ids, len(ids))
		}
	}
	var bds []models.Birthday
	q.Order("birth_month asc, birth_day asc").Find(&bds)

	// 附带多标签信息
	tagsByBid := s.birthdaysTagsMap(uid, bds)
	out := make([]birthdayWithTag, 0, len(bds))
	for _, b := range bds {
		tags := tagsByBid[b.ID]
		if tags == nil {
			tags = []models.Tag{}
		}
		out = append(out, birthdayWithTag{Birthday: b, Tags: tags})
	}
	OK(w, out)
}

// birthdaysTagsMap 返回一组生日的标签 map[birthdayID][]Tag
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
	s.db.Where("id IN ? AND user_id = ?", tagIDs, uid).Find(&tags)
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
	tagIDs, err := s.validateTagIDs(uid, req.TagIDs)
	if err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	bd := models.Birthday{
		UserID:     uid,
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
	if err := s.db.Where("id = ? AND user_id = ?", id, uid).First(&bd).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
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
	tagIDs, err := s.validateTagIDs(uid, req.TagIDs)
	if err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.notFound"))
		return
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
	if err := s.db.Where("id = ? AND user_id = ?", id, uid).First(&bd).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	// 删除关联（BirthdayTag 不级联，手动删）
	s.db.Where("birthday_id = ?", bd.ID).Delete(&models.BirthdayTag{})
	s.db.Delete(&bd)
	audit.Log(s.db, uid, userFromContext(r).Username, "delete_birthday", bd.Name, ipFromContext(r))
	OK(w, nil)
}
