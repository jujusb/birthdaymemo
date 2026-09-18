<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import * as api from '@/api'
import type { Language, ReminderSetting, EmailStatus } from '@/api/types'
import { ApiError } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useI18nStore } from '@/stores/i18n'
import { useThemeStore } from '@/stores/theme'
import { THEME_COLORS, DEFAULT_THEME_COLOR } from '@/stores/themes'
import { useToast } from '@/composables/useToast'
import AboutSection from '@/components/AboutSection.vue'
import SharingPanel from '@/components/SharingPanel.vue'

const auth = useAuthStore()
const i18n = useI18nStore()
const t = i18n.t
const theme = useThemeStore()
const toast = useToast()

const reminder = ref<ReminderSetting | null>(null)
const language = ref('')
const themeChoice = ref<'dark' | 'light'>('light')
const themeColorChoice = ref<string>(DEFAULT_THEME_COLOR)
const topbarRangeDays = ref(7)
const availableLanguages = ref<Language[]>([])
const loading = ref(false)
const saving = ref(false)

// 任务3：本人生日
const selfBirthYear = ref<number>(0)
const selfBirthMonth = ref<number>(0)
const selfBirthDay = ref<number>(0)
const monthList = Array.from({ length: 12 }, (_, i) => i + 1)
function dayListForMonth(): number[] {
  const m = selfBirthMonth.value
  if (!m) return Array.from({ length: 31 }, (_, i) => i + 1)
  let days = 31
  if ([4, 6, 9, 11].includes(m)) days = 30
  else if (m === 2) {
    const y = selfBirthYear.value
    const isLeap = y > 0 && (y % 4 === 0 && (y % 100 !== 0 || y % 400 === 0))
    days = isLeap ? 29 : 28
  }
  return Array.from({ length: days }, (_, i) => i + 1)
}
function onSelfMonthChange() {
  // 若当前日超出该月天数，则截断
  const max = selfBirthMonth.value ? dayListForMonth().length : 31
  if (selfBirthDay.value > max) selfBirthDay.value = max
}
function clearSelfBirthday() {
  selfBirthYear.value = 0
  selfBirthMonth.value = 0
  selfBirthDay.value = 0
}

// 邮箱确认流程
const confirmedEmail = ref('')
const pendingEmail = ref('')
const emailInput = ref('')
const emailStatus = ref<EmailStatus | null>(null)
const sendingConfirm = ref(false)

// 修改密码（复用首次登录强制修改的校验规则）
const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const changingPassword = ref(false)
const ruleLength = computed(() => newPassword.value.length >= 8)
const ruleUpper = computed(() => /[A-Z]/.test(newPassword.value))
const ruleLower = computed(() => /[a-z]/.test(newPassword.value))
const ruleDigit = computed(() => /[0-9]/.test(newPassword.value))
const showRules = computed(() => newPassword.value.length > 0)
const allPassed = computed(
  () => ruleLength.value && ruleUpper.value && ruleLower.value && ruleDigit.value,
)
// 任务2：确认密码不一致时显示提示
const confirmMismatch = computed(
  () =>
    confirmPassword.value !== '' &&
    newPassword.value !== '' &&
    newPassword.value !== confirmPassword.value,
)
function validatePassword(pw: string): boolean {
  return pw.length >= 8 && /[A-Z]/.test(pw) && /[a-z]/.test(pw) && /[0-9]/.test(pw)
}
async function changePassword() {
  if (!oldPassword.value) {
    toast.error(t('common.required'))
    return
  }
  if (!validatePassword(newPassword.value)) {
    toast.error(t('error.invalidInput'))
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    toast.error(t('login.confirmPassword'))
    return
  }
  changingPassword.value = true
  try {
    await api.changePassword(oldPassword.value, newPassword.value)
    toast.success(t('login.passwordChanged'))
    oldPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    changingPassword.value = false
  }
}

const dayKeyList = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat']
const dayNames = computed(() => dayKeyList.map((k) => t('calendar.' + k)))
const hours = Array.from({ length: 24 }, (_, i) => i)
const monthlyDays = Array.from({ length: 28 }, (_, i) => i + 1)

const emailStatusLabel = computed(() => {
  if (confirmedEmail.value) return t('email.statusVerified')
  if (pendingEmail.value) return t('email.statusPending')
  return t('email.statusNone')
})
const emailStatusClass = computed(() => {
  if (confirmedEmail.value) return 'verified'
  if (pendingEmail.value) return 'pending'
  return 'none'
})
const cooldownLeft = computed(() => emailStatus.value?.cooldown_left ?? 3)
const confirmBtnLabel = computed(() => {
  if (pendingEmail.value) return t('email.resendConfirm')
  return t('email.requestConfirm')
})

