<script setup lang="ts">
import { useToast } from '@/composables/useToast'

const { toasts, remove } = useToast()

// 每种类型的图标和图标背景色（使用 CSS 变量）
const ICONS: Record<string, string> = {
  info: 'ℹ',
  success: '✓',
  warning: '!',
  error: '✕',
}
</script>

<template>
  <div class="toast-wrap" aria-live="polite">
    <transition-group name="toast">
      <div
        v-for="t in toasts"
        :key="t.id"
        class="toast"
        :class="[t.type, t.state]"
        role="status"
        @click="remove(t.id)"
      >
        <span class="toast-icon" :class="t.type">{{ ICONS[t.type] }}</span>
        <span class="toast-msg">{{ t.message }}</span>
      </div>
    </transition-group>
  </div>
</template>

<style scoped>
.toast-wrap {
  position: fixed;
  top: 24px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 3000;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  pointer-events: none;
}
.toast {
  pointer-events: auto;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 240px;
  max-width: 480px;
  padding: 12px 18px 12px 14px;
  background: var(--color-bg-elevated);
  color: var(--color-text);
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.14), 0 2px 6px rgba(0, 0, 0, 0.06);
  border-left: 4px solid var(--color-primary);
  font-size: 14px;
  /* 入场动画初始状态 */
  opacity: 0;
  transform: translateY(-12px) scale(0.98);
  transition: opacity 0.22s ease, transform 0.22s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.toast.visible {
  opacity: 1;
  transform: translateY(0) scale(1);
}
.toast.leaving {
  opacity: 0;
  transform: translateY(-8px) scale(0.98);
}
.toast.error {
  border-left-color: var(--color-danger);
}
.toast.success {
  border-left-color: var(--color-success);
}
.toast.warning {
  border-left-color: var(--color-warning);
}
.toast-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  font-size: 13px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
  background: var(--color-primary);
}
.toast-icon.success {
  background: var(--color-success);
}
.toast-icon.error {
  background: var(--color-danger);
}
.toast-icon.warning {
  background: var(--color-warning);
}
.toast-msg {
  flex: 1;
  word-break: break-word;
  line-height: 1.4;
}
</style>
