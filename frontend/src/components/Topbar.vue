<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import * as api from '@/api'
import type { UpcomingBirthday } from '@/api/types'
import { useAuthStore } from '@/stores/auth'
import { useI18nStore } from '@/stores/i18n'
import { useThemeStore } from '@/stores/theme'
import { useUiStore } from '@/stores/ui'

const auth = useAuthStore()
const i18n = useI18nStore()
const t = i18n.t
const theme = useThemeStore()
const router = useRouter()
const ui = useUiStore()

const upcoming = ref<UpcomingBirthday[]>([])
let timer: number | undefined

const upcomingCount = computed(() => upcoming.value.length)
const upcomingText = computed(() =>
  upcomingCount.value === 0
    ? t('topbar.noUpcoming')
    : t('topbar.upcoming', { count: upcomingCount.value }),
)

async function loadUpcoming() {
  try {
    upcoming.value = await api.getUpcoming()
  } catch {
    // 401 handled globally; ignore other errors in topbar
  }
}

onMounted(() => {
  loadUpcoming()
  timer = window.setInterval(loadUpcoming, 5 * 60 * 1000)
})
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})

watch(() => ui.birthdayRefreshKey, loadUpcoming)

function toggleTheme() {
  theme.setTheme(theme.theme === 'dark' ? 'light' : 'dark')
}

async function onLogout() {
  try {
    await auth.logout()
  } catch {
    // ignore
  } finally {
    router.replace({ name: 'login' })
  }
}
</script>

<template>
  <header class="topbar">
    <div class="left">
      <span class="welcome">{{ t('topbar.welcome', { name: auth.user?.username ?? '' }) }}</span>
      <span v-if="auth.isGuest" class="guest-badge">👁️ {{ t('role.guest') }}</span>
      <span class="upcoming" :class="{ none: upcomingCount === 0 }">
        <span class="dot" v-if="upcomingCount > 0"></span>
        {{ upcomingText }}
      </span>
    </div>

    <nav class="nav">
      <RouterLink
        to="/"
        class="nav-link icon-link"
        active-class=""
        exact-active-class="router-link-active"
        :title="t('topbar.calendar')"
        aria-label="calendar"
      >
        <span aria-hidden="true">📅</span>
        <span class="nav-label">{{ t('topbar.calendar') }}</span>
      </RouterLink>
      <RouterLink v-if="!auth.isGuest" to="/export" class="nav-link icon-link" :title="t('export.title')" aria-label="export">
        <span aria-hidden="true">📄</span>
        <span class="nav-label">{{ t('export.title') }}</span>
      </RouterLink>
      <RouterLink v-if="!auth.isGuest" to="/settings" class="nav-link icon-link" :title="t('topbar.settings')" aria-label="settings">
        <span aria-hidden="true">⚙️</span>
        <span class="nav-label">{{ t('topbar.settings') }}</span>
      </RouterLink>

      <RouterLink v-if="auth.isAdmin" to="/admin" class="nav-link icon-link" :title="t('admin.title')" aria-label="admin">
        <span aria-hidden="true">🛠️</span>
        <span class="nav-label">{{ t('admin.title') }}</span>
      </RouterLink>

      <button class="nav-link theme-toggle" :title="t('settings.theme')" @click="toggleTheme">
        <span aria-hidden="true">{{ theme.theme === 'dark' ? '☀️' : '🌙' }}</span>
      </button>

      <button class="nav-link logout icon-link" :title="t('login.logout')" @click="onLogout">
        <span aria-hidden="true">🚪</span>
        <span class="nav-label">{{ t('login.logout') }}</span>
      </button>
    </nav>
  </header>
</template>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 12px;
  height: 52px;
  background: var(--color-bg-elevated);
  border-bottom: 1px solid var(--color-border);
  box-shadow: var(--shadow-sm);
  position: sticky;
  top: 0;
  z-index: 50;
  /* 不允许换行：换行会导致顶栏变高、挡住内容 */
  flex-wrap: nowrap;
  overflow: hidden;
}
.left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  overflow: hidden;
}
.welcome {
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.guest-badge {
  font-size: 11px;
  color: var(--color-primary);
  border: 1px solid var(--color-primary);
  border-radius: 10px;
  padding: 1px 8px;
  white-space: nowrap;
}
.upcoming {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--color-text-muted);
  white-space: nowrap;
}
.upcoming .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-warning);
  flex-shrink: 0;
}
.upcoming.none {
  color: var(--color-text-muted);
}
.nav {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-wrap: nowrap;
  flex-shrink: 0;
}
.nav-link {
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  color: var(--color-text);
  border: 1px solid transparent;
  background: transparent;
  cursor: pointer;
  font-size: 13px;
  white-space: nowrap;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.nav-link:hover {
  background: var(--color-bg-sunken);
  border-color: var(--color-border);
}
.nav-link.router-link-active {
  color: var(--color-primary);
  background: var(--color-bg-sunken);
  border-color: var(--color-primary);
}
.icon-link {
  font-size: 15px;
}
.theme-toggle {
  font-size: 15px;
}
.logout:hover {
  color: var(--color-danger);
  border-color: var(--color-danger);
}

/* 平板及小屏幕：隐藏 nav-link 的文字标签，只显示图标 */
@media (max-width: 768px) {
  .topbar {
    padding: 0 8px;
    gap: 4px;
  }
  .welcome {
    font-size: 13px;
    max-width: 140px;
  }
  .upcoming {
    display: none;
  }
  .nav-label {
    display: none;
  }
  .nav-link {
    padding: 6px 8px;
  }
}

/* 极小屏幕：进一步紧凑 */
@media (max-width: 400px) {
  .welcome {
    max-width: 90px;
    font-size: 12px;
  }
  .nav {
    gap: 0;
  }
  .nav-link {
    padding: 6px 6px;
  }
}
</style>
