package web

import (
	"encoding/json"
	"net/http"

	"github.com/mcbill1/birthdaymemo/internal/database"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/models"
)

// settingsResponse 用户设置响应
type settingsResponse struct {
	Reminder       models.ReminderSetting `json:"reminder"`
	Language       string                 `json:"language"`
	Theme          string                 `json:"theme"`
	ThemeColor     string                 `json:"theme_color"`
	Email          string                 `json:"email"`         // 已确认邮箱
	PendingEmail   string                 `json:"pending_email"` // 待确认邮箱
	TopbarRange    int                    `json:"topbar_range_days"`
	AvailableLangs []i18n.Language        `json:"available_languages"`
	// 任务3：本人生日
	SelfBirthYear  int `json:"self_birth_year"`
	SelfBirthMonth int `json:"self_birth_month"`
	SelfBirthDay   int `json:"self_birth_day"`
	// 标签名称最大字符数（服务端配置，可通过环境变量调整）
	TagNameMaxLength int `json:"tag_name_max_length"`
}

// settingsRequest 用户设置请求
type settingsRequest struct {
	Reminder       models.ReminderSetting `json:"reminder"`
	Language       string                 `json:"language"`
	Theme          string                 `json:"theme"`
	ThemeColor     string                 `json:"theme_color"`
	TopbarRange    int                    `json:"topbar_range_days"`
	SelfBirthYear  int                    `json:"self_birth_year"`
	SelfBirthMonth int                    `json:"self_birth_month"`
	SelfBirthDay   int                    `json:"self_birth_day"`
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r)
	var rs models.ReminderSetting
	if err := s.db.Where("user_id = ?", u.ID).First(&rs).Error; err != nil {
		database.EnsureUserSettings(s.db, u.ID)
		s.db.Where("user_id = ?", u.ID).First(&rs)
	}
	themeColor := u.ThemeColor
	if themeColor == "" {
		themeColor = models.ThemeColorBlue
	}
	OK(w, settingsResponse{
		Reminder:       rs,
		Language:       u.Language,
		Theme:          u.Theme,
		ThemeColor:     themeColor,
		Email:          u.Email,
		PendingEmail:   u.PendingEmail,
		TopbarRange:    u.TopbarRangeDays,
		AvailableLangs: i18n.Available(),
		SelfBirthYear:    u.SelfBirthYear,
		SelfBirthMonth:   u.SelfBirthMonth,
		SelfBirthDay:     u.SelfBirthDay,
		TagNameMaxLength: tagNameLimit(s.cfg),
	})
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r)
	var req settingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}

	// 校验提醒设置
	if req.Reminder.Mode != models.ReminderModeDaysBefore && req.Reminder.Mode != models.ReminderModePeriodic {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "settings.reminderMode"))
		return
	}
	if req.Reminder.RemindHour < 0 || req.Reminder.RemindHour > 23 {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "settings.remindTime"))
		return
	}
	if req.Reminder.Mode == models.ReminderModeDaysBefore {
		if req.Reminder.DaysBefore < 1 || req.Reminder.DaysBefore > 365 {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "settings.daysBefore"))
			return
		}
	}
	if req.Reminder.Mode == models.ReminderModePeriodic {
		if req.Reminder.SubMode != models.ReminderSubWeekly && req.Reminder.SubMode != models.ReminderSubMonthly {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "settings.reminderMode"))
			return
		}
		if req.Reminder.SubMode == models.ReminderSubWeekly && (req.Reminder.WeeklyDay < 0 || req.Reminder.WeeklyDay > 6) {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "settings.weeklyDay"))
			return
		}
		if req.Reminder.SubMode == models.ReminderSubMonthly && (req.Reminder.MonthlyDay < 1 || req.Reminder.MonthlyDay > 28) {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "settings.monthlyDay"))
			return
		}
	}

	// 语言校验
	langValid := false
	for _, l := range i18n.Available() {
		if l.Code == req.Language {
			langValid = true
			break
		}
	}
	if !langValid {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "settings.language"))
		return
	}
	theme := req.Theme
	if theme != models.ThemeDark && theme != models.ThemeLight {
		theme = models.ThemeLight
	}
	themeColor := req.ThemeColor
	if !models.IsValidThemeColor(themeColor) {
		themeColor = models.ThemeColorBlue
	}
	topbar := req.TopbarRange
	if topbar < 1 {
		topbar = 30
	}
	if topbar > 365 {
		topbar = 365
	}
	// 任务3：本人生日校验（允许全 0 = 未设置；若设置则月份/日必须合法，年份可 0）
	selfYear := req.SelfBirthYear
	selfMonth := req.SelfBirthMonth
	selfDay := req.SelfBirthDay
	if selfMonth != 0 || selfDay != 0 {
		if selfMonth < 1 || selfMonth > 12 {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "validation.invalidMonth"))
			return
		}
		// 校验日：根据月份判断
		daysInMonth := daysInMonthFor(selfYear, selfMonth)
		if selfDay < 1 || selfDay > daysInMonth {
			Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "validation.invalidDayForMonth"))
			return
		}
	} else {
		// 未设置时全部清零
		selfYear, selfMonth, selfDay = 0, 0, 0
	}
	// 邮箱不再通过通用设置直接修改，改由确认流程管理（见 email_handlers.go）

	// 保存提醒设置
	var rs models.ReminderSetting
	if err := s.db.Where("user_id = ?", u.ID).First(&rs).Error; err != nil {
		rs.UserID = u.ID
	}
	rs.Mode = req.Reminder.Mode
	rs.SubMode = req.Reminder.SubMode
	rs.DaysBefore = req.Reminder.DaysBefore
	rs.RemindHour = req.Reminder.RemindHour
	rs.WeeklyDay = req.Reminder.WeeklyDay
	rs.MonthlyDay = req.Reminder.MonthlyDay
	rs.EmailEnabled = req.Reminder.EmailEnabled
	s.db.Save(&rs)

	s.db.Model(&models.User{}).Where("id = ?", u.ID).Updates(map[string]any{
		"language":          req.Language,
		"theme":             theme,
		"theme_color":       themeColor,
		"topbar_range_days": topbar,
		"self_birth_year":   selfYear,
		"self_birth_month":  selfMonth,
		"self_birth_day":    selfDay,
	})

	OKMsg(w, i18n.T(s.cfg.Language, "settings.saved"), nil)
}

