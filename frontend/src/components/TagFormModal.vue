<script setup lang="ts">
import { ref, computed } from 'vue'
import * as api from '@/api'
import type { Tag } from '@/api/types'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useUiStore } from '@/stores/ui'
import { useToast } from '@/composables/useToast'

const props = defineProps<{ tag?: Tag | null }>()
const emit = defineEmits<{ close: []; saved: [tag: Tag] }>()

const i18n = useI18nStore()
const t = i18n.t
const ui = useUiStore()
const toast = useToast()

const PRESET_COLORS = [
  '#4A90D9', '#E91E63', '#4CAF50', '#FF9800', '#9C27B0',
  '#00BCD4', '#FF5722', '#795548', '#607D8B', '#FFC107',
]

const name = ref(props.tag?.name ?? '')
const color = ref(props.tag?.color ?? PRESET_COLORS[0])
const saving = ref(false)

const isEdit = computed(() => !!props.tag)
const nameCount = computed(() => name.value.length)
const NAME_LIMIT = 10

function close() {
  emit('close')
}

async function save() {
  const trimmed = name.value.trim()
  if (!trimmed) {
    toast.error(t('common.required'))
    return
  }
  if (trimmed.length > NAME_LIMIT) {
    toast.error(t('tag.nameLimit'))
    return
  }
  saving.value = true
  try {
    let saved: Tag
    if (isEdit.value && props.tag) {
      saved = await api.updateTag(props.tag.id, trimmed, color.value)
    } else {
      saved = await api.createTag(trimmed, color.value)
    }
    ui.bumpTags()
    toast.success(t('common.success'))
    emit('saved', saved)
  } catch (e) {
    toast.error((e as ApiError).message || t('common.failed'))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="modal-backdrop" @click.self="close">
    <div class="modal" role="dialog" aria-modal="true">
      <h3 class="modal-title">{{ isEdit ? t('common.edit') : t('tag.new') }}</h3>

      <div class="col mt-16">
        <label>{{ t('tag.name') }}</label>
        <input
          v-model="name"
          type="text"
          :maxlength="NAME_LIMIT"
          :placeholder="t('tag.name')"
        />
        <span class="hint">{{ nameCount }} / {{ NAME_LIMIT }} · {{ t('tag.nameLimit') }}</span>
      </div>

      <div class="col mt-16">
        <label>{{ t('tag.color') }}</label>
        <div class="color-row">
          <input v-model="color" type="color" class="color-input" />
          <span class="color-preview">
            <span class="tag-dot" :style="{ background: color }"></span>
            {{ color }}
          </span>
        </div>
        <div class="preset-row">
          <button
            v-for="c in PRESET_COLORS"
            :key="c"
            type="button"
            class="preset"
            :class="{ active: c.toLowerCase() === color.toLowerCase() }"
            :style="{ background: c }"
            :title="c"
            @click="color = c"
          ></button>
        </div>
      </div>

      <div class="row between mt-24 actions">
        <button type="button" @click="close">{{ t('common.cancel') }}</button>
        <button type="button" class="primary" :disabled="saving" @click="save">
          {{ saving ? t('common.loading') : t('common.save') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-title {
  font-size: 16px;
  font-weight: 600;
}
.hint {
  font-size: 12px;
  color: var(--color-text-muted);
}
.color-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.color-input {
  width: 44px;
  height: 36px;
  padding: 2px;
  cursor: pointer;
}
.color-preview {
  display: flex;
  align-items: center;
  font-size: 13px;
  color: var(--color-text-muted);
}
.preset-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}
.preset {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: 2px solid transparent;
  padding: 0;
  cursor: pointer;
}
.preset.active {
  border-color: var(--color-text);
}
.actions {
  justify-content: flex-end;
}
</style>
