import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useAuthStore } from './stores/auth'
import { useI18nStore } from './stores/i18n'
import { useThemeStore } from './stores/theme'
import { setUnauthorizedHandler } from './api/client'
import { detectGuestOnlyMode, isGuestOnlyMode } from './utils/guestOnly'
import './styles/main.css'

async function bootstrap() {
  const app = createApp(App)
  const pinia = createPinia()
  app.use(pinia)

  const auth = useAuthStore()
  const i18n = useI18nStore()
  const theme = useThemeStore()

  // Detect `--guest-only` backend before the router's initial navigation:
  // only public share pages (`#/s/:token`) exist there.
  await detectGuestOnlyMode()

  if (isGuestOnlyMode()) {
    // No session APIs in guest mode (`/api/me` 404s); share views init
    // their own language. Best-effort init so first paint is localized.
    try {
      const nav = (navigator.language || 'en').slice(0, 2)
      await i18n.init(nav === 'zh' ? 'zh' : 'en')
    } catch { /* ignore, ShareView retries */ }
    theme.init('light', 'blue')
  } else {
    // 关键：必须在 router 安装之前拉取用户信息
    // 否则 app.use(router) 会触发初始导航，beforeEach 守卫会在 fetchMe 完成前运行，
    // 导致已登录用户在刷新页面时被错误地重定向到登录页
    await auth.fetchMe()

    if (auth.isAuthenticated) {
      await i18n.init(auth.user?.language || 'en')
      theme.init(auth.user?.theme || 'light', auth.user?.theme_color || 'blue')
    } else {
      await i18n.init('en')
      theme.init('light', 'blue')
    }
  }

  setUnauthorizedHandler(() => {
    auth.setUser(null)
    // Guest-only 服务没有登录页：未授权直接显示不可用页
    if (isGuestOnlyMode()) {
      router.replace({ name: 'guest-blocked' })
      return
    }
    // 401 时携带当前路径作为 redirect，登录后跳回
    const current = router.currentRoute.value.fullPath
    if (current && current !== '/login' && !current.startsWith('/login')) {
      router.replace({ name: 'login', query: { redirect: current } })
    } else {
      router.replace({ name: 'login' })
    }
  })

  // 在用户信息就绪后再安装路由，触发初始导航
  app.use(router)
  app.mount('#app')
}

bootstrap()
