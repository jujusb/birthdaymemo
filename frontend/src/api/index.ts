import { get, post, put, del, postRaw, getRaw } from './client'
import type {
  User,
  Birthday,
  BirthdayWithTag,
  Tag,
  ShareLink,
  TagGrant,
  PublicShareData,
  ReminderSetting,
  PdfSetting,
  PdfRequest,
  PresetFont,
  SmtpSetting,
  EmailTemplate,
  SystemConfig,
  OperationLog,
  CalendarMonthData,
  CalendarYearData,
  LoginData,
  CaptchaData,
  LanguagesData,
  I18nData,
  SettingsData,
  EmailStatus,
  UpcomingBirthday,
  PdfPreviewData,
} from './types'

// --- 认证 ---
export const login = (username: string, password: string, captcha_id?: string, captcha_code?: string) =>
  post<LoginData>('/api/login', { username, password, captcha_id, captcha_code })

export const logout = () => post<void>('/api/logout')
export const getMe = () => get<User>('/api/me')
export const changePassword = (old_password: string, new_password: string) =>
  post<void>('/api/change-password', { old_password, new_password })
export const getCaptcha = () => get<CaptchaData>('/api/captcha')

// --- 语言/i18n ---
export const getLanguages = () => get<LanguagesData>('/api/languages')
export const getI18n = (lang: string) => get<I18nData>(`/api/i18n?lang=${encodeURIComponent(lang)}`)

// --- 顶栏 / 日历 ---
export const getUpcoming = () => get<UpcomingBirthday[]>('/api/upcoming')
export const getCalendarMonth = (year: number, month: number) =>
  get<CalendarMonthData>(`/api/calendar?view=month&year=${year}&month=${month}`)
export const getCalendarYear = (year: number) =>
  get<CalendarYearData>(`/api/calendar?view=year&year=${year}`)

// --- 生日 ---
export const listBirthdays = (opts?: { tag_ids?: number[] }) => {
  const params = new URLSearchParams()
  if (opts?.tag_ids && opts.tag_ids.length > 0) {
    params.set('tag_ids', opts.tag_ids.join(','))
  }
  const qs = params.toString()
  return get<BirthdayWithTag[]>(qs ? `/api/birthdays?${qs}` : '/api/birthdays')
}
export const createBirthday = (b: Partial<Birthday>) => post<Birthday>('/api/birthdays', b)
export const updateBirthday = (id: number, b: Partial<Birthday>) =>
  put<Birthday>(`/api/birthdays/${id}`, b)
export const deleteBirthday = (id: number) => del<void>(`/api/birthdays/${id}`)

// --- 标签 ---
export const listTags = () => get<Tag[]>('/api/tags')
export const createTag = (name: string, color: string) => post<Tag>('/api/tags', { name, color })
export const updateTag = (id: number, name: string, color: string) =>
  put<Tag>(`/api/tags/${id}`, { name, color })
export const deleteTag = (id: number) => del<void>(`/api/tags/${id}`)

// --- 用户设置 ---
export const getSettings = () => get<SettingsData>('/api/settings')
export const updateSettings = (s: {
  reminder: ReminderSetting
  language: string
  theme: 'dark' | 'light'
  theme_color?: string
  topbar_range_days: number
  self_birth_year?: number
  self_birth_month?: number
  self_birth_day?: number
}) => put<void>('/api/settings', s)

// --- 邮箱确认流程 ---
export const getEmailStatus = () => get<EmailStatus>('/api/settings/email/status')
export const requestEmailConfirm = (email: string) =>
  post<EmailStatus>('/api/settings/email/confirm', { email })

// --- PDF 设置 ---
export const getPdfSettings = () => get<PdfSetting>('/api/pdf-settings')
export const updatePdfSettings = (p: PdfSetting) => put<PdfSetting>('/api/pdf-settings', p)
export const listPresetFonts = () => get<PresetFont[]>('/api/pdf/fonts')
export const pdfPreview = (req: PdfRequest) => post<PdfPreviewData>('/api/pdf/preview', req)
export const pdfExport = (req: PdfRequest) => postRaw('/api/pdf/export', req)

// --- 只读分享链接（owner 管理） ---
export const listShareLinks = () => get<ShareLink[]>('/api/share-links')
export const createShareLink = (b: {
  name: string
  scope_mode: 'all' | 'tags'
  tag_ids?: number[]
  expires_at?: string | null
  slug?: string
}) => post<ShareLink>('/api/share-links', b)
export const updateShareLink = (
  id: number,
  b: { slug?: string; name?: string; scope_mode?: 'all' | 'tags'; tag_ids?: number[] },
) => put<ShareLink>(`/api/share-links/${id}`, b)
export const deleteShareLink = (id: number) => del<void>(`/api/share-links/${id}`)
export const rotateShareLink = (id: number) => post<ShareLink>(`/api/share-links/${id}/rotate`)

// --- 标签委托授权 ---
export const listGrants = (type?: 'owned' | 'received') =>
  get<TagGrant[]>(type ? `/api/grants?type=${type}` : '/api/grants')
