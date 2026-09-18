package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mcbill1/birthdaymemo/internal/audit"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/models"
)

type createGrantRequest struct {
	GranteeUsername string `json:"grantee_username"`
	TagID           uint   `json:"tag_id"`
	Permission      string `json:"permission"` // view | edit
}

type grantResponse struct {
	ID              uint   `json:"id"`
	TagID           uint   `json:"tag_id"`
	TagName         string `json:"tag_name"`
	OwnerUsername   string `json:"owner_username"`
	GranteeUsername string `json:"grantee_username"`
	Permission      string `json:"permission"`
}

func (s *Server) grantToResponse(g models.TagGrant) grantResponse {
	var tag models.Tag
	s.db.First(&tag, g.TagID)
	var owner, grantee models.User
	s.db.First(&owner, g.OwnerUserID)
	s.db.First(&grantee, g.GranteeUserID)
	return grantResponse{
		ID: g.ID, TagID: g.TagID, TagName: tag.Name,
		OwnerUsername: owner.Username, GranteeUsername: grantee.Username,
		Permission: g.Permission,
	}
}

func (s *Server) createGrant(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	var req createGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	perm := strings.TrimSpace(req.Permission)
	if perm == "" {
		perm = models.GrantEdit
	}
	if perm != models.GrantView && perm != models.GrantEdit {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	// tag 必须属于调用者（admin 可为他人代建：tag 属主即 owner）
	var tag models.Tag
	if err := s.db.First(&tag, req.TagID).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	var me models.User
	s.db.First(&me, uid)
	ownerID := tag.UserID
	if tag.UserID != uid && me.Role != models.RoleAdmin {
		Fail(w, CodeForbidden, i18n.T(s.cfg.Language, "error.forbidden"))
		return
	}
	username := strings.TrimSpace(req.GranteeUsername)
	if username == "" {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	var grantee models.User
	if err := s.db.Where("username = ?", username).First(&grantee).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	if grantee.ID == ownerID {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	if grantee.Role == models.RoleGuest {
		// 登录型 guest 只降级为 view
		perm = models.GrantView
	}
	var existing models.TagGrant
	if err := s.db.Where("grantee_user_id = ? AND tag_id = ?", grantee.ID, tag.ID).First(&existing).Error; err == nil {
		existing.Permission = perm
		s.db.Save(&existing)
		audit.Log(s.db, uid, userFromContext(r).Username, "update_grant", username+":"+tag.Name, ipFromContext(r))
		OK(w, s.grantToResponse(existing))
		return
	}
	g := models.TagGrant{OwnerUserID: ownerID, GranteeUserID: grantee.ID, TagID: tag.ID, Permission: perm}
	if err := s.db.Create(&g).Error; err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	audit.Log(s.db, uid, userFromContext(r).Username, "create_grant", username+":"+tag.Name, ipFromContext(r))
	OK(w, s.grantToResponse(g))
}

func (s *Server) listGrants(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	kind := r.URL.Query().Get("type") // owned | received | all(默认)
	var grants []models.TagGrant
	switch kind {
	case "received":
		s.db.Where("grantee_user_id = ?", uid).Order("id desc").Find(&grants)
	case "owned":
		s.db.Where("owner_user_id = ?", uid).Order("id desc").Find(&grants)
	default:
		s.db.Where("owner_user_id = ? OR grantee_user_id = ?", uid, uid).Order("id desc").Find(&grants)
	}
	out := make([]grantResponse, 0, len(grants))
	for _, g := range grants {
		out = append(out, s.grantToResponse(g))
	}
	OK(w, out)
}

func (s *Server) updateGrant(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	id := chi.URLParam(r, "id")
	var g models.TagGrant
	if err := s.db.First(&g, id).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	var me models.User
	s.db.First(&me, uid)
	if g.OwnerUserID != uid && me.Role != models.RoleAdmin {
		Fail(w, CodeForbidden, i18n.T(s.cfg.Language, "error.forbidden"))
		return
	}
	var req struct {
		Permission string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	perm := strings.TrimSpace(req.Permission)
	if perm != models.GrantView && perm != models.GrantEdit {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	g.Permission = perm
	s.db.Save(&g)
	audit.Log(s.db, uid, userFromContext(r).Username, "update_grant", perm, ipFromContext(r))
	OK(w, s.grantToResponse(g))
}

func (s *Server) deleteGrant(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	id := chi.URLParam(r, "id")
	var g models.TagGrant
	if err := s.db.First(&g, id).Error; err != nil {
		Fail(w, CodeNotFound, i18n.T(s.cfg.Language, "error.notFound"))
		return
	}
	var me models.User
	s.db.First(&me, uid)
	// owner、grantee 本人（主动退出）、admin 可删
	if g.OwnerUserID != uid && g.GranteeUserID != uid && me.Role != models.RoleAdmin {
		Fail(w, CodeForbidden, i18n.T(s.cfg.Language, "error.forbidden"))
		return
	}
	s.db.Delete(&g)
	audit.Log(s.db, uid, userFromContext(r).Username, "delete_grant", "", ipFromContext(r))
	OK(w, nil)
}
