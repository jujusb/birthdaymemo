package database

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/mcbill1/birthdaymemo/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBPath 返回可执行文件同目录下的数据库路径
func DBPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "birthdaymemo.db"), nil
}

// Init 打开 SQLite 并执行迁移与初始化默认数据
func Init() (*gorm.DB, error) {
	path, err := DBPath()
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.ReminderSetting{},
		&models.PdfSetting{},
		&models.Birthday{},
		&models.Tag{},
		&models.BirthdayTag{},
		&models.SmtpSetting{},
		&models.EmailTemplate{},
		&models.OperationLog{},
		&models.Setting{},
		&models.LoginAttempt{},
		&models.ReminderSent{},
		&models.UserSession{},
		&models.EmailConfirmation{},
	); err != nil {
		return nil, err
	}

	if err := seedDefaults(db); err != nil {
		return nil, err
	}
	return db, nil
}

// seedDefaults 初始化系统默认设置（仅在缺失时写入）
func seedDefaults(db *gorm.DB) error {
	// SMTP 默认空行
	var smtp models.SmtpSetting
	if err := db.First(&smtp, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			db.Create(&models.SmtpSetting{ID: 1, Encryption: models.SMTPEncryptionStartTLS, Port: 587})
		} else {
			return err
		}
	}

	// 邮件模板默认
	var tpl models.EmailTemplate
	if err := db.First(&tpl, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			db.Create(&models.EmailTemplate{
				ID:                 1,
				Subject:            models.DefaultEmailTemplateSubject,
				Body:               models.DefaultEmailTemplateBody,
				SelfAdvanceSubject: models.DefaultSelfAdvanceSubject,
				SelfAdvanceBody:    models.DefaultSelfAdvanceBody,
				SelfTodaySubject:   models.DefaultSelfTodaySubject,
				SelfTodayBody:      models.DefaultSelfTodayBody,
			})
		} else {
			return err
		}
	} else {
		// 任务5：补全已有记录中缺失的本人生日模板字段（旧数据库迁移）
		needSave := false
		if tpl.SelfAdvanceSubject == "" {
			tpl.SelfAdvanceSubject = models.DefaultSelfAdvanceSubject
			needSave = true
		}
		if tpl.SelfAdvanceBody == "" {
			tpl.SelfAdvanceBody = models.DefaultSelfAdvanceBody
			needSave = true
		}
		if tpl.SelfTodaySubject == "" {
			tpl.SelfTodaySubject = models.DefaultSelfTodaySubject
			needSave = true
		}
		if tpl.SelfTodayBody == "" {
			tpl.SelfTodayBody = models.DefaultSelfTodayBody
			needSave = true
		}
		if needSave {
			db.Save(&tpl)
		}
	}

	// 支持语言默认 en,zh
	var s models.Setting
	if err := db.Where("key = ?", models.SettingKeySupportedLanguages).First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			b, _ := json.Marshal([]string{"en", "zh"})
			db.Create(&models.Setting{Key: models.SettingKeySupportedLanguages, Value: string(b)})
		} else {
			return err
		}
	}
	return nil
}

// EnsureUserSettings 为新用户初始化提醒与 PDF 设置
func EnsureUserSettings(db *gorm.DB, userID uint) error {
	var rs models.ReminderSetting
	if err := db.Where("user_id = ?", userID).First(&rs).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&models.ReminderSetting{
				UserID:       userID,
				Mode:         models.ReminderModeDaysBefore,
				DaysBefore:   3,
				RemindHour:   8,
				EmailEnabled: false,
			}).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	var ps models.PdfSetting
	if err := db.Where("user_id = ?", userID).First(&ps).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&models.PdfSetting{
				UserID:         userID,
				TitleText:      models.DefaultPdfTitleText,
				SubtitleText:   models.DefaultPdfSubtitleText,
				BackgroundType: models.PdfBgWhite,
				TableEffect:    models.PdfEffectNone,
				TableOpacity:   100,
			}).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
