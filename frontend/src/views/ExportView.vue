<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
import * as api from '@/api'
import type { PdfSetting, PdfRange, PdfRequest, PresetFont, CalendarMonthData, CalendarDayBirthday } from '@/api/types'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useToast } from '@/composables/useToast'

// 分享模式：客人通过公开链接使用导出窗口（免登录，仅限分享范围的数据）
// 未传 shareToken 时为所有者模式（需登录，可保存设计）
const props = defineProps<{ shareToken?: string }>()
const isShare = computed(() => !!props.shareToken)
const router = useRouter()

// 旧版硬编码中文默认标题（与后端 models.DefaultPdfTitleText 一致）：
// 存量用户若从未改过标题，非中文 UI 下显示为本地化默认标题
const LEGACY_DEFAULT_TITLE = '[{month}] 当月 {count} 人过生日'

const i18n = useI18nStore()
const t = i18n.t
const toast = useToast()

const today = new Date()
const currentYear = today.getFullYear()
const monthKeyList = ['jan', 'feb', 'mar', 'apr', 'may', 'jun', 'jul', 'aug', 'sep', 'oct', 'nov', 'dec']
const monthNames = computed(() => monthKeyList.map((k) => t('calendar.' + k)))
// 预览表头星期：按 UI 语言本地化（与后端 PDF 一致，而非硬编码中文）
const dayKeyList = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat']
const dayNames = computed(() => dayKeyList.map((k) => t('calendar.' + k)))

// 渐变预设
interface GradientPreset {
  key: string
  start: string
  end: string
}
const gradientPresets: GradientPreset[] = [
  { key: 'sunset', start: '#FF6B6B', end: '#FFE66D' },
  { key: 'ocean', start: '#4A90D9', end: '#9C27B0' },
  { key: 'lavender', start: '#e0c3fc', end: '#8ec5fc' },
  { key: 'aurora', start: '#00C9FF', end: '#92FE9D' },
  { key: 'peach', start: '#FCE38A', end: '#F38181' },
  { key: 'mint', start: '#a8e063', end: '#56ab2f' },
  { key: 'sky', start: '#2980b9', end: '#6dd5fa' },
  { key: 'rose', start: '#ee9ca7', end: '#ffdde1' },
]

// 颜色色板
const colorSwatches = [
  '#ffffff', '#f5f7fa', '#e3f2fd', '#f3e5f5', '#fce4ec', '#fff8e1', '#e8f5e9', '#e0f7fa',
  '#eceff1', '#cfd8dc', '#b0bec5', '#90a4ae', '#78909c', '#546e7a', '#37474f', '#000000',
  '#e74c3c', '#ff7043', '#ff9800', '#ffc107', '#ffeb3b', '#cddc39', '#8bc34a', '#4caf50',
  '#2196f3', '#1976d2', '#0288d1', '#00bcd4', '#009688', '#26a69a', '#3f51b5', '#673ab7',
  '#9c27b0', '#ab47bc', '#e91e63', '#f06292', '#ec407a', '#d81b60', '#ad1457', '#880e4f',
]

// 默认设置（subtitle 固定，前端不可配置）
const settings = ref<PdfSetting>({
  id: 0,
  user_id: 0,
  title_text: '[{month}] 当月 {count} 人过生日',
  subtitle_text: '🎂BirthDayMemo🎂',
  background_type: 'white',
  background_color: '#ffffff',
  gradient_start: '#e0c3fc',
  gradient_end: '#8ec5fc',
  blur: 0,
  // 任务6：外框默认白色背景 + 黑色边框开启
  table_effect: 'none',
  table_opacity: 100,
  table_bg_color: '#ffffff',
  table_border_enabled: true,
  table_border_color: '#000000',
  table_border_opacity: 100,
  // 任务6：内格默认白色背景 + 黑色边框开启
  cell_effect: 'none',
  cell_opacity: 100,
  cell_bg_color: '#ffffff',
  cell_border_enabled: true,
  cell_border_color: '#000000',
  cell_border_opacity: 100,
  text_color: '#333333',
  show_age: false,
})

const presetFonts = ref<PresetFont[]>([])
const titleFontSel = ref<string>('_default')
const tableFontSel = ref<string>('_default')
const titleFontData = ref<string>('')
const tableFontData = ref<string>('')
const titleFontName = ref('')
const tableFontName = ref('')

const bgImageData = ref<string>('')
const bgImageName = ref('')

// 日期范围（指定月份 / 整年 / 学年：进行中的学年，9 月起 12 页）
const rangeType = ref<'month' | 'year' | 'school_year'>('month')
const rangeMonth = ref(today.getMonth() + 1)

// 进行中的学年起始年份：9 月及之后从当年 9 月起，否则从去年 9 月起（与后端一致）
const schoolStartYear = computed(() =>
  today.getMonth() + 1 >= 9 ? today.getFullYear() : today.getFullYear() - 1,
)
const schoolYearLabel = computed(() => {
  const y = schoolStartYear.value
  return `${y}/${y + 1}`
})

const saving = ref(false)
const downloading = ref(false)

// 预览数据：日历数据 + 脚注
interface PreviewCell {
  day: number
  inMonth: boolean
  birthdays: CalendarDayBirthday[]
  index?: number // 脚注索引 [N]
}
interface PreviewFootnote {
  index: number
  day: number
  names: string[]
}
interface PreviewMonth {
  year: number
  month: number
  title: string
  cells: PreviewCell[][] // 6行 × 7列
  footnotes: PreviewFootnote[]
}

const previewMonths = ref<PreviewMonth[]>([])
const previewLoading = ref(false)

const monthOptions = computed(() =>
  monthNames.value.map((name, i) => ({ value: i + 1, label: name })),
)

const currentRange = computed<PdfRange>(() => {
  // lang 告诉后端用哪种语言渲染星期表头/脚注/默认标题
  if (rangeType.value === 'school_year') return { type: 'school_year', lang: i18n.lang }
  if (rangeType.value === 'year') return { type: 'year', lang: i18n.lang }
  return { type: 'month', month: rangeMonth.value, lang: i18n.lang }
})

// 选中的渐变预设（用于高亮）
const selectedPresetKey = computed(() => {
  const p = gradientPresets.find(
    (x) => x.start.toLowerCase() === settings.value.gradient_start.toLowerCase() &&
              x.end.toLowerCase() === settings.value.gradient_end.toLowerCase(),
  )
  return p?.key ?? ''
})

function selectPreset(p: GradientPreset) {
  settings.value.gradient_start = p.start
  settings.value.gradient_end = p.end
}

// 颜色选择弹层
// 任务6：新增外框边框色、内格背景色、内格边框色
type ColorTarget = 'bg' | 'gradientStart' | 'gradientEnd' | 'tableBg' | 'tableBorder' | 'cellBg' | 'cellBorder' | 'text' | null
const openColorTarget = ref<ColorTarget>(null)

function toggleColorPicker(target: ColorTarget, e: MouseEvent) {
  e.stopPropagation()
  openColorTarget.value = openColorTarget.value === target ? null : target
}
function applyColor(c: string) {
  switch (openColorTarget.value) {
    case 'bg': settings.value.background_color = c; break
    case 'gradientStart': settings.value.gradient_start = c; break
    case 'gradientEnd': settings.value.gradient_end = c; break
    case 'tableBg':
      settings.value.table_bg_color = c
      // 任务6：白色背景默认开启黑色边框，其他默认不开启
      settings.value.table_border_enabled = c.toLowerCase() === '#ffffff'
      break
    case 'tableBorder': settings.value.table_border_color = c; break
    case 'cellBg':
      settings.value.cell_bg_color = c
      settings.value.cell_border_enabled = c.toLowerCase() === '#ffffff'
      break
    case 'cellBorder': settings.value.cell_border_color = c; break
    case 'text': settings.value.text_color = c; break
  }
}
function stopPropagation(e: MouseEvent) {
  e.stopPropagation()
}
function closeColorPicker() {
  openColorTarget.value = null
}
onMounted(() => document.addEventListener('mousedown', closeColorPicker))
onBeforeUnmount(() => document.removeEventListener('mousedown', closeColorPicker))

