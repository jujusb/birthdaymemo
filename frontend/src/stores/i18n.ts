import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as api from '@/api'
import type { Language } from '@/api/types'

export const useI18nStore = defineStore('i18n', () => {
  const lang = ref<string>('en')
  const strings = ref<Record<string, string>>({})
  const available = ref<Language[]>([])
  //缓存各语言的 strings，避免第二次切回时 strings 不更新
  const stringsCache = new Map<string, Record<string, string>>()

  const t = computed(() => (key: string, vars?: Record<string, string | number>): string => {
    let s = strings.value[key] ?? key
    if (vars) {
      for (const [k, v] of Object.entries(vars)) {
        s = s.split(`{${k}}`).join(String(v))
      }
    }
    return s
  })

  async function loadLanguages() {
    const data = await api.getLanguages()
    available.value = data.available
    if (!lang.value || !available.value.find((l) => l.code === lang.value)) {
      lang.value = data.default
    }
  }

  async function load(l: string) {
    // 命中缓存：恢复 strings 引用（触发响应式更新）
    if (stringsCache.has(l)) {
      strings.value = stringsCache.get(l)!
      lang.value = l
      document.documentElement.lang = l
      return
    }
    const data = await api.getI18n(l)
    stringsCache.set(l, data.strings)
    strings.value = data.strings
    lang.value = data.lang
    document.documentElement.lang = l
  }

  async function init(initialLang: string) {
    await loadLanguages()
    const target = initialLang || lang.value
    await load(target)
  }

  return { lang, strings, available, t, loadLanguages, load, init }
})
