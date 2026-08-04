// 主题色预设（10 种）
// 每个预设包含亮/暗两种模式的 primary 与 primary-hover 颜色

export interface ThemeColorDef {
  code: string
  // 亮色模式（light）
  light: { primary: string; primaryHover: string }
  // 暗色模式（dark）
  dark: { primary: string; primaryHover: string }
}

export const THEME_COLORS: ThemeColorDef[] = [
  {
    code: 'blue',
    light: { primary: '#4A90D9', primaryHover: '#3a7bc8' },
    dark: { primary: '#5fa8e8', primaryHover: '#6fb3f0' },
  },
  {
    code: 'purple',
    light: { primary: '#9C27B0', primaryHover: '#7b1fa2' },
    dark: { primary: '#BA68C8', primaryHover: '#CE93D8' },
  },
  {
    code: 'pink',
    light: { primary: '#E91E63', primaryHover: '#C2185B' },
    dark: { primary: '#F06292', primaryHover: '#F48FB1' },
  },
  {
    code: 'red',
    light: { primary: '#F44336', primaryHover: '#D32F2F' },
    dark: { primary: '#EF5350', primaryHover: '#E57373' },
  },
  {
    code: 'orange',
    light: { primary: '#FF9800', primaryHover: '#F57C00' },
    dark: { primary: '#FFB74D', primaryHover: '#FFCC80' },
  },
  {
    code: 'amber',
    light: { primary: '#FFC107', primaryHover: '#FFA000' },
    dark: { primary: '#FFCA28', primaryHover: '#FFD54F' },
  },
  {
    code: 'green',
    light: { primary: '#4CAF50', primaryHover: '#388E3C' },
    dark: { primary: '#66BB6A', primaryHover: '#81C784' },
  },
  {
    code: 'teal',
    light: { primary: '#009688', primaryHover: '#00796B' },
    dark: { primary: '#4DB6AC', primaryHover: '#80CBC4' },
  },
  {
    code: 'cyan',
    light: { primary: '#00BCD4', primaryHover: '#0097A7' },
    dark: { primary: '#4DD0E1', primaryHover: '#80DEEA' },
  },
  {
    code: 'indigo',
    light: { primary: '#3F51B5', primaryHover: '#303F9F' },
    dark: { primary: '#5C6BC0', primaryHover: '#7986CB' },
  },
]

export const DEFAULT_THEME_COLOR = 'blue'

export function getThemeColor(code: string): ThemeColorDef {
  return THEME_COLORS.find((c) => c.code === code) ?? THEME_COLORS[0]
}

// 应用主题色：根据 theme 与 themeColor 设置 CSS 变量
export function applyThemeColor(theme: 'dark' | 'light', colorCode: string) {
  const def = getThemeColor(colorCode)
  const palette = theme === 'dark' ? def.dark : def.light
  const root = document.documentElement
  root.style.setProperty('--color-primary', palette.primary)
  root.style.setProperty('--color-primary-hover', palette.primaryHover)
}
