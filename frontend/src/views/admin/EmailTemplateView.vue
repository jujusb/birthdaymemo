<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useToast } from '@/composables/useToast'

const i18n = useI18nStore()
const t = i18n.t
const toast = useToast()

const subject = ref('')
const body = ref('')
const selfAdvanceSubject = ref('')
const selfAdvanceBody = ref('')
const selfTodaySubject = ref('')
const selfTodayBody = ref('')
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const tpl = await api.getEmailTemplate()
    subject.value = tpl.subject
    body.value = tpl.body
    selfAdvanceSubject.value = tpl.self_advance_subject
    selfAdvanceBody.value = tpl.self_advance_body
    selfTodaySubject.value = tpl.self_today_subject
    selfTodayBody.value = tpl.self_today_body
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
    await api.updateEmailTemplate(
      subject.value,
      body.value,
      selfAdvanceSubject.value,
      selfAdvanceBody.value,
      selfTodaySubject.value,
      selfTodayBody.value,
    )
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="tpl-view">
    <h2 class="page-title">{{ t('admin.emailTemplate') }}</h2>

    <div v-if="loading" class="loading">{{ t('common.loading') }}</div>

    <div v-else class="card mt-16 form-card">
      <div class="col mt-8">
        <label>{{ t('admin.emailSubject') }}</label>
        <input v-model="subject" type="text" />
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.emailBody') }}</label>
        <textarea v-model="body" rows="12"></textarea>
      </div>

      <div class="vars-hint mt-16">
        <div class="vars-title">{{ t('admin.templateVars') }}</div>
        <pre class="vars-example">亲爱的 {user}：

共有 {count} 位朋友即将过生日：
[Start loop]
您的{birthday_person_N}好友即将迎来{birthday_sex_N}的{birthday_age_N}岁生日 [ {birthday_date_N} ]
Tag：{tags_N}
[End loop]
请记得送上祝福！</pre>
      </div>

      <!-- 任务5：本人生日提醒模板 -->
      <div class="section-divider"></div>
      <h3 class="sub-section-title">{{ t('admin.selfBirthdayTemplate') }}</h3>
      <span class="hint">{{ t('admin.selfBirthdayHint') }}</span>

      <div class="col mt-16">
        <label>{{ t('admin.selfAdvanceSubject') }}</label>
        <input v-model="selfAdvanceSubject" type="text" />
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.selfAdvanceBody') }}</label>
        <textarea v-model="selfAdvanceBody" rows="12"></textarea>
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.selfTodaySubject') }}</label>
        <input v-model="selfTodaySubject" type="text" />
      </div>

      <div class="col mt-16">
        <label>{{ t('admin.selfTodayBody') }}</label>
        <textarea v-model="selfTodayBody" rows="12"></textarea>
      </div>

      <div class="vars-hint mt-16">
        <div class="vars-title">{{ t('admin.selfTemplateVars') }}</div>
        <pre class="vars-example">{user} 用户名
{self_birthday_date} 本人生日日期
{self_birthday_age} 即将到达的岁数
{self_birthday_last} 距离生日的剩余天数（仅提前提醒模板）</pre>
      </div>

      <div class="row mt-24 actions">
        <button class="primary" :disabled="saving" @click="save">
          {{ saving ? t('common.loading') : t('common.save') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tpl-view {
  max-width: 720px;
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
textarea {
  resize: vertical;
  font-family: inherit;
}
.vars-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  background: var(--color-bg-sunken);
  padding: 10px 12px;
  border-radius: var(--radius-sm);
}
.vars-title {
  margin-bottom: 6px;
  line-height: 1.5;
}
.vars-example {
  margin: 0;
  padding: 8px 10px;
  background: var(--color-bg-elevated, rgba(0,0,0,0.05));
  border-radius: var(--radius-sm);
  font-family: ui-monospace, "SF Mono", Menlo, Consolas, monospace;
  font-size: 11.5px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--color-text);
}
.actions {
  justify-content: flex-start;
}
.section-divider {
  height: 1px;
  background: var(--color-border);
  margin: 24px 0 16px;
}
.sub-section-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}
.hint {
  font-size: 12px;
  color: var(--color-text-muted);
}
</style>