async function load() {
  loading.value = true
  try {
    const s = await api.getSettings()
    reminder.value = { ...s.reminder }
    confirmedEmail.value = s.email
    pendingEmail.value = s.pending_email
    emailInput.value = s.email || s.pending_email || ''
    language.value = s.language
    themeChoice.value = s.theme
    themeColorChoice.value = s.theme_color || DEFAULT_THEME_COLOR
    topbarRangeDays.value = s.topbar_range_days
    availableLanguages.value = s.available_languages
    // 任务3：本人生日
    selfBirthYear.value = s.self_birth_year || 0
    selfBirthMonth.value = s.self_birth_month || 0
    selfBirthDay.value = s.self_birth_day || 0
    // 单独获取邮箱状态（含冷却次数）
    try {
      emailStatus.value = await api.getEmailStatus()
    } catch {
      // 忽略，状态已从 settings 获取
    }
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    loading.value = false
  }
}

onMounted(load)

async function sendConfirm() {
  const addr = emailInput.value.trim()
  if (!addr) {
    toast.error(t('email.address'))
    return
  }
  sendingConfirm.value = true
  try {
    const res = await api.requestEmailConfirm(addr)
    pendingEmail.value = addr
    if (res) {
      emailStatus.value = res
    }
    toast.success(t('email.confirmSent'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    sendingConfirm.value = false
  }
}

// 选择主题色时立即预览（不写库）
function previewThemeColor(code: string) {
  themeColorChoice.value = code
  theme.setThemeColor(code)
}

async function save() {
  if (!reminder.value) return
  saving.value = true
  try {
    const langChanged = language.value !== i18n.lang
    const themeChanged = themeChoice.value !== theme.theme
    const colorChanged = themeColorChoice.value !== theme.themeColor
    await api.updateSettings({
      reminder: reminder.value,
      language: language.value,
      theme: themeChoice.value,
      theme_color: themeColorChoice.value,
      topbar_range_days: topbarRangeDays.value,
      self_birth_year: selfBirthYear.value,
      self_birth_month: selfBirthMonth.value,
      self_birth_day: selfBirthDay.value,
    })
    if (langChanged) await i18n.load(language.value)
    if (themeChanged) theme.setTheme(themeChoice.value)
    if (colorChanged) theme.setThemeColor(themeColorChoice.value)
    if (auth.user) {
      auth.user.language = language.value
      auth.user.theme = themeChoice.value
      auth.user.theme_color = themeColorChoice.value
      auth.user.email = confirmedEmail.value
      auth.user.topbar_range_days = topbarRangeDays.value
    }
    toast.success(t('settings.saved'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="settings-view">
    <h2 class="page-title">{{ t('settings.title') }}</h2>

    <div v-if="loading" class="loading">{{ t('common.loading') }}</div>

    <template v-else-if="reminder">
      <!-- 任务3：本人生日 -->
      <section class="card mt-16">
        <h3 class="section-title">{{ t('settings.selfBirthday') }}</h3>
        <span class="hint">{{ t('settings.selfBirthdayHint') }}</span>
        <div class="row gap-16 mt-16 self-birthday-row">
          <div class="col grow">
            <label>{{ t('birthday.year') }}</label>
            <input
              v-model.number="selfBirthYear"
              type="number"
              min="0"
              max="9999"
              :placeholder="t('birthday.year')"
            />
          </div>
          <div class="col grow">
            <label>{{ t('birthday.month') }}</label>
            <select v-model.number="selfBirthMonth" @change="onSelfMonthChange">
              <option :value="0">--</option>
              <option v-for="m in monthList" :key="m" :value="m">{{ m }}</option>
            </select>
          </div>
          <div class="col grow">
            <label>{{ t('birthday.day') }}</label>
            <select v-model.number="selfBirthDay">
              <option :value="0">--</option>
              <option v-for="d in dayListForMonth()" :key="d" :value="d">{{ d }}</option>
            </select>
          </div>
          <button type="button" class="self-clear-btn" @click="clearSelfBirthday">
            {{ t('common.reset') }}
          </button>
          <button
            type="button"
            class="primary self-save-btn"
            :disabled="saving"
            @click="save"
          >
            {{ saving ? t('common.loading') : t('common.save') }}
          </button>
        </div>
      </section>

      <!-- Reminder -->
      <section class="card mt-16 reminder-section">
        <div v-if="!confirmedEmail" class="reminder-mask">
          <div class="mask-text">{{ t('settings.reminderMaskHint') }}</div>
        </div>
        <div :class="{ 'reminder-masked': !confirmedEmail }">
          <h3 class="section-title">{{ t('settings.reminder') }}</h3>

          <div class="col mt-8">
            <label>{{ t('settings.reminderMode') }}</label>
            <div class="row gap-8">
              <label class="radio-opt">
                <input v-model.number="reminder.mode" type="radio" :value="1" />
                {{ t('settings.modeDaysBefore') }}
              </label>
              <label class="radio-opt">
                <input v-model.number="reminder.mode" type="radio" :value="2" />
                {{ t('settings.modePeriodic') }}
              </label>
            </div>
          </div>

          <!-- Mode 1: days before -->
          <div v-if="reminder.mode === 1" class="row gap-16 mt-16">
            <div class="col grow">
              <label>{{ t('settings.daysBefore') }}</label>
              <input v-model.number="reminder.days_before" type="number" min="0" max="365" />
            </div>
            <div class="col grow">
              <label>{{ t('settings.remindTime') }}</label>
              <select v-model.number="reminder.remind_hour">
                <option v-for="h in hours" :key="h" :value="h">{{ h }}:00</option>
              </select>
            </div>
          </div>

          <!-- Mode 2: periodic -->
          <div v-else class="col mt-16 gap-16">
            <div class="row gap-16">
              <div class="col grow">
                <label>{{ t('settings.reminderMode') }}</label>
                <select v-model="reminder.sub_mode">
                  <option value="weekly">{{ t('settings.weeklyDay') }}</option>
                  <option value="monthly">{{ t('settings.monthlyDay') }}</option>
                </select>
              </div>
              <div v-if="reminder.sub_mode === 'weekly'" class="col grow">
                <label>{{ t('settings.weeklyDay') }}</label>
                <select v-model.number="reminder.weekly_day">
                  <option v-for="(dn, i) in dayNames" :key="i" :value="i">{{ dn }}</option>
                </select>
              </div>
              <div v-else class="col grow">
                <label>{{ t('settings.monthlyDay') }}</label>
                <select v-model.number="reminder.monthly_day">
                  <option v-for="d in monthlyDays" :key="d" :value="d">{{ d }}</option>
                </select>
              </div>
              <div class="col grow">
                <label>{{ t('settings.remindTime') }}</label>
                <select v-model.number="reminder.remind_hour">
                  <option v-for="h in hours" :key="h" :value="h">{{ h }}:00</option>
                </select>
              </div>
            </div>
          </div>

          <div class="row mt-16">
            <label class="check-opt">
              <input v-model="reminder.email_enabled" type="checkbox" />
              {{ t('settings.emailEnabled') }}
            </label>
          </div>
        </div>
      </section>

      <!-- Email Address & Confirmation -->
      <section class="card mt-16">
        <h3 class="section-title">{{ t('email.address') }}</h3>
        <div class="row gap-8 mt-8 email-status-row">
          <span class="email-status-badge" :class="emailStatusClass">{{ emailStatusLabel }}</span>
          <span v-if="confirmedEmail" class="email-confirmed-text">{{ confirmedEmail }}</span>
        </div>
        <div v-if="pendingEmail" class="email-pending-hint">{{ t('email.pending') }} ({{ pendingEmail }})</div>
        <div class="row gap-8 mt-16">
          <input
            v-model="emailInput"
            type="email"
            :placeholder="t('email.address')"
            class="grow"
          />
          <button
            type="button"
            class="primary"
            :disabled="sendingConfirm || cooldownLeft <= 0"
            @click="sendConfirm"
          >
            {{ sendingConfirm ? t('common.loading') : confirmBtnLabel }}
          </button>
        </div>
        <span class="hint">{{ t('email.address') }} · {{ cooldownLeft }}/3</span>
      </section>

      <!-- Language & Theme -->
      <section class="card mt-16">
        <h3 class="section-title">{{ t('settings.language') }} &amp; {{ t('settings.theme') }}</h3>

        <div class="col mt-8">
          <label>{{ t('settings.language') }}</label>
          <select v-model="language">
            <option v-for="l in availableLanguages" :key="l.code" :value="l.code">{{ l.name }}</option>
          </select>
          <span class="hint">{{ t('settings.languageHint') }}</span>
        </div>

        <div class="col mt-16">
          <label>{{ t('settings.theme') }}</label>
          <div class="row gap-8">
            <label class="radio-opt">
              <input v-model="themeChoice" type="radio" value="light" />
              {{ t('settings.light') }}
            </label>
            <label class="radio-opt">
              <input v-model="themeChoice" type="radio" value="dark" />
              {{ t('settings.dark') }}
            </label>
          </div>
        </div>

        <div class="col mt-16">
          <label>{{ t('settings.themeColor') }}</label>
          <div class="theme-color-grid">
            <button
              v-for="c in THEME_COLORS"
              :key="c.code"
              type="button"
              class="theme-color-swatch"
              :class="{ active: themeColorChoice === c.code }"
              :style="{ background: themeChoice === 'dark' ? c.dark.primary : c.light.primary }"
              :title="t('themeColor.' + c.code)"
              @click="previewThemeColor(c.code)"
            ></button>
          </div>
          <span class="hint">{{ t('settings.themeColorHint') }}</span>
        </div>

        <div class="col mt-16">
          <label>{{ t('settings.topbarRange') }}</label>
          <input v-model.number="topbarRangeDays" type="number" min="1" max="365" />
        </div>
      </section>

      <div class="row mt-16 actions">
        <button class="primary" :disabled="saving" @click="save">
          {{ saving ? t('common.loading') : t('common.save') }}
        </button>
      </div>

      <!-- 修改密码 -->
      <section class="card mt-16">
        <h3 class="section-title">{{ t('login.changePassword') }}</h3>
        <div class="col mt-8">
          <label>{{ t('login.password') }}</label>
          <input
            v-model="oldPassword"
            type="password"
            autocomplete="current-password"
            :placeholder="t('login.password')"
          />
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
          <span v-else class="hint">8+ · A-Z · a-z · 0-9</span>
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
        <div class="row mt-16 actions">
          <button
            class="primary"
            :disabled="changingPassword || !allPassed"
            @click="changePassword"
          >
            {{ changingPassword ? t('common.loading') : t('login.changePassword') }}
          </button>
        </div>
      </section>

      <!-- 分享与委托 -->
      <SharingPanel />

      <!-- 关于 -->
      <AboutSection />
    </template>
  </div>
</template>

<style scoped>
.settings-view {
  max-width: 720px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
}
.page-title {
  font-size: 20px;
  font-weight: 700;
}
.section-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}
.loading {
  text-align: center;
  padding: 32px;
  color: var(--color-text-muted);
}
.radio-opt {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 14px;
  color: var(--color-text);
  cursor: pointer;
}
.radio-opt input {
  width: auto;
}
.check-opt {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--color-text);
  cursor: pointer;
}
.check-opt input {
  width: auto;
}
.hint {
  font-size: 12px;
  color: var(--color-text-muted);
}
.actions {
  justify-content: flex-end;
}
.email-status-row {
  align-items: center;
}
.email-status-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.email-status-badge.verified {
  background: rgba(22, 163, 74, 0.15);
  color: #16a34a;
}
.email-status-badge.pending {
  background: rgba(217, 119, 6, 0.15);
  color: #d97706;
}
.email-status-badge.none {
  background: var(--color-bg-sunken);
  color: var(--color-text-muted);
}
.email-confirmed-text {
  font-size: 13px;
  color: var(--color-text);
  word-break: break-all;
}
.email-pending-hint {
  font-size: 12px;
  color: #d97706;
  margin-top: 6px;
}
.theme-color-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.theme-color-swatch {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 2px solid var(--color-border);
  cursor: pointer;
  padding: 0;
  transition: transform 0.15s, border-color 0.15s, box-shadow 0.15s;
}
.theme-color-swatch:hover {
  transform: scale(1.1);
}
.theme-color-swatch.active {
  border-color: var(--color-text);
  box-shadow: 0 0 0 2px var(--color-bg-elevated), 0 0 0 4px var(--color-text);
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
.self-birthday-row {
  align-items: flex-end;
}
.self-clear-btn {
  flex-shrink: 0;
  padding: 8px 12px;
  font-size: 13px;
}
.self-save-btn {
  flex-shrink: 0;
  padding: 8px 16px;
  font-size: 13px;
}
/* 任务5（本批次）：未绑定邮箱时遮罩提醒设置 */
.reminder-section {
  position: relative;
}
.reminder-mask {
  position: absolute;
  inset: 0;
  z-index: 5;
  background: rgba(128, 128, 128, 0.55);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius);
  pointer-events: auto;
  cursor: not-allowed;
}
.reminder-mask .mask-text {
  font-size: 15px;
  font-weight: 600;
  color: #fff;
  background: rgba(0, 0, 0, 0.55);
  padding: 10px 18px;
  border-radius: var(--radius);
  text-align: center;
  max-width: 80%;
  line-height: 1.5;
}
.reminder-masked {
  filter: grayscale(1);
  opacity: 0.6;
  pointer-events: none;
  user-select: none;
}
</style>
