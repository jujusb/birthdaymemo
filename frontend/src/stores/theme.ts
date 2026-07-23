import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { applyThemeColor, DEFAULT_THEME_COLOR } from './themes'

export type Theme = 'dark' | 'light'

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>('light')
  const themeColor = ref<string>(DEFAULT_THEME_COLOR)

  function applyTheme(t: Theme) {
    document.documentElement.setAttribute('data-theme', t)
    applyThemeColor(t, themeColor.value)
  }

  function setTheme(t: Theme) {
    theme.value = t
    applyTheme(t)
  }

  function setThemeColor(c: string) {
    themeColor.value = c
    applyTheme(theme.value)
  }

  function init(initial: Theme, color: string = DEFAULT_THEME_COLOR) {
    theme.value = initial
    themeColor.value = color || DEFAULT_THEME_COLOR
    applyTheme(initial)
  }

  watch(theme, (t) => applyTheme(t))
  watch(themeColor, () => applyTheme(theme.value))

  return { theme, themeColor, setTheme, setThemeColor, init }
})
