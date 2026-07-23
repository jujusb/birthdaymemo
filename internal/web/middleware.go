package web

import (
	"net/http"

	"github.com/mcbill1/birthdaymemo/internal/auth"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/logger"
	"github.com/mcbill1/birthdaymemo/internal/models"
	"gorm.io/gorm"
)

// SessionCookieName 会话 Cookie 名称
const SessionCookieName = "bdm_session"

// SecurityHeaders 添加安全响应头：
//   - X-Content-Type-Options: nosniff —— 阻止 MIME 嗅探
//   - X-Frame-Options: DENY —— 防止点击劫持
//   - Referrer-Policy: strict-origin-when-cross-origin —— 限制 Referer 泄漏
//   - X-XSS-Protection: 1; mode=block —— 旧浏览器反射型 XSS 防护
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

// IsHTTPS 判断当前请求是否为 HTTPS（含反向代理透传）
func IsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if xfp := r.Header.Get("X-Forwarded-Proto"); xfp == "https" {
		return true
	}
	return false
}

// RequireAuth 校验登录状态，未登录返回 401
func RequireAuth(db *gorm.DB, lang string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIP(r)
			r = withIP(r, ip)

			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				FailStatus(w, http.StatusUnauthorized, CodeUnauthorized, i18n.T(lang, "error.unauthorized"))
				return
			}
			s := auth.GetSession(cookie.Value)
			if s == nil {
				FailStatus(w, http.StatusUnauthorized, CodeUnauthorized, i18n.T(lang, "error.unauthorized"))
				return
			}
			auth.TouchSession(cookie.Value)
			r = withSession(r, s)

			var u models.User
			if err := db.First(&u, s.UserID).Error; err != nil {
				auth.DestroySession(cookie.Value)
				FailStatus(w, http.StatusUnauthorized, CodeUnauthorized, i18n.T(lang, "error.unauthorized"))
				return
			}
			r = withUser(r, &u)
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin 要求当前用户为管理员
func RequireAdmin(lang string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := userFromContext(r)
			if u == nil || u.Role != models.RoleAdmin {
				FailStatus(w, http.StatusForbidden, CodeForbidden, i18n.T(lang, "error.forbidden"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLog 记录请求日志
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := userFromContext(r); u != nil {
			logger.Info("%s %s user=%s ip=%s", r.Method, r.URL.Path, u.Username, ipFromContext(r))
		}
		next.ServeHTTP(w, r)
	})
}
