<script setup lang="ts">
import { computed } from 'vue'
import { useConfirm } from '@/composables/useConfirm'
import { useI18nStore } from '@/stores/i18n'

const { state, resolveConfirm } = useConfirm()
const i18n = useI18nStore()
const t = i18n.t

// 计算弹窗位置：若 x/y 提供，则定位到坐标下方；否则居中
const positionStyle = computed(() => {
  if (state.value.x != null && state.value.y != null) {
    return {
      left: state.value.x + 'px',
      top: state.value.y + 'px',
      transform: 'translate(0, 0)',
    }
  }
  return {
    left: '50%',
    top: '50%',
    transform: 'translate(-50%, -50%)',
  }
})

function onConfirm() {
  resolveConfirm(true)
}
function onCancel() {
  resolveConfirm(false)
}
function onBackdropClick() {
  resolveConfirm(false)
}
</script>

<template>
  <div v-if="state.show" class="confirm-backdrop" @click="onBackdropClick">
    <div
      class="confirm-popover"
      :style="positionStyle"
      :class="{ centered: state.x == null }"
      @click.stop
      role="alertdialog"
      aria-modal="true"
    >
      <div class="confirm-msg">{{ state.message }}</div>
      <div class="confirm-actions">
        <button class="cancel-btn" @click="onCancel">{{ t('common.cancel') }}</button>
        <button class="primary danger" @click="onConfirm">{{ t('common.confirm') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.confirm-backdrop {
  position: fixed;
  inset: 0;
  z-index: 3000;
  background: transparent;
  /* 不阻挡其他位置点击，但点击空白处关闭 */
}
.confirm-popover {
  position: fixed;
  min-width: 220px;
  max-width: 320px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.22);
  padding: 14px 16px;
  animation: confirm-pop 0.14s ease-out;
}
.confirm-popover.centered {
  /* 居中时使用更大的 max-width，避免在内容过多时过窄 */
  max-width: 90vw;
}
@keyframes confirm-pop {
  from { opacity: 0; transform: translate(0, -4px) scale(0.96); }
  to { opacity: 1; }
}
.confirm-popover.centered {
  animation-name: confirm-pop-center;
}
@keyframes confirm-pop-center {
  from { opacity: 0; transform: translate(-50%, -50%) scale(0.96); }
  to { opacity: 1; transform: translate(-50%, -50%) scale(1); }
}
.confirm-msg {
  font-size: 13px;
  color: var(--color-text);
  line-height: 1.5;
  margin-bottom: 12px;
  word-break: break-word;
}
.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
.confirm-actions button {
  padding: 5px 14px;
  font-size: 13px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background: var(--color-bg-elevated);
  color: var(--color-text);
  cursor: pointer;
}
.confirm-actions button:hover {
  border-color: var(--color-primary);
}
.confirm-actions .danger {
  background: var(--color-danger, #e74c3c);
  border-color: var(--color-danger, #e74c3c);
  color: #fff;
}
.confirm-actions .danger:hover {
  filter: brightness(0.92);
}
</style>
