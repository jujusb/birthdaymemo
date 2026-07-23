package reminder

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mcbill1/birthdaymemo/internal/audit"
	"github.com/mcbill1/birthdaymemo/internal/email"
	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/logger"
	"github.com/mcbill1/birthdaymemo/internal/models"
	"gorm.io/gorm"
)

// Scheduler 提醒调度器，每分钟检查一次是否到提醒时间。
type Scheduler struct {
	db     *gorm.DB
	sender *email.Sender
	stop   chan struct{}
	wg     sync.WaitGroup
}

// NewScheduler 创建调度器。
func NewScheduler(db *gorm.DB, sender *email.Sender) *Scheduler {
	return &Scheduler{db: db, sender: sender, stop: make(chan struct{})}
}

// Start 启动调度器 goroutine。
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.run()
}

// Stop 停止调度器并等待退出。
func (s *Scheduler) Stop() {
	close(s.stop)
	s.wg.Wait()
}

// run 主循环，每分钟检查一次。
func (s *Scheduler) run() {
	defer s.wg.Done()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

// tick 执行一次检查，遍历所有用户并按其提醒设置处理。
func (s *Scheduler) tick() {
	now := time.Now()
	var users []models.User
	if err := s.db.Find(&users).Error; err != nil {
		logger.Error("reminder: load users failed: %v", err)
		return
	}
	for i := range users {
		s.processUser(&users[i], now)
	}
}

// processUser 按用户提醒设置评估是否触发并发送邮件。
func (s *Scheduler) processUser(u *models.User, now time.Time) {
	var rs models.ReminderSetting
	if err := s.db.Where("user_id = ?", u.ID).First(&rs).Error; err != nil {
		return
	}

	// 任务7：本人生日提醒（独立于 EmailEnabled，按 ReminderSetting 触发；与好友邮件分别发送）
	// 任务1（本批次）：当日提醒固定早上 8 点，提前提醒按 ReminderSetting.RemindHour
	// 仍要求用户已确认邮箱才能发送
	if u.Email != "" && u.HasSelfBirthday() {
		s.processSelfBirthday(u, &rs, now)
	}

	// 等到提醒小时后才触发好友提醒
	if now.Hour() < rs.RemindHour {
		return
	}

	// 好友生日提醒：仅当 EmailEnabled 时发送
	if !rs.EmailEnabled {
		return
	}
	if u.Email == "" {
		return
	}

	var key string
	var entries []birthdayItem
	switch rs.Mode {
	case models.ReminderModeDaysBefore:
		key = "d:" + now.Format("2006-01-02")
		entries = s.findDaysBefore(u.ID, now, rs.DaysBefore)
	case models.ReminderModePeriodic:
		switch rs.SubMode {
		case models.ReminderSubWeekly:
			if int(now.Weekday()) != rs.WeeklyDay {
				return
			}
			year, week := now.ISOWeek()
			key = fmt.Sprintf("w:%d-%02d", year, week)
			entries = s.findUpcoming(u.ID, now, 7)
		case models.ReminderSubMonthly:
			if now.Day() != rs.MonthlyDay {
				return
			}
			key = fmt.Sprintf("m:%d-%02d", now.Year(), int(now.Month()))
			entries = s.findUpcoming(u.ID, now, 30)
		default:
			return
		}
	default:
		return
	}
	if key == "" || len(entries) == 0 {
		return
	}

	// 去重：若该 SentKey 已发送过，则跳过
	var sent models.ReminderSent
	if err := s.db.Where("user_id = ? AND sent_key = ?", u.ID, key).First(&sent).Error; err == nil {
		return
	}

	if err := s.sendEmail(u, entries); err != nil {
		logger.Error("reminder: send to %s failed: %v", u.Email, err)
		// 发送失败不记录，下一分钟重试
		return
	}
	// 标记已发送，避免重启后重复
	s.db.Create(&models.ReminderSent{UserID: u.ID, SentKey: key})
	logger.Info("reminder: sent %d birthday(s) to %s", len(entries), u.Email)
	// 审计日志：记录提醒邮件发送
	detail := u.Email
	if len(entries) > 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name)
		}
		detail = fmt.Sprintf("%s → %s", u.Email, strings.Join(names, ", "))
	}
	audit.Log(s.db, u.ID, u.Username, "send_reminder_email", detail, "")
}

