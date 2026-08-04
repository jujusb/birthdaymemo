<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import * as api from '@/api'
import type { CalendarMonthData, CalendarYearData, CalendarDayBirthday } from '@/api/types'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useToast } from '@/composables/useToast'
import BirthdayFormModal from '@/components/BirthdayFormModal.vue'
import BirthdayListSidebar from '@/components/BirthdayListSidebar.vue'

const i18n = useI18nStore()
const t = i18n.t
const toast = useToast()

const today = new Date()
const view = ref<'month' | 'year'>('month')
const year = ref(today.getFullYear())
const month = ref(today.getMonth() + 1)

const monthData = ref<CalendarMonthData | null>(null)
const yearData = ref<CalendarYearData | null>(null)
const loading = ref(false)
const showBirthdayModal = ref(false)

// 日历"还有N人"弹窗状态
const popupDay = ref<{ day: number; birthdays: CalendarDayBirthday[]; x: number; y: number } | null>(null)

const monthKeyList = ['jan', 'feb', 'mar', 'apr', 'may', 'jun', 'jul', 'aug', 'sep', 'oct', 'nov', 'dec']
const dayKeyList = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat']

const monthNames = computed(() => monthKeyList.map((k) => t('calendar.' + k)))
const dayNames = computed(() => dayKeyList.map((k) => t('calendar.' + k)))

const currentMonthName = computed(() => monthNames.value[month.value - 1])

function genderColor(g: string): string {
  if (g === 'male') return 'var(--color-male)'
  if (g === 'female') return 'var(--color-female)'
  return 'var(--color-gender-none)'
}

// 日历独立显示所有生日，不受左侧标签筛选影响
const birthdaysByDay = computed<Record<number, CalendarDayBirthday[]>>(() => {
  const map: Record<number, CalendarDayBirthday[]> = {}
  if (!monthData.value) return map
  for (const b of monthData.value.birthdays) {
    if (!map[b.day]) map[b.day] = []
    map[b.day].push(b)
  }
  return map
})

interface DayCell {
  day: number
  birthdays: CalendarDayBirthday[]
  isToday: boolean
  isSelfBirthday: boolean // 本人生日当日
}

const cells = computed<(DayCell | null)[]>(() => {
  if (!monthData.value) return []
  const arr: (DayCell | null)[] = []
  for (let i = 0; i < monthData.value.first_weekday; i++) arr.push(null)
  const isThisMonth = year.value === today.getFullYear() && month.value === today.getMonth() + 1
  const selfDay = monthData.value.self_birthday_day || 0
  for (let d = 1; d <= monthData.value.days_in_month; d++) {
    arr.push({
      day: d,
      birthdays: birthdaysByDay.value[d] || [],
      isToday: isThisMonth && d === today.getDate(),
      isSelfBirthday: d === selfDay && selfDay > 0,
    })
  }
  return arr
})

async function loadMonth() {
  loading.value = true
  try {
    monthData.value = await api.getCalendarMonth(year.value, month.value)
  } catch (e) {
    toast.error((e as ApiError).message)
    monthData.value = null
  } finally {
    loading.value = false
  }
}

async function loadYear() {
  loading.value = true
  try {
    yearData.value = await api.getCalendarYear(year.value)
  } catch (e) {
    toast.error((e as ApiError).message)
    yearData.value = null
  } finally {
    loading.value = false
  }
}

function reload() {
  popupDay.value = null
  if (view.value === 'month') loadMonth()
  else loadYear()
}

onMounted(reload)
// 日历不依赖左侧标签筛选，仅响应日期/视图变化
watch([year, month, view], reload)

function prev() {
  if (view.value === 'month') {
    month.value--
    if (month.value < 1) {
      month.value = 12
      year.value--
    }
  } else {
    year.value--
  }
}
function next() {
  if (view.value === 'month') {
    month.value++
    if (month.value > 12) {
      month.value = 1
      year.value++
    }
  } else {
    year.value++
  }
}
function goToday() {
  year.value = today.getFullYear()
  month.value = today.getMonth() + 1
  view.value = 'month'
}
function toggleView() {
  view.value = view.value === 'month' ? 'year' : 'month'
}
function selectMonth(m: number) {
  month.value = m
  view.value = 'month'
}

// 点击"还有N人"时在当前位置弹窗显示全部生日
// 弹窗位置会自动 clamp 到视口内，避免移动端超出屏幕显示不全
function showMore(e: MouseEvent, day: number, birthdays: CalendarDayBirthday[]) {
  const target = e.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()
  // 弹窗尺寸预估（与 CSS .day-popup 一致）
  const POPUP_W = 220
  const POPUP_H_MAX = 280
  const VW = window.innerWidth
  const VH = window.innerHeight
  const MARGIN = 8
  // 期望位置：以点击元素左下角为基准
  let x = rect.left
  let y = rect.bottom + 4
  // 横向 clamp：右边超出 -> 往左移；左边超出 -> 贴左边
  if (x + POPUP_W > VW - MARGIN) x = VW - POPUP_W - MARGIN
  if (x < MARGIN) x = MARGIN
  // 纵向 clamp：底部超出 -> 显示在点击元素上方
  if (y + POPUP_H_MAX > VH - MARGIN) {
    const aboveY = rect.top - POPUP_H_MAX - 4
    y = aboveY > MARGIN ? aboveY : Math.max(MARGIN, VH - POPUP_H_MAX - MARGIN)
  }
  popupDay.value = { day, birthdays, x, y }
}

