package models

import "time"

// 用户角色
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// 主题
const (
	ThemeDark  = "dark"
	ThemeLight = "light"
)

// 主题色预设代码（10 种）
const (
	ThemeColorBlue   = "blue"
	ThemeColorPurple = "purple"
	ThemeColorPink   = "pink"
	ThemeColorRed    = "red"
	ThemeColorOrange = "orange"
	ThemeColorAmber  = "amber"
	ThemeColorGreen  = "green"
	ThemeColorTeal   = "teal"
	ThemeColorCyan   = "cyan"
	ThemeColorIndigo = "indigo"
)

// PresetThemeColors 主题色预设列表（10 种）
var PresetThemeColors = []string{
	ThemeColorBlue, ThemeColorPurple, ThemeColorPink, ThemeColorRed, ThemeColorOrange,
	ThemeColorAmber, ThemeColorGreen, ThemeColorTeal, ThemeColorCyan, ThemeColorIndigo,
}

// IsValidThemeColor 校验主题色代码是否合法
func IsValidThemeColor(code string) bool {
	for _, c := range PresetThemeColors {
		if c == code {
			return true
		}
	}
	return false
}

// User 用户
type User struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	Username           string    `json:"username" gorm:"uniqueIndex;size:20;not null"`
	PasswordHash       string    `json:"-" gorm:"not null"`
	Email              string    `json:"email" gorm:"size:100"`         // 已确认的提醒邮件收件人，可空
	PendingEmail       string    `json:"pending_email" gorm:"size:100"` // 待确认的邮箱地址
	Role               string    `json:"role" gorm:"size:10;not null;default:user"`
	MustChangePassword bool      `json:"must_change_password" gorm:"not null;default:false"`
	Language           string    `json:"language" gorm:"size:10;not null;default:en"`
	Theme              string    `json:"theme" gorm:"size:10;not null;default:light"`
	ThemeColor         string    `json:"theme_color" gorm:"size:10;not null;default:blue"` // 主题色（10 种预设）
	TopbarRangeDays    int       `json:"topbar_range_days" gorm:"not null;default:30"`
	// 任务3：本人生日（独立于好友生日列表，0 表示未设置）
	SelfBirthYear  int `json:"self_birth_year" gorm:"default:0"`
	SelfBirthMonth int `json:"self_birth_month" gorm:"default:0"`
	SelfBirthDay   int `json:"self_birth_day" gorm:"default:0"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// HasSelfBirthday 是否已设置本人生日
func (u User) HasSelfBirthday() bool {
	return u.SelfBirthMonth > 0 && u.SelfBirthDay > 0
}

// 提醒模式
const (
	ReminderModeDaysBefore = 1 // 生日到期前N天
	ReminderModePeriodic   = 2 // 周期性：每周/每月
	ReminderSubWeekly      = "weekly"
	ReminderSubMonthly     = "monthly"
)

// ReminderSetting 用户提醒设置（与用户一对一）
type ReminderSetting struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"uniqueIndex;not null"`
	Mode         int       `json:"mode" gorm:"not null;default:1"`              // 1=到期前N天 2=周期
	SubMode      string    `json:"sub_mode" gorm:"size:10;default:weekly"`      // mode=2: weekly|monthly
	DaysBefore   int       `json:"days_before" gorm:"not null;default:3"`       // mode=1: 提前天数
	RemindHour   int       `json:"remind_hour" gorm:"not null;default:8"`       // 提醒小时 0-23
	WeeklyDay    int       `json:"weekly_day" gorm:"not null;default:0"`        // mode=2 weekly: 0=周日..6=周六
	MonthlyDay   int       `json:"monthly_day" gorm:"not null;default:1"`       // mode=2 monthly: 几号 1-28
	EmailEnabled bool      `json:"email_enabled" gorm:"not null;default:false"` // 是否邮件提醒
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PdfBackgroundType PDF 背景类型
const (
	PdfBgWhite    = "white"
	PdfBgSolid    = "solid"
	PdfBgGradient = "gradient"
	PdfBgImage    = "image"
)

// PdfTableEffect 表格特效
const (
	PdfEffectNone    = "none"
	PdfEffectAcrylic = "acrylic" // 磨砂玻璃
	PdfEffectFrosted = "frosted" // 液态玻璃
)

// DefaultPdfTitleText 默认大行文字（变量 {month}=月份英文, {count}=当月人数）
const DefaultPdfTitleText = "[{month}] 当月 {count} 人过生日"

// DefaultPdfSubtitleText 默认小行文字
// 注：fpdf 库不支持 BMP 之外的字形（如 emoji），因此默认值不使用 emoji
const DefaultPdfSubtitleText = "BirthDayMemo"

// PdfSetting 用户 PDF 导出设置
type PdfSetting struct {
	ID     uint `json:"id" gorm:"primaryKey"`
	UserID uint `json:"user_id" gorm:"uniqueIndex;not null"`
	// 上方文字
	TitleText    string `json:"title_text" gorm:"size:200;default:[{month}] 当月 {count} 人过生日"`
	SubtitleText string `json:"subtitle_text" gorm:"size:200;default:🎂 BirthDayMemo 🎂"`
	// 背景
	BackgroundType  string `json:"background_type" gorm:"size:10;not null;default:white"`
	BackgroundColor string `json:"background_color" gorm:"size:20;default:#ffffff"`
	GradientStart   string `json:"gradient_start" gorm:"size:20;default:#e0c3fc"`
	GradientEnd     string `json:"gradient_end" gorm:"size:20;default:#8ec5fc"`
	Blur            int    `json:"blur" gorm:"not null;default:0"` // 0-20 模糊程度
	// 任务6：表格背景（外框）— 框住所有日期格的大圆角框
	TableEffect         string `json:"table_effect" gorm:"size:10;not null;default:none"`
	TableOpacity        int    `json:"table_opacity" gorm:"not null;default:100"` // 0-100 透明度
	TableBgColor        string `json:"table_bg_color" gorm:"size:20;default:#ffffff"`
	TableBorderEnabled  bool   `json:"table_border_enabled" gorm:"not null;default:true"`
	TableBorderColor    string `json:"table_border_color" gorm:"size:20;default:#000000"`
	TableBorderOpacity  int    `json:"table_border_opacity" gorm:"not null;default:100"` // 0-100
	// 任务6：日期表格（内格）— 每个日期的小方格
	CellEffect         string `json:"cell_effect" gorm:"size:10;not null;default:none"`
	CellOpacity        int    `json:"cell_opacity" gorm:"not null;default:100"` // 0-100 透明度
	CellBgColor        string `json:"cell_bg_color" gorm:"size:20;default:#ffffff"`
	CellBorderEnabled  bool   `json:"cell_border_enabled" gorm:"not null;default:true"`
	CellBorderColor    string `json:"cell_border_color" gorm:"size:20;default:#000000"`
	CellBorderOpacity  int    `json:"cell_border_opacity" gorm:"not null;default:100"` // 0-100
	// 文字
	TextColor string    `json:"text_color" gorm:"size:20;default:#333333"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
