import { ref } from 'vue'

export type ToastType = 'info' | 'success' | 'error' | 'warning'

export interface ToastItem {
  id: number
  message: string
  type: ToastType
  duration: number
  // 动画状态：entering/visible/leaving
  state: 'entering' | 'visible' | 'leaving'
}

const toasts = ref<ToastItem[]>([])
let nextId = 1

// 默认时长（毫秒）：错误/警告更长，信息/成功较短
const DEFAULT_DURATION: Record<ToastType, number> = {
  info: 3000,
  success: 2500,
  warning: 4000,
  error: 5000,
}

function remove(id: number) {
  const idx = toasts.value.findIndex((t) => t.id === id)
  if (idx < 0) return
  // 触发离场动画
  toasts.value[idx].state = 'leaving'
  setTimeout(() => {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }, 200)
}

function show(message: string, type: ToastType = 'info', duration?: number) {
  const id = nextId++
  const dur = duration ?? DEFAULT_DURATION[type]
  const item: ToastItem = { id, message, type, duration: dur, state: 'entering' }
  toasts.value.push(item)
  // 进入动画 16ms 后切到 visible（触发 CSS transition）
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      const cur = toasts.value.find((t) => t.id === id)
      if (cur) cur.state = 'visible'
    })
  })
  setTimeout(() => remove(id), dur)
}

function success(message: string) {
  show(message, 'success')
}

function error(message: string) {
  show(message, 'error')
}

function warning(message: string) {
  show(message, 'warning')
}

function info(message: string) {
  show(message, 'info')
}

export function useToast() {
  return { toasts, show, success, error, warning, info, remove }
}
