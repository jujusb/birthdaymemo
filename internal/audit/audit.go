package audit

import (
	"time"

	"github.com/mcbill1/birthdaymemo/internal/models"
	"gorm.io/gorm"
)

// Log 写入一条操作日志
func Log(db *gorm.DB, userID uint, username, action, detail, ip string) {
	db.Create(&models.OperationLog{
		UserID:   userID,
		Username: username,
		Action:   action,
		Detail:   detail,
		IP:       ip,
	})
}

// CleanExpired 清理超过保留天数的操作日志，retentionDays=0 表示不清理
func CleanExpired(db *gorm.DB, retentionDays int) {
	if retentionDays <= 0 {
		return
	}
	threshold := time.Now().AddDate(0, 0, -retentionDays)
	db.Where("created_at < ?", threshold).Delete(&models.OperationLog{})
}
