package models

import "time"

// 性别
const (
	GenderMale   = "male"
	GenderFemale = "female"
	GenderNone   = "none"
)

// 性别色
const (
	GenderColorMale   = "#4A90D9" // 蓝
	GenderColorFemale = "#E91E63" // 粉
	GenderColorNone   = "#9E9E9E" // 灰
)

// GenderColor 返回性别对应颜色
func GenderColor(g string) string {
	switch g {
	case GenderMale:
		return GenderColorMale
	case GenderFemale:
		return GenderColorFemale
	default:
		return GenderColorNone
	}
}

// Birthday 生日记录
type Birthday struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"index;not null"`
	Name       string    `json:"name" gorm:"size:50;not null"`
	Gender     string    `json:"gender" gorm:"size:10;not null;default:none"`
	BirthYear  int       `json:"birth_year" gorm:"default:0"` // 0=未知年份
	BirthMonth int       `json:"birth_month" gorm:"not null"` // 1-12
	BirthDay   int       `json:"birth_day" gorm:"not null"`   // 1-31
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BirthdayTag 生日-标签 多对多关联
type BirthdayTag struct {
	BirthdayID uint      `json:"birthday_id" gorm:"primaryKey"`
	TagID      uint      `json:"tag_id" gorm:"primaryKey"`
	CreatedAt  time.Time `json:"created_at"`
}

// Age 计算当前年龄（年份未知返回 -1）
func (b Birthday) Age(now time.Time) int {
	if b.BirthYear == 0 {
		return -1
	}
	age := now.Year() - b.BirthYear
	t := time.Date(now.Year(), time.Month(b.BirthMonth), b.BirthDay, 0, 0, 0, 0, now.Location())
	if now.Before(t) {
		age--
	}
	return age
}

// UpcomingAge 计算下一个生日后的年龄
func (b Birthday) UpcomingAge(now time.Time) int {
	if b.BirthYear == 0 {
		return -1
	}
	age := now.Year() - b.BirthYear
	t := time.Date(now.Year(), time.Month(b.BirthMonth), b.BirthDay, 0, 0, 0, 0, now.Location())
	if now.Before(t) || now.Equal(t) {
		// 今年生日还没到，下一个生日是今年，年龄 age；若已过则 age+1
		return age
	}
	return age + 1
}

// Tag 标签
type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	Name      string    `json:"name" gorm:"size:10;not null"` // 限制10字符
	Color     string    `json:"color" gorm:"size:20;not null;default:#4A90D9"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PresetTagColors 标签预设色板（10 色柔和配色）
var PresetTagColors = []string{
	"#4A90D9", // 蓝
	"#E91E63", // 粉
	"#4CAF50", // 绿
	"#FF9800", // 橙
	"#9C27B0", // 紫
	"#00BCD4", // 青
	"#FF5722", // 红橙
	"#795548", // 棕
	"#607D8B", // 蓝灰
	"#FFC107", // 琥珀
}
