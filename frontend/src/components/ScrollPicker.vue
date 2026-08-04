<script setup lang="ts">
import { computed } from 'vue'
import { VueScrollPicker } from 'vue-scroll-picker'
import 'vue-scroll-picker/style.css'

// 兼容旧接口：{label, value} → 转换为 vue-scroll-picker 的 {name, value}
interface Option {
  label: string | number
  value: number | string
}

const props = defineProps<{
  options: Option[]
  modelValue: number | string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: number | string] }>()

// 转换选项格式
const pickerOptions = computed(() =>
  props.options.map((o) => ({ name: String(o.label), value: o.value })),
)

// VueScrollPicker 的 update 事件可能传 undefined
function onUpdate(val: unknown) {
  if (val === null || val === undefined) return
  emit('update:modelValue', val as number | string)
}
</script>

<template>
  <VueScrollPicker
    :options="pickerOptions"
    :model-value="modelValue"
    @update:model-value="onUpdate"
  />
</template>

<style scoped>
/* 全局样式由 vue-scroll-picker/style.css 提供 */
/* 这里做容器适配 + 暗色模式覆盖（vue-scroll-picker 默认硬编码浅色） */
:deep(.vue-scroll-picker) {
  width: 100%;
  /* 默认（浅色模式）也使用主题变量，避免与 app 主题脱节 */
  background: var(--color-bg-elevated);
}

/* 选项文字颜色：使用主题文本色 */
:deep(.vue-scroll-picker-item) {
  color: var(--color-text);
}
:deep(.vue-scroll-picker-item[aria-selected=true]) {
  color: var(--color-primary);
}
:deep(.vue-scroll-picker-item[data-value=""]),
:deep(.vue-scroll-picker-item[aria-disabled=true]) {
  color: var(--color-text-muted);
}
:deep(.vue-scroll-picker-item[data-value=""][aria-selected=true]),
:deep(.vue-scroll-picker-item[aria-disabled=true][aria-selected=true]) {
  color: var(--color-text-muted);
}

/* 上下渐变层：使用主题 elevated 背景，让其半透明渐变到背景色 */
:deep(.vue-scroll-picker-layer-top) {
  background: linear-gradient(
    180deg,
    var(--color-bg-elevated) 10%,
    rgba(0, 0, 0, 0)
  );
  border-bottom-color: var(--color-border);
}
:deep(.vue-scroll-picker-layer-bottom) {
  background: linear-gradient(
    0deg,
    var(--color-bg-elevated) 10%,
    rgba(0, 0, 0, 0)
  );
  border-top-color: var(--color-border);
}
</style>
