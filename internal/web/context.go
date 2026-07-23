package web

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/mcbill1/birthdaymemo/internal/auth"
	"github.com/mcbill1/birthdaymemo/internal/models"
)

type ctxKey int

const (
	keySession ctxKey = iota
	keyUser
	keyIP
)

// ClientIP 从请求中提取真实客户端 IP（支持代理头）
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// withSession 在上下文存储会话
func withSession(r *http.Request, s *auth.Session) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), keySession, s))
}

// sessionFromContext 取出会话
func sessionFromContext(r *http.Request) *auth.Session {
	if v, ok := r.Context().Value(keySession).(*auth.Session); ok {
		return v
	}
	return nil
}

// withUser 在上下文存储用户
func withUser(r *http.Request, u *models.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), keyUser, u))
}

// userFromContext 取出当前登录用户
func userFromContext(r *http.Request) *models.User {
	if v, ok := r.Context().Value(keyUser).(*models.User); ok {
		return v
	}
	return nil
}

// currentUserID 取出当前用户 ID
func currentUserID(r *http.Request) uint {
	if u := userFromContext(r); u != nil {
		return u.ID
	}
	return 0
}

// withIP 在上下文存储 IP
func withIP(r *http.Request, ip string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), keyIP, ip))
}

// ipFromContext 取出 IP
func ipFromContext(r *http.Request) string {
	if v, ok := r.Context().Value(keyIP).(string); ok {
		return v
	}
	return ""
}
