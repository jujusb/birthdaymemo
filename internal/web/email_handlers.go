package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mcbill1/birthdaymemo/internal/audit"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/logger"
	"github.com/mcbill1/birthdaymemo/internal/models"
)

// 邮箱确认相关常量
const (
	emailConfirmTTL      = 24 * time.Hour // 确认链接有效期
	emailConfirmCooldown = 3              // 每天最多发送次数
	emailTokenBytes      = 32             // token 字节数（64 hex 字符）
)

// emailRegex 简易邮箱格式校验
var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// emailConfirmRequest 邮箱确认请求
type emailConfirmRequest struct {
	Email string `json:"email"`
}

// emailStatusResponse 邮箱状态响应
type emailStatusResponse struct {
	Email        string `json:"email"`         // 已确认邮箱
	PendingEmail string `json:"pending_email"` // 待确认邮箱
	Status       string `json:"status"`        // verified | pending | none
	CooldownLeft int    `json:"cooldown_left"` // 今日剩余可发送次数
}

// requestEmailConfirmation 用户请求确认邮箱：校验、限流、生成 token、发送确认邮件
// POST /api/settings/email/confirm
func (s *Server) requestEmailConfirmation(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r)
	if u == nil {
		FailStatus(w, http.StatusUnauthorized, CodeUnauthorized, i18n.T(s.cfg.Language, "error.unauthorized"))
		return
	}
	var req emailConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	email := strings.TrimSpace(req.Email)
	if email == "" {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "email.address"))
		return
	}
	if !emailRegex.MatchString(email) {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "validation.invalidName"))
		return
	}
	// 已是该用户的已确认邮箱，无需重复确认
	if email == u.Email {
		OKMsg(w, i18n.T(s.cfg.Language, "email.verified"), nil)
		return
	}

	// 冷却：今日已发送次数
	todayStart := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
	var sentToday int64
	s.db.Model(&models.EmailConfirmation{}).
		Where("user_id = ? AND created_at >= ?", u.ID, todayStart).
		Count(&sentToday)
	cooldownLeft := emailConfirmCooldown - int(sentToday)
	if cooldownLeft <= 0 {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "email.cooldown"))
		return
	}

	// 生成 token 与确认记录
	token, err := randomHexToken(emailTokenBytes)
	if err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	now := time.Now()
	ec := models.EmailConfirmation{
		UserID:    u.ID,
		Email:     email,
		Token:     token,
		ExpiresAt: now.Add(emailConfirmTTL),
		CreatedAt: now,
	}
	if err := s.db.Create(&ec).Error; err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}

	// 记录待确认邮箱到用户表
	s.db.Model(&models.User{}).Where("id = ?", u.ID).Update("pending_email", email)

	// 发送确认邮件
	confirmURL := buildConfirmURL(r, s.cfg.ExternalURL) + "/api/email/confirm?token=" + token
	if err := s.sendConfirmationEmail(u, email, confirmURL); err != nil {
		logger.Error("email confirm: send to %s failed: %v", email, err)
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "validation.smtpNotConfigured"))
		return
	}
	logger.Info("email confirm sent: user='%s' to=%s ip=%s", u.Username, email, ipFromContext(r))
	audit.Log(s.db, u.ID, u.Username, "request_email_confirm", email, ipFromContext(r))
	OKMsg(w, i18n.T(s.cfg.Language, "email.confirmSent"), emailStatusResponse{
		Email:        u.Email,
		PendingEmail: email,
		Status:       "pending",
		CooldownLeft: cooldownLeft - 1,
	})
}

// confirmEmail 用户点击确认链接后调用（公开接口）
// GET /api/email/confirm?token=xxx
func (s *Server) confirmEmail(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		renderConfirmHTML(w, false, s.cfg.Language)
		return
	}
	var ec models.EmailConfirmation
	if err := s.db.Where("token = ?", token).First(&ec).Error; err != nil {
		renderConfirmHTML(w, false, s.cfg.Language)
		return
	}
	if ec.ConfirmedAt != nil || time.Now().After(ec.ExpiresAt) {
		renderConfirmHTML(w, false, s.cfg.Language)
		return
	}
	// 标记确认
	now := time.Now()
	ec.ConfirmedAt = &now
	s.db.Save(&ec)
	// 更新用户：邮箱正式生效，清空待确认邮箱
	s.db.Model(&models.User{}).Where("id = ?", ec.UserID).Updates(map[string]any{
		"email":         ec.Email,
		"pending_email": "",
	})
	// 查询用户名用于审计日志
	var confirmUser models.User
	s.db.First(&confirmUser, ec.UserID)
	logger.Info("email confirmed: user='%s' email=%s", confirmUser.Username, ec.Email)
	audit.Log(s.db, ec.UserID, confirmUser.Username, "confirm_email", ec.Email, ipFromContext(r))
	renderConfirmHTML(w, true, s.cfg.Language)
}