const currentColorValue = computed(() => {
  switch (openColorTarget.value) {
    case 'bg': return settings.value.background_color
    case 'gradientStart': return settings.value.gradient_start
    case 'gradientEnd': return settings.value.gradient_end
    case 'tableBg': return settings.value.table_bg_color
    case 'tableBorder': return settings.value.table_border_color
    case 'cellBg': return settings.value.cell_bg_color
    case 'cellBorder': return settings.value.cell_border_color
    case 'text': return settings.value.text_color
    default: return '#ffffff'
  }
})

const colorPickerTitle = computed(() => {
  switch (openColorTarget.value) {
    case 'bg': return t('export.bgSolid')
    case 'gradientStart': return t('export.gradientStart')
    case 'gradientEnd': return t('export.gradientEnd')
    case 'tableBg': return t('export.tableBgColor')
    case 'tableBorder': return t('export.tableBorderColor')
    case 'cellBg': return t('export.cellBgColor')
    case 'cellBorder': return t('export.cellBorderColor')
    case 'text': return t('export.textColor')
    default: return ''
  }
})

// 允许的图片 MIME 类型
const ALLOWED_IMAGE_TYPES = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp', 'image/gif', 'image/bmp']
const ALLOWED_IMAGE_EXTS = ['.jpg', '.jpeg', '.png', '.webp', '.gif', '.bmp']

function isImageFile(file: File): boolean {
  if (file.type && ALLOWED_IMAGE_TYPES.includes(file.type.toLowerCase())) return true
  const name = file.name.toLowerCase()
  return ALLOWED_IMAGE_EXTS.some((ext) => name.endsWith(ext))
}

function fileToDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

// 字体上传
async function onCustomFontUpload(target: 'title' | 'table', e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  try {
    const dataUrl = await fileToDataURL(f)
    const idx = dataUrl.indexOf('base64,')
    const base64 = idx >= 0 ? dataUrl.substring(idx + 7) : dataUrl
    if (target === 'title') {
      titleFontData.value = base64
      titleFontName.value = f.name
      titleFontSel.value = '_custom'
    } else {
      tableFontData.value = base64
      tableFontName.value = f.name
      tableFontSel.value = '_custom'
    }
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    ;(e.target as HTMLInputElement).value = ''
  }
}

function onFontSelChange(target: 'title' | 'table') {
  const sel = target === 'title' ? titleFontSel.value : tableFontSel.value
  if (sel !== '_custom') {
    if (target === 'title') { titleFontData.value = ''; titleFontName.value = '' }
    else { tableFontData.value = ''; tableFontName.value = '' }
  }
}

// ============ 图片裁剪 ============
const cropModal = ref<{
  show: boolean
  imgSrc: string
  imgWidth: number
  imgHeight: number
  dispWidth: number
  dispHeight: number
  rectX: number
  rectY: number
  rectW: number
  rectH: number
}>({
  show: false, imgSrc: '', imgWidth: 0, imgHeight: 0,
  dispWidth: 0, dispHeight: 0, rectX: 0, rectY: 0, rectW: 0, rectH: 0,
})

const cropImgRef = ref<HTMLImageElement | null>(null)
let dragState: { dragging: boolean; startX: number; startY: number; origX: number; origY: number } | null = null

const CROP_MAX_DISPLAY = 500
const A4_LANDSCAPE_RATIO = 297 / 210

async function onBgImage(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  if (!isImageFile(f)) {
    toast.error(t('validation.onlyImageAllowed'))
    ;(e.target as HTMLInputElement).value = ''
    return
  }
  try {
    const dataUrl = await fileToDataURL(f)
    const img = new Image()
    img.onload = () => {
      const maxW = CROP_MAX_DISPLAY
      const maxH = 380
      let dispW = img.naturalWidth
      let dispH = img.naturalHeight
      const scale = Math.min(maxW / dispW, maxH / dispH, 1)
      dispW = Math.round(dispW * scale)
      dispH = Math.round(dispH * scale)

      let rectW = dispW * 0.9
      let rectH = rectW / A4_LANDSCAPE_RATIO
      if (rectH > dispH * 0.95) {
        rectH = dispH * 0.95
        rectW = rectH * A4_LANDSCAPE_RATIO
      }
      const rectX = (dispW - rectW) / 2
      const rectY = (dispH - rectH) / 2

      cropModal.value = {
        show: true, imgSrc: dataUrl,
        imgWidth: img.naturalWidth, imgHeight: img.naturalHeight,
        dispWidth: dispW, dispHeight: dispH,
        rectX, rectY, rectW, rectH,
      }
    }
    img.onerror = () => toast.error(t('validation.onlyImageAllowed'))
    img.src = dataUrl
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    ;(e.target as HTMLInputElement).value = ''
  }
}

function onCropPointerDown(e: PointerEvent) {
  e.preventDefault()
  e.stopPropagation()
  const target = e.target as HTMLElement
  if (target.classList.contains('crop-rect') || target.classList.contains('crop-rect-inner')) {
    dragState = {
      dragging: true, startX: e.clientX, startY: e.clientY,
      origX: cropModal.value.rectX, origY: cropModal.value.rectY,
    }
    target.setPointerCapture?.(e.pointerId)
  }
}
function onCropPointerMove(e: PointerEvent) {
  if (!dragState?.dragging) return
  const dx = e.clientX - dragState.startX
  const dy = e.clientY - dragState.startY
  let newX = dragState.origX + dx
  let newY = dragState.origY + dy
  const maxX = cropModal.value.dispWidth - cropModal.value.rectW
  const maxY = cropModal.value.dispHeight - cropModal.value.rectH
  newX = Math.max(0, Math.min(maxX, newX))
  newY = Math.max(0, Math.min(maxY, newY))
  cropModal.value.rectX = newX
  cropModal.value.rectY = newY
}
function onCropPointerUp() {
  if (dragState) { dragState.dragging = false; dragState = null }
}
function cancelCrop() {
  cropModal.value.show = false
}

// 压缩图片到 1MB 以下
async function compressImageToUnder1MB(canvas: HTMLCanvasElement): Promise<string> {
  const MAX_BYTES = 950 * 1024 // 950KB 留余量
  // 先尝试 PNG（无损）
  let dataUrl = canvas.toDataURL('image/png')
  let base64 = dataUrl.split(',')[1] || ''
  let bytes = base64.length * 0.75
  if (bytes <= MAX_BYTES) return base64

  // PNG 太大，转 JPEG 逐级降质
  let quality = 0.92
  while (quality >= 0.4) {
    dataUrl = canvas.toDataURL('image/jpeg', quality)
    base64 = dataUrl.split(',')[1] || ''
    bytes = base64.length * 0.75
    if (bytes <= MAX_BYTES) return base64
    quality -= 0.1
  }
  // 最后兜底：缩小尺寸再试
  const scale = 0.7
  const c2 = document.createElement('canvas')
  c2.width = Math.round(canvas.width * scale)
  c2.height = Math.round(canvas.height * scale)
  const ctx2 = c2.getContext('2d')!
  ctx2.drawImage(canvas, 0, 0, c2.width, c2.height)
  dataUrl = c2.toDataURL('image/jpeg', 0.7)
  return (dataUrl.split(',')[1] || '')
}

async function applyCrop() {
  const cm = cropModal.value
  const scaleX = cm.imgWidth / cm.dispWidth
  const scaleY = cm.imgHeight / cm.dispHeight
  const srcX = Math.round(cm.rectX * scaleX)
  const srcY = Math.round(cm.rectY * scaleY)
  const srcW = Math.round(cm.rectW * scaleX)
  const srcH = Math.round(cm.rectH * scaleY)

  const canvas = document.createElement('canvas')
  canvas.width = srcW
  canvas.height = srcH
  const ctx = canvas.getContext('2d')
  if (!ctx) { toast.error('Canvas not supported'); return }
  const img = cropImgRef.value
  if (!img) { toast.error('Image not loaded'); return }
  ctx.drawImage(img, srcX, srcY, srcW, srcH, 0, 0, srcW, srcH)

  // 压缩到 1MB 以下
  try {
    const base64 = await compressImageToUnder1MB(canvas)
    bgImageData.value = base64
    bgImageName.value = `crop_${srcW}x${srcH}.png`
    cropModal.value.show = false
  } catch (err) {
    toast.error((err as Error).message)
  }
}

