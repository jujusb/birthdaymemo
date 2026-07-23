// 通用 API 响应
export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

// 用户
export interface User {
  id: number
  username: string
  email: string
  role: 'admin' | 'user'
  must_change_password: boolean
  language: string
  theme: 'dark' | 'light'
  theme_color: string
  topbar_range_days: number
  created_at: string
  updated_at: string
}

// 生日记录
export interface Birthday {
  id: number
  user_id: number
  name: string
  gender: 'male' | 'female' | 'none'
  birth_year: number // 0 表示旧数据"未知"（前端表单已不再允许选"未知"，编辑时会回填为当前年）
  birth_month: number
  birth_day: number
  tag_ids?: number[] // 多标签 ID 列表
  created_at: string
  updated_at: string
}

// 标签
export interface Tag {
  id: number
  user_id: number
  name: string
  color: string
  created_at: string
  updated_at: string
}

// 提醒设置
export interface ReminderSetting {
  id: number
  user_id: number
  mode: 1 | 2 // 1=到期前N天 2=周期
  sub_mode: 'weekly' | 'monthly'
  days_before: number
  remind_hour: number
  weekly_day: number // 0=周日..6=周六
  monthly_day: number
  email_enabled: boolean
}

// PDF 设置
export interface PdfSetting {
  id: number
  user_id: number
  // 上方文字
  title_text: string
  subtitle_text: string
  // 背景
  background_type: 'white' | 'solid' | 'gradient' | 'image'
  background_color: string
  gradient_start: string
  gradient_end: string
  blur: number
  table_effect: 'none' | 'acrylic' | 'frosted'
  table_opacity: number
  table_bg_color: string
  table_border_enabled: boolean
  table_border_color: string
  table_border_opacity: number
  cell_effect: 'none' | 'acrylic' | 'frosted'
  cell_opacity: number
  cell_bg_color: string
  cell_border_enabled: boolean
  cell_border_color: string
  cell_border_opacity: number
  text_color: string
}

// PDF 预设字体
export interface PresetFont {
  key: string
  name: string
  style: string
  family: string
}

// SMTP 设置
export interface SmtpSetting {
  id: number
  host: string
  port: number
  username: string
  from: string
  encryption: 'none' | 'tls' | 'starttls'
  has_password: boolean
}

// 邮件模板
export interface EmailTemplate {
  id: number
  subject: string
  body: string
  self_advance_subject: string
  self_advance_body: string
  self_today_subject: string
  self_today_body: string
  updated_at: string
}

// 系统配置
export interface SystemConfig {
  language: string
  listen_address: string
  listen_port: number
  operation_log_retention_days: number
  external_url: string
}

// 操作日志
export interface OperationLog {
  id: number
  user_id: number
  username: string
  action: string
  detail: string
  ip: string
  created_at: string
}

// 日历
export interface CalendarDayBirthday {
  id: number
  name: string
  gender: 'male' | 'female' | 'none'
  color: string
  day: number
  age: number
  upcoming_age: number
  tags: Tag[]
}

export interface CalendarMonthData {
  year: number
  month: number
  first_weekday: number // 0=周日
  days_in_month: number
  birthdays: CalendarDayBirthday[] // 按 day 分组在前端做
  self_birthday_day: number
}

export interface CalendarYearMonth {
  month: number
  count: number
  genders: { male: number; female: number; none: number }
}

export interface CalendarYearData {
  year: number
  months: CalendarYearMonth[]
}

// 登录响应
export interface LoginData {
  user: User
  must_change_password: boolean
  needs_captcha: boolean
  locked: boolean
  locked_until: string | null
}

// 验证码
export interface CaptchaData {
  id: string
  image: string // data:image/png;base64,...
}

// 语言
export interface Language {
  code: string
  name: string
}

export interface LanguagesData {
  available: Language[]
  default: string
}

export interface I18nData {
  lang: string
  strings: { [key: string]: string }
}

// 设置响应
export interface SettingsData {
  reminder: ReminderSetting
  language: string
  theme: 'dark' | 'light'
  theme_color: string
  email: string
  pending_email: string
  topbar_range_days: number
  available_languages: Language[]
  self_birth_year: number
  self_birth_month: number
  self_birth_day: number
}

// 邮箱确认状态
export interface EmailStatus {
  email: string
  pending_email: string
  status: 'verified' | 'pending' | 'none'
  cooldown_left: number
}

// 顶栏近期生日
export interface UpcomingBirthday {
  id: number
  name: string
  date: string // YYYY-MM-DD
  days_until: number
  upcoming_age: number
  gender: 'male' | 'female' | 'none'
  color: string
  tags: Tag[]
}

// PDF 预览/导出
// 注：year 不从前端传入，由后端使用 time.Now().Year() 自动填充
export interface PdfRange {
  type: 'month' | 'year'
  month?: number
}

export interface PdfRequest {
  range: PdfRange
  settings: PdfSetting
  // 预设字体 key（小请求）
  title_font_key?: string
  table_font_key?: string
  // 自定义上传字体（base64 TTF）
  title_font?: string
  table_font?: string
  bg_image?: string
  subtitle_image?: string
}

export interface PdfPreviewData {
  pdf: string // data:application/pdf;base64,...
}

// 生日列表带 tags（多标签）
export interface BirthdayWithTag extends Birthday {
  tags: Tag[]
}