// daysInMonthFor 返回某年某月的天数（年份为 0 时按平年算）
func daysInMonthFor(year, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if year > 0 && (year%4 == 0 && (year%100 != 0 || year%400 == 0)) {
			return 29
		}
		return 28
	}
	return 0
}

// pdfSettingsRequest PDF 设置请求
type pdfSettingsRequest struct {
	models.PdfSetting
}

func (s *Server) getPdfSettings(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r)
	var ps models.PdfSetting
	if err := s.db.Where("user_id = ?", u.ID).First(&ps).Error; err != nil {
		database.EnsureUserSettings(s.db, u.ID)
		s.db.Where("user_id = ?", u.ID).First(&ps)
	}
	OK(w, ps)
}

func (s *Server) updatePdfSettings(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r)
	var req models.PdfSetting
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, CodeBadRequest, i18n.T(s.cfg.Language, "error.badRequest"))
		return
	}
	var ps models.PdfSetting
	if err := s.db.Where("user_id = ?", u.ID).First(&ps).Error; err != nil {
		ps.UserID = u.ID
	}
	ps.TitleText = req.TitleText
	if ps.TitleText == "" {
		ps.TitleText = models.DefaultPdfTitleText
	}
	ps.SubtitleText = req.SubtitleText
	if ps.SubtitleText == "" {
		ps.SubtitleText = models.DefaultPdfSubtitleText
	}
	ps.BackgroundType = req.BackgroundType
	ps.BackgroundColor = req.BackgroundColor
	ps.GradientStart = req.GradientStart
	ps.GradientEnd = req.GradientEnd
	ps.Blur = clamp(req.Blur, 0, 20)
	ps.TableEffect = req.TableEffect
	ps.TableOpacity = clamp(req.TableOpacity, 0, 100)
	ps.TableBgColor = req.TableBgColor
	ps.TextColor = req.TextColor
	ps.ShowAge = req.ShowAge
	s.db.Save(&ps)
	OK(w, ps)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
