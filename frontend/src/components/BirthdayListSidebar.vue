<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import * as api from '@/api'
import type { Tag, BirthdayWithTag } from '@/api/types'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useUiStore } from '@/stores/ui'
import { useToast } from '@/composables/useToast'
import { showConfirm } from '@/composables/useConfirm'
import { precomputePinyin, matchWithPinyin, type PinyinData } from '@/utils/pinyinInitial'
import TagFormModal from './TagFormModal.vue'
import BirthdayFormModal from './BirthdayFormModal.vue'
import { useAuthStore } from '@/stores/auth'

// readonly: 分享预览 / Guest Mode / guest 账号（隐藏一切写入口）
// showAge: 年龄显示开关（由 CalendarView 的 🎂 按钮控制，默认显示）
const props = defineProps<{ readonly?: boolean; showAge?: boolean }>()
const auth = useAuthStore()
const effectiveReadonly = computed(() => !!props.readonly || auth.isReadOnly)
const showAge = computed(() => props.showAge ?? true)

const i18n = useI18nStore()
const t = i18n.t
const ui = useUiStore()
const toast = useToast()

const tags = ref<Tag[]>([])
const birthdays = ref<BirthdayWithTag[]>([])
const showTagModal = ref(false)
const editingTag = ref<Tag | null>(null)
const showBirthdayModal = ref(false)
const editingBirthday = ref<BirthdayWithTag | null>(null)
const loadingList = ref(false)
const sortBy = ref<'date' | 'name'>('date')

// 搜索关键词（按姓名过滤）
const searchQuery = ref('')
// 防抖后的搜索关键词（200ms 延迟，避免快速输入时频繁过滤）
const searchQueryDebounced = ref('')
let searchDebounceTimer: number | undefined
watch(searchQuery, (val) => {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  searchDebounceTimer = window.setTimeout(() => {
    searchQueryDebounced.value = val
  }, 200)
})
onBeforeUnmount(() => {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
})

// 预计算的拼音数据：在生日列表变化时一次性计算所有名称的拼音，避免每次输入时重复计算
const pinyinMap = computed<Map<number, PinyinData>>(() => {
  const m = new Map<number, PinyinData>()
  for (const b of birthdays.value) {
    m.set(b.id, precomputePinyin(b.name))
  }
  return m
})

// 标签展开状态（每个生日的标签列表是否展开）
const expandedTagSet = ref<Set<number>>(new Set())

const today = new Date()
today.setHours(0, 0, 0, 0)

// 本人生日信息（置顶显示）
const selfBirthYear = ref<number>(0)
const selfBirthMonth = ref<number>(0)
const selfBirthDay = ref<number>(0)

async function loadSelfBirthday() {
  try {
    const s = await api.getSettings()
    selfBirthYear.value = s.self_birth_year || 0
    selfBirthMonth.value = s.self_birth_month || 0
    selfBirthDay.value = s.self_birth_day || 0
  } catch {
    // 忽略，不阻塞侧边栏渲染
  }
}

// 本人生日：下一次生日日期
const selfNextDate = computed<Date | null>(() => {
  if (!selfBirthMonth.value || !selfBirthDay.value) return null
  const y = today.getFullYear()
  let d = new Date(y, selfBirthMonth.value - 1, selfBirthDay.value)
  if (d < today) {
    d = new Date(y + 1, selfBirthMonth.value - 1, selfBirthDay.value)
  }
  return d
})

// 本人生日：距今天数
const selfDaysLeft = computed<number>(() => {
  const d = selfNextDate.value
  if (!d) return -1
  return Math.ceil((d.getTime() - today.getTime()) / 86400000)
})

// 本人生日：是否当日
const isSelfToday = computed(() => selfDaysLeft.value === 0)

// 本人生日：根据距离天数选择文本
const selfBirthdayText = computed<string>(() => {
  const days = selfDaysLeft.value
  if (days < 0) return ''
  if (days === 0) {
    return t('sidebar.selfToday')
  }
  return t('sidebar.selfDaysLeft', { count: days })
})

