<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import * as api from '@/api'
import type { CaptchaData } from '@/api/types'
import { ApiError } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useI18nStore } from '@/stores/i18n'
import { useThemeStore } from '@/stores/theme'
import { useToast } from '@/composables/useToast'
import ToastContainer from '@/components/ToastContainer.vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const i18n = useI18nStore()
const t = i18n.t
const theme = useThemeStore()
const toast = useToast()

const form = ref({ username: '', password: '', captchaCode: '' })
const captcha = ref<CaptchaData | null>(null)
const showCaptcha = ref(false)
const error = ref('')

async function refreshCaptcha() {
  try {
    captcha.value = await api.getCaptcha()
  } catch {
    captcha.value = null
  }
}

onMounted(() => {
  refreshCaptcha()
})

// 解析 redirect 参数：优先取 query.redirect，确保是站内路径
function resolveRedirect(): string | null {
  const r = route.query.redirect
  if (typeof r === 'string' && r.startsWith('/') && !r.startsWith('//')) {
    return r
  }
  return null
}

async function onSubmit() {
  error.value = ''
  // 1. 优先校验验证码是否填写（如已显示验证码）
  if (showCaptcha.value && !form.value.captchaCode) {
    error.value = t('login.captchaPlaceholder')
    return
  }
  // 2. 校验用户名和密码是否填写
  if (!form.value.username || !form.value.password) {
    error.value = t('common.required')
    return
  }
  try {
    const data = await auth.login(
      form.value.username,
      form.value.password,
      captcha.value?.id,
      form.value.captchaCode || undefined,
    )
    await i18n.init(data.user.language)
    theme.init(data.user.theme, data.user.theme_color || 'blue')
    if (data.must_change_password) {
      // 需要改密时，保留 redirect 参数，改密成功后再跳转
      router.replace({ name: 'change-password', query: route.query })
    } else {
      const target = resolveRedirect()
      router.replace(target || { name: 'calendar' })
    }
  } catch (e) {
    const err = e as ApiError
    error.value = err.message || t('login.invalid')
    toast.error(error.value)
    // 根据后端返回的 needs_captcha 标志决定是否显示验证码
    // 后端策略：10 分钟内失败 ≥5 次才会要求验证码
    const needCaptcha = err.data && err.data.needs_captcha === true
    if (needCaptcha && !showCaptcha.value) {
      showCaptcha.value = true
      form.value.captchaCode = ''
      await refreshCaptcha()
    } else if (showCaptcha.value) {
      // 已显示验证码则刷新一张新的
      form.value.captchaCode = ''
      await refreshCaptcha()
    }
  }
}
</script>

<template>
  <div class="login-page">
    <ToastContainer />
    <form class="login-card card" @submit.prevent="onSubmit">
      <img class="login-logo" src="/logo512.png" :alt="t('app.name')" />
      <h1 class="login-title">{{ t('app.name') }}</h1>
      <h2 class="login-subtitle">{{ t('login.title') }}</h2>

      <div v-if="error" class="error-box">{{ error }}</div>

      <div class="col">
        <label>{{ t('login.username') }}</label>
        <input
          v-model="form.username"
          type="text"
          autocomplete="username"
          :placeholder="t('login.username')"
        />
      </div>

      <div class="col mt-16">
        <label>{{ t('login.password') }}</label>
        <input
          v-model="form.password"
          type="password"
          autocomplete="current-password"
          :placeholder="t('login.password')"
        />
      </div>

      <div v-if="showCaptcha" class="col mt-16">
        <label>{{ t('login.captcha') }}</label>
        <div class="captcha-row">
          <input
            v-model="form.captchaCode"
            type="text"
            maxlength="4"
            autocomplete="off"
            :placeholder="t('login.captchaPlaceholder')"
          />
          <img
            v-if="captcha"
            class="captcha-img"
            :src="captcha.image"
            :alt="t('login.captcha')"
            :title="t('common.preview')"
            @click="refreshCaptcha"
          />
          <button type="button" class="refresh-btn" @click="refreshCaptcha">↻</button>
        </div>
      </div>

      <button type="submit" class="primary submit-btn mt-24" :disabled="auth.loading">
        {{ auth.loading ? t('common.loading') : t('login.submit') }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-hover));
}
.login-card {
  width: 100%;
  max-width: 380px;
  padding: 32px 28px;
  box-shadow: var(--shadow-md);
}
.login-logo {
  width: 72px;
  height: 72px;
  display: block;
  margin: 0 auto 12px;
  border-radius: 16px;
  object-fit: contain;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
.login-title {
  font-size: 22px;
  font-weight: 700;
  text-align: center;
  color: var(--color-primary);
}
.login-subtitle {
  font-size: 14px;
  font-weight: 400;
  text-align: center;
  color: var(--color-text-muted);
  margin-top: 4px;
  margin-bottom: 20px;
}
.error-box {
  background: rgba(231, 76, 60, 0.1);
  color: var(--color-danger);
  border: 1px solid var(--color-danger);
  border-radius: var(--radius-sm);
  padding: 8px 12px;
  font-size: 13px;
  margin-bottom: 16px;
}
.captcha-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.captcha-img {
  height: 36px;
  width: 110px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  cursor: pointer;
  object-fit: cover;
}
.refresh-btn {
  flex-shrink: 0;
  padding: 6px 10px;
}
.submit-btn {
  width: 100%;
  padding: 10px;
  font-size: 15px;
}
</style>