// processSelfBirthday 任务4/7：本人生日提醒（独立于 EmailEnabled）
//   - 任务1（本批次）：当日提醒固定早上 8 点发送（不再受 ReminderSetting.RemindHour 控制）
//   - 提前提醒仍按 ReminderSetting.RemindHour 触发，仅 mode=1（days_before）模式
//   - SentKey：sa:YYYY-MM-DD（提前）/ st:YYYY-MM-DD（当日），与好友提醒互不冲突
func (s *Scheduler) processSelfBirthday(u *models.User, rs *models.ReminderSetting, now time.Time) {
	today := startOfDay(now)

	// 计算下一次本人生日日期
	selfBd := models.Birthday{
		BirthYear:  u.SelfBirthYear,
		BirthMonth: u.SelfBirthMonth,
		BirthDay:   u.SelfBirthDay,
	}
	next := nextBirthdayDate(selfBd, today)
	days := int(next.Sub(today).Hours() / 24)
	if days < 0 {
		return
	}

	// 1. 当日提醒：生日就是今天，固定早 8 点发送（< 8 点跳过）
	if days == 0 {
		if now.Hour() < 8 {
			return
		}
		key := "st:" + today.Format("2006-01-02")
		var sent models.ReminderSent
		if err := s.db.Where("user_id = ? AND sent_key = ?", u.ID, key).First(&sent).Error; err == nil {
			// 当日已发过
		} else if err := s.sendSelfBirthdayEmail(u, next, 0, true); err != nil {
			logger.Error("reminder: self-today send to %s failed: %v", u.Email, err)
		} else {
			s.db.Create(&models.ReminderSent{UserID: u.ID, SentKey: key})
			logger.Info("reminder: self-today sent to %s", u.Email)
			audit.Log(s.db, u.ID, u.Username, "send_reminder_email",
				fmt.Sprintf("%s → self birthday today", u.Email), "")
		}
		return
	}

	// 2. 提前提醒：仅 mode=1（days_before）模式触发，且需到提醒小时
	if rs.Mode != models.ReminderModeDaysBefore {
		return
	}
	if now.Hour() < rs.RemindHour {
		return
	}
	if days != rs.DaysBefore {
		return
	}
	key := "sa:" + today.Format("2006-01-02")
	var sent models.ReminderSent
	if err := s.db.Where("user_id = ? AND sent_key = ?", u.ID, key).First(&sent).Error; err == nil {
		return
	}
	if err := s.sendSelfBirthdayEmail(u, next, days, false); err != nil {
		logger.Error("reminder: self-advance send to %s failed: %v", u.Email, err)
		return
	}
	s.db.Create(&models.ReminderSent{UserID: u.ID, SentKey: key})
	logger.Info("reminder: self-advance sent to %s (%d days)", u.Email, days)
	audit.Log(s.db, u.ID, u.Username, "send_reminder_email",
		fmt.Sprintf("%s → self birthday in %d days", u.Email, days), "")
}

// sendSelfBirthdayEmail 渲染并发送本人生日提醒邮件
//   - isToday=true：使用 SelfTodaySubject/SelfTodayBody 模板
//   - isToday=false：使用 SelfAdvanceSubject/SelfAdvanceBody 模板（含 {self_birthday_last} 变量）
func (s *Scheduler) sendSelfBirthdayEmail(u *models.User, nextBirthday time.Time, daysLeft int, isToday bool) error {
	var smtp models.SmtpSetting
	if err := s.db.First(&smtp, 1).Error; err != nil {
		return fmt.Errorf("load smtp: %w", err)
	}
	if smtp.Host == "" || smtp.Username == "" || smtp.Password == "" {
		return fmt.Errorf("%s", i18n.T(u.Language, "validation.smtpNotConfigured"))
	}
	var tpl models.EmailTemplate
	if err := s.db.First(&tpl, 1).Error; err != nil {
		return fmt.Errorf("load email template: %w", err)
	}

	// 计算岁数：年份未知显示 ?
	ageStr := "?"
	if u.SelfBirthYear > 0 {
		ageStr = strconv.Itoa(nextBirthday.Year() - u.SelfBirthYear)
	}

	vars := map[string]string{
		"user":               u.Username,
		"self_birthday_date": nextBirthday.Format("2006-01-02"),
		"self_birthday_age":  ageStr,
		"self_birthday_last": strconv.Itoa(daysLeft),
	}

	var subject, body string
	if isToday {
		subject = renderSimpleTemplate(tpl.SelfTodaySubject, vars)
		body = renderSimpleTemplate(tpl.SelfTodayBody, vars)
	} else {
		subject = renderSimpleTemplate(tpl.SelfAdvanceSubject, vars)
		body = renderSimpleTemplate(tpl.SelfAdvanceBody, vars)
	}
	return s.sender.Send(smtp, u.Email, subject, body)
}

