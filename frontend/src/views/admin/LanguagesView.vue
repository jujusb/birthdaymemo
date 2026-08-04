<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useToast } from '@/composables/useToast'

interface LangItem {
  code: string
  name: string
}

const i18n = useI18nStore()
const t = i18n.t
const toast = useToast()

const languages = ref<LangItem[]>([])
const loading = ref(false)
const scanning = ref(false)
const scanErrors = ref<string[]>([])

const validateCode = ref('')
const validateContent = ref('')
const validating = ref(false)

async function load() {
  loading.value = true
  try {
    const data = await api.adminLanguages()
    languages.value = (data.available || []) as LangItem[]
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function scan() {
  scanning.value = true
  scanErrors.value = []
  try {
    const data = await api.scanLanguages()
    languages.value = (data.available || []) as LangItem[]
    scanErrors.value = data.errors || []
    if (scanErrors.value.length) {
      toast.error(t('common.failed'))
    } else {
      toast.success(t('common.success'))
    }
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    scanning.value = false
  }
}

async function validate() {
  if (!validateCode.value || !validateContent.value) {
    toast.error(t('common.required'))
    return
  }
  validating.value = true
  try {
    await api.validateLanguageFile(validateContent.value, validateCode.value)
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    validating.value = false
  }
}

async function exportSample() {
  try {
    const resp = await api.exportSampleLanguage()
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'en.json'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}
</script>

<template>
  <div class="lang-view">
    <div class="row between mb-16">
      <h2 class="page-title">{{ t('admin.languages') }}</h2>
      <div class="row gap-8">
        <button @click="exportSample">{{ t('admin.exportSample') }}</button>
        <button :disabled="scanning" @click="scan">
          {{ scanning ? t('common.loading') : t('admin.scanLanguages') }}
        </button>
      </div>
    </div>

    <p class="hint">{{ t('admin.languagesHint') }}</p>

    <div v-if="loading" class="loading">{{ t('common.loading') }}</div>

    <div v-else class="card mt-16">
      <div v-for="l in languages" :key="l.code" class="lang-item row between">
        <span class="lang-code">{{ l.code }}</span>
        <span class="lang-name grow">{{ l.name }}</span>
      </div>
      <div v-if="languages.length === 0" class="empty">{{ t('common.none') }}</div>
    </div>

    <div v-if="scanErrors.length" class="card mt-16 errors">
      <div v-for="(err, i) in scanErrors" :key="i" class="err-line">{{ err }}</div>
    </div>

    <div class="card mt-16 validate-card">
      <div class="col mt-8">
        <label>{{ t('settings.language') }}</label>
        <input v-model="validateCode" type="text" placeholder="en" />
      </div>
      <div class="col mt-16">
        <label>{{ t('common.preview') }}</label>
        <textarea v-model="validateContent" rows="8"></textarea>
      </div>
      <div class="row mt-16 actions">
        <button class="primary" :disabled="validating" @click="validate">
          {{ validating ? t('common.loading') : t('common.confirm') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.lang-view {
  max-width: 640px;
  margin: 0 auto;
}
.page-title {
  font-size: 20px;
  font-weight: 700;
}
.hint {
  font-size: 13px;
  color: var(--color-text-muted);
}
.loading {
  text-align: center;
  padding: 32px;
  color: var(--color-text-muted);
}
.lang-item {
  padding: 8px 4px;
  border-bottom: 1px solid var(--color-border);
}
.lang-item:last-child {
  border-bottom: none;
}
.lang-code {
  font-family: monospace;
  background: var(--color-bg-sunken);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 13px;
}
.lang-name {
  color: var(--color-text);
}
.empty {
  color: var(--color-text-muted);
  padding: 12px 4px;
}
.errors {
  border-left: 3px solid var(--color-danger);
}
.err-line {
  font-size: 13px;
  color: var(--color-danger);
  padding: 4px 0;
}
.validate-card {
  padding: 20px;
}
.section-title {
  font-size: 15px;
  font-weight: 600;
}
textarea {
  resize: vertical;
  font-family: monospace;
}
.actions {
  justify-content: flex-start;
}
</style>
