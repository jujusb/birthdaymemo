<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useI18nStore } from '@/stores/i18n'
import { useThemeStore } from '@/stores/theme'
import ToastContainer from '@/components/ToastContainer.vue'

const auth = useAuthStore()
const i18n = useI18nStore()
const t = i18n.t
const theme = useThemeStore()
const router = useRouter()

// 侧边栏收缩状态：移动端默认收缩，桌面端默认展开
const collapsed = ref(false)
const isMobile = ref(false)

function checkViewport() {
  isMobile.value = window.innerWidth < 768
  collapsed.value = isMobile.value
}

onMounted(() => {
  checkViewport()
  window.addEventListener('resize', checkViewport)
})
onUnmounted(() => {
  window.removeEventListener('resize', checkViewport)
})

function toggleSidebar() {
  collapsed.value = !collapsed.value
}

const navItems = computed(() => [
  { to: '/admin/users', label: t('admin.users'), icon: '👤' },
  { to: '/admin/system', label: t('admin.systemConfig'), icon: '⚙' },
  { to: '/admin/smtp', label: t('admin.smtp'), icon: '✉' },
  { to: '/admin/template', label: t('admin.emailTemplate'), icon: '📝' },
  { to: '/admin/languages', label: t('admin.languages'), icon: '🌐' },
  { to: '/admin/logs', label: t('admin.operationLogs'), icon: '📋' },
])

async function onLogout() {
  try {
    await auth.logout()
  } catch {
    // ignore
  } finally {
    router.replace({ name: 'login' })
  }
}

function toggleTheme() {
  theme.setTheme(theme.theme === 'dark' ? 'light' : 'dark')
}
</script>

<template>
  <div class="admin-layout" :class="{ collapsed }">
    <ToastContainer />
    <aside class="sidebar">
      <div class="sidebar-head">
        <span v-if="!collapsed" class="title">{{ t('admin.title') }}</span>
        <button class="toggle-btn" @click="toggleSidebar" :title="t('admin.sidebar')">
          <span>{{ collapsed ? '▶' : '◀' }}</span>
        </button>
      </div>

      <nav class="nav">
        <RouterLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="nav-item"
          :title="item.label"
        >
          <span class="icon">{{ item.icon }}</span>
          <span v-if="!collapsed" class="label">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <RouterLink to="/" class="nav-item" :title="t('admin.backToApp')">
          <span class="icon">🏠</span>
          <span v-if="!collapsed" class="label">{{ t('admin.backToApp') }}</span>
        </RouterLink>
        <button class="nav-item" :title="t('settings.theme')" @click="toggleTheme">
          <span class="icon">{{ theme.theme === 'dark' ? '☀' : '🌙' }}</span>
          <span v-if="!collapsed" class="label">{{ t('settings.theme') }}</span>
        </button>
        <button class="nav-item" :title="t('login.logout')" @click="onLogout">
          <span class="icon">🚪</span>
          <span v-if="!collapsed" class="label">{{ t('login.logout') }}</span>
        </button>
      </div>
    </aside>

    <main class="content">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.admin-layout {
  display: flex;
  height: 100vh;
}
.sidebar {
  width: 220px;
  flex-shrink: 0;
  background: var(--color-bg-elevated);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  transition: width 0.2s ease;
  overflow: hidden;
}
.admin-layout.collapsed .sidebar {
  width: 56px;
}
.sidebar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  border-bottom: 1px solid var(--color-border);
  gap: 8px;
}
.title {
  font-weight: 700;
  font-size: 15px;
  white-space: nowrap;
  color: var(--color-primary);
}
.toggle-btn {
  border: 1px solid var(--color-border);
  background: transparent;
  border-radius: var(--radius-sm);
  padding: 2px 8px;
  cursor: pointer;
  font-size: 12px;
  color: var(--color-text-muted);
  flex-shrink: 0;
}
.toggle-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}
.nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 8px;
  gap: 2px;
  overflow-y: auto;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  color: var(--color-text);
  text-decoration: none;
  font-size: 14px;
  border: 1px solid transparent;
  background: transparent;
  cursor: pointer;
  text-align: left;
  width: 100%;
  white-space: nowrap;
}
.nav-item:hover {
  background: var(--color-bg-sunken);
  border-color: var(--color-border);
}
.nav-item.router-link-active {
  color: var(--color-primary);
  background: var(--color-bg-sunken);
  border-color: var(--color-primary);
}
.icon {
  flex-shrink: 0;
  font-size: 16px;
  width: 20px;
  text-align: center;
}
.label {
  overflow: hidden;
  text-overflow: ellipsis;
}
.sidebar-footer {
  padding: 8px;
  border-top: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.content {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 16px;
}

@media (max-width: 768px) {
  .content {
    padding: 12px;
  }
}
</style>
