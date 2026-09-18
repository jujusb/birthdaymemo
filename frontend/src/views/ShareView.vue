<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import * as api from '@/api'
import type { PublicBirthday, PublicShareData, CalendarMonthData, CalendarYearData } from '@/api/types'
import { useI18nStore } from '@/stores/i18n'

const route = useRoute()
const router = useRouter()
const i18n = useI18nStore()
const t = i18n.t

const token = computed(() => String(route.params.token || ''))
const share = ref<PublicShareData | null>(null)
const monthData = ref<CalendarMonthData | null>(null)
const yearData = ref<CalendarYearData | null>(null)
const loading = ref(true)
const notFound = ref(false)
const view = ref<'month' | 'year'>('month')
const today = new Date()
const year = ref(today.getFullYear())
const month = ref(today.getMonth() + 1)
// Guest Mode display toggle for public screens: larger text
const kiosk = ref(false)
// Show upcoming ages in calendar and list (toggleable)
const showAge = ref(true)
// Show birth years in calendar and list (toggleable)
const showYear = ref(true)
const selectedTagIds = ref<number[]>([])

const monthKeyList = ['jan', 'feb', 'mar', 'apr', 'may', 'jun', 'jul', 'aug', 'sep', 'oct', 'nov', 'dec']
const dayKeyList = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat']
const monthNames = computed(() => monthKeyList.map((k) => t('calendar.' + k)))
const dayNames = computed(() => dayKeyList.map((k) => t('calendar.' + k)))

const filteredBirthdays = computed(() => {
  if (!share.value) return []
  if (selectedTagIds.value.length === 0) return share.value.birthdays
  return share.value.birthdays.filter((b) =>
    selectedTagIds.value.every((id) => b.tags.some((tg) => tg.id === id)),
  )
})

function toggleTag(id: number) {
  const i = selectedTagIds.value.indexOf(id)
  if (i >= 0) selectedTagIds.value.splice(i, 1)
  else selectedTagIds.value.push(id)
}

function goExport() {
  router.push({ name: 'share-export', params: { token: token.value } })
}

// Upcoming age for a public birthday: prefer backend value, fall back to
// local computation (next birthday year - birth year). Returns -1 when unknown.
function upcomingAgeOf(b: PublicBirthday): number {
  if (typeof b.upcoming_age === 'number') return b.upcoming_age
  if (!b.birth_year) return -1
  const t = new Date()
  t.setHours(0, 0, 0, 0)
  const y = t.getFullYear()
  let next = new Date(y, b.birth_month - 1, b.birth_day)
  if (next < t) next = new Date(y + 1, b.birth_month - 1, b.birth_day)
  return next.getFullYear() - b.birth_year
}

interface ShareDayItem extends PublicBirthday {
  upcomingAge: number
}

// 日历格显示的年龄：跟随当前查看的年份（切到明年就显示明年满的岁数），
// 仅当年视图沿用基于今天算出的 upcomingAge；出生年份未知时回退该值
const currentYear = today.getFullYear()
function displayAge(b: ShareDayItem): number {
  if (year.value === currentYear) return b.upcomingAge
  if ((b.birth_year || 0) > 0) return year.value - b.birth_year
  return b.upcomingAge
}

async function loadShare() {
  loading.value = true
  notFound.value = false
  try {
    share.value = await api.getPublicShare(token.value)
  } catch {
    notFound.value = true
    share.value = null
  } finally {
    loading.value = false
  }
}

async function loadCalendar() {
  if (!token.value || notFound.value) return
  try {
    if (view.value === 'month') {
      monthData.value = await api.getPublicShareCalendarMonth(token.value, year.value, month.value)
    } else {
      yearData.value = await api.getPublicShareCalendarYear(token.value, year.value)
    }
  } catch {
    // expired/revoked mid-view
    notFound.value = true
  }
}

