package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mcbill1/birthdaymemo/internal/audit"
	"github.com/mcbill1/birthdaymemo/internal/auth"
	"github.com/mcbill1/birthdaymemo/internal/config"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/logger"
	"github.com/mcbill1/birthdaymemo/internal/models"
	"github.com/mcbill1/birthdaymemo/internal/security"
	"gorm.io/gorm"
)

// loginRequest 登录请求
type loginRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

// loginData 登录响应数据
type loginData struct {
	User               *models.User `json:"user"`
	MustChangePassword bool         `json:"must_change_password"`
	NeedsCaptcha       bool         `json:"needs_captcha"`
	Locked             bool         `json:"locked"`
	LockedUntil        *time.Time   `json:"locked_until,omitempty"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	ip := ClientIP(r)
	lang := s.cfg.Language
	username := strings.TrimSpace(req.Username)

	// loginFail 统一失败响应：携带 needs_captcha 标志供前端决定是否显示验证码
	loginFail := func(code int, msg string) {
		FailWithData(w, code, msg, map[string]any{"needs_captcha": security.RequireCaptcha(s.db, ip)})
	}

	// 1. IP 锁定检查（优先）
	if locked, until := security.IsLocked(s.db, ip); locked {
		logger.Info("login blocked: user='%s' ip=%s locked until %s", username, ip, until.Format("2006-01-02 15:04:05"))
		Fail(w, CodeForbidden, fmt.Sprintf(i18n.T(lang, "login.locked"), until.Format("2006-01-02 15:04")))
		return
	}

	// 2. 验证码检查（仅在需要时）
	needsCaptcha := security.RequireCaptcha(s.db, ip)
	if needsCaptcha {
		// 2a. 优先校验是否填写验证码
		if req.CaptchaID == "" || req.CaptchaCode == "" {
			logger.Info("login failed: captcha empty user='%s' ip=%s", username, ip)
			loginFail(CodeBadRequest, i18n.T(lang, "login.wrongCaptcha"))
			return
		}
		// 2b. 校验验证码是否正确
		if !auth.VerifyCaptcha(req.CaptchaID, req.CaptchaCode) {
			logger.Info("login failed: wrong captcha user='%s' ip=%s", username, ip)
			loginFail(CodeBadRequest, i18n.T(lang, "login.wrongCaptcha"))
			return
		}
	}

	// 3. 校验用户名和密码是否填写
	if username == "" {
		logger.Info("login failed: username empty ip=%s", ip)
		loginFail(CodeBadRequest, i18n.T(lang, "common.required"))
		return
	}
	if req.Password == "" {
		logger.Info("login failed: password empty user='%s' ip=%s", username, ip)
		loginFail(CodeBadRequest, i18n.T(lang, "common.required"))
		return
	}

	// 4. 校验登录信息（凭证）
	var u models.User
	if err := s.db.Where("username = ?", username).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Info("login failed: user not found user='%s' ip=%s", username, ip)
			locked, until := security.RecordFailure(s.db, ip)
			if locked {
				logger.Info("login locked: ip=%s until %s", ip, until.Format("2006-01-02 15:04:05"))
				Fail(w, CodeForbidden, fmt.Sprintf(i18n.T(lang, "login.locked"), until.Format("2006-01-02 15:04")))
				return
			}
			loginFail(CodeBadRequest, i18n.T(lang, "login.invalid"))
			return
		}
		Fail(w, CodeInternal, i18n.T(lang, "error.internal"))
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		logger.Info("login failed: wrong password user='%s' ip=%s", u.Username, ip)
		locked, until := security.RecordFailure(s.db, ip)
		if locked {
			logger.Info("login locked: ip=%s until %s", ip, until.Format("2006-01-02 15:04:05"))
			Fail(w, CodeForbidden, fmt.Sprintf(i18n.T(lang, "login.locked"), until.Format("2006-01-02 15:04")))
			return
		}
		loginFail(CodeBadRequest, i18n.T(lang, "login.invalid"))
		return
	}

	// 5. 登录成功
	security.RecordSuccess(s.db, ip)
	token := auth.CreateSession(&u)
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   IsHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
	logger.Info("login success: user='%s' ip=%s", u.Username, ip)
	audit.Log(s.db, u.ID, u.Username, "login", "login success", ip)
	OK(w, loginData{User: &u, MustChangePassword: u.MustChangePassword, NeedsCaptcha: false})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		auth.DestroySession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   IsHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	u := userFromContext(r)
	if u != nil {
		logger.Info("logout: user='%s' ip=%s", u.Username, ipFromContext(r))
		audit.Log(s.db, u.ID, u.Username, "logout", "logout", ipFromContext(r))
	}
	OK(w, nil)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r)
	OK(w, u)
}

// captchaResponse 验证码响应
type captchaResponse struct {
	ID    string `json:"id"`
	Image string `json:"image"` // base64 PNG
}

func (s *Server) handleCaptcha(w http.ResponseWriter, r *http.Request) {
	id, _, img, err := auth.CreateCaptcha()
	if err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	OK(w, captchaResponse{ID: id, Image: "data:image/png;base64," + img})
}

// changePasswordRequest 修改密码请求
type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	u := userFromContext(r)
	if u == nil {
		FailStatus(w, http.StatusUnauthorized, CodeUnauthorized, i18n.T(s.cfg.Language, "error.unauthorized"))
		return
	}

	// 首次登录修改初始密码时不校验旧密码（MustChangePassword=true）
	if !u.MustChangePassword {
		if !auth.CheckPassword(u.PasswordHash, req.OldPassword) {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "login.invalid"))
			return
		}
	}
	if err := config.ValidatePassword(req.NewPassword, s.cfg.Language); err != nil {
		Fail(w, CodeBadRequest, err.Error())
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	if err := s.db.Model(&models.User{}).Where("id = ?", u.ID).
		Updates(map[string]any{"password_hash": hash, "must_change_password": false}).Error; err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	// 保留当前会话，仅销毁其他设备的会话（避免改密后立即被踢出登录）
	currentToken := ""
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		currentToken = cookie.Value
	}
	auth.DestroyUserSessionsExcept(u.ID, currentToken)
	logger.Info("password changed: user='%s' ip=%s", u.Username, ipFromContext(r))
	audit.Log(s.db, u.ID, u.Username, "change_password", "password changed", ipFromContext(r))
	OKMsg(w, i18n.T(s.cfg.Language, "login.passwordChanged"), nil)
}
