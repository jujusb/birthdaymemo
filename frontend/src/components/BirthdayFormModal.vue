<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import * as api from '@/api'
import type { Birthday, BirthdayWithTag, Tag } from '@/api/types'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useUiStore } from '@/stores/ui'
import { useToast } from '@/composables/useToast'
import TagFormModal from './TagFormModal.vue'
import ScrollPicker from './ScrollPicker.vue'

const props = defineProps<{ birthday?: BirthdayWithTag | null }>()
const emit = defineEmits<{ close: []; saved: [] }>()

const i18n = useI18nStore()
const t = i18n.t
const ui = useUiStore()
const toast = useToast()

const isEdit = computed(() => !!props.birthday)

const now = new Date()
const name = ref(props.birthday?.name ?? '')
const gender = ref<Birthday['gender']>(props.birthday?.gender ?? 'none')
// 默认全部为当前日期
// 若旧数据 birth_year 为 0（未知），编辑时回填为当前年，避免出现"未知"选项被选中却不在列表中的情况
const year = ref<number>(
  props.birthday && props.birthday.birth_year && props.birthday.birth_year > 0
    ? props.birthday.birth_year
    : now.getFullYear()
)
const month = ref<number>(props.birthday?.birth_month ?? now.getMonth() + 1)
const day = ref<number>(props.birthday?.birth_day ?? now.getDate())
// 多标签：编辑时从 tags 数组初始化
const tagIds = ref<number[]>(props.birthday?.tags?.map((tg) => tg.id) ?? [])

const tags = ref<Tag[]>([])
const saving = ref(false)
const showTagModal = ref(false)

// 共享生日的 owner 标签不在 listTags（仅本人）中，合并进来以便委托编辑时可见/可保留
const allTags = computed(() => {
  const map = new Map<number, Tag>()
  for (const tg of tags.value) map.set(tg.id, tg)
  for (const tg of props.birthday?.tags ?? []) map.set(tg.id, tg)
  return [...map.values()]
})
const isSharedEdit = computed(() => !!props.birthday?.shared)

const MONTHS = Array.from({ length: 12 }, (_, i) => i + 1)

// 年份选项：1900 .. 当前年+1（不再有"未知"选项；旧数据 birth_year=0 会在编辑时回填为当前年）
const yearOptions = computed(() => {
  const cy = new Date().getFullYear()
  const years: { label: string | number; value: number }[] = []
  for (let y = 1900; y <= cy + 1; y++) {
    years.push({ label: y, value: y })
  }
  return years
})

function daysInMonth(y: number, m: number): number {
  // 年份始终 > 0（已在 year ref 中回填），无需 yearForCalc 兜底
  return new Date(y, m, 0).getDate()
}

const daysInCurrentMonth = computed(() => daysInMonth(year.value, month.value))
const dayOptions = computed(() => Array.from({ length: daysInCurrentMonth.value }, (_, i) => i + 1))

// 滚动选择器选项（iOS 风格）
const monthPickerOptions = MONTHS.map((m) => ({ label: m, value: m }))
const dayPickerOptions = computed(() =>
  dayOptions.value.map((d) => ({ label: d, value: d })),
)

// Clamp day when month/year changes
watch(daysInCurrentMonth, (max) => {
  if (day.value > max) day.value = max
})

