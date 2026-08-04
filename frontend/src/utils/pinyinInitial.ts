//名称搜索匹配工具（基于 pinyin-pro）
import { pinyin } from 'pinyin-pro'

export interface PinyinData {
  pinyin: string
  initials: string
}

// 缓存：名称 -> 拼音数据，避免重复计算同名
const pinyinCache = new Map<string, PinyinData>()

//预计算单个名称的拼音数据
export function precomputePinyin(name: string): PinyinData {
  if (!name) return { pinyin: '', initials: '' }
  const cached = pinyinCache.get(name)
  if (cached) return cached
  // pinyin-pro: pattern='first' 获取首字母，toneType='none' 去音调
  const initials = pinyin(name, { pattern: 'first', toneType: 'none', type: 'array' })
    .join('')
    .toLowerCase()
  const full = pinyin(name, { toneType: 'none', type: 'array' })
    .join('')
    .toLowerCase()
  const data: PinyinData = { pinyin: full, initials }
  pinyinCache.set(name, data)
  return data
}

export function matchWithPinyin(name: string, data: PinyinData, query: string): boolean {
  if (!query) return true
  if (name.toLowerCase().includes(query)) return true
  if (data.pinyin.includes(query)) return true
  if (data.initials.startsWith(query)) return true
  if (query.length === 1 && data.initials.charAt(0) === query) return true
  return false
}
