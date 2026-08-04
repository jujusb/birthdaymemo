package security

import (
	"time"

	"github.com/mcbill1/birthdaymemo/internal/models"
	"gorm.io/gorm"
)

// 登录失败限制策略（按 IP，不针对用户名）：
//   - 10 分钟内失败 ≥ 5 次：要求图形验证码
//   - 10 分钟内失败 ≥ 15 次：锁定 IP 1 小时
//   - 锁定解锁后计数重置，重新累计
//   - 登录成功后该 IP 的所有计数清零
const (
	captchaThreshold = 5                // 10 分钟内失败 ≥5 次启用验证码
	ipLockThreshold  = 15               // 10 分钟内失败 ≥15 次锁定 IP
	failWindow       = 10 * time.Minute // 滑动窗口：距上次失败 10 分钟内视为同一窗口
	ipLockDuration   = 1 * time.Hour    // IP 锁定时长
)

// Status 登录限制状态
type Status struct {
	NeedsCaptcha bool
	Locked       bool
	LockedUntil  time.Time
	FailCount    int
}

// GetStatus 查询某 IP 的登录限制状态
func GetStatus(db *gorm.DB, ip string) Status {
	var la models.LoginAttempt
	if err := db.Where("ip = ?", ip).First(&la).Error; err != nil {
		return Status{}
	}
	now := time.Now()
	s := Status{FailCount: la.FailCount}

	// 仍在锁定期内
	if now.Before(la.LockedUntil) {
		s.Locked = true
		s.LockedUntil = la.LockedUntil
		return s
	}

	// 若上次失败距今超过窗口，则不计入需要验证码
	// （FailCount 会在下次 RecordFailure 时重置）
	if !la.UpdatedAt.IsZero() && now.Sub(la.UpdatedAt) > failWindow {
		return s
	}

	// 窗口内失败 ≥5：要求验证码
	if la.FailCount >= captchaThreshold {
		s.NeedsCaptcha = true
	}
	return s
}

// RequireCaptcha 是否需要验证码
func RequireCaptcha(db *gorm.DB, ip string) bool {
	return GetStatus(db, ip).NeedsCaptcha
}

// IsLocked 是否被锁定
func IsLocked(db *gorm.DB, ip string) (bool, time.Time) {
	s := GetStatus(db, ip)
	return s.Locked, s.LockedUntil
}

// RecordFailure 记录一次失败登录。
// 返回 (locked, lockedUntil)。
//   - 若距上次失败 > 10 min，重置窗口与计数后递增；
//   - 若曾锁定且已解锁，重置窗口与计数后递增；
//   - 若窗口内累计 ≥15 次，设置锁定 1 小时。
func RecordFailure(db *gorm.DB, ip string) (bool, time.Time) {
	var la models.LoginAttempt
	err := db.Where("ip = ?", ip).First(&la).Error
	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		la = models.LoginAttempt{
			IP:          ip,
			FailCount:   1,
			WindowStart: now,
			UpdatedAt:   now,
		}
		if err := db.Create(&la).Error; err != nil {
			return false, now
		}
		return false, time.Time{}
	}
	if err != nil {
		return false, now
	}

	// 曾锁定且已解锁：重置窗口与计数
	if !la.LockedUntil.IsZero() && now.After(la.LockedUntil) {
		la.LockedUntil = time.Time{}
		la.WindowStart = now
		la.FailCount = 0
	} else if now.Sub(la.UpdatedAt) > failWindow {
		// 距上次失败超过窗口：重置窗口与计数
		la.WindowStart = now
		la.FailCount = 0
	}

	la.FailCount++
	la.UpdatedAt = now

	locked := false
	until := time.Time{}
	if la.FailCount >= ipLockThreshold {
		la.LockedUntil = now.Add(ipLockDuration)
		locked = true
		until = la.LockedUntil
	}
	db.Save(&la)
	return locked, until
}

// RecordSuccess 登录成功后重置该 IP 记录
func RecordSuccess(db *gorm.DB, ip string) {
	db.Model(&models.LoginAttempt{}).Where("ip = ?", ip).Updates(map[string]any{
		"fail_count":   0,
		"window_start": time.Now(),
		"locked_until": time.Time{},
	})
}