onMounted(async () => {
  try {
    const nav = (navigator.language || 'en').slice(0, 2)
    await i18n.init(nav === 'zh' ? 'zh' : 'en')
  } catch { /* ignore */ }
  await loadShare()
  await loadCalendar()
})
watch([view, year, month], loadCalendar)

function prev() {
  if (view.value === 'month') {
    month.value--
    if (month.value < 1) {
      month.value = 12
      year.value--
    }
  } else year.value--
}
function next() {
  if (view.value === 'month') {
    month.value++
    if (month.value > 12) {
      month.value = 1
      year.value++
    }
  } else year.value++
}

const birthdaysByDay = computed<Record<number, ShareDayItem[]>>(() => {
  const map: Record<number, ShareDayItem[]> = {}
  if (view.value !== 'month' || !monthData.value) return map
  // month endpoint already scoped; intersect with tag filter by id
  const fullById = new Map(filteredBirthdays.value.map((b) => [b.id, b]))
  for (const b of monthData.value.birthdays) {
    const full = fullById.get(b.id)
    if (!full) continue
    if (b.day !== full.birth_day) continue
    if (!map[b.day]) map[b.day] = []
    // Calendar endpoint carries the authoritative upcoming age (handles Feb 29 etc.)
    const upcomingAge = typeof b.upcoming_age === 'number' ? b.upcoming_age : upcomingAgeOf(full)
    map[b.day].push({ ...full, upcomingAge })
  }
  return map
})
</script>

