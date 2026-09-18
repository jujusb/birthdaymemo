import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as api from '@/api'
import type { User } from '@/api/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(false)

  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isGuest = computed(() => user.value?.role === 'guest')
  // 只读：guest 登录账号不可写；匿名分享页另由 ShareView 控制
  const isReadOnly = computed(() => user.value?.role === 'guest')
  const mustChangePassword = computed(() => !!user.value?.must_change_password)

  async function fetchMe() {
    try {
      user.value = await api.getMe()
    } catch {
      user.value = null
    }
  }

  async function login(username: string, password: string, captcha_id?: string, captcha_code?: string) {
    loading.value = true
    try {
      const data = await api.login(username, password, captcha_id, captcha_code)
      user.value = data.user
      return data
    } finally {
      loading.value = false
    }
  }

  async function logout() {
    try {
      await api.logout()
    } finally {
      user.value = null
    }
  }

  async function changePassword(old_password: string, new_password: string) {
    await api.changePassword(old_password, new_password)
    if (user.value) user.value.must_change_password = false
  }

  function setUser(u: User | null) {
    user.value = u
  }

  return {
    user,
    loading,
    isAuthenticated,
    isAdmin,
    isGuest,
    isReadOnly,
    mustChangePassword,
    fetchMe,
    login,
    logout,
    changePassword,
    setUser,
  }
})
