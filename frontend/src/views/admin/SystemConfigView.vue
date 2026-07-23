<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api'
import type { SystemConfig } from '@/api/types'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useToast } from '@/composables/useToast'

const i18n = useI18nStore()
const t = i18n.t
const toast = useToast()

const config = ref<SystemConfig | null>(null)
const loading = ref(false)
const saving = ref(false)
const needRestart = ref(false)

const addresses = ['0.0.0.0', '127.0.0.1', '::', '::1']

async function load() {
  loading.value = true
  try {
    config.value = await api.getSystemConfig()
    needRestart.value = false
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function save() {
  if (!config.value) return
  saving.value = true
  try {
    const res = await api.updateSystemConfig(config.value)
    needRestart.value = res.need_restart
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="sys-view">
    <h2 class="page-title">{{ t('admin.systemConfig') }}</h2>

    <div v-if="loading" class="loading">{{ t('common.loading') }}</div>

    <div v-else-if="config" class="card mt-16 form-card">
      <div class="col mt-8">
        <label>{{ t('settings.language') }}</label>
        <select v-model="config.language">
          <option v-for="l in i18n.available" :key="l.code" :value="l.code">{{ l.name }}</option>
        </select>
        <span class="hint">{{ t('settings.languageHint') }}</span>
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.listenAddress') }}</label>
        <select v-model="config.listen_address">
          <option v-for="a in addresses" :key="a" :value="a">{{ a }}</option>
        </select>
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.listenPort') }}</label>
        <input v-model.number="config.listen_port" type="number" min="1" max="65535" />
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.logRetention') }}</label>
        <input v-model.number="config.operation_log_retention_days" type="number" min="1" />
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.externalUrl') }}</label>
        <input v-model.trim="config.external_url" type="text" :placeholder="t('admin.externalUrlPlaceholder')" />
        <span class="hint">{{ t('admin.externalUrlHint') }}</span>
      </div>

      <div v-if="needRestart" class="warning mt-16">{{ t('admin.needRestart') }}</div>

      <div class="row mt-24 actions">
        <button class="primary" :disabled="saving" @click="save">
          {{ saving ? t('common.loading') : t('common.save') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sys-view {
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
.warning {
  background: rgba(243, 156, 18, 0.12);
  color: var(--color-warning);
  border: 1px solid var(--color-warning);
  border-radius: var(--radius-sm);
  padding: 8px 12px;
  font-size: 13px;
}
.actions {
  justify-content: flex-start;
}
</style>
