import { ref } from 'vue'

// 支持位置定位（x/y 优先），无位置时居中显示
export interface ConfirmState {
  show: boolean
  x: number | null // 若为 null，则居中显示
  y: number | null
  message: string
  // 解析后回调：true = 确认，false = 取消
  resolve: ((ok: boolean) => void) | null
}

const state = ref<ConfirmState>({
  show: false,
  x: null,
  y: null,
  message: '',
  resolve: null,
})

/**
 * 显示确认弹窗，返回 Promise<boolean>
 * @param message 提示文案
 * @param opts.x 可选：弹窗 x 坐标（不传则居中）
 * @param opts.y 可选：弹窗 y 坐标（不传则居中）
 */
export function showConfirm(message: string, opts?: { x?: number; y?: number }): Promise<boolean> {
  return new Promise<boolean>((resolve) => {
    state.value = {
      show: true,
      x: opts?.x ?? null,
      y: opts?.y ?? null,
      message,
      resolve,
    }
  })
}

export function resolveConfirm(ok: boolean) {
  if (state.value.resolve) {
    state.value.resolve(ok)
  }
  state.value = { ...state.value, show: false, resolve: null }
}

export function useConfirm() {
  return { state, showConfirm, resolveConfirm }
}
