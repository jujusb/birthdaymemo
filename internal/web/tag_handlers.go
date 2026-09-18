package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mcbill1/birthdaymemo/internal/config"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/models"
)

// tagRequest 标签请求
type tagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// tagNameLimit 取生效的标签名称最大字符数（老配置文件缺字段时回退默认值）
func tagNameLimit(cfg *config.Config) int {
	if cfg == nil || cfg.TagNameMaxLength < 1 {
		return config.DefaultTagNameMaxLength
	}
	if cfg.TagNameMaxLength > config.MaxTagNameMaxLength {
		return config.MaxTagNameMaxLength
	}
	return cfg.TagNameMaxLength
}

// validateTagName 校验标签名称长度，失败返回本地化错误信息（空串=通过）
func validateTagName(cfg *config.Config, name string, lang string) string {
	max := tagNameLimit(cfg)
	if len([]rune(name)) < 1 || len([]rune(name)) > max {
		return strings.ReplaceAll(i18n.T(lang, "tag.nameLimit"), "{max}", strconv.Itoa(max))
	}
	return ""
}

func (s *Server) listTags(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	var tags []models.Tag
	s.db.Where("user_id = ?", uid).Order("id asc").Find(&tags)
	OK(w, tags)
}

func (s *Server) createTag(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	var req tagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	name := strings.TrimSpace(req.Name)
	if errMsg := validateTagName(s.cfg, name, s.cfg.Language); errMsg != "" {
		Fail(w, CodeBadRequest, errMsg)
		return
	}
	color := strings.TrimSpace(req.Color)
	if color == "" {
		color = models.PresetTagColors[0]
	}
	tag := models.Tag{UserID: uid, Name: name, Color: color}
	if err := s.db.Create(&tag).Error; err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	OK(w, tag)
}

func (s *Server) updateTag(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	id := chi.URLParam(r, "id")
	var tag models.Tag
	if err := s.db.Where("id = ? AND user_id = ?", id, uid).First(&tag).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	var req tagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	name := strings.TrimSpace(req.Name)
	if errMsg := validateTagName(s.cfg, name, s.cfg.Language); errMsg != "" {
		Fail(w, CodeBadRequest, errMsg)
		return
	}
	tag.Name = name
	if c := strings.TrimSpace(req.Color); c != "" {
		tag.Color = c
	}
	s.db.Save(&tag)
	OK(w, tag)
}

func (s *Server) deleteTag(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	id := chi.URLParam(r, "id")
	var tag models.Tag
	if err := s.db.Where("id = ? AND user_id = ?", id, uid).First(&tag).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	// 级联删除 BirthdayTag 关联（多对多）
	s.db.Where("tag_id = ?", tag.ID).Delete(&models.BirthdayTag{})
	// 级联删除该标签的委托授权
	s.db.Where("tag_id = ?", tag.ID).Delete(&models.TagGrant{})
	s.db.Delete(&tag)
	OK(w, nil)
}