function closePopup() {
  popupDay.value = null
}

function genderPercents(g: { male: number; female: number; none: number }) {
  const total = g.male + g.female + g.none
  if (total === 0) return null
  return {
    male: (g.male / total) * 100,
    female: (g.female / total) * 100,
    none: (g.none / total) * 100,
  }
}

function onBirthdaySaved() {
  showBirthdayModal.value = false
  reload()
}
</script>

<template>
  <div class="calendar-wrap">
    <BirthdayListSidebar />
    <div class="calendar-view">
      <div class="cal-header row between">
        <div class="row gap-8">
          <button class="icon" @click="prev" :title="t('calendar.prev')">‹</button>
          <button class="month-title" @click="toggleView">
            <template v-if="view === 'month'">{{ currentMonthName }} {{ year }}</template>
            <template v-else>{{ year }}</template>
          </button>
          <button class="icon" @click="next" :title="t('calendar.next')">›</button>
        </div>

        <div class="row gap-8">
          <button @click="goToday">{{ t('calendar.today') }}</button>
          <div class="view-toggle">
            <button :class="{ active: view === 'month' }" @click="view = 'month'">{{ t('calendar.monthView') }}</button>
            <button :class="{ active: view === 'year' }" @click="view = 'year'">{{ t('calendar.yearView') }}</button>
          </div>
          <button class="primary" @click="showBirthdayModal = true">+ {{ t('birthday.new') }}</button>
        </div>
      </div>

      <div v-if="loading" class="loading">{{ t('common.loading') }}</div>

      <!-- Month view -->
      <div v-else-if="view === 'month' && monthData" class="month-grid">
        <div v-for="(dn, i) in dayNames" :key="i" class="day-name">{{ dn }}</div>
        <template v-for="(cell, idx) in cells" :key="idx">
          <div v-if="cell === null" class="day-cell empty"></div>
          <div
            v-else
            class="day-cell"
            :class="{ today: cell.isToday, 'self-birthday': cell.isSelfBirthday }"
          >
            <span v-if="cell.isSelfBirthday" class="self-bd-text">{{ t('calendar.selfBirthday') }}</span>
            <span v-else class="day-num">{{ cell.day }}</span>
            <div class="bd-list">
              <div
                v-for="b in cell.birthdays.slice(0, 2)"
                :key="b.id"
                class="bd-item"
                :title="t('calendar.birthdayOn')"
              >
                <span class="tag-dot" :style="{ background: genderColor(b.gender) }"></span>
                <span class="bd-name">{{ b.name }}</span>
              </div>
              <a
                v-if="cell.birthdays.length > 2"
                class="more-link"
                @click="showMore($event, cell.day, cell.birthdays)"
              >
                {{ t('calendar.more', { count: cell.birthdays.length - 2 }) }}
              </a>
            </div>
          </div>
        </template>
      </div>

      <!-- 还有N人弹窗 -->
      <div v-if="popupDay" class="popup-overlay" @click="closePopup">
        <div
          class="day-popup"
          :style="{ left: popupDay.x + 'px', top: popupDay.y + 'px' }"
          @click.stop
        >
          <div class="popup-head">
            <span>{{ t('calendar.dayN', { day: popupDay.day }) }}</span>
            <button class="popup-close" @click="closePopup">×</button>
          </div>
          <div class="popup-list">
            <div v-for="b in popupDay.birthdays" :key="b.id" class="popup-item">
              <span class="tag-dot" :style="{ background: genderColor(b.gender) }"></span>
              <span class="popup-name">{{ b.name }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Year view -->
      <div v-else-if="view === 'year' && yearData" class="year-grid">
        <div
          v-for="m in yearData.months"
          :key="m.month"
          class="year-cell card"
          @click="selectMonth(m.month)"
        >
          <div class="row between">
            <span class="ym-name">{{ monthNames[m.month - 1] }}</span>
            <span class="ym-count" v-if="m.count > 0">{{ m.count }}</span>
          </div>
          <div class="ym-text">
            {{ m.count > 0 ? t('calendar.yearHasBirthday', { count: m.count }) : t('calendar.noBirthday') }}
          </div>
          <div v-if="genderPercents(m.genders)" class="gender-bar">
            <span class="seg male" :style="{ width: genderPercents(m.genders)!.male + '%' }"></span>
            <span class="seg female" :style="{ width: genderPercents(m.genders)!.female + '%' }"></span>
            <span class="seg none" :style="{ width: genderPercents(m.genders)!.none + '%' }"></span>
          </div>
        </div>
      </div>

      <BirthdayFormModal v-if="showBirthdayModal" @close="showBirthdayModal = false" @saved="onBirthdaySaved" />
    </div>
  </div>
</template>

<style scoped>
.calendar-wrap {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.calendar-view {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.cal-header {
  flex-wrap: wrap;
  gap: 8px;
}
.icon {
  font-size: 18px;
  padding: 2px 10px;
  line-height: 1;
}
.month-title {
  background: transparent;
  border: 1px solid transparent;
  font-size: 16px;
  font-weight: 600;
  padding: 4px 12px;
  min-width: 160px;
  text-align: center;
}
.month-title:hover {
  border-color: var(--color-border);
}
.view-toggle {
  display: inline-flex;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.view-toggle button {
  border: none;
  border-radius: 0;
  background: transparent;
}
.view-toggle button.active {
  background: var(--color-primary);
  color: #fff;
}
.loading {
  text-align: center;
  padding: 40px;
  color: var(--color-text-muted);
}
.month-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 4px;
}
.day-name {
  text-align: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-muted);
  padding: 6px 0;
}
.day-cell {
  min-height: 84px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: 4px 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.day-cell.empty {
  background: transparent;
  border: none;
}
.day-cell.today {
  border-color: var(--color-primary);
  background: var(--color-bg-sunken);
}
.day-num {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-muted);
}
/* 本人生日当日 - RGB 幻彩斜 45° 滚动彩虹边框 */
.day-cell.self-birthday {
  position: relative;
  border: 2px solid transparent;
  background-clip: padding-box;
  z-index: 1;
}
.day-cell.self-birthday::before {
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
  animation: rainbow-slide 3s linear infinite;
  z-index: 0;
  pointer-events: none;
}
.day-cell.self-birthday::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: linear-gradient(
    45deg,
    rgba(255, 0, 64, 0.08),
    rgba(255, 140, 0, 0.08),
    rgba(255, 215, 0, 0.08),
    rgba(0, 255, 127, 0.08),
    rgba(0, 191, 255, 0.08),
    rgba(138, 43, 226, 0.08),
    rgba(255, 0, 64, 0.08)
  );
  background-size: 400% 400%;
  animation: rainbow-slide 3s linear infinite;
  z-index: 0;
  pointer-events: none;
}
@keyframes rainbow-slide {
  0% {
    background-position: 0% 50%;
  }
  100% {
    background-position: 400% 50%;
  }
}
.self-bd-text {
  font-size: 10px;
  font-weight: 700;
  color: var(--color-primary);
  display: block;
  width: 100%;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  position: relative;
  z-index: 1;
}
.day-cell.self-birthday .bd-list {
  position: relative;
  z-index: 1;
}
.bd-list {
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.bd-item {
  display: flex;
  align-items: center;
  font-size: 11px;
  line-height: 1.3;
  overflow: hidden;
}
.bd-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.more-link {
  font-size: 11px;
  color: var(--color-primary);
  cursor: pointer;
  margin-top: 2px;
}
/* 还有N人弹窗 */ 
.popup-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: transparent;
}
.day-popup {
  position: fixed;
  min-width: 160px;
  max-width: 260px;
  max-height: 280px;
  overflow-y: auto;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.16);
  padding: 8px;
  z-index: 1001;
}
.popup-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
  padding: 4px 6px 8px;
  border-bottom: 1px solid var(--color-border);
  margin-bottom: 6px;
}
.popup-close {
  background: transparent;
  border: none;
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  color: var(--color-text-muted);
  padding: 0 4px;
}
.popup-close:hover {
  color: var(--color-danger);
}
.popup-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.popup-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  background: var(--color-bg-sunken);
}
.popup-item:hover {
  background: var(--color-bg-hover);
}
.popup-name {
  font-size: 13px;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 移动端：弹窗占视口大部分宽度，避免被裁切 */
@media (max-width: 640px) {
  .day-popup {
    min-width: 0;
    width: calc(100vw - 16px);
    max-width: calc(100vw - 16px);
    left: 8px !important;
  }
}
.year-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}
.year-cell {
  cursor: pointer;
  padding: 12px;
  transition: border-color 0.15s;
}
.year-cell:hover {
  border-color: var(--color-primary);
}
.ym-name {
  font-weight: 600;
}
.ym-count {
  background: var(--color-primary);
  color: #fff;
  border-radius: 10px;
  padding: 0 8px;
  font-size: 12px;
}
.ym-text {
  font-size: 12px;
  color: var(--color-text-muted);
  margin: 6px 0;
}
.gender-bar {
  display: flex;
  height: 6px;
  border-radius: 3px;
  overflow: hidden;
  background: var(--color-bg-sunken);
}
.gender-bar .seg {
  display: block;
  height: 100%;
}
.gender-bar .male {
  background: var(--color-male);
}
.gender-bar .female {
  background: var(--color-female);
}
.gender-bar .none {
  background: var(--color-gender-none);
}

@media (max-width: 640px) {
  .month-grid {
    grid-template-columns: repeat(7, 1fr);
  }
  .day-cell {
    min-height: 56px;
  }
  .bd-name {
    max-width: 100%;
  }
  .year-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .calendar-wrap {
    flex-direction: column;
  }
}
</style>
