import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { isGuestOnlyMode } from '@/utils/guestOnly'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { public: true },
  },
  {
    path: '/s/:token',
    name: 'share',
    component: () => import('@/views/ShareView.vue'),
    meta: { public: true },
  },
  {
    // 分享页的导出窗口：复用 ExportView（shareToken 模式：公开数据、无保存）
    path: '/s/:token/export',
    name: 'share-export',
    component: () => import('@/views/ExportView.vue'),
    props: (route) => ({ shareToken: String(route.params.token || '') }),
    meta: { public: true },
  },
  {
    path: '/change-password',
    name: 'change-password',
    component: () => import('@/views/ChangePasswordView.vue'),
  },
  {
    path: '/',
    component: () => import('@/views/MainLayout.vue'),
    children: [
      { path: '', name: 'calendar', component: () => import('@/views/CalendarView.vue') },
      { path: 'settings', name: 'settings', component: () => import('@/views/SettingsView.vue') },
      { path: 'export', name: 'export', component: () => import('@/views/ExportView.vue') },
    ],
  },
  {
    path: '/admin',
    component: () => import('@/views/admin/AdminLayout.vue'),
    children: [
      { path: '', redirect: { name: 'admin-users' } },
      { path: 'users', name: 'admin-users', component: () => import('@/views/admin/UsersView.vue') },
      { path: 'system', name: 'admin-system', component: () => import('@/views/admin/SystemConfigView.vue') },
      { path: 'smtp', name: 'admin-smtp', component: () => import('@/views/admin/SmtpView.vue') },
      { path: 'template', name: 'admin-template', component: () => import('@/views/admin/EmailTemplateView.vue') },
      { path: 'languages', name: 'admin-languages', component: () => import('@/views/admin/LanguagesView.vue') },
      { path: 'logs', name: 'admin-logs', component: () => import('@/views/admin/OperationLogsView.vue') },
    ],
  },
  {
    path: '/not-available',
    name: 'guest-blocked',
    component: () => import('@/views/GuestBlockedView.vue'),
    meta: { public: true },
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to) => {
  // `--guest-only` backend: only public share pages exist. Everything else
  // (login, calendar, settings, admin, ...) must not be accessible.
  // This check runs first so guest mode never redirects to the login page
  // (which itself is blocked).
  if (isGuestOnlyMode()) {
    if (to.name === 'share' || to.name === 'share-export' || to.name === 'guest-blocked') return true
    return { name: 'guest-blocked' }
  }
  const auth = useAuthStore()
  // 已登录用户访问登录页 → 跳转主页
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'calendar' }
  }
  if (to.meta.public) return true
  if (!auth.isAuthenticated) {
    // 携带原始目标路径，登录成功后跳回
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (auth.mustChangePassword && to.name !== 'change-password') {
    return { name: 'change-password' }
  }
  // guest 只读账号：不可访问导出（写接口）与管理后台
  if (auth.isGuest && (to.name === 'export' || to.path.startsWith('/admin'))) {
    return { name: 'calendar' }
  }
  // 管理员路由检查（无权限跳回首页）
  if (to.path.startsWith('/admin') && !auth.isAdmin) {
    return { name: 'calendar' }
  }
  return true
})

export default router