async function loadTags() {
  try {
    tags.value = await api.listTags()
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

onMounted(loadTags)
// refresh tags when something else bumped them
watch(() => ui.tagRefreshKey, loadTags)

function close() {
  emit('close')
}

function toggleTag(id: number) {
  const idx = tagIds.value.indexOf(id)
  if (idx >= 0) {
    tagIds.value.splice(idx, 1)
  } else {
    tagIds.value.push(id)
  }
}

function onTagCreated(tag: Tag) {
  tags.value.push(tag)
  if (!tagIds.value.includes(tag.id)) {
    tagIds.value.push(tag.id)
  }
  //新建标签时通知其他组件刷新标签列表
  ui.bumpTags()
  showTagModal.value = false
}

async function save() {
  const trimmed = name.value.trim()
  if (!trimmed) {
    toast.error(t('common.required'))
    return
  }
  if (!month.value || !day.value) {
    toast.error(t('common.required'))
    return
  }
  saving.value = true
  try {
    const payload: Partial<Birthday> = {
      name: trimmed,
      gender: gender.value,
      birth_year: year.value,
      birth_month: month.value,
      birth_day: day.value,
      tag_ids: tagIds.value,
    }
    if (isEdit.value && props.birthday) {
      await api.updateBirthday(props.birthday.id, payload)
    } else {
      await api.createBirthday(payload)
    }
    // 新建/编辑生日后通知顶栏与侧栏刷新
    ui.bumpBirthdays()
    toast.success(t('common.success'))
    emit('saved')
  } catch (e) {
    toast.error((e as ApiError).message || t('common.failed'))
  } finally {
    saving.value = false
  }
}

const genderOptions = computed(() => [
  { value: 'male' as const, color: 'var(--color-male)', label: t('birthday.male') },
  { value: 'female' as const, color: 'var(--color-female)', label: t('birthday.female') },
  { value: 'none' as const, color: 'var(--color-gender-none)', label: t('birthday.unknown') },
])
</script>

<template>
  <div class="modal-backdrop" @click.self="close">
    <div class="modal" role="dialog" aria-modal="true">
      <h3 class="modal-title">{{ isEdit ? t('birthday.edit') : t('birthday.new') }}</h3>

      <div class="col mt-16">
        <label>{{ t('birthday.name') }} <span class="req">*</span></label>
        <input v-model="name" type="text" :placeholder="t('birthday.name')" />
      </div>

      <div class="col mt-16">
        <label>{{ t('birthday.gender') }}</label>
        <div class="gender-row">
          <label
            v-for="opt in genderOptions"
            :key="opt.value"
            class="gender-opt"
            :class="{ active: gender === opt.value }"
          >
            <input v-model="gender" type="radio" :value="opt.value" />
            <span class="tag-dot" :style="{ background: opt.color }"></span>
            {{ opt.label }}
          </label>
        </div>
      </div>

      <div class="col mt-16">
        <label>{{ t('birthday.date') }} <span class="req">*</span></label>
        <div class="date-row">
          <div class="col grow">
            <span class="field-label">{{ t('picker.selectYear') }}</span>
            <ScrollPicker
              :options="yearOptions"
              :modelValue="year"
              @update:modelValue="year = $event as number"
            />
          </div>
          <div class="col grow">
            <span class="field-label">{{ t('picker.month') }}</span>
            <ScrollPicker
              :options="monthPickerOptions"
              :modelValue="month"
              @update:modelValue="month = $event as number"
            />
          </div>
          <div class="col grow">
            <span class="field-label">{{ t('picker.day') }}</span>
            <ScrollPicker
              :options="dayPickerOptions"
              :modelValue="day"
              @update:modelValue="day = $event as number"
            />
          </div>
        </div>
      </div>

      <div class="col mt-16">
        <label>{{ t('birthday.tags') }}</label>
        <span class="hint">{{ t('birthday.multiTagHint') }}</span>
        <span v-if="isSharedEdit" class="hint shared-hint">👥 {{ t('grant.sharedEditHint', { name: props.birthday?.owner_username ?? '' }) }}</span>
        <div v-if="allTags.length === 0" class="tag-empty">{{ t('tag.untagged') }}</div>
        <div v-else class="tag-check-grid">
          <label
            v-for="tg in allTags"
            :key="tg.id"
            class="tag-check"
            :class="{ checked: tagIds.includes(tg.id) }"
          >
            <input
              type="checkbox"
              :value="tg.id"
              :checked="tagIds.includes(tg.id)"
              @change="toggleTag(tg.id)"
            />
            <span class="tag-dot" :style="{ background: tg.color }"></span>
            <span class="tag-check-name">{{ tg.name }}</span>
          </label>
        </div>
        <div class="row gap-8 mt-8">
          <button type="button" @click="showTagModal = true">+ {{ t('tag.new') }}</button>
        </div>
      </div>

      <div class="row between mt-24 actions">
        <button type="button" @click="close">{{ t('common.cancel') }}</button>
        <button type="button" class="primary" :disabled="saving" @click="save">
          {{ saving ? t('common.loading') : t('common.save') }}
        </button>
      </div>

      <TagFormModal v-if="showTagModal" @close="showTagModal = false" @saved="onTagCreated" />
    </div>
  </div>
</template>

<style scoped>
.modal-title {
  font-size: 16px;
  font-weight: 600;
}
.req {
  color: var(--color-danger);
}
.hint {
  font-size: 12px;
  color: var(--color-text-muted);
}
.shared-hint {
  display: block;
  color: var(--color-primary);
  margin-top: 2px;
}
.gender-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.gender-opt {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  margin: 0;
  font-size: 14px;
  color: var(--color-text);
}
.gender-opt.active {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: #fff;
}
.gender-opt.active .tag-dot {
  filter: brightness(1.2);
}
.gender-opt input {
  width: auto;
  margin: 0;
}
.date-row {
  display: flex;
  gap: 8px;
}
.field-label {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 4px;
}
.mt-8 {
  margin-top: 8px;
}
.tag-empty {
  font-size: 13px;
  color: var(--color-text-muted);
  padding: 8px 0;
}
.tag-check-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 6px;
}
.tag-check {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  margin: 0;
  font-size: 14px;
  color: var(--color-text);
  user-select: none;
}
.tag-check:hover {
  border-color: var(--color-primary);
}
.tag-check.checked {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: #fff;
}
.tag-check.checked .tag-dot {
  filter: brightness(1.2);
}
.tag-check input {
  width: auto;
  margin: 0;
}
.tag-check-name {
  white-space: nowrap;
}
.actions {
  justify-content: flex-end;
}
</style>
