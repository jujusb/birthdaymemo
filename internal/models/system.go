package models

import "time"

// SMTP 加密方式
const (
	SMTPEncryptionNone     = "none"
	SMTPEncryptionTLS      = "tls"
	SMTPEncryptionStartTLS = "starttls"
)

// SmtpSetting SMTP 设置（系统级，单行）
type SmtpSetting struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Host       string    `json:"host" gorm:"size:100"`
	Port       int       `json:"port" gorm:"default:587"`
	Username   string    `json:"username" gorm:"size:100"`
	Password   string    `json:"-" gorm:"size:255"` // 不返回前端
	From       string    `json:"from" gorm:"size:100"`
	Encryption string    `json:"encryption" gorm:"size:10;default:starttls"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// HasPassword 用于前端判断是否已配置密码
func (s SmtpSetting) HasPassword() bool {
	return s.Password != ""
}

// EmailTemplate 提醒邮件模板（系统级，单行）
type EmailTemplate struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Subject string `json:"subject" gorm:"size:200;not null"`
	Body    string `json:"body" gorm:"type:text;not null"`
	// 任务5：本人生日提醒（提前 N 天）
	SelfAdvanceSubject string `json:"self_advance_subject" gorm:"size:200;not null;default:''"`
	SelfAdvanceBody    string `json:"self_advance_body" gorm:"type:text;not null;default:''"`
	// 任务5：本人生日提醒（当日）
	SelfTodaySubject string    `json:"self_today_subject" gorm:"size:200;not null;default:''"`
	SelfTodayBody    string    `json:"self_today_body" gorm:"type:text;not null;default:''"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// DefaultEmailTemplateSubject 默认邮件主题
const DefaultEmailTemplateSubject = "[BirthDayMemo] {user} 的生日提醒"

// DefaultEmailTemplateBody 默认邮件正文，支持变量:
// 静态变量: {user} 收件用户名, {count} 即将过生日的总人数
// 循环块: [Start loop] 与 [End loop] 之间的内容会按每个生日人重复一次
//
//	{birthday_person_N} 姓名, {birthday_date_N} 生日日期,
//	{birthday_age_N} 岁数（年份未知显示 ?）, {birthday_sex_N} 性别代词（他/她/TA）,
//	{tags_N} 同人标签合并（逗号分隔）
const DefaultEmailTemplateBody = `亲爱的 {user}：

这是一封来自 BirthDayMemo 的生日提醒邮件。

共有 {count} 位朋友即将过生日：
[Start loop]
您的{birthday_person_N}好友即将迎来{birthday_sex_N}的{birthday_age_N}岁生日 [ {birthday_date_N} ]
Tag：{tags_N}
[End loop]
请记得送上祝福！

— BirthDayMemo`

// DefaultSelfAdvanceSubject 本人生日提前提醒默认主题
const DefaultSelfAdvanceSubject = "[BirthDayMemo] {user}，你的生日即将到来"

// DefaultSelfAdvanceBody 本人生日提前提醒默认正文
// 支持变量: {user} 用户名, {self_birthday_date} 生日日期, {self_birthday_age} 岁数, {self_birthday_last} 剩余天数
const DefaultSelfAdvanceBody = `提前提醒
亲爱的 {user}，你好！

温馨提醒：你的 {self_birthday_age} 岁生日即将到来（{self_birthday_date}），距离现在仅剩 {self_birthday_last} 天啦！🎈

是时候准备庆祝计划了——挑选蛋糕、邀约好友、或者给自己安排一份特别的礼物。别忘了给家人打个电话，他们一定也在惦记着你。

提前祝生日快乐，愿新的一岁遇见更多美好与惊喜！✨

— BirthDayMemo`

// DefaultSelfTodaySubject 本人生日当日提醒默认主题
const DefaultSelfTodaySubject = "[BirthDayMemo] {user}，生日快乐！"

// DefaultSelfTodayBody 本人生日当日提醒默认正文
const DefaultSelfTodayBody = `🎉 生日快乐！今日正日 🎉

{user}，今天就是你的 {self_birthday_date} 啦！

恭喜你正式迈入 {self_birthday_age} 岁的新征程！🎂 这是属于你的一天，全世界都在为你让路。无论今天怎么安排，都请务必让自己开心——吃顿好的，做件喜欢的事，和在乎的人说说话。

岁月漫长，但你值得所有温柔与热烈。新一岁，继续闪闪发光吧！🌟

— BirthDayMemo`

// OperationLog 操作日志
type OperationLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index"`
	Username  string    `json:"username" gorm:"size:20"`
	Action    string    `json:"action" gorm:"size:50;not null"`
	Detail    string    `json:"detail" gorm:"type:text"`
	IP        string    `json:"ip" gorm:"size:45"`
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

// Setting 系统键值设置
type Setting struct {
	Key   string `json:"key" gorm:"primaryKey;size:50"`
	Value string `json:"value" gorm:"type:text"`
}

// SettingKey 支持的语言列表(JSON 数组)
const SettingKeySupportedLanguages = "supported_languages"

// LoginAttempt 登录失败记录（按 IP 统计，不针对用户名）
type LoginAttempt struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	IP          string    `json:"ip" gorm:"uniqueIndex;size:45"`
	FailCount   int       `json:"fail_count" gorm:"not null;default:0"` // 当前滑动窗口内失败次数
	WindowStart time.Time `json:"window_start"`                         // 当前窗口起始时间
	LockedUntil time.Time `json:"locked_until"`                         // IP 锁定截止时间（零值表示未锁定）
	UpdatedAt   time.Time `json:"updated_at"`                           // 最近一次失败时间，用于滑动窗口判断
}

// ReminderSent 提醒发送记录（按 SentKey 去重，避免重启后重复发送）
// SentKey 规则：
//   - 模式1（前N天）："d:YYYY-MM-DD"，每日一次
//   - 模式2周（每周）："w:YYYY-WW"，每周一次
//   - 模式2月（每月）："m:YYYY-MM"，每月一次
type ReminderSent struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"uniqueIndex:idx_user_sent_key;not null"`
	SentKey   string    `json:"sent_key" gorm:"uniqueIndex:idx_user_sent_key;size:30;not null"`
	CreatedAt time.Time `json:"created_at"`
}

// UserSession 持久化会话（数据库存储，支持服务器重启后保持登录）
type UserSession struct {
	Token        string    `json:"-" gorm:"primaryKey;size:64"`
	UserID       uint      `json:"user_id" gorm:"index;not null"`
	Username     string    `json:"username" gorm:"size:20"`
	Role         string    `json:"role" gorm:"size:10"`
	LastActivity time.Time `json:"last_activity"`
	CreatedAt    time.Time `json:"created_at"`
}

// EmailConfirmation 邮箱确认记录
// 用户设置邮箱时发送带 token 的确认邮件，点击确认链接后才正式生效
type EmailConfirmation struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	UserID      uint       `json:"user_id" gorm:"index;not null"`
	Email       string     `json:"email" gorm:"size:100;not null"` // 待确认邮箱
	Token       string     `json:"-" gorm:"uniqueIndex;size:64;not null"`
	ConfirmedAt *time.Time `json:"confirmed_at"`            // 确认时间，nil=未确认
	ExpiresAt   time.Time  `json:"expires_at"`              // 过期时间（24h）
	CreatedAt   time.Time  `json:"created_at" gorm:"index"` // 创建时间，用于冷却计数
}
