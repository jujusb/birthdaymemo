<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ApiError } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useI18nStore } from '@/stores/i18n'
import { useToast } from '@/composables/useToast'
import ToastContainer from '@/components/ToastContainer.vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const i18n = useI18nStore()
const t = i18n.t
const toast = useToast()

const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const saving = ref(false)

// If user must change password, backend does NOT require old password
const requireOld = computed(() => !auth.mustChangePassword)

// 密码规则动态校验：分别检测长度、大写、小写、数字
const ruleLength = computed(() => newPassword.value.length >= 8)
const ruleUpper = computed(() => /[A-Z]/.test(newPassword.value))
const ruleLower = computed(() => /[a-z]/.test(newPassword.value))
const ruleDigit = computed(() => /[0-9]/.test(newPassword.value))
// 任一规则未满足时显示规则区
const showRules = computed(() => newPassword.value.length > 0)
const allPassed = computed(() => ruleLength.value && ruleUpper.value && ruleLower.value && ruleDigit.value)
// 任务2：确认密码不一致时显示提示
const confirmMismatch = computed(
  () =>
    confirmPassword.value !== '' &&
    newPassword.value !== '' &&
    newPassword.value !== confirmPassword.value,
)

function validate(pw: string): boolean {
  return pw.length >= 8 && /[A-Z]/.test(pw) && /[a-z]/.test(pw) && /[0-9]/.test(pw)
}

async function onSubmit() {
  if (requireOld.value && !oldPassword.value) {
    toast.error(t('common.required'))
    return
  }
  if (!validate(newPassword.value)) {
    toast.error(t('error.invalidInput'))
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    toast.error(t('login.confirmPassword'))
    return
  }
  saving.value = true
  try {
    await auth.changePassword(requireOld.value ? oldPassword.value : '', newPassword.value)
    toast.success(t('login.passwordChanged'))
    // 改密成功后：若有 redirect 参数则跳转，否则回首页
    const r = route.query.redirect
    if (typeof r === 'string' && r.startsWith('/') && !r.startsWith('//')) {
      router.replace(r)
    } else {
      router.replace({ name: 'calendar' })
    }
  } catch (e) {
    const err = e as ApiError
    if (err.status === 401) {
      toast.error(t('login.expired'))
      router.replace({ name: 'login' })
    } else {
      toast.error(err.message || t('common.failed'))
    }
  } finally {
    saving.value = false
  }
}

function goLogin() {
  router.replace({ name: 'login' })
}
</script>

<template>
  <div class="cp-page">
    <ToastContainer />
    <form class="cp-card card" @submit.prevent="onSubmit">
      <h1 class="cp-title">{{ t('login.changePassword') }}</h1>
      <p v-if="requireOld" class="cp-hint">{{ t('login.mustChangePassword') }}</p>

      <div v-if="requireOld" class="col mt-16">
        <label>{{ t('login.password') }}</label>
        <input v-model="oldPassword" type="password" autocomplete="current-password" />
      </div>

      <div class="col mt-16">
        <label>{{ t('login.newPassword') }}</label>
        <input v-model="newPassword" type="password" autocomplete="new-password" />
        <ul v-if="showRules" class="pw-rules">
          <li :class="{ passed: ruleLength }">
            <span class="ico">{{ ruleLength ? '\u2713' : '\u2022' }}</span>
            {{ t('validation.passwordRuleLength') }}
          </li>
          <li :class="{ passed: ruleUpper }">
            <span class="ico">{{ ruleUpper ? '\u2713' : '\u2022' }}</span>
            {{ t('validation.passwordRuleUpper') }}
          </li>
          <li :class="{ passed: ruleLower }">
            <span class="ico">{{ ruleLower ? '\u2713' : '\u2022' }}</span>
            {{ t('validation.passwordRuleLower') }}
          </li>
          <li :class="{ passed: ruleDigit }">
            <span class="ico">{{ ruleDigit ? '\u2713' : '\u2022' }}</span>
            {{ t('validation.passwordRuleDigit') }}
          </li>
        </ul>
        <span v-else class="pw-hint">8+ · A-Z · a-z · 0-9</span>
      </div>

      <div class="col mt-16">
        <label :class="{ 'label-error': confirmMismatch }">{{ t('login.confirmPassword') }}</label>
        <input
          v-model="confirmPassword"
          type="password"
          autocomplete="new-password"
          :class="{ inputError: confirmMismatch }"
        />
        <span v-if="confirmMismatch" class="mismatch-hint">{{ t('login.passwordMismatch') }}</span>
      </div>

      <div class="row between mt-24 actions">
        <button type="button" @click="goLogin">{{ t('common.back') }}</button>
        <button type="submit" class="primary" :disabled="saving || !allPassed">
          {{ saving ? t('common.loading') : t('login.changePassword') }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.cp-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: var(--color-bg);
}
.cp-card {
  width: 100%;
  max-width: 400px;
  padding: 28px;
}
.cp-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 4px;
}
.cp-hint {
  font-size: 13px;
  color: var(--color-text-muted);
}
.pw-hint {
  font-size: 12px;
  color: var(--color-text-muted);
}
.pw-rules {
  list-style: none;
  margin: 6px 0 0;
  padding: 0;
  font-size: 12px;
  display: grid;
  gap: 2px;
}
.pw-rules li {
  color: var(--color-danger, #d9534f);
  display: flex;
  align-items: center;
  gap: 6px;
  transition: color 0.15s ease;
}
.pw-rules li.passed {
  color: var(--color-success, #2a8f2a);
}
.pw-rules li .ico {
  display: inline-block;
  width: 12px;
  text-align: center;
  font-weight: 700;
}
.actions {
  justify-content: flex-end;
}
.label-error {
  color: var(--color-danger, #d9534f);
}
.inputError {
  border-color: var(--color-danger, #d9534f) !important;
}
.mismatch-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-danger, #d9534f);
}
</style>