// getEmailStatus 返回当前用户的邮箱确认状态
// GET /api/settings/email/status
func (s *Server) getEmailStatus(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r)
	if u == nil {
		FailStatus(w, http.StatusUnauthorized, CodeUnauthorized, i18n.T(s.cfg.Language, "error.unauthorized"))
		return
	}
	status := "none"
	if u.Email != "" {
		status = "verified"
	} else if u.PendingEmail != "" {
		status = "pending"
	}
	// 计算今日剩余次数
	todayStart := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
	var sentToday int64
	s.db.Model(&models.EmailConfirmation{}).
		Where("user_id = ? AND created_at >= ?", u.ID, todayStart).
		Count(&sentToday)
	cooldownLeft := emailConfirmCooldown - int(sentToday)
	if cooldownLeft < 0 {
		cooldownLeft = 0
	}
	OK(w, emailStatusResponse{
		Email:        u.Email,
		PendingEmail: u.PendingEmail,
		Status:       status,
		CooldownLeft: cooldownLeft,
	})
}

// sendConfirmationEmail 发送含确认链接的测试邮件
func (s *Server) sendConfirmationEmail(u *models.User, to, confirmURL string) error {
	var smtp models.SmtpSetting
	if err := s.db.First(&smtp, 1).Error; err != nil {
		return fmt.Errorf("load smtp: %w", err)
	}
	if smtp.Host == "" || smtp.Username == "" || smtp.Password == "" || smtp.From == "" {
		return fmt.Errorf("%s", i18n.T(u.Language, "validation.smtpNotConfigured"))
	}
	lang := u.Language
	if lang == "" {
		lang = s.cfg.Language
	}
	// 渲染变量 {username}
	vars := map[string]string{"username": u.Username}
	renderStatic := func(tpl string) string {
		out := tpl
		for k, v := range vars {
			out = strings.ReplaceAll(out, "{"+k+"}", v)
		}
		return out
	}
	subject := renderStatic(i18n.T(lang, "email.testSubject"))
	intro := renderStatic(i18n.T(lang, "email.confirmIntro"))
	btn := renderStatic(i18n.T(lang, "email.confirmButton"))
	ignore := renderStatic(i18n.T(lang, "email.confirmIgnore"))
	body := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n", intro, confirmURL, ignore, btn)
	return s.smtpSender.Send(smtp, to, subject, body)
}

// buildConfirmURL 根据请求构造外部可访问的基础 URL（协议+主机）。
// 优先使用配置的 ExternalURL，未配置时回退到请求 Host。
func buildConfirmURL(r *http.Request, externalURL string) string {
	if externalURL != "" {
		return externalURL
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	if host == "" {
		host = r.Header.Get("X-Forwarded-Host")
	}
	return scheme + "://" + host
}

// randomHexToken 生成安全的随机 hex token
func randomHexToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// renderConfirmHTML 返回简单的确认结果 HTML 页面
func renderConfirmHTML(w http.ResponseWriter, success bool, lang string) {
	var msg, cls string
	if success {
		msg = i18n.T(lang, "email.confirmSuccess")
		cls = "ok"
	} else {
		msg = i18n.T(lang, "email.confirmInvalid")
		cls = "err"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>BirthDayMemo</title>
<style>
  body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f5f6f8;color:#222;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}
  .card{background:#fff;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,.08);padding:40px;max-width:420px;text-align:center}
  .icon{font-size:48px;margin-bottom:12px}
  .msg{font-size:18px;font-weight:600}
  .ok{color:#16a34a}
  .err{color:#dc2626}
</style>
</head>
<body>
  <div class="card">
    <div class="icon">%s</div>
    <div class="msg %s">%s</div>
  </div>
</body>
</html>`, lang, map[bool]string{true: "✅", false: "⚠️"}[success], cls, msg)))
}