export const createGrant = (grantee_username: string, tag_id: number, permission: 'view' | 'edit') =>
  post<TagGrant>('/api/grants', { grantee_username, tag_id, permission })
export const updateGrant = (id: number, permission: 'view' | 'edit') =>
  put<TagGrant>(`/api/grants/${id}`, { permission })
export const deleteGrant = (id: number) => del<void>(`/api/grants/${id}`)

// --- 公开分享（免登录） ---
export const getPublicShare = (token: string) =>
  get<PublicShareData>(`/api/public/s/${encodeURIComponent(token)}`)
export const getPublicShareCalendarMonth = (token: string, year: number, month: number) =>
  get<CalendarMonthData>(
    `/api/public/s/${encodeURIComponent(token)}/calendar?view=month&year=${year}&month=${month}`,
  )
export const getPublicShareCalendarYear = (token: string, year: number) =>
  get<CalendarYearData>(
    `/api/public/s/${encodeURIComponent(token)}/calendar?view=year&year=${year}`,
  )

// --- 公开分享的 PDF 导出（免登录，仅限分享范围内的数据；只读/纯生成） ---
export const getPublicPdfFonts = () => get<PresetFont[]>('/api/public/pdf/fonts')
export const getPublicSharePdfSettings = (token: string) =>
  get<PdfSetting>(`/api/public/s/${encodeURIComponent(token)}/pdf-settings`)
export const publicSharePdfExport = (token: string, req: PdfRequest) =>
  postRaw(`/api/public/s/${encodeURIComponent(token)}/pdf/export`, req)

// --- 管理员：用户 ---
export const listUsers = () => get<User[]>('/api/admin/users')
export const createUser = (username: string, password: string, role?: 'user' | 'guest' | 'admin') =>
  post<User>('/api/admin/users', { username, password, role })
export const deleteUser = (id: number) => del<void>(`/api/admin/users/${id}`)
export const resetUserPassword = (id: number, password: string) =>
  post<void>(`/api/admin/users/${id}/reset-password`, { password })

// --- 管理员：系统配置 ---
export const getSystemConfig = () => get<SystemConfig>('/api/admin/system-config')
export const updateSystemConfig = (c: SystemConfig) =>
  put<{ need_restart: boolean }>('/api/admin/system-config', c)

// --- 管理员：SMTP ---
export const getSmtp = () => get<SmtpSetting>('/api/admin/smtp')
export const updateSmtp = (s: Partial<SmtpSetting> & { password?: string }) =>
  put<SmtpSetting>('/api/admin/smtp', s)
export const testSmtp = () => post<void>('/api/admin/smtp/test')

// --- 管理员：邮件模板 ---
export const getEmailTemplate = () => get<EmailTemplate>('/api/admin/email-template')
export const updateEmailTemplate = (
  subject: string,
  body: string,
  selfAdvanceSubject: string,
  selfAdvanceBody: string,
  selfTodaySubject: string,
  selfTodayBody: string,
) =>
  put<EmailTemplate>('/api/admin/email-template', {
    subject,
    body,
    self_advance_subject: selfAdvanceSubject,
    self_advance_body: selfAdvanceBody,
    self_today_subject: selfTodaySubject,
    self_today_body: selfTodayBody,
  })

// --- 管理员：语言 ---
export const adminLanguages = () => get<{ available: any[] }>('/api/admin/languages')
export const scanLanguages = () =>
  post<{ available: any[]; errors: string[] }>('/api/admin/languages/scan')
export const validateLanguageFile = (content: string, code: string) =>
  post<void>('/api/admin/languages/validate', { content, code })
export const exportSampleLanguage = () => getRaw('/api/admin/languages/sample')

// --- 管理员：操作日志 ---
export interface OperationLogFilter {
  users?: string[]
  actions?: string[]
  detail?: string
  ip?: string
}
export const listOperationLogs = (page = 1, size = 50, filter: OperationLogFilter = {}) => {
  const params = new URLSearchParams()
  params.set('page', String(page))
  params.set('size', String(size))
  if (filter.users && filter.users.length > 0) params.set('users', filter.users.join(','))
  if (filter.actions && filter.actions.length > 0) params.set('actions', filter.actions.join(','))
  if (filter.detail) params.set('detail', filter.detail)
  if (filter.ip) params.set('ip', filter.ip)
  return get<{ logs: OperationLog[]; total: number; page: number; size: number }>(
    `/api/admin/operation-logs?${params.toString()}`,
  )
}

// --- 管理员：日志文件 ---
export interface LogFileInfo {
  name: string
  size: number
  mod_time: string
}
export interface LogFileContent {
  name: string
  content: string
  size: number
}
export const listLogFiles = () => get<LogFileInfo[]>('/api/admin/log-files')
export const getLogFile = (name: string, tail?: number) => {
  const qs = tail && tail > 0 ? `?tail=${tail}` : ''
  return get<LogFileContent>(`/api/admin/log-files/${encodeURIComponent(name)}${qs}`)
}
