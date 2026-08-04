package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/mcbill1/birthdaymemo/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	bcryptCost        = 10
	sessionTTL        = 7 * 24 * time.Hour // 7 天有效
	sessionTokenBytes = 32
)

// Session 服务端会话（运行时缓存）
type Session struct {
	Token        string
	UserID       uint
	Username     string
	Role         string
	LastActivity time.Time
}

// db 全局数据库引用，由 SetDB 初始化
var db *gorm.DB

// SetDB 设置数据库引用，必须在所有会话操作前调用
func SetDB(database *gorm.DB) {
	db = database
}

// HashPassword 使用 bcrypt 加密密码
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword 校验密码
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// randomToken 生成安全随机 token
func randomToken() string {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		// 回退到时间戳保证不返回空（极端情况）
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}

// CreateSession 创建会话并返回令牌（持久化到数据库）
func CreateSession(u *models.User) string {
	token := randomToken()
	now := time.Now()
	s := &models.UserSession{
		Token:        token,
		UserID:       u.ID,
		Username:     u.Username,
		Role:         u.Role,
		LastActivity: now,
		CreatedAt:    now,
	}
	if db != nil {
		db.Create(s)
	}
	return token
}

// GetSession 返回会话，若不存在或已超时则返回 nil
func GetSession(token string) *Session {
	if db == nil {
		return nil
	}
	var us models.UserSession
	if err := db.Where("token = ?", token).First(&us).Error; err != nil {
		return nil
	}
	if time.Since(us.LastActivity) > sessionTTL {
		db.Delete(&us)
		return nil
	}
	return &Session{
		Token:        us.Token,
		UserID:       us.UserID,
		Username:     us.Username,
		Role:         us.Role,
		LastActivity: us.LastActivity,
	}
}

// TouchSession 更新会话活跃时间（会话存在时）
func TouchSession(token string) {
	if db == nil {
		return
	}
	db.Model(&models.UserSession{}).Where("token = ?", token).
		Update("last_activity", time.Now())
}

// DestroySession 销毁会话
func DestroySession(token string) {
	if db == nil {
		return
	}
	db.Where("token = ?", token).Delete(&models.UserSession{})
}

// DestroyUserSessions 销毁指定用户的所有会话（重置密码后调用）
func DestroyUserSessions(userID uint) {
	if db == nil {
		return
	}
	db.Where("user_id = ?", userID).Delete(&models.UserSession{})
}

// DestroyUserSessionsExcept 销毁指定用户除当前会话外的所有会话
// 用于改密成功后保留当前登录，强制其他设备下线
func DestroyUserSessionsExcept(userID uint, currentToken string) {
	if db == nil {
		return
	}
	q := db.Where("user_id = ?", userID)
	if currentToken != "" {
		q = q.Where("token <> ?", currentToken)
	}
	q.Delete(&models.UserSession{})
}
