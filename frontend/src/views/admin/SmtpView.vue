<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api'
import type { SmtpSetting } from '@/api/types'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useToast } from '@/composables/useToast'

const i18n = useI18nStore()
const t = i18n.t
const toast = useToast()

const form = ref<SmtpSetting>({
  id: 0,
  host: '',
  port: 587,
  username: '',
  from: '',
  encryption: 'none',
  has_password: false,
})
const password = ref('')
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)

const encryptions: SmtpSetting['encryption'][] = ['none', 'tls', 'starttls']

async function load() {
  loading.value = true
  try {
    const s = await api.getSmtp()
    form.value = { ...s }
    password.value = ''
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  saving.value = true
  try {
    const payload: Partial<SmtpSetting> & { password?: string } = {
      host: form.value.host,
      port: form.value.port,
      username: form.value.username,
      from: form.value.from,
      encryption: form.value.encryption,
    }
    if (password.value) payload.password = password.value
    const saved = await api.updateSmtp(payload)
    form.value = { ...saved }
    password.value = ''
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    saving.value = false
  }
}

async function test() {
  testing.value = true
  try {
    await api.testSmtp()
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    testing.value = false
  }
}
</script>

<template>
  <div class="smtp-view">
    <h2 class="page-title">{{ t('admin.smtp') }}</h2>

    <div v-if="loading" class="loading">{{ t('common.loading') }}</div>

    <div v-else class="card mt-16 form-card">
      <div class="col mt-8">
        <label>{{ t('admin.smtpHost') }}</label>
        <input v-model="form.host" type="text" placeholder="smtp.example.com" />
      </div>

      <div class="row gap-16 mt-16">
        <div class="col grow">
          <label>{{ t('admin.smtpPort') }}</label>
          <input v-model.number="form.port" type="number" min="1" max="65535" />
        </div>
        <div class="col grow">
          <label>{{ t('admin.smtpEncryption') }}</label>
          <select v-model="form.encryption">
            <option v-for="enc in encryptions" :key="enc" :value="enc">{{ enc }}</option>
          </select>
        </div>
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.smtpUsername') }}</label>
        <input v-model="form.username" type="text" autocomplete="username" />
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.smtpPassword') }}</label>
        <input v-model="password" type="password" autocomplete="new-password" :placeholder="form.has_password ? '••••••' : ''" />
        <span v-if="form.has_password" class="hint">{{ t('common.yes') }}</span>
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.smtpFrom') }}</label>
        <input v-model="form.from" type="email" placeholder="noreply@example.com" />
      </div>

      <div class="row gap-8 mt-24 actions">
        <button class="primary" :disabled="saving" @click="save">
          {{ saving ? t('common.loading') : t('common.save') }}
        </button>
        <button :disabled="testing" @click="test">
          {{ testing ? t('common.loading') : t('admin.smtpTest') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.smtp-view {
  max-width: 560px;
  margin: 0 auto;
}
.page-title {
  font-size: 20px;
  font-weight: 700;
}
.loading {
  text-align: center;
  padding: 32px;
  color: var(--color-text-muted);
}
.form-card {
  padding: 20px;
}
.hint {
  font-size: 12px;
  color: var(--color-text-muted);
}
.actions {
  justify-content: flex-start;
}
</style>
