import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useAuthStore } from './stores/auth'
import { useI18nStore } from './stores/i18n'
import { useThemeStore } from './stores/theme'
import { setUnauthorizedHandler } from './api/client'
import './styles/main.css'

async function bootstrap() {
  const app = createApp(App)
  const pinia = createPinia()
  app.use(pinia)

  const auth = useAuthStore()
  const i18n = useI18nStore()
  const theme = useThemeStore()

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

  setUnauthorizedHandler(() => {
    auth.setUser(null)
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