// 本人生日：根据距离天数选择小字描述
const selfBirthdayDesc = computed<string>(() => {
  const days = selfDaysLeft.value
  if (days < 0) return ''
  if (days === 0) {
    return t('sidebar.selfTodayDesc')
  }
  if (days <= 30) {
    return t('sidebar.selfRange30', { count: days })
  }
  if (days <= 100) {
    return t('sidebar.selfRange100', { count: days })
  }
  if (days <= 200) {
    return t('sidebar.selfRange200', { count: days })
  }
  if (days <= 300) {
    return t('sidebar.selfRange300', { count: days })
  }
  return t('sidebar.selfRange365', { count: days })
})

async function loadTags() {
  try {
    tags.value = await api.listTags()
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

async function loadBirthdays() {
  loadingList.value = true
  try {
    const opts =
      ui.selectedTagIds.length > 0 ? { tag_ids: ui.selectedTagIds } : undefined
    birthdays.value = await api.listBirthdays(opts)
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    loadingList.value = false
  }
}

onMounted(() => {
  loadTags()
  loadBirthdays()
  loadSelfBirthday()
})
watch(() => ui.tagRefreshKey, loadTags)
// 标签筛选变化、tag 刷新、生日数据变化时刷新列表
watch(
  () => [ui.selectedTagIds.length, ui.tagRefreshKey, ui.birthdayRefreshKey],
  loadBirthdays,
)
// 本人生日信息随生日数据变化刷新
watch(() => ui.birthdayRefreshKey, loadSelfBirthday)

function toggleTag(id: number) {
  ui.toggleSelectedTag(id)
}
function clearTags() {
  ui.clearSelectedTags()
}

function openNew() {
  editingTag.value = null
  showTagModal.value = true
}
function openEdit(tag: Tag) {
  editingTag.value = tag
  showTagModal.value = true
}
function onTagSaved() {
  showTagModal.value = false
  editingTag.value = null
  loadTags()
  ui.bumpTags()
}

// 用自定义 ConfirmDialog 替换浏览器 confirm()
// 若上下文菜单位置可用，则在菜单位置附近显示，否则居中
async function removeTag(tag: Tag) {
  const ok = await showConfirm(t('tag.deleteConfirm') + ' "' + tag.name + '"?', {
    x: contextMenu.value.show ? contextMenu.value.x : undefined,
    y: contextMenu.value.show ? contextMenu.value.y + 80 : undefined,
  })
  if (!ok) return
  try {
    await api.deleteTag(tag.id)
    if (ui.selectedTagIds.includes(tag.id)) {
      ui.toggleSelectedTag(tag.id)
    }
    ui.bumpTags()
    ui.bumpBirthdays()
    loadTags()
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

// 下一个生日日期，用于排序
function nextDate(b: BirthdayWithTag): Date {
  const y = today.getFullYear()
  let d = new Date(y, b.birth_month - 1, b.birth_day)
  if (d < today) {
    d = new Date(y + 1, b.birth_month - 1, b.birth_day)
  }
  return d
}

// 按搜索关键词过滤 + 排序（使用防抖后的查询和预计算的拼音数据）
const sortedBirthdays = computed(() => {
  const q = searchQueryDebounced.value.trim().toLowerCase()
  let arr = birthdays.value
  if (q) {
    const pm = pinyinMap.value
    // 使用预计算的拼音数据进行快速匹配，避免每次输入时调用 pinyin-pro
    arr = arr.filter((b) => {
      const pd = pm.get(b.id)
      if (pd) return matchWithPinyin(b.name, pd, q)
      return b.name.toLowerCase().includes(q)
    })
  }
  arr = [...arr]
  if (sortBy.value === 'name') {
    arr.sort((a, b) => a.name.localeCompare(b.name, i18n.lang))
  } else {
    arr.sort((a, b) => nextDate(a).getTime() - nextDate(b).getTime())
  }
  return arr
})

function genderColor(g: string): string {
  if (g === 'male') return 'var(--color-male)'
  if (g === 'female') return 'var(--color-female)'
  return 'var(--color-gender-none)'
}

function formatDate(b: BirthdayWithTag): string {
  const d = nextDate(b)
  const days = Math.ceil((d.getTime() - today.getTime()) / 86400000)
  const dateStr = `${b.birth_month}/${b.birth_day}`
  if (days === 0) return `${dateStr} · ${t('list.today')}`
  return `${dateStr} · ${days}d`
}

// Upcoming age (age they turn on their next birthday); -1 when birth year unknown
function upcomingAgeOf(b: BirthdayWithTag): number {
  if (!b.birth_year) return -1
  return nextDate(b).getFullYear() - b.birth_year
}

// 默认只显示前 2 个，多的折叠为 +N；展开后显示全部
const FOLD_TAG_COUNT = 2
function visibleTags(b: BirthdayWithTag): Tag[] {
  if (!b.tags || b.tags.length === 0) return []
  if (expandedTagSet.value.has(b.id)) return b.tags
  return b.tags.slice(0, FOLD_TAG_COUNT)
}
function hiddenTagCount(b: BirthdayWithTag): number {
  if (!b.tags || b.tags.length === 0) return 0
  if (expandedTagSet.value.has(b.id)) return 0
  return Math.max(0, b.tags.length - FOLD_TAG_COUNT)
}
function toggleExpandTags(b: BirthdayWithTag) {
  const s = new Set(expandedTagSet.value)
  if (s.has(b.id)) {
    s.delete(b.id)
  } else {
    s.add(b.id)
  }
  expandedTagSet.value = s
}

// ============ 上下文菜单（右键 / 长按） ============
// 用于标签和生日条目的"修改/删除"操作
interface ContextMenuState {
  show: boolean
  x: number
  y: number
  type: 'tag' | 'birthday'
  tag?: Tag
  birthday?: BirthdayWithTag
}
const contextMenu = ref<ContextMenuState>({ show: false, x: 0, y: 0, type: 'tag' })

// 长按定时器
let longPressTimer: number | undefined
const LONG_PRESS_MS = 500

function openContextMenu(x: number, y: number, type: 'tag' | 'birthday', tag?: Tag, birthday?: BirthdayWithTag) {
  // clamp 到视口内
  const VW = window.innerWidth
  const VH = window.innerHeight
  const MENU_W = 140
  const MENU_H = 80
  const MARGIN = 4
  let cx = x
  let cy = y
  if (cx + MENU_W > VW - MARGIN) cx = VW - MENU_W - MARGIN
  if (cx < MARGIN) cx = MARGIN
  if (cy + MENU_H > VH - MARGIN) cy = VH - MENU_H - MARGIN
  if (cy < MARGIN) cy = MARGIN
  contextMenu.value = { show: true, x: cx, y: cy, type, tag, birthday }
}

function closeContextMenu() {
  contextMenu.value = { ...contextMenu.value, show: false }
}

// 标签：右键
function onTagContextMenu(e: MouseEvent, tag: Tag) {
  e.preventDefault()
  if (effectiveReadonly.value) return
  openContextMenu(e.clientX, e.clientY, 'tag', tag)
}

// 标签：长按（移动端）
function onTagTouchStart(e: TouchEvent, tag: Tag) {
  if (effectiveReadonly.value) return
  const touch = e.touches[0]
  const x = touch.clientX
  const y = touch.clientY
  longPressTimer = window.setTimeout(() => {
    openContextMenu(x, y, 'tag', tag)
  }, LONG_PRESS_MS)
}
function onTagTouchEnd() {
  if (longPressTimer) {
    window.clearTimeout(longPressTimer)
    longPressTimer = undefined
  }
}

// 生日条目是否可编辑：本人（非共享）恒可编辑；共享需 can_edit（≥1 tag 被授予 edit）
function canEditBd(b: BirthdayWithTag): boolean {
  if (!b.shared) return true
  return b.can_edit ?? false
}
// 生日条目是否可删除：仅本人（非共享）；被委托人不可删除
function canDeleteBd(b: BirthdayWithTag): boolean {
  return !b.shared
}

// 生日条目：右键
function onBdContextMenu(e: MouseEvent, b: BirthdayWithTag) {
  e.preventDefault()
  if (effectiveReadonly.value) return
  if (!canEditBd(b) && !canDeleteBd(b)) return
  openContextMenu(e.clientX, e.clientY, 'birthday', undefined, b)
}

// 生日条目：长按
function onBdTouchStart(e: TouchEvent, b: BirthdayWithTag) {
  if (effectiveReadonly.value) return
  if (!canEditBd(b) && !canDeleteBd(b)) return
  const touch = e.touches[0]
  const x = touch.clientX
  const y = touch.clientY
  longPressTimer = window.setTimeout(() => {
    openContextMenu(x, y, 'birthday', undefined, b)
  }, LONG_PRESS_MS)
}
function onBdTouchEnd() {
  if (longPressTimer) {
    window.clearTimeout(longPressTimer)
    longPressTimer = undefined
  }
}

// 菜单动作
function onMenuEdit() {
  if (contextMenu.value.type === 'tag' && contextMenu.value.tag) {
    openEdit(contextMenu.value.tag)
  } else if (contextMenu.value.type === 'birthday' && contextMenu.value.birthday) {
    editingBirthday.value = contextMenu.value.birthday
    showBirthdayModal.value = true
  }
  closeContextMenu()
}

function onMenuDelete() {
  if (contextMenu.value.type === 'tag' && contextMenu.value.tag) {
    const tag = contextMenu.value.tag
    closeContextMenu()
    void removeTag(tag)
  } else if (contextMenu.value.type === 'birthday' && contextMenu.value.birthday) {
    const b = contextMenu.value.birthday
    closeContextMenu()
    void deleteBirthday(b)
  }
}

// showConfirm
async function deleteBirthday(b: BirthdayWithTag) {
  const ok = await showConfirm(t('birthday.deleteConfirm') + ' "' + b.name + '"?', {
    x: contextMenu.value.show ? contextMenu.value.x : undefined,
    y: contextMenu.value.show ? contextMenu.value.y + 80 : undefined,
  })
  if (!ok) return
  try {
    await api.deleteBirthday(b.id)
    ui.bumpTags()
    ui.bumpBirthdays()
    loadBirthdays()
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

function onBirthdaySaved() {
  showBirthdayModal.value = false
  editingBirthday.value = null
  loadBirthdays()
}

// 全局点击关闭菜单
function onGlobalClick() {
  if (contextMenu.value.show) closeContextMenu()
}
onMounted(() => document.addEventListener('click', onGlobalClick))
onBeforeUnmount(() => {
  document.removeEventListener('click', onGlobalClick)
  if (longPressTimer) window.clearTimeout(longPressTimer)
})
</script>

<template>
  <aside class="sidebar">
    <!-- 1. 搜索 -->
    <div class="sidebar-section">
      <div class="search-wrap">
        <span class="search-icon">🔍</span>
        <input
          v-model="searchQuery"
          type="text"
          :placeholder="t('common.search')"
          class="search-input"
        />
        <button v-if="searchQuery" class="search-clear" @click="searchQuery = ''">×</button>
      </div>
    </div>

    <!-- 2. 标签筛选 -->
    <div class="sidebar-section">
      <div class="section-head row between">
        <span class="title">{{ t('list.filterTags') }}</span>
        <button v-if="!effectiveReadonly" class="small" @click="openNew" :title="t('tag.new')">+</button>
      </div>
      <div class="tag-filter-grid">
        <label
          v-for="tag in tags"
          :key="tag.id"
          class="tag-chip"
          :class="{ checked: ui.selectedTagIds.includes(tag.id) }"
          @contextmenu.prevent="onTagContextMenu($event, tag)"
          @touchstart="onTagTouchStart($event, tag)"
          @touchend="onTagTouchEnd"
          @touchmove="onTagTouchEnd"
          @touchcancel="onTagTouchEnd"
        >
          <input
            type="checkbox"
            :checked="ui.selectedTagIds.includes(tag.id)"
            @change="toggleTag(tag.id)"
          />
          <span class="tag-dot" :style="{ background: tag.color }"></span>
          <span class="chip-name">{{ tag.name }}</span>
        </label>
        <span v-if="tags.length === 0" class="empty">{{ t('tag.untagged') }}</span>
      </div>
      <button v-if="ui.selectedTagIds.length > 0" class="clear-btn" @click="clearTags">
        {{ t('list.allTags') }}
      </button>
      <span v-if="!effectiveReadonly" class="hint">{{ t('list.rowHint') }}</span>
    </div>

    <!-- 3. 生日列表 -->
    <div class="sidebar-section grow">
      <div class="section-head row between">
        <span class="title">{{ t('list.title') }}</span>
        <div class="sort-toggle">
          <button
            :class="{ active: sortBy === 'date' }"
            :title="t('list.sortByDate')"
            @click="sortBy = 'date'"
          >📅</button>
          <button
            :class="{ active: sortBy === 'name' }"
            :title="t('list.sortByName')"
            @click="sortBy = 'name'"
          >🔤</button>
        </div>
      </div>
      <div class="bd-list-scroll">
        <div
          v-if="selfBirthdayText"
          class="self-bd-card"
          :class="{ 'self-today': isSelfToday }"
        >
          <div class="self-bd-title">{{ selfBirthdayText }}</div>
          <div class="self-bd-desc">{{ selfBirthdayDesc }}</div>
        </div>
        <div v-if="loadingList" class="empty">{{ t('common.loading') }}</div>
        <div v-else-if="sortedBirthdays.length === 0" class="empty">
          {{ searchQuery ? t('list.empty') : t('list.empty') }}
        </div>
        <div v-else class="bd-list">
          <div
            v-for="b in sortedBirthdays"
            :key="b.id"
            class="bd-row"
            @contextmenu.prevent="onBdContextMenu($event, b)"
            @touchstart="onBdTouchStart($event, b)"
            @touchend="onBdTouchEnd"
            @touchmove="onBdTouchEnd"
            @touchcancel="onBdTouchEnd"
          >
            <span class="gender-dot" :style="{ background: genderColor(b.gender) }"></span>
            <div class="bd-info grow">
              <div class="bd-name-row">
                <span class="bd-name">{{ b.name }}</span>
                <span v-if="b.shared" class="shared-badge" :title="b.owner_username">👥{{ b.owner_username }}</span>
                <div
                  v-if="b.tags && b.tags.length > 0"
                  class="bd-tags"
                  :class="{ expanded: expandedTagSet.has(b.id) }"
                  @click.stop="toggleExpandTags(b)"
                >
                  <span
                    v-for="tg in visibleTags(b)"
                    :key="tg.id"
                    class="mini-tag"
                    :style="{ color: tg.color, borderColor: tg.color }"
                  >{{ tg.name }}</span>
                  <span v-if="hiddenTagCount(b) > 0" class="mini-tag more-chip">+{{ hiddenTagCount(b) }}</span>
                </div>
                <span class="bd-date">{{ formatDate(b) }}</span>
                <span
                  v-if="showAge && upcomingAgeOf(b) > 0"
                  class="bd-age"
                  :title="t('common.turnsAge', { age: upcomingAgeOf(b) })"
                >{{ upcomingAgeOf(b) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      <span v-if="!effectiveReadonly" class="hint">{{ t('list.rowHint') }}</span>
    </div>

    <!-- 上下文菜单（fixed 定位，可超出 sidebar overflow） -->
    <div
      v-if="contextMenu.show"
      class="context-menu"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
      @click.stop
      @contextmenu.prevent
    >
      <button
        v-if="contextMenu.type === 'tag' || (contextMenu.birthday && canEditBd(contextMenu.birthday))"
        class="context-item"
        @click="onMenuEdit"
      >
        <span class="ctx-icon">✎</span>{{ t('common.edit') }}
      </button>
      <button
        v-if="contextMenu.type === 'tag' || (contextMenu.birthday && canDeleteBd(contextMenu.birthday))"
        class="context-item danger"
        @click="onMenuDelete"
      >
        <span class="ctx-icon">🗑</span>{{ t('common.delete') }}
      </button>
    </div>

    <TagFormModal v-if="showTagModal && !effectiveReadonly" :tag="editingTag" @close="showTagModal = false" @saved="onTagSaved" />
    <BirthdayFormModal v-if="showBirthdayModal && !effectiveReadonly" :birthday="editingBirthday" @close="showBirthdayModal = false" @saved="onBirthdaySaved" />
  </aside>
</template>

<style scoped>
.sidebar {
  width: 260px;
  flex-shrink: 0;
  background: var(--color-bg-elevated);
  border-right: 1px solid var(--color-border);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: hidden;
  max-height: calc(100vh - 84px);
  position: relative;
}
.sidebar-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.sidebar-section.grow {
  flex: 1;
  min-height: 0;
}
.section-head {
  align-items: center;
}
.title {
  font-weight: 600;
  font-size: 13px;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.small {
  font-size: 16px;
  width: 24px;
  height: 24px;
  padding: 0;
  line-height: 1;
  border-radius: var(--radius-sm);
}
.hint {
  font-size: 11px;
  color: var(--color-text-muted);
  opacity: 0.7;
  margin-top: 2px;
}

/* 搜索框 */
.search-wrap {
  position: relative;
  display: flex;
  align-items: center;
}
.search-icon {
  position: absolute;
  left: 8px;
  font-size: 12px;
  pointer-events: none;
  opacity: 0.6;
}
.search-input {
  width: 100%;
  padding: 6px 28px 6px 26px;
  font-size: 13px;
  border-radius: var(--radius-sm);
}
.search-input:focus {
  outline: none;
}
.search-clear {
  position: absolute;
  right: 4px;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  background: transparent;
  font-size: 16px;
  line-height: 1;
  color: var(--color-text-muted);
  cursor: pointer;
}
.search-clear:hover {
  color: var(--color-danger);
}
.shared-badge {
  font-size: 10px;
  color: var(--color-primary);
  border: 1px solid var(--color-primary);
  border-radius: 8px;
  padding: 0 5px;
  margin-left: 6px;
  white-space: nowrap;
}

/* 标签 */
.tag-filter-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.tag-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  font-size: 12px;
  cursor: pointer;
  user-select: none;
  color: var(--color-text);
  background: transparent;
  position: relative;
  -webkit-user-select: none;
  -webkit-touch-callout: none;
}
.tag-chip:hover {
  border-color: var(--color-primary);
}
.tag-chip.checked {
  border-color: var(--color-primary);
  background: rgba(0, 122, 255, 0.08);
}
.tag-chip input {
  display: none;
}
.chip-name {
  white-space: nowrap;
}
.clear-btn {
  align-self: flex-start;
  font-size: 12px;
  background: transparent;
  border: none;
  color: var(--color-primary);
  cursor: pointer;
  padding: 2px 4px;
}
.empty {
  color: var(--color-text-muted);
  font-size: 13px;
  padding: 8px 0;
}

/* 排序切换 */
.sort-toggle {
  display: inline-flex;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.sort-toggle button {
  border: none;
  background: transparent;
  padding: 2px 6px;
  cursor: pointer;
  font-size: 12px;
}
.sort-toggle button.active {
  background: var(--color-primary);
}

/* 生日列表 */
.bd-list-scroll {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}
.bd-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.self-bd-card {
  position: relative;
  padding: 10px 12px;
  margin-bottom: 8px;
  background: var(--color-bg-sunken);
  border: 2px solid var(--color-primary);
  border-radius: var(--radius);
  text-align: center;
  z-index: 1;
}
.self-bd-card .self-bd-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-primary);
  line-height: 1.4;
}
.self-bd-card .self-bd-desc {
  font-size: 10px;
  color: var(--color-text-muted);
  margin-top: 4px;
  line-height: 1.5;
}
.self-bd-card.self-today {
  border: 2px solid transparent;
  background-clip: padding-box;
}
.self-bd-card.self-today::before {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: inherit;
  padding: 2px;
  background: linear-gradient(
    45deg,
    #ff0040,
    #ff8c00,
    #ffd700,
    #00ff7f,
    #00bfff,
    #8a2be2,
    #ff0040
  );
  background-size: 400% 400%;
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  animation: self-rainbow-slide 3s linear infinite;
  z-index: 0;
  pointer-events: none;
}
.self-bd-card.self-today::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: linear-gradient(
    45deg,
    rgba(255, 0, 64, 0.1),
    rgba(255, 140, 0, 0.1),
    rgba(255, 215, 0, 0.1),
    rgba(0, 255, 127, 0.1),
    rgba(0, 191, 255, 0.1),
    rgba(138, 43, 226, 0.1),
    rgba(255, 0, 64, 0.1)
  );
  background-size: 400% 400%;
  animation: self-rainbow-slide 3s linear infinite;
  z-index: 0;
  pointer-events: none;
}
@keyframes self-rainbow-slide {
  0% {
    background-position: 0% 50%;
  }
  100% {
    background-position: 400% 50%;
  }
}
.self-bd-card.self-today .self-bd-title,
.self-bd-card.self-today .self-bd-desc {
  position: relative;
  z-index: 1;
}
.bd-row {
  display: flex;
  gap: 8px;
  padding: 8px 10px;
  background: var(--color-bg-sunken);
  border-radius: var(--radius-sm);
  align-items: flex-start;
  cursor: pointer;
  -webkit-user-select: none;
  user-select: none;
  -webkit-touch-callout: none;
}
.bd-row:hover {
  background: var(--color-bg-hover);
}
.gender-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-top: 6px;
  flex-shrink: 0;
}
.bd-info {
  min-width: 0;
}
.bd-name-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.bd-name {
  font-size: 13px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 0;
  max-width: 80px;
}
.bd-date {
  font-size: 11px;
  color: var(--color-text-muted);
  white-space: nowrap;
  flex-shrink: 0;
  margin-left: auto;
}
.bd-age {
  font-size: 10px;
  font-weight: 600;
  color: var(--color-text-muted);
  background: var(--color-bg-sunken);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 0 5px;
  line-height: 1.5;
  white-space: nowrap;
  flex-shrink: 0;
}
.bd-tags {
  display: flex;
  flex-wrap: nowrap;
  gap: 4px;
  overflow: hidden;
  flex: 1;
  min-width: 0;
  cursor: pointer;
  -webkit-user-select: none;
  user-select: none;
}
.bd-tags.expanded {
  flex-wrap: wrap;
  overflow: visible;
}
.mini-tag {
  font-size: 10px;
  padding: 1px 5px;
  border: 1px solid;
  border-radius: 8px;
  white-space: nowrap;
  flex-shrink: 0;
}
.mini-tag.more-chip {
  border-color: var(--color-border);
  color: var(--color-text-muted);
  background: var(--color-bg-sunken);
  cursor: pointer;
}

/* 上下文菜单 */
.context-menu {
  position: fixed;
  z-index: 2000;
  min-width: 140px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
  padding: 4px;
  animation: ctx-pop 0.12s ease-out;
}
@keyframes ctx-pop {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: translateY(0); }
}
.context-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  border: none;
  background: transparent;
  color: var(--color-text);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  border-radius: var(--radius-sm);
}
.context-item:hover {
  background: var(--color-bg-sunken);
}
.context-item.danger {
  color: var(--color-danger);
}
.context-item.danger:hover {
  background: var(--color-danger);
  color: #fff;
}
.ctx-icon {
  font-size: 14px;
  width: 16px;
  text-align: center;
}

@media (max-width: 768px) {
  .sidebar {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid var(--color-border);
    max-height: 320px;
  }
}
@media print {
  /* Printing is handled by the calendar / share views; the sidebar
     duplicates that info and only wastes paper. */
  .sidebar {
    display: none !important;
  }
}
</style>