// renderSimpleTemplate 渲染模板（仅替换静态变量，无循环块）
func renderSimpleTemplate(tpl string, vars map[string]string) string {
	out := tpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

// birthdayItem 提醒邮件中一条生日条目
type birthdayItem struct {
	BirthdayID uint // 任务7：用于查询标签
	Name       string
	Gender     string // 任务7：性别，用于渲染代词 {birthday_sex_N}
	BirthYear  int
	BirthMonth int
	BirthDay   int
	Date       time.Time    // 下一次生日日期
	Age        int          // 即将到达的年龄，-1 表示年份未知
	Tags       []models.Tag // 任务7：此人的所有标签（已合并）
}

// findDaysBefore 返回下个生日距今恰好 daysBefore 天的生日。
func (s *Scheduler) findDaysBefore(userID uint, now time.Time, daysBefore int) []birthdayItem {
	var bds []models.Birthday
	if err := s.db.Where("user_id = ?", userID).Find(&bds).Error; err != nil {
		return nil
	}
	today := startOfDay(now)
	var out []birthdayItem
	for _, b := range bds {
		next := nextBirthdayDate(b, today)
		days := int(next.Sub(today).Hours() / 24)
		if days == daysBefore {
			out = append(out, birthdayItem{
				BirthdayID: b.ID,
				Name:       b.Name,
				Gender:     b.Gender,
				BirthYear:  b.BirthYear,
				BirthMonth: b.BirthMonth,
				BirthDay:   b.BirthDay,
				Date:       next,
				Age:        upcomingAge(b, next),
			})
		}
	}
	// 任务7：加载标签
	s.fillTags(userID, out)
	return out
}

// findUpcoming 返回接下来 days 天内（含今日）的生日。
func (s *Scheduler) findUpcoming(userID uint, now time.Time, days int) []birthdayItem {
	var bds []models.Birthday
	if err := s.db.Where("user_id = ?", userID).Find(&bds).Error; err != nil {
		return nil
	}
	today := startOfDay(now)
	end := today.AddDate(0, 0, days-1)
	var out []birthdayItem
	for _, b := range bds {
		next := nextBirthdayDate(b, today)
		if !next.Before(today) && !next.After(end) {
			out = append(out, birthdayItem{
				BirthdayID: b.ID,
				Name:       b.Name,
				Gender:     b.Gender,
				BirthYear:  b.BirthYear,
				BirthMonth: b.BirthMonth,
				BirthDay:   b.BirthDay,
				Date:       next,
				Age:        upcomingAge(b, next),
			})
		}
	}
	// 任务7：加载标签
	s.fillTags(userID, out)
	return out
}

// fillTasks 加载生日条目的标签（任务7：用于循环变量 {tags_N}）
func (s *Scheduler) fillTags(userID uint, items []birthdayItem) {
	if len(items) == 0 {
		return
	}
	bids := make([]uint, 0, len(items))
	for _, it := range items {
		bids = append(bids, it.BirthdayID)
	}
	var bts []models.BirthdayTag
	s.db.Where("birthday_id IN ?", bids).Find(&bts)
	if len(bts) == 0 {
		return
	}
	tagIDSet := make(map[uint]struct{})
	for _, bt := range bts {
		tagIDSet[bt.TagID] = struct{}{}
	}
	tagIDs := make([]uint, 0, len(tagIDSet))
	for id := range tagIDSet {
		tagIDs = append(tagIDs, id)
	}
	var tags []models.Tag
	s.db.Where("id IN ? AND user_id = ?", tagIDs, userID).Find(&tags)
	tagMap := make(map[uint]models.Tag, len(tags))
	for _, t := range tags {
		tagMap[t.ID] = t
	}
	bidTagsMap := make(map[uint][]models.Tag, len(items))
	for _, bt := range bts {
		if t, ok := tagMap[bt.TagID]; ok {
			bidTagsMap[bt.BirthdayID] = append(bidTagsMap[bt.BirthdayID], t)
		}
	}
	for i := range items {
		items[i].Tags = bidTagsMap[items[i].BirthdayID]
		if items[i].Tags == nil {
			items[i].Tags = []models.Tag{}
		}
	}
}

// startOfDay 返回当天 00:00 时刻
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// nextBirthdayDate 计算从 today 起下一次生日的日期。
func nextBirthdayDate(b models.Birthday, today time.Time) time.Time {
	year := today.Year()
	t := time.Date(year, time.Month(b.BirthMonth), b.BirthDay, 0, 0, 0, 0, today.Location())
	if t.Before(today) {
		t = time.Date(year+1, time.Month(b.BirthMonth), b.BirthDay, 0, 0, 0, 0, today.Location())
	}
	return t
}

// upcomingAge 计算下一次生日时的年龄（年份未知返回 -1）。
func upcomingAge(b models.Birthday, next time.Time) int {
	if b.BirthYear == 0 {
		return -1
	}
	return next.Year() - b.BirthYear
}

// sendEmail 渲染模板并发送提醒邮件给用户。
// 任务7：使用 [Start loop]/[End loop] 循环块语法
// 循环变量：{birthday_person_N}、{birthday_date_N}、{birthday_age_N}、{birthday_sex_N}、{tags_N}
// 静态变量：{user}、{count}
func (s *Scheduler) sendEmail(u *models.User, entries []birthdayItem) error {
	var smtp models.SmtpSetting
	if err := s.db.First(&smtp, 1).Error; err != nil {
		return fmt.Errorf("load smtp: %w", err)
	}
	if smtp.Host == "" || smtp.Username == "" || smtp.Password == "" {
		return fmt.Errorf("%s", i18n.T(u.Language, "validation.smtpNotConfigured"))
	}
	var tpl models.EmailTemplate
	if err := s.db.First(&tpl, 1).Error; err != nil {
		return fmt.Errorf("load email template: %w", err)
	}

	// 任务7：构建循环变量（每人一份），代词按用户语言取自 i18n
	persons := make([]email.PersonVars, 0, len(entries))
	for _, e := range entries {
		var ageStr string
		if e.Age >= 0 {
			ageStr = strconv.Itoa(e.Age)
		} else {
			ageStr = "?"
		}
		// 合并标签名
		tagNames := make([]string, 0, len(e.Tags))
		for _, t := range e.Tags {
			tagNames = append(tagNames, t.Name)
		}
		persons = append(persons, email.PersonVars{
			Name: e.Name,
			Date: e.Date.Format("2006-01-02"),
			Age:  ageStr,
			Sex:  pronounForLanguage(u.Language, e.Gender),
			Tags: strings.Join(tagNames, ", "),
		})
	}

	// 静态变量（仅保留新模板所需的非循环变量）
	staticVars := map[string]string{
		"user":  u.Username,
		"count": strconv.Itoa(len(entries)),
	}
	// 任务7：使用支持循环块的渲染器
	subject := email.RenderTemplateLoop(tpl.Subject, staticVars, persons)
	body := email.RenderTemplateLoop(tpl.Body, staticVars, persons)
	return s.sender.Send(smtp, u.Email, subject, body)
}

// pronounForLanguage 按语言返回性别代词字符串
// 任务7：{birthday_sex_N} 变量值
//   - 中文：男→他，女→她，未知→TA
//   - 英文：男→his，女→her，未知→their
func pronounForLanguage(lang, gender string) string {
	switch gender {
	case models.GenderMale:
		return i18n.T(lang, "email.pronounMale")
	case models.GenderFemale:
		return i18n.T(lang, "email.pronounFemale")
	default:
		return i18n.T(lang, "email.pronounUnknown")
	}
}