<template>
  <div class="share-view" :class="{ kiosk }">
    <header class="share-head">
      <div>
        <h1>{{ share?.name || t('app.name') }}</h1>
        <p class="sub">{{ t('share.readonlyHint') }}</p>
      </div>
      <div class="row gap-8">
        <button @click="view = 'month'" :class="{ active: view === 'month' }">{{ t('calendar.monthView') }}</button>
        <button @click="view = 'year'" :class="{ active: view === 'year' }">{{ t('calendar.yearView') }}</button>
        <button @click="kiosk = !kiosk" :title="t('share.kioskHint')">🖥️ {{ t('share.kiosk') }}</button>
        <button @click="showAge = !showAge" :class="{ active: showAge }" :title="t('share.showAgeHint')">🎂 {{ t('share.showAge') }}</button>
        <button @click="showYear = !showYear" :class="{ active: showYear }" :title="t('share.showYearHint')">📆 {{ t('share.showYear') }}</button>
        <button @click="goExport" :title="t('export.title')">📄 {{ t('export.title') }}</button>
      </div>
    </header>

    <div v-if="loading" class="loading">{{ t('common.loading') }}</div>
    <div v-else-if="notFound" class="expired card">
      <h2>{{ t('share.expiredTitle') }}</h2>
      <p>{{ t('share.expiredDesc') }}</p>
    </div>
    <template v-else>
      <div v-if="share && share.tags.length > 0" class="tag-row">
        <button
          v-for="tg in share.tags"
          :key="tg.id"
          class="tag-chip"
          :class="{ checked: selectedTagIds.includes(tg.id) }"
          @click="toggleTag(tg.id)"
        >
          <span class="tag-dot" :style="{ background: tg.color }"></span>{{ tg.name }}
        </button>
        <button v-if="selectedTagIds.length > 0" class="clear-btn" @click="selectedTagIds = []">
          {{ t('list.allTags') }}
        </button>
      </div>

      <div class="cal-nav row between">
        <button @click="prev">‹</button>
        <strong>{{ view === 'month' ? `${monthNames[month - 1]} ${year}` : year }}</strong>
        <button @click="next">›</button>
      </div>

      <div v-if="view === 'month' && monthData" class="month-grid">
        <div v-for="(dn, i) in dayNames" :key="i" class="day-name">{{ dn }}</div>
        <div v-for="n in monthData.first_weekday" :key="'e' + n" class="day-cell empty"></div>
        <div v-for="d in monthData.days_in_month" :key="d" class="day-cell">
          <span class="day-num">{{ d }}</span>
          <div v-for="b in birthdaysByDay[d] || []" :key="b.id" class="bd-item">
            <span class="tag-dot" :style="{ background: b.color }"></span>{{ b.name }}
            <span
              v-if="showAge && displayAge(b) > 0"
              class="age-badge"
              :title="t('common.turnsAge', { age: displayAge(b) })"
            >{{ displayAge(b) }}</span>
            <span
              v-if="showYear && (b.birth_year || 0) > 0"
              class="age-badge"
              :title="t('common.bornIn', { year: b.birth_year })"
            >{{ b.birth_year }}</span>
          </div>
        </div>
      </div>

      <div v-else-if="view === 'year' && yearData" class="year-grid">
        <div v-for="m in yearData.months" :key="m.month" class="year-cell card">
          <strong>{{ monthNames[m.month - 1] }}</strong>
          <span>{{ m.count > 0 ? t('calendar.yearHasBirthday', { count: m.count }) : t('calendar.noBirthday') }}</span>
        </div>
      </div>

      <section class="list card">
        <h3>{{ t('list.title') }} ({{ filteredBirthdays.length }})</h3>
        <div v-if="filteredBirthdays.length === 0" class="empty">{{ t('list.empty') }}</div>
        <div v-for="b in filteredBirthdays" :key="b.id" class="list-row">
          <span class="tag-dot" :style="{ background: b.color }"></span>
          <span class="grow">{{ b.name }} · {{ b.birth_month }}/{{ b.birth_day }}</span>
          <span
            v-if="showAge && upcomingAgeOf(b) > 0"
            class="age-badge"
            :title="t('common.turnsAge', { age: upcomingAgeOf(b) })"
          >{{ t('common.turnsAge', { age: upcomingAgeOf(b) }) }}</span>
          <span
            v-if="showYear && (b.birth_year || 0) > 0"
            class="age-badge"
            :title="t('common.bornIn', { year: b.birth_year })"
          >{{ b.birth_year }}</span>
          <span class="muted">{{ b.days_until }}d</span>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.share-view {
  max-width: 1000px;
  margin: 0 auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.share-view.kiosk {
  font-size: 18px;
  max-width: 1400px;
}
.share-view.kiosk .day-cell {
  min-height: 110px;
}
.share-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.share-head h1 {
  font-size: 20px;
  margin: 0;
}
.sub {
  color: var(--color-text-muted);
  font-size: 13px;
  margin: 2px 0 0;
}
button.active {
  background: var(--color-primary);
  color: #fff;
}
.tag-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.tag-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 4px 10px;
}
.tag-chip.checked {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: #fff;
}
.cal-nav {
  align-items: center;
}
.month-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 4px;
}
.day-name {
  text-align: center;
  font-size: 12px;
  color: var(--color-text-muted);
  font-weight: 600;
}
.day-cell {
  min-height: 84px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  padding: 4px 6px;
  background: var(--color-bg-elevated);
}
.day-cell.empty {
  border: none;
  background: transparent;
}
.day-num {
  font-size: 12px;
  color: var(--color-text-muted);
  font-weight: 600;
}
.bd-item {
  font-size: 12px;
  display: flex;
  gap: 4px;
  align-items: center;
}
.age-badge {
  display: inline-block;
  min-width: 20px;
  text-align: center;
  font-size: 11px;
  font-weight: 600;
  line-height: 1.4;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--color-bg-sunken);
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
  white-space: nowrap;
}
.list-row .age-badge {
  flex-shrink: 0;
}
.year-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}
.year-cell {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.list {
  padding: 12px;
}
.list-row {
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 6px 0;
  border-bottom: 1px solid var(--color-border);
}
.muted {
  color: var(--color-text-muted);
  font-size: 12px;
}
.expired {
  padding: 32px;
  text-align: center;
}
.loading {
  text-align: center;
  padding: 40px;
  color: var(--color-text-muted);
}
@media (max-width: 640px) {
  .year-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