function clearBgImage() {
  bgImageData.value = ''
  bgImageName.value = ''
}

async function loadSettings() {
  try {
    // 分享模式：读取所有者的 PDF 设计作为初始值（只读，客人无法保存）
    const s = isShare.value
      ? await api.getPublicSharePdfSettings(props.shareToken as string)
      : await api.getPdfSettings()
    // 旧版中文默认值：非中文 UI 下替换为本地化默认标题（不自动保存，点保存后才持久化）
    let titleText = s.title_text
    if (!titleText || (titleText === LEGACY_DEFAULT_TITLE && i18n.lang !== 'zh')) {
      titleText = t('export.defaultTitle')
    }
    settings.value = { ...settings.value, ...s, title_text: titleText }
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

async function loadPresetFonts() {
  try {
    presetFonts.value = isShare.value ? await api.getPublicPdfFonts() : await api.listPresetFonts()
  } catch { /* 静默 */ }
}

// ============ 前端 HTML 预览（模拟 PDF 布局） ============
// 标题变量替换：{month} 使用英文月份缩写（JAN/FEB/...），与 PDF 后端一致
function applyTitleVars(text: string, month: number, count: number): string {
  const monthShort = monthKeyList[month - 1]?.toUpperCase() || ''
  return text
    .replace(/\{month\}/g, monthShort)
    .replace(/\{monthShort\}/g, monthShort)
    .replace(/\{monthNum\}/g, String(month))
    .replace(/\{count\}/g, String(count))
    .replace(/\{year\}/g, String(currentYear))
}

// 截断名字
function truncateName(name: string, maxLen: number): string {
  if (!name) return ''
  return name.length > maxLen ? name.substring(0, maxLen - 1) + '…' : name
}

// 性别色（与 PDF 后端 genderRGB 一致）
// male=#4A90D9 / female=#E91E63 / none=#9E9E9E
function genderColor(g: string): string {
  if (g === 'male') return '#4A90D9'
  if (g === 'female') return '#E91E63'
  return '#9E9E9E'
}

// 预览用的年龄后缀（如 " (35 · 1990)"），与后端 PDF 的 ageSuffix 对齐：
// 出生年份 = 页面年份 - 即将到的年龄；年份未知或未开启时返回空串
function previewAgeSuffix(b: CalendarDayBirthday, pageYear: number): string {
  if (!settings.value.show_age || b.upcoming_age <= 0) return ''
  const birthYear = pageYear - b.upcoming_age
  if (birthYear <= 0) return ''
  return ` (${b.upcoming_age} · ${birthYear})`
}

// 构建单月预览数据
function buildPreviewMonth(data: CalendarMonthData): PreviewMonth {
  const year = data.year
  const month = data.month
  // 按日分组
  const dayMap = new Map<number, CalendarDayBirthday[]>()
  for (const b of data.birthdays) {
    if (!dayMap.has(b.day)) dayMap.set(b.day, [])
    dayMap.get(b.day)!.push(b)
  }
  // 标题
  const count = data.birthdays.length
  const title = applyTitleVars(settings.value.title_text, month, count)

  // 6×7 网格
  const cells: PreviewCell[][] = []
  let footnoteCounter = 0
  const footnotes: PreviewFootnote[] = []
  for (let row = 0; row < 6; row++) {
    const rowCells: PreviewCell[] = []
    for (let col = 0; col < 7; col++) {
      const dayNum = row * 7 + col - data.first_weekday + 1
      const inMonth = dayNum >= 1 && dayNum <= data.days_in_month
      const birthdays = inMonth ? (dayMap.get(dayNum) || []) : []
      let index: number | undefined
      if (birthdays.length > 2) {
        footnoteCounter++
        index = footnoteCounter
        footnotes.push({
          index: footnoteCounter,
          day: dayNum,
          names: birthdays.map((b) => b.name + previewAgeSuffix(b, year)),
        })
      }
      rowCells.push({ day: dayNum, inMonth, birthdays, index })
    }
    cells.push(rowCells)
  }
  return { year, month, title, cells, footnotes }
}

// 加载预览数据（分享模式走公开日历接口，仅限分享范围）
async function loadPreviewData() {
  previewLoading.value = true
  try {
    const loadMonth = (y: number, m: number) =>
      isShare.value
        ? api.getPublicShareCalendarMonth(props.shareToken as string, y, m)
        : api.getCalendarMonth(y, m)
    // [year, month] 目标页：整年 1-12 月；学年为进行中学年的 9 月起 12 页
    const targets: Array<[number, number]> = []
    if (rangeType.value === 'school_year') {
      const sy = schoolStartYear.value
      for (let i = 0; i < 12; i++) {
        let m = 9 + i
        let y = sy
        if (m > 12) {
          m -= 12
          y++
        }
        targets.push([y, m])
      }
    } else if (rangeType.value === 'year') {
      for (let m = 1; m <= 12; m++) targets.push([currentYear, m])
    } else {
      targets.push([currentYear, rangeMonth.value])
    }
    const months: PreviewMonth[] = []
    for (const [y, m] of targets) {
      months.push(buildPreviewMonth(await loadMonth(y, m)))
    }
    previewMonths.value = months
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    previewLoading.value = false
  }
}

// 配置或日期范围变化时实时刷新预览数据
// 注：仅当日历数据未加载或月份切换时重新请求 API；配置变化（颜色/字体等）通过 computed 自动反映
watch([rangeType, rangeMonth], () => {
  loadPreviewData()
})

// 年龄开关影响预计算的脚注名单，切换时重建预览
watch(() => settings.value.show_age, () => {
  loadPreviewData()
})

// 标题文字变化时，更新已加载的预览标题（不重新请求 API）
watch(() => settings.value.title_text, () => {
  if (previewMonths.value.length === 0) return
  previewMonths.value = previewMonths.value.map((m) => {
    const count = m.cells.flat().reduce((sum, c) => sum + c.birthdays.length, 0)
    return { ...m, title: applyTitleVars(settings.value.title_text, m.month, count) }
  })
})

// 构建下载请求
function buildRequest(r: PdfRange): PdfRequest {
  const req: PdfRequest = {
    range: r,
    settings: settings.value,
    bg_image: bgImageData.value || undefined,
    // 任务5：副标题使用浏览器字体渲染的图片，避免 PDF 渲染 emoji 失败
    subtitle_image: subtitleImageBase64.value || undefined,
  }
  if (titleFontSel.value !== '_default' && titleFontSel.value !== '_custom') {
    req.title_font_key = titleFontSel.value
  } else if (titleFontSel.value === '_custom' && titleFontData.value) {
    req.title_font = titleFontData.value
  }
  if (tableFontSel.value !== '_default' && tableFontSel.value !== '_custom') {
    req.table_font_key = tableFontSel.value
  } else if (tableFontSel.value === '_custom' && tableFontData.value) {
    req.table_font = tableFontData.value
  }
  return req
}

onMounted(async () => {
  if (isShare.value) {
    // 分享页直达导出窗口时（如刷新），按浏览器语言初始化（与 ShareView 一致）
    try {
      const nav = (navigator.language || 'en').slice(0, 2)
      await i18n.init(nav === 'zh' ? 'zh' : 'en')
    } catch { /* ignore */ }
  }
  await Promise.all([loadSettings(), loadPresetFonts()])
  await loadPreviewData()
})

function goBackToShare() {
  router.push({ name: 'share', params: { token: props.shareToken } })
}

async function saveSettings() {
  saving.value = true
  try {
    const saved = await api.updatePdfSettings(settings.value)
    settings.value = { ...settings.value, ...saved }
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    saving.value = false
  }
}

async function download() {
  downloading.value = true
  try {
    const req = buildRequest(currentRange.value)
    const resp = isShare.value
      ? await api.publicSharePdfExport(props.shareToken as string, req)
      : await api.pdfExport(req)
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    const rt = currentRange.value.type
    const suffix = rt === 'year' ? 'year' : rt === 'school_year' ? 'school' : `month${currentRange.value.month}`
    a.download = `birthdays-${suffix}.pdf`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    downloading.value = false
  }
}

// ============ 预览样式（响应式） ============
const pageBgStyle = computed(() => {
  const s = settings.value
  switch (s.background_type) {
    case 'solid':
      return { background: s.background_color }
    case 'gradient':
      return { background: `linear-gradient(135deg, ${s.gradient_start}, ${s.gradient_end})` }
    case 'image':
      return bgImageData.value
        ? {
            backgroundImage: `url(data:image/png;base64,${bgImageData.value})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            filter: s.blur > 0 ? `blur(${s.blur}px)` : 'none',
          }
        : { background: '#ffffff' }
    default:
      return { background: '#ffffff' }
  }
})

// 任务6：外框/内格透明度独立计算
const outerAlpha = computed(() => Math.max(0, Math.min(1, settings.value.table_opacity / 100)))
const cellAlpha = computed(() => Math.max(0, Math.min(1, settings.value.cell_opacity / 100)))

const tableBgColor = computed(() => settings.value.table_bg_color)
const cellBgColor = computed(() => settings.value.cell_bg_color)
const textColor = computed(() => settings.value.text_color)

function rgba(hex: string, alpha: number): string {
  // #RRGGBB → rgba(r,g,b,a)
  const m = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  if (!m) return `rgba(255,255,255,${alpha})`
  const r = parseInt(m[1], 16)
  const g = parseInt(m[2], 16)
  const b = parseInt(m[3], 16)
  return `rgba(${r},${g},${b},${alpha})`
}

// 任务6：表格特效的 backdrop-filter — 外框/内格各自独立
// - acrylic（磨砂玻璃）: 强模糊 + 饱和度提升（Windows Acrylic 风格）
// - frosted（液态玻璃）: 轻模糊 + 亮度提升（参考 Liquid Glass Vue 多层叠加思路，见 CSS）
function effectFilter(effect: string): string {
  switch (effect) {
    case 'acrylic':
      return 'blur(16px) saturate(1.8) brightness(1.05)'
    case 'frosted':
      return 'blur(8px) brightness(1.1) contrast(1.05)'
    default:
      return 'none'
  }
}
const outerEffectFilter = computed(() => effectFilter(settings.value.table_effect))
const cellEffectFilter = computed(() => effectFilter(settings.value.cell_effect))

// 任务6：外框样式（背景 + 透明度 + 可选边框 + 特效）
// 任务3：液态玻璃特效通过类绑定实现（不在 style 中加非法属性）
const outerBgStyle = computed(() => {
  return {
    background: rgba(tableBgColor.value, outerAlpha.value),
    backdropFilter: outerEffectFilter.value,
    WebkitBackdropFilter: outerEffectFilter.value,
  } as Record<string, string>
})

// 任务6：内格样式（背景 + 透明度 + 可选边框 + 特效）
const cellBgStyle = computed(() => {
  return {
    background: rgba(cellBgColor.value, cellAlpha.value),
    backdropFilter: cellEffectFilter.value,
    WebkitBackdropFilter: cellEffectFilter.value,
  } as Record<string, string>
})

// 任务3：液态玻璃特效类（用于 CSS 多层叠加效果）
const outerGlassClass = computed(() => {
  if (settings.value.table_effect === 'frosted') return 'glass-frosted'
  if (settings.value.table_effect === 'acrylic') return 'glass-acrylic'
  return ''
})
const cellGlassClass = computed(() => {
  if (settings.value.cell_effect === 'frosted') return 'glass-frosted'
  if (settings.value.cell_effect === 'acrylic') return 'glass-acrylic'
  return ''
})

// 任务6：自动从背景提取主色（白底→白；纯色→纯色；渐变→起止色平均；图片→采样平均）
function parseHexToRgb(hex: string): [number, number, number] {
  const m = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  if (!m) return [255, 255, 255]
  return [parseInt(m[1], 16), parseInt(m[2], 16), parseInt(m[3], 16)]
}
function rgbToHex(r: number, g: number, b: number): string {
  const toHex = (n: number) => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, '0')
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`
}
function extractDominantColorSync(): string {
  const s = settings.value
  switch (s.background_type) {
    case 'solid':
      return s.background_color
    case 'gradient': {
      const [r1, g1, b1] = parseHexToRgb(s.gradient_start)
      const [r2, g2, b2] = parseHexToRgb(s.gradient_end)
      return rgbToHex((r1 + r2) / 2, (g1 + g2) / 2, (b1 + b2) / 2)
    }
    default:
      return '#ffffff'
  }
}
async function extractDominantColorFromImage(base64: string): Promise<string> {
  return new Promise((resolve) => {
    const img = new Image()
    img.onload = () => {
      try {
        const size = 40
        const canvas = document.createElement('canvas')
        canvas.width = size
        canvas.height = size
        const ctx = canvas.getContext('2d')!
        ctx.drawImage(img, 0, 0, size, size)
        const data = ctx.getImageData(0, 0, size, size).data
        let r = 0, g = 0, b = 0, count = 0
        for (let i = 0; i < data.length; i += 4) {
          r += data[i]
          g += data[i + 1]
          b += data[i + 2]
          count++
        }
        resolve(rgbToHex(r / count, g / count, b / count))
      } catch {
        resolve('#ffffff')
      }
    }
    img.onerror = () => resolve('#ffffff')
    img.src = `data:image/png;base64,${base64}`
  })
}

// 标记用户是否手动改过外框/内格颜色（改过后不再被自动提取覆盖）
let userTouchedTableBg = false
let userTouchedCellBg = false
function applyColorWithTrack(c: string) {
  switch (openColorTarget.value) {
    case 'tableBg':
      userTouchedTableBg = true
      break
    case 'cellBg':
      userTouchedCellBg = true
      break
  }
  applyColor(c)
}

// 背景变化时自动提取主色（仅未手动改过的字段会被更新）
watch(() => [
  settings.value.background_type,
  settings.value.background_color,
  settings.value.gradient_start,
  settings.value.gradient_end,
  bgImageData.value,
], async () => {
  let c: string
  if (settings.value.background_type === 'image' && bgImageData.value) {
    c = await extractDominantColorFromImage(bgImageData.value)
  } else {
    c = extractDominantColorSync()
  }
  if (!userTouchedTableBg) {
    settings.value.table_bg_color = c
    settings.value.table_border_enabled = c.toLowerCase() === '#ffffff'
  }
  if (!userTouchedCellBg) {
    settings.value.cell_bg_color = c
    settings.value.cell_border_enabled = c.toLowerCase() === '#ffffff'
  }
}, { deep: true })

// 任务5：副标题使用浏览器字体渲染为图片，发送给后端嵌入 PDF
// 这样预览和下载都使用浏览器字体，避免 fpdf 无法渲染 emoji 的问题
const subtitleImageBase64 = ref<string>('')
function renderSubtitleToImage(text: string, textColor: string): string {
  // 副标题区域：PDF 中是 277mm × 6mm 的盒子（与 pdf.go 中 innerW × 6 一致）
  const MM_TO_PX = 3.7795 // 96 DPI
  const SCALE = 2         // 2x 高清渲染
  const widthMm = 277
  const heightMm = 6
  const widthPx = Math.round(widthMm * MM_TO_PX * SCALE)
  const heightPx = Math.round(heightMm * MM_TO_PX * SCALE)
  const canvas = document.createElement('canvas')
  canvas.width = widthPx
  canvas.height = heightPx
  const ctx = canvas.getContext('2d')
  if (!ctx) return ''
  ctx.scale(SCALE, SCALE)
  // 使用浏览器默认字体栈（与 .preview-subtitle 的 font-family 一致）
  const fontSizePx = 4 * MM_TO_PX // 约 4mm 高字
  ctx.font = `${fontSizePx}px -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", "Hiragino Sans GB", sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillStyle = textColor
  ctx.fillText(text, widthMm * MM_TO_PX / 2, heightMm * MM_TO_PX / 2)
  return canvas.toDataURL('image/png').split(',')[1] || ''
}
watch([() => settings.value.subtitle_text, () => settings.value.text_color], () => {
  const text = settings.value.subtitle_text || 'BirthDayMemo'
  subtitleImageBase64.value = renderSubtitleToImage(text, settings.value.text_color)
}, { immediate: true })

// 任务6：外框边框样式（可选）
const outerBorderStyle = computed(() => {
  if (!settings.value.table_border_enabled) return {}
  return {
    borderColor: rgba(settings.value.table_border_color, Math.min(1, settings.value.table_border_opacity / 100)),
    borderWidth: '1.5px',
    borderStyle: 'solid',
  }
})

// 任务6：内格边框样式（可选）
const cellBorderStyle = computed(() => {
  if (!settings.value.cell_border_enabled) return {}
  return {
    borderColor: rgba(settings.value.cell_border_color, Math.min(1, settings.value.cell_border_opacity / 100)),
    borderWidth: '0.8px',
    borderStyle: 'solid',
  }
})

const cropRectStyle = computed(() => ({
  left: cropModal.value.rectX + 'px',
  top: cropModal.value.rectY + 'px',
  width: cropModal.value.rectW + 'px',
  height: cropModal.value.rectH + 'px',
}))

const cropImageDispStyle = computed(() => ({
  width: cropModal.value.dispWidth + 'px',
  height: cropModal.value.dispHeight + 'px',
}))

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}
</script>

<template>
  <div class="export-view">
    <div v-if="isShare" class="row mb-8">
      <button @click="goBackToShare">‹ {{ t('common.back') }}</button>
    </div>
    <h2 class="page-title">{{ t('export.title') }}</h2>

    <div class="export-layout">
      <!-- Left: config -->
      <section class="card config-panel">
        <!-- 上方文字（仅标题） -->
        <div class="section-block">
          <h3 class="section-title">{{ t('export.topText') }}</h3>
          <div class="col gap-6">
            <div class="col">
              <label class="lbl">{{ t('export.titleText') }}</label>
              <input v-model="settings.title_text" type="text" class="txt-input" />
              <span class="hint">{{ t('export.titleTextHint') }}</span>
            </div>
          </div>
        </div>

        <!-- 背景 -->
        <div class="section-block">
          <h3 class="section-title">{{ t('export.background') }}</h3>
          <select v-model="settings.background_type" class="txt-input">
            <option value="white">{{ t('export.bgWhite') }}</option>
            <option value="solid">{{ t('export.bgSolid') }}</option>
            <option value="gradient">{{ t('export.bgGradient') }}</option>
            <option value="image">{{ t('export.bgImage') }}</option>
          </select>

          <div v-if="settings.background_type === 'solid'" class="col mt-8">
            <label class="lbl">{{ t('export.bgSolid') }}</label>
            <div class="color-picker-wrap">
              <button class="color-chip" :style="{ background: settings.background_color }" @mousedown="toggleColorPicker('bg', $event)"></button>
              <span class="color-hex">{{ settings.background_color }}</span>
              <div v-if="openColorTarget === 'bg'" class="color-popover" @mousedown="stopPropagation">
                <div class="popover-title">{{ colorPickerTitle }}</div>
                <div class="swatch-grid">
                  <button v-for="c in colorSwatches" :key="c" class="swatch" :class="{ active: currentColorValue.toLowerCase() === c.toLowerCase() }" :style="{ background: c }" @click="applyColor(c)"></button>
                </div>
                <div class="custom-row">
                  <input type="color" :value="currentColorValue" class="custom-color-input" @input="applyColor(($event.target as HTMLInputElement).value)" />
                  <input type="text" :value="currentColorValue" class="hex-input" @change="applyColor(($event.target as HTMLInputElement).value)" />
                </div>
              </div>
            </div>
          </div>

          <div v-if="settings.background_type === 'gradient'" class="col mt-8 gap-6">
            <label class="lbl">{{ t('export.gradientPreset') }}</label>
            <div class="preset-grid">
              <button v-for="p in gradientPresets" :key="p.key" class="preset-btn" :class="{ active: selectedPresetKey === p.key }" :style="{ background: `linear-gradient(135deg, ${p.start}, ${p.end})` }" :title="t('export.preset' + capitalize(p.key))" @click="selectPreset(p)"></button>
            </div>
            <div class="color-row-wrap">
              <div class="color-item">
                <label class="lbl">{{ t('export.gradientStart') }}</label>
                <div class="color-chip-row">
                  <button class="color-chip" :style="{ background: settings.gradient_start }" @mousedown="toggleColorPicker('gradientStart', $event)"></button>
                  <span class="color-hex">{{ settings.gradient_start }}</span>
                </div>
              </div>
              <div class="color-item">
                <label class="lbl">{{ t('export.gradientEnd') }}</label>
                <div class="color-chip-row">
                  <button class="color-chip" :style="{ background: settings.gradient_end }" @mousedown="toggleColorPicker('gradientEnd', $event)"></button>
                  <span class="color-hex">{{ settings.gradient_end }}</span>
                </div>
              </div>
              <div v-if="openColorTarget === 'gradientStart' || openColorTarget === 'gradientEnd'" class="color-popover" @mousedown="stopPropagation">
                <div class="popover-title">{{ colorPickerTitle }}</div>
                <div class="swatch-grid">
                  <button v-for="c in colorSwatches" :key="c" class="swatch" :class="{ active: currentColorValue.toLowerCase() === c.toLowerCase() }" :style="{ background: c }" @click="applyColor(c)"></button>
                </div>
                <div class="custom-row">
                  <input type="color" :value="currentColorValue" class="custom-color-input" @input="applyColor(($event.target as HTMLInputElement).value)" />
                  <input type="text" :value="currentColorValue" class="hex-input" @change="applyColor(($event.target as HTMLInputElement).value)" />
                </div>
              </div>
            </div>
          </div>

          <div v-if="settings.background_type === 'image'" class="col mt-8 gap-6">
            <div class="row gap-8">
              <label class="upload-btn">{{ t('common.upload') }}
                <input type="file" accept="image/jpeg,image/png,image/webp,image/gif,image/bmp" @change="onBgImage" hidden />
              </label>
              <span class="filename">{{ bgImageName || t('export.noFontFile') }}</span>
              <button v-if="bgImageName" class="small danger" @click="clearBgImage">×</button>
            </div>
            <span class="hint">{{ t('export.bgImageHint') }}</span>
            <div v-if="bgImageData" class="bg-preview">
              <img :src="'data:image/png;base64,' + bgImageData" alt="bg preview" />
            </div>
            <div class="col">
              <label class="lbl">{{ t('export.blur') }}: {{ settings.blur }}</label>
              <input v-model.number="settings.blur" type="range" min="0" max="20" />
            </div>
          </div>
        </div>

        <!-- 任务6：表格背景（外框） — 框住所有日期格的大圆角框 -->
        <div class="section-block">
          <h3 class="section-title">{{ t('export.tableEffectOuter') }}</h3>
          <div class="col gap-6">
            <select v-model="settings.table_effect" class="txt-input">
              <option value="none">{{ t('export.effectNone') }}</option>
              <option value="acrylic">{{ t('export.effectAcrylic') }}</option>
              <option value="frosted">{{ t('export.effectFrosted') }}</option>
            </select>
            <div class="col">
              <label class="lbl">{{ t('export.opacity') }}: {{ settings.table_opacity }}%</label>
              <input v-model.number="settings.table_opacity" type="range" min="0" max="100" />
            </div>
            <div class="col">
              <label class="lbl">{{ t('export.tableBgColor') }}</label>
              <div class="color-picker-wrap">
                <button class="color-chip" :style="{ background: settings.table_bg_color }" @mousedown="toggleColorPicker('tableBg', $event)"></button>
                <span class="color-hex">{{ settings.table_bg_color }}</span>
                <div v-if="openColorTarget === 'tableBg'" class="color-popover" @mousedown="stopPropagation">
                  <div class="popover-title">{{ colorPickerTitle }}</div>
                  <div class="swatch-grid">
                    <button v-for="c in colorSwatches" :key="c" class="swatch" :class="{ active: currentColorValue.toLowerCase() === c.toLowerCase() }" :style="{ background: c }" @click="applyColorWithTrack(c)"></button>
                  </div>
                  <div class="custom-row">
                    <input type="color" :value="currentColorValue" class="custom-color-input" @input="applyColorWithTrack(($event.target as HTMLInputElement).value)" />
                    <input type="text" :value="currentColorValue" class="hex-input" @change="applyColorWithTrack(($event.target as HTMLInputElement).value)" />
                  </div>
                </div>
              </div>
            </div>
            <label class="check-opt">
              <input v-model="settings.table_border_enabled" type="checkbox" />
              {{ t('export.tableBorderEnabled') }}
            </label>
            <div v-if="settings.table_border_enabled" class="col">
              <label class="lbl">{{ t('export.tableBorderColor') }}</label>
              <div class="color-picker-wrap">
                <button class="color-chip" :style="{ background: settings.table_border_color }" @mousedown="toggleColorPicker('tableBorder', $event)"></button>
                <span class="color-hex">{{ settings.table_border_color }}</span>
                <div v-if="openColorTarget === 'tableBorder'" class="color-popover" @mousedown="stopPropagation">
                  <div class="popover-title">{{ colorPickerTitle }}</div>
                  <div class="swatch-grid">
                    <button v-for="c in colorSwatches" :key="c" class="swatch" :class="{ active: currentColorValue.toLowerCase() === c.toLowerCase() }" :style="{ background: c }" @click="applyColor(c)"></button>
                  </div>
                  <div class="custom-row">
                    <input type="color" :value="currentColorValue" class="custom-color-input" @input="applyColor(($event.target as HTMLInputElement).value)" />
                    <input type="text" :value="currentColorValue" class="hex-input" @change="applyColor(($event.target as HTMLInputElement).value)" />
                  </div>
                </div>
              </div>
              <label class="lbl mt-8">{{ t('export.tableBorderOpacity') }}: {{ settings.table_border_opacity }}%</label>
              <input v-model.number="settings.table_border_opacity" type="range" min="0" max="100" />
            </div>
          </div>
        </div>

        <!-- 任务6：日期表格（内格） — 每个日期的小方格 -->
        <div class="section-block">
          <h3 class="section-title">{{ t('export.tableEffectCell') }}</h3>
          <div class="col gap-6">
            <select v-model="settings.cell_effect" class="txt-input">
              <option value="none">{{ t('export.effectNone') }}</option>
              <option value="acrylic">{{ t('export.effectAcrylic') }}</option>
              <option value="frosted">{{ t('export.effectFrosted') }}</option>
            </select>
            <div class="col">
              <label class="lbl">{{ t('export.cellOpacity') }}: {{ settings.cell_opacity }}%</label>
              <input v-model.number="settings.cell_opacity" type="range" min="0" max="100" />
            </div>
            <div class="col">
              <label class="lbl">{{ t('export.cellBgColor') }}</label>
              <div class="color-picker-wrap">
                <button class="color-chip" :style="{ background: settings.cell_bg_color }" @mousedown="toggleColorPicker('cellBg', $event)"></button>
                <span class="color-hex">{{ settings.cell_bg_color }}</span>
                <div v-if="openColorTarget === 'cellBg'" class="color-popover" @mousedown="stopPropagation">
                  <div class="popover-title">{{ colorPickerTitle }}</div>
                  <div class="swatch-grid">
                    <button v-for="c in colorSwatches" :key="c" class="swatch" :class="{ active: currentColorValue.toLowerCase() === c.toLowerCase() }" :style="{ background: c }" @click="applyColorWithTrack(c)"></button>
                  </div>
                  <div class="custom-row">
                    <input type="color" :value="currentColorValue" class="custom-color-input" @input="applyColorWithTrack(($event.target as HTMLInputElement).value)" />
                    <input type="text" :value="currentColorValue" class="hex-input" @change="applyColorWithTrack(($event.target as HTMLInputElement).value)" />
                  </div>
                </div>
              </div>
            </div>
            <label class="check-opt">
              <input v-model="settings.cell_border_enabled" type="checkbox" />
              {{ t('export.cellBorderEnabled') }}
            </label>
            <div v-if="settings.cell_border_enabled" class="col">
              <label class="lbl">{{ t('export.cellBorderColor') }}</label>
              <div class="color-picker-wrap">
                <button class="color-chip" :style="{ background: settings.cell_border_color }" @mousedown="toggleColorPicker('cellBorder', $event)"></button>
                <span class="color-hex">{{ settings.cell_border_color }}</span>
                <div v-if="openColorTarget === 'cellBorder'" class="color-popover" @mousedown="stopPropagation">
                  <div class="popover-title">{{ colorPickerTitle }}</div>
                  <div class="swatch-grid">
                    <button v-for="c in colorSwatches" :key="c" class="swatch" :class="{ active: currentColorValue.toLowerCase() === c.toLowerCase() }" :style="{ background: c }" @click="applyColor(c)"></button>
                  </div>
                  <div class="custom-row">
                    <input type="color" :value="currentColorValue" class="custom-color-input" @input="applyColor(($event.target as HTMLInputElement).value)" />
                    <input type="text" :value="currentColorValue" class="hex-input" @change="applyColor(($event.target as HTMLInputElement).value)" />
                  </div>
                </div>
              </div>
              <label class="lbl mt-8">{{ t('export.cellBorderOpacity') }}: {{ settings.cell_border_opacity }}%</label>
              <input v-model.number="settings.cell_border_opacity" type="range" min="0" max="100" />
            </div>
          </div>
        </div>

        <!-- 文字颜色 -->
        <div class="section-block">
          <h3 class="section-title">{{ t('export.textColor') }}</h3>
          <div class="color-picker-wrap">
            <button class="color-chip" :style="{ background: settings.text_color }" @mousedown="toggleColorPicker('text', $event)"></button>
            <span class="color-hex">{{ settings.text_color }}</span>
            <div v-if="openColorTarget === 'text'" class="color-popover" @mousedown="stopPropagation">
              <div class="popover-title">{{ colorPickerTitle }}</div>
              <div class="swatch-grid">
                <button v-for="c in colorSwatches" :key="c" class="swatch" :class="{ active: currentColorValue.toLowerCase() === c.toLowerCase() }" :style="{ background: c }" @click="applyColor(c)"></button>
              </div>
              <div class="custom-row">
                <input type="color" :value="currentColorValue" class="custom-color-input" @input="applyColor(($event.target as HTMLInputElement).value)" />
                <input type="text" :value="currentColorValue" class="hex-input" @change="applyColor(($event.target as HTMLInputElement).value)" />
              </div>
            </div>
          </div>
        </div>

        <!-- 字体设置 -->
        <div class="section-block">
          <h3 class="section-title">{{ t('export.fontSettings') }}</h3>
          <span class="hint">{{ t('export.fontHint') }}</span>
          <div class="col gap-6 mt-8">
            <div class="col">
              <label class="lbl">{{ t('export.titleFont') }}</label>
              <select v-model="titleFontSel" class="txt-input" @change="onFontSelChange('title')">
                <option value="_default">{{ t('export.presetFontDefault') }}</option>
                <option v-for="f in presetFonts" :key="f.key" :value="f.key">{{ f.name }} ({{ f.style }})</option>
                <option value="_custom">{{ t('export.fontCustom') }}...</option>
              </select>
              <div v-if="titleFontSel === '_custom'" class="row gap-8 mt-8">
                <label class="upload-btn">{{ t('common.upload') }}
                  <input type="file" accept=".ttf,.otf,.ttc" @change="onCustomFontUpload('title', $event)" hidden />
                </label>
                <span class="filename">{{ titleFontName || t('export.noFontFile') }}</span>
              </div>
            </div>
            <div class="col">
              <label class="lbl">{{ t('export.tableFont') }}</label>
              <select v-model="tableFontSel" class="txt-input" @change="onFontSelChange('table')">
                <option value="_default">{{ t('export.presetFontDefault') }}</option>
                <option v-for="f in presetFonts" :key="f.key" :value="f.key">{{ f.name }} ({{ f.style }})</option>
                <option value="_custom">{{ t('export.fontCustom') }}...</option>
              </select>
              <div v-if="tableFontSel === '_custom'" class="row gap-8 mt-8">
                <label class="upload-btn">{{ t('common.upload') }}
                  <input type="file" accept=".ttf,.otf,.ttc" @change="onCustomFontUpload('table', $event)" hidden />
                </label>
                <span class="filename">{{ tableFontName || t('export.noFontFile') }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 日期选择 -->
        <div class="section-block">
          <h3 class="section-title">{{ t('export.dateRange') }}</h3>
          <div class="row gap-8 mb-8">
            <label class="radio-opt">
              <input v-model="rangeType" type="radio" value="month" />
              {{ t('export.specificMonth') }}
            </label>
            <label class="radio-opt">
              <input v-model="rangeType" type="radio" value="year" />
              {{ t('export.fullYear') }}
            </label>
            <label class="radio-opt">
              <input v-model="rangeType" type="radio" value="school_year" />
              {{ t('export.schoolYear') }} {{ schoolYearLabel }}
            </label>
          </div>
          <div v-if="rangeType === 'month'" class="row gap-8">
            <select v-model.number="rangeMonth" class="txt-input">
              <option v-for="opt in monthOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
            </select>
          </div>
          <label class="check-opt mt-8">
            <input v-model="settings.show_age" type="checkbox" />
            {{ t('export.showAge') }}
          </label>
          <span class="hint">{{ t('export.showAgeHint') }}</span>
        </div>

        <div class="row gap-8 actions">
          <button v-if="!isShare" class="primary" :disabled="saving" @click="saveSettings">
            {{ saving ? t('common.loading') : t('common.save') }}
          </button>
          <button :disabled="downloading" @click="download">
            {{ downloading ? t('common.loading') : t('export.download') }}
          </button>
        </div>
      </section>

      <!-- Right: HTML 模拟预览 -->
      <section class="card preview-panel">
        <div class="preview-header">
          <h3 class="section-title">{{ t('common.preview') }}</h3>
          <span v-if="previewLoading" class="hint">{{ t('common.loading') }}</span>
        </div>
        <div class="preview-scroll">
          <div v-for="(pm, idx) in previewMonths" :key="idx" class="preview-page">
            <!-- 背景层（仅这一层应用模糊，不影响内容） -->
            <div class="preview-bg-layer" :style="pageBgStyle"></div>
            <!-- 内容层（永远不模糊） -->
            <div class="preview-content">
              <!-- 标题 -->
              <div class="preview-title" :style="{ color: textColor }">{{ pm.title }}</div>
              <!-- 副标题（任务5：使用浏览器字体，与 PDF 一致） -->
              <div class="preview-subtitle" :style="{ color: textColor }">{{ settings.subtitle_text || 'BirthDayMemo' }}</div>
              <!-- 日历网格 -->
              <div class="calendar-wrap">
                <div class="calendar-outer" :class="outerGlassClass" :style="[outerBgStyle, outerBorderStyle]">
                  <!-- 表头 -->
                  <div class="calendar-header" :style="{ color: textColor }">
                    <div v-for="(wd, i) in dayNames" :key="i" class="header-cell">{{ wd }}</div>
                  </div>
                  <!-- 6×7 网格 -->
                  <div class="calendar-grid">
                    <template v-for="(row, ri) in pm.cells" :key="ri">
                      <div
                        v-for="(cell, ci) in row"
                        :key="ri + '-' + ci"
                        class="calendar-cell"
                        :class="[cell.inMonth ? cellGlassClass : '', { 'empty-cell': !cell.inMonth }]"
                        :style="cell.inMonth ? [cellBgStyle, cellBorderStyle] : null"
                      >
                        <template v-if="cell.inMonth">
                          <div class="cell-date" :style="{ color: textColor }">{{ cell.day }}</div>
                          <div v-if="cell.index" class="cell-index" :style="{ color: textColor }">[{{ cell.index }}]</div>
                          <div class="cell-names">
                            <div
                              v-for="(b, bi) in (cell.birthdays.length > 2 ? cell.birthdays.slice(0, 2) : cell.birthdays)"
                              :key="bi"
                              class="cell-name"
                              :style="{ color: textColor }"
                            >
                              <span class="cell-gender-dot" :style="{ background: genderColor(b.gender) }"></span>
                              <span class="cell-name-text">{{ truncateName(b.name, settings.show_age ? 4 : 6) + previewAgeSuffix(b, pm.year) }}</span>
                            </div>
                          </div>
                        </template>
                      </div>
                    </template>
                  </div>
                </div>
              </div>
              <!-- 脚注 -->
              <div v-if="pm.footnotes.length > 0" class="footnotes" :style="{ color: textColor }">
                <div v-for="fn in pm.footnotes" :key="fn.index" class="footnote-item">
                  {{ t('export.footnote', { index: fn.index, day: fn.day, names: fn.names.join('、') }) }}
                </div>
              </div>
            </div>
          </div>
          <div v-if="!previewLoading && previewMonths.length === 0" class="preview-empty">
            {{ t('common.preview') }}
          </div>
        </div>
      </section>
    </div>

    <!-- 图片裁剪模态框 -->
    <div v-if="cropModal.show" class="crop-backdrop" @click.self="cancelCrop">
      <div class="crop-modal">
        <h3 class="crop-title">{{ t('export.cropImage') }}</h3>
        <p class="crop-hint">{{ t('export.cropHint') }}</p>
        <div
          class="crop-canvas"
          :style="{ width: cropModal.dispWidth + 'px', height: cropModal.dispHeight + 'px' }"
          @pointerdown="onCropPointerDown"
          @pointermove="onCropPointerMove"
          @pointerup="onCropPointerUp"
          @pointercancel="onCropPointerUp"
        >
          <img ref="cropImgRef" :src="cropModal.imgSrc" class="crop-img" :style="cropImageDispStyle" draggable="false" />
          <div class="crop-mask-top" :style="{ height: cropModal.rectY + 'px' }"></div>
          <div class="crop-mask-bottom" :style="{ top: (cropModal.rectY + cropModal.rectH) + 'px' }"></div>
          <div class="crop-mask-left" :style="{ top: cropModal.rectY + 'px', height: cropModal.rectH + 'px', width: cropModal.rectX + 'px' }"></div>
          <div class="crop-mask-right" :style="{ top: cropModal.rectY + 'px', height: cropModal.rectH + 'px', left: (cropModal.rectX + cropModal.rectW) + 'px' }"></div>
          <div class="crop-rect" :style="cropRectStyle">
            <div class="crop-rect-inner"></div>
          </div>
        </div>
        <div class="crop-actions">
          <button @click="cancelCrop">{{ t('common.cancel') }}</button>
          <button class="primary" @click="applyCrop">{{ t('common.apply') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.export-view {
  display: flex;
  flex-direction: column;
}
.page-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 16px;
}
.export-layout {
  display: grid;
  grid-template-columns: 420px 1fr;
  gap: 16px;
  align-items: start;
}
.section-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 8px;
}
.config-panel {
  max-height: calc(100vh - 120px);
  overflow-y: auto;
  padding: 16px;
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.config-panel::-webkit-scrollbar {
  display: none;
}
.section-block {
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 14px;
  margin-bottom: 14px;
}
.section-block:last-of-type {
  border-bottom: none;
  margin-bottom: 0;
}
.hint {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-top: 4px;
}
.lbl {
  font-size: 13px;
  color: var(--color-text);
  margin-bottom: 4px;
  display: block;
}
.radio-opt {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--color-text);
  cursor: pointer;
  font-size: 14px;
}
.radio-opt input {
  width: auto;
}
.txt-input {
  width: 100%;
  padding: 6px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text);
  font-size: 14px;
}
.upload-btn {
  display: inline-flex;
  align-items: center;
  padding: 6px 14px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 14px;
  color: var(--color-text);
  background: var(--color-bg-elevated);
  white-space: nowrap;
  position: relative;
  overflow: hidden;
}
.upload-btn:hover {
  border-color: var(--color-primary);
}
.upload-btn input[type="file"] {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}
.filename {
  font-size: 12px;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 180px;
}
.bg-preview {
  margin-top: 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
  max-height: 120px;
}
.bg-preview img {
  width: 100%;
  display: block;
  object-fit: cover;
  max-height: 120px;
}
.small {
  font-size: 12px;
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background: var(--color-bg-elevated);
  color: var(--color-text);
  cursor: pointer;
}
.small:hover {
  border-color: var(--color-primary);
}

/* 颜色选择器 */
.color-picker-wrap {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
}
.color-chip {
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  flex-shrink: 0;
}
.color-chip:hover {
  border-color: var(--color-primary);
}
.color-hex {
  font-size: 12px;
  color: var(--color-text-muted);
  font-family: monospace;
}
.color-row-wrap {
  position: relative;
  display: flex;
  gap: 12px;
}
.color-item {
  flex: 1;
}
.color-chip-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.color-popover {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  z-index: 50;
  width: 248px;
  padding: 10px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.18);
}
.popover-title {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 8px;
}
.swatch-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 4px;
}
.swatch {
  width: 22px;
  height: 22px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 4px;
  cursor: pointer;
  padding: 0;
}
.swatch:hover {
  transform: scale(1.12);
  border-color: var(--color-primary);
}
.swatch.active {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary);
}
.custom-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--color-border);
}
.custom-color-input {
  width: 32px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  background: transparent;
}
.hex-input {
  flex: 1;
  padding: 4px 6px;
  font-size: 12px;
  font-family: monospace;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-sunken);
  color: var(--color-text);
}

/* 渐变预设 */
.preset-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 6px;
}
.preset-btn {
  height: 28px;
  border: 2px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  padding: 0;
}
.preset-btn:hover {
  border-color: var(--color-primary);
}
.preset-btn.active {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 1px var(--color-primary);
}

.actions {
  justify-content: flex-start;
  padding-top: 8px;
}

/* ============ HTML 模拟预览 ============ */
.preview-panel {
  position: sticky;
  top: 70px;
  min-height: 400px;
  max-height: calc(100vh - 90px);
  display: flex;
  flex-direction: column;
  padding: 16px;
}
.preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.preview-header .section-title {
  margin-bottom: 0;
}
.preview-scroll {
  flex: 1;
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.preview-scroll::-webkit-scrollbar {
  display: none;
}
.preview-empty {
  padding: 40px;
  text-align: center;
  color: var(--color-text-muted);
}
/* A4 横版比例 297:210 ≈ 1.414 */
/* container-type 必须在父容器上，cqw 单位才能正确引用此容器尺寸 */
.preview-page {
  width: 100%;
  aspect-ratio: 297 / 210;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  margin-bottom: 12px;
  position: relative;
  overflow: hidden;
  box-sizing: border-box;
  container-type: inline-size;
}
/* 背景层：独立一层，仅此层应用模糊滤镜，不影响内容 */
.preview-bg-layer {
  position: absolute;
  inset: 0;
  z-index: 0;
  background-repeat: no-repeat;
}
/* 内容层：永远不模糊，z-index 高于背景层 */
.preview-content {
  position: relative;
  z-index: 1;
  width: 100%;
  height: 100%;
  padding: 1.5%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}
.preview-title {
  text-align: center;
  font-size: 2.2cqw;
  font-weight: 700;
  margin-bottom: 0.4%;
}
.preview-subtitle {
  text-align: center;
  font-size: 1.4cqw;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  margin-bottom: 1%;
}
.calendar-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.calendar-outer {
  flex: 1;
  border-radius: 4px;
  padding: 0.8%;
  display: flex;
  flex-direction: column;
  gap: 0.5%;
  box-sizing: border-box;
}
.calendar-header {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 0.5%;
  margin-bottom: 0.5%;
}
.header-cell {
  text-align: center;
  font-size: 1.3cqw;
  font-weight: 600;
  padding: 0.4% 0;
}
.calendar-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  grid-template-rows: repeat(6, 1fr);
  gap: 0.5%;
}
.calendar-cell {
  border-radius: 2px;
  padding: 4%;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  overflow: hidden;
  min-width: 0;
}
/* 任务3：液态玻璃（Liquid Glass）多层叠加效果 — 参考 Liquid Glass Vue */
/* 通过 inset box-shadow 模拟"光晕边框"（lit bezel），不依赖 border 属性 */
/* 这样边框开关与液态玻璃效果可以独立控制 */
.glass-frosted {
  box-shadow:
    inset 0 1px 1px rgba(255, 255, 255, 0.5),
    inset 0 -1px 1px rgba(0, 0, 0, 0.06),
    0 1px 3px rgba(0, 0, 0, 0.08);
}
.glass-acrylic {
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.25),
    0 2px 8px rgba(0, 0, 0, 0.06);
}
/* 任务4：非当月日期不预留方格（保留网格位置但完全透明） */
.calendar-cell.empty-cell {
  background: transparent !important;
  border: none !important;
  backdrop-filter: none !important;
  -webkit-backdrop-filter: none !important;
  box-shadow: none !important;
  padding: 0;
}
.cell-date {
  font-size: 1.6cqw;
  font-weight: 700;
  text-align: center;
  margin-bottom: 2%;
}
.cell-index {
  position: absolute;
  top: 4%;
  right: 6%;
  font-size: 0.9cqw;
  font-weight: 500;
}
.cell-names {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 1px;
}
.cell-name {
  font-size: 1.1cqw;
  text-align: left;
  width: 100%;
  display: flex;
  align-items: center;
  gap: 4px;
  overflow: hidden;
}
.cell-gender-dot {
  width: 1.1cqw;
  height: 1.1cqw;
  min-width: 4px;
  min-height: 4px;
  border-radius: 50%;
  flex-shrink: 0;
}
.cell-name-text {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.footnotes {
  margin-top: 0.8%;
  font-size: 1cqw;
  line-height: 1.4;
}
.footnote-item {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 图片裁剪模态框 */
.crop-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}
.crop-modal {
  background: var(--color-bg-elevated);
  border-radius: var(--radius);
  padding: 16px;
  max-width: 90vw;
  max-height: 90vh;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.3);
}
.crop-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 4px;
}
.crop-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 12px;
}
.crop-canvas {
  position: relative;
  background: #000;
  overflow: hidden;
  touch-action: none;
  cursor: grab;
  user-select: none;
}
.crop-canvas:active {
  cursor: grabbing;
}
.crop-img {
  display: block;
  user-select: none;
  pointer-events: none;
  position: absolute;
  top: 0;
  left: 0;
}
.crop-mask-top,
.crop-mask-bottom,
.crop-mask-left,
.crop-mask-right {
  position: absolute;
  background: rgba(0, 0, 0, 0.5);
  pointer-events: none;
}
.crop-mask-top {
  top: 0;
  left: 0;
  right: 0;
}
.crop-mask-bottom {
  left: 0;
  right: 0;
  bottom: 0;
}
.crop-mask-left {
  top: 0;
  left: 0;
}
.crop-mask-right {
  top: 0;
  right: 0;
}
.crop-rect {
  position: absolute;
  border: 2px solid #fff;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.4);
  cursor: move;
  box-sizing: border-box;
}
.crop-rect-inner {
  width: 100%;
  height: 100%;
  border: 1px dashed rgba(255, 255, 255, 0.5);
}
.crop-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}

.mt-8 { margin-top: 8px; }
.mb-8 { margin-bottom: 8px; }
.gap-6 { gap: 6px; }
.gap-8 { gap: 8px; }
.row { display: flex; align-items: center; }
.col { display: flex; flex-direction: column; }
.grow { flex: 1 1 0; }
.check-opt {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--color-text);
  cursor: pointer;
  font-size: 14px;
}
.check-opt input {
  width: auto;
}

@media (max-width: 900px) {
  .export-layout {
    grid-template-columns: 1fr;
  }
  .preview-panel {
    position: static;
    max-height: none;
  }
}
</style>
