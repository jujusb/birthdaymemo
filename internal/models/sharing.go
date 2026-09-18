package models

import (
	"strings"
	"time"
)

// 分享范围
const (
	ShareScopeAll  = "all"
	ShareScopeTags = "tags"
)

// 授权类型
const (
	GrantView = "view"
	GrantEdit = "edit"
)

// ShareLink 只读分享链接：匿名访客通过 unguessable token（或可选的自定义 slug）查看指定范围的日历
type ShareLink struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Token       string     `json:"-" gorm:"uniqueIndex;size:64;not null"`
	Slug        string     `json:"slug" gorm:"size:32;not null;default:''"` // 可选自定义短址，全局唯一；空表示未设置
	OwnerUserID uint       `json:"owner_user_id" gorm:"index;not null"`
	Name        string     `json:"name" gorm:"size:50;not null;default:''"`
	ScopeMode   string     `json:"scope_mode" gorm:"size:10;not null;default:all"` // all | tags
	TagIDs      string     `json:"tag_ids" gorm:"size:500;not null;default:'[]'"`  // ScopeMode=tags 时的 JSON []uint
	ExpiresAt   *time.Time `json:"expires_at" gorm:"index"`
	Revoked     bool       `json:"revoked" gorm:"not null;default:false"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// 分享 slug 规则：3–32 位小写字母/数字/连字符（token 固定 64 位，二者长度无交集，公开展示时不会歧义）
const (
	ShareSlugMinLen = 3
	ShareSlugMaxLen = 32
)

// NormalizeShareSlug 规范化并校验自定义 slug：去空白转小写，合法返回 (slug, true)
func NormalizeShareSlug(s string) (string, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) < ShareSlugMinLen || len(s) > ShareSlugMaxLen {
		return "", false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return "", false
		}
	}
	return s, true
}

// Expired 是否已过期
func (s ShareLink) Expired() bool {
	return s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt)
}

// TagGrant 标签级委托：Owner 将自己某个 Tag 的查看/编辑权授予 Grantee
type TagGrant struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	OwnerUserID   uint      `json:"owner_user_id" gorm:"index;not null"`
	GranteeUserID uint      `json:"grantee_user_id" gorm:"uniqueIndex:idx_grantee_tag;not null"`
	TagID         uint      `json:"tag_id" gorm:"uniqueIndex:idx_grantee_tag;not null"`
	Permission    string    `json:"permission" gorm:"size:10;not null;default:edit"` // view | edit
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
