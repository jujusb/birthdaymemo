<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import * as api from '@/api'
import type { OperationLog } from '@/api/types'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useToast } from '@/composables/useToast'

const i18n = useI18nStore()
const t = i18n.t
const toast = useToast()

const activeTab = ref<'audit' | 'files'>('audit')

// ----- 审计日志 -----
const logs = ref<OperationLog[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(50)
const loading = ref(false)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size.value)))

// 筛选状态
const filterUsers = ref<string[]>([])
const filterActions = ref<string[]>([])
const filterDetail = ref('')
const filterIp = ref('')
// 用户名候选列表（来自当前已加载日志和后端）
const userCandidates = ref<string[]>([])
// 操作类型候选列表（与后端 actionLabelMap 对应）
const actionOptions = computed<{ value: string; label: string }[]>(() => [
  { value: 'login', label: t('audit.actionLogin') },
  { value: 'logout', label: t('audit.actionLogout') },
  { value: 'change_password', label: t('audit.actionChangePassword') },
  { value: 'create_user', label: t('audit.actionCreateUser') },
  { value: 'delete_user', label: t('audit.actionDeleteUser') },
  { value: 'reset_password', label: t('audit.actionResetPassword') },
  { value: 'update_system_config', label: t('audit.actionUpdateSystemConfig') },
  { value: 'update_smtp', label: t('audit.actionUpdateSmtp') },
  { value: 'update_email_template', label: t('audit.actionUpdateEmailTemplate') },
  { value: 'scan_languages', label: t('audit.actionScanLanguages') },
  { value: 'request_email_confirm', label: t('audit.actionRequestEmailConfirm') },
  { value: 'confirm_email', label: t('audit.actionConfirmEmail') },
  { value: 'create_birthday', label: t('audit.actionCreateBirthday') },
  { value: 'delete_birthday', label: t('audit.actionDeleteBirthday') },
  { value: 'send_reminder_email', label: t('audit.actionSendReminderEmail') },
])

// 用户名输入框（添加新用户名筛选）
const userInput = ref('')
function addUserFilter() {
  const v = userInput.value.trim()
  if (v && !filterUsers.value.includes(v)) {
    filterUsers.value.push(v)
    userCandidates.value = Array.from(new Set([...userCandidates.value, v]))
  }
  userInput.value = ''
}
function removeUserFilter(u: string) {
  filterUsers.value = filterUsers.value.filter((x) => x !== u)
}
function toggleActionFilter(a: string) {
  const idx = filterActions.value.indexOf(a)
  if (idx >= 0) filterActions.value.splice(idx, 1)
  else filterActions.value.push(a)
}

function buildFilter(): api.OperationLogFilter {
  return {
    users: filterUsers.value.length > 0 ? filterUsers.value : undefined,
    actions: filterActions.value.length > 0 ? filterActions.value : undefined,
    detail: filterDetail.value.trim() || undefined,
    ip: filterIp.value.trim() || undefined,
  }
}

async function loadLogs() {
  loading.value = true
  try {
    const data = await api.listOperationLogs(page.value, size.value, buildFilter())
    logs.value = data.logs || []
    total.value = data.total
    page.value = data.page
    size.value = data.size
    // 从结果中收集用户名候选
    const names = new Set<string>(userCandidates.value)
    for (const l of logs.value) {
      if (l.username) names.add(l.username)
    }
    userCandidates.value = Array.from(names)
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    loading.value = false
  }
}

function applyFilter() {
  page.value = 1
  loadLogs()
}
function resetFilter() {
  filterUsers.value = []
  filterActions.value = []
  filterDetail.value = ''
  filterIp.value = ''
  page.value = 1
  loadLogs()
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    loadLogs()
  }
}
function nextPage() {
  if (page.value < totalPages.value) {
    page.value++
    loadLogs()
  }
}
function goToPage(p: number) {
  if (p < 1 || p > totalPages.value || p === page.value) return
  page.value = p
  loadLogs()
}
function goToFirst() { goToPage(1) }
function goToLast() { goToPage(totalPages.value) }

// 分页器：返回应显示的页码数组，包含 "..." 占位
const pageWindow = computed<(number | string)[]>(() => {
  const cur = page.value
  const last = totalPages.value
  if (last <= 7) {
    return Array.from({ length: last }, (_, i) => i + 1)
  }
  const out: (number | string)[] = []
  out.push(1)
  const left = Math.max(2, cur - 2)
  const right = Math.min(last - 1, cur + 2)
  if (left > 2) out.push('...')
  for (let i = left; i <= right; i++) out.push(i)
  if (right < last - 1) out.push('...')
  out.push(last)
  return out
})

// 跳转页数输入
const jumpPageInput = ref<string>('')
function submitJump() {
  const p = parseInt(jumpPageInput.value, 10)
  if (!isNaN(p)) {
    goToPage(p)
    jumpPageInput.value = ''
  }
}

function formatTime(s: string): string {
  try {
    return new Date(s).toLocaleString()
  } catch {
    return s
  }
}

// action 翻译映射：将后端 action 字符串映射到 i18n key
const actionLabelMap: Record<string, string> = {
  login: 'audit.actionLogin',
  logout: 'audit.actionLogout',
  change_password: 'audit.actionChangePassword',
  create_user: 'audit.actionCreateUser',
  delete_user: 'audit.actionDeleteUser',
  reset_password: 'audit.actionResetPassword',
  update_system_config: 'audit.actionUpdateSystemConfig',
  update_smtp: 'audit.actionUpdateSmtp',
  update_email_template: 'audit.actionUpdateEmailTemplate',
  scan_languages: 'audit.actionScanLanguages',
  request_email_confirm: 'audit.actionRequestEmailConfirm',
  confirm_email: 'audit.actionConfirmEmail',
  create_birthday: 'audit.actionCreateBirthday',
  delete_birthday: 'audit.actionDeleteBirthday',
  send_reminder_email: 'audit.actionSendReminderEmail',
}

function actionLabel(action: string): string {
  const key = actionLabelMap[action]
  if (key && i18n.strings[key]) return i18n.strings[key]
  return action
}

// ----- 日志文件 -----
const logFiles = ref<api.LogFileInfo[]>([])
const currentLogFile = ref<string>('')
const logContent = ref<string>('')
const logContentSize = ref(0)
const autoRefresh = ref(true)
const filesLoading = ref(false)
let refreshTimer: number | undefined

async function loadFileList() {
  try {
    const list = await api.listLogFiles()
    logFiles.value = list || []
    // 默认选中 latest.log
    if (!currentLogFile.value && logFiles.value.length > 0) {
      const latest = logFiles.value.find((f) => f.name === 'latest.log')
      currentLogFile.value = latest ? latest.name : logFiles.value[0].name
      await loadFileContent()
    }
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

async function loadFileContent() {
  if (!currentLogFile.value) return
  filesLoading.value = true
  try {
    // 只拉取最后 500 行避免过大
    const data = await api.getLogFile(currentLogFile.value, 500)
    logContent.value = data.content || ''
    logContentSize.value = data.size || 0
    await nextTick()
    scrollToBottom()
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    filesLoading.value = false
  }
}

const logContentRef = ref<HTMLElement | null>(null)
function scrollToBottom() {
  if (logContentRef.value) {
    logContentRef.value.scrollTop = logContentRef.value.scrollHeight
  }
}

function selectFile(name: string) {
  currentLogFile.value = name
  loadFileContent()
}

function startAutoRefresh() {
  stopAutoRefresh()
  if (autoRefresh.value && currentLogFile.value === 'latest.log') {
    refreshTimer = window.setInterval(loadFileContent, 3000)
  }
}
function stopAutoRefresh() {
  if (refreshTimer !== undefined) {
    window.clearInterval(refreshTimer)
    refreshTimer = undefined
  }
}

watch(autoRefresh, (v) => {
  if (v) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
})

watch(currentLogFile, () => {
  stopAutoRefresh()
  if (autoRefresh.value) startAutoRefresh()
})

// 切换到文件 tab 时加载
watch(activeTab, (v) => {
  if (v === 'files' && logFiles.value.length === 0) {
    loadFileList()
  }
})

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1024 / 1024).toFixed(2) + ' MB'
}

// 解析单行日志，给 INFO/ERROR 着色
function formatLogLine(line: string): { text: string; level: 'info' | 'error' | 'plain' } {
  const m = line.match(/^\[([\d-: ]+)\] \[(INFO|ERROR)\] (.*)$/)
  if (m) {
    return {
      text: `[${m[1]}] [${m[2]}] ${m[3]}`,
      level: m[2] === 'ERROR' ? 'error' : 'info',
    }
  }
  return { text: line, level: 'plain' }
}

const logLines = computed(() => {
  if (!logContent.value) return [] as { text: string; level: 'info' | 'error' | 'plain' }[]
  return logContent.value.split('\n').map(formatLogLine)
})

onMounted(() => {
  loadLogs()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<template>
  <div class="logs-view">
    <h2 class="page-title">{{ t('admin.operationLogs') }}</h2>

    <!-- Tabs -->
    <div class="tabs">
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'audit' }"
        @click="activeTab = 'audit'"
      >
        {{ t('audit.tabAudit') }}
      </button>
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'files' }"
        @click="activeTab = 'files'"
      >
        {{ t('audit.tabFiles') }}
      </button>
    </div>

    <!-- 审计日志 -->
    <div v-if="activeTab === 'audit'">
      <!-- 筛选区 -->
      <div class="card filter-card">
        <div class="filter-title">{{ t('audit.filterTitle') }}</div>
        <div class="filter-grid">
          <!-- 用户名多选 -->
          <div class="filter-item">
            <label>{{ t('audit.filterUser') }}</label>
            <div class="multi-input">
              <div class="chips">
                <span v-for="u in filterUsers" :key="u" class="chip">
                  {{ u }}
                  <button class="chip-x" @click="removeUserFilter(u)">×</button>
                </span>
              </div>
              <input
                v-model="userInput"
                :placeholder="t('audit.filterPlaceholder')"
                list="user-candidates"
                @keydown.enter.prevent="addUserFilter"
              />
              <datalist id="user-candidates">
                <option v-for="u in userCandidates" :key="u" :value="u" />
              </datalist>
            </div>
          </div>
          <!-- 操作多选 -->
          <div class="filter-item">
            <label>{{ t('audit.filterAction') }}</label>
            <div class="action-list">
              <label v-for="opt in actionOptions" :key="opt.value" class="action-opt">
                <input
                  type="checkbox"
                  :checked="filterActions.includes(opt.value)"
                  @change="toggleActionFilter(opt.value)"
                />
                <span>{{ opt.label }}</span>
              </label>
            </div>
          </div>
          <!-- 详情关键字 -->
          <div class="filter-item">
            <label>{{ t('audit.filterDetail') }}</label>
            <input v-model="filterDetail" :placeholder="t('audit.filterPlaceholder')" />
          </div>
          <!-- IP 关键字 -->
          <div class="filter-item">
            <label>{{ t('audit.filterIp') }}</label>
            <input v-model="filterIp" :placeholder="t('audit.filterPlaceholder')" />
          </div>
        </div>
        <div class="filter-actions">
          <button class="primary" @click="applyFilter">{{ t('audit.filterApply') }}</button>
          <button @click="resetFilter">{{ t('audit.filterReset') }}</button>
        </div>
      </div>

      <div v-if="loading" class="loading">{{ t('common.loading') }}</div>
      <div v-else class="card audit-card">
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th class="time-col">{{ t('audit.colTime') }}</th>
                <th class="user-col">{{ t('audit.colUser') }}</th>
                <th class="action-col">{{ t('audit.colAction') }}</th>
                <th class="detail-col">{{ t('audit.colDetail') }}</th>
                <th class="ip-col">{{ t('audit.colIp') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="log in logs" :key="log.id" class="audit-row">
                <td class="mono time-cell" :data-label="t('audit.colTime')">{{ formatTime(log.created_at) }}</td>
                <td class="user-cell" :data-label="t('audit.colUser')">{{ log.username || '—' }}</td>
                <td class="action-cell" :data-label="t('audit.colAction')">
                  <span class="action-tag">{{ actionLabel(log.action) }}</span>
                </td>
                <td class="detail-cell" :data-label="t('audit.colDetail')">{{ log.detail || '—' }}</td>
                <td class="mono ip-cell" :data-label="t('audit.colIp')">{{ log.ip || '—' }}</td>
              </tr>
              <tr v-if="logs.length === 0">
                <td colspan="5" class="empty">{{ t('common.none') }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 分页器：首页 ... 当前页前后 ... 末页 -->
        <div class="row pager">
          <button class="pager-btn" :disabled="page <= 1" @click="prevPage">‹ {{ t('audit.pagePrev') }}</button>
          <div class="page-numbers">
            <template v-for="(p, idx) in pageWindow" :key="idx">
              <span v-if="p === '...'" class="ellipsis">…</span>
              <button
                v-else-if="p !== page"
                class="page-num"
                @click="goToPage(p as number)"
              >{{ p }}</button>
              <span v-else class="page-num current">
                <input
                  v-model="jumpPageInput"
                  class="jump-input"
                  :placeholder="String(page)"
                  @keydown.enter="submitJump"
                  @blur="jumpPageInput = ''"
                />
              </span>
            </template>
          </div>
          <button class="pager-btn" :disabled="page >= totalPages" @click="nextPage">{{ t('audit.pageNext') }} ›</button>
        </div>
        <div class="row between page-meta">
          <button class="pager-mini" :disabled="page <= 1" @click="goToFirst">« {{ t('audit.pageFirst') }}</button>
          <span class="page-info">{{ page }} / {{ totalPages }} ({{ total }})</span>
          <button class="pager-mini" :disabled="page >= totalPages" @click="goToLast">{{ t('audit.pageLast') }} »</button>
        </div>
      </div>
    </div>

    <!-- 日志文件 -->
    <div v-else class="files-layout">
      <!-- 左侧文件列表 -->
      <aside class="card file-list-panel">
        <div class="file-list-head">
          <span>{{ t('audit.fileList') }}</span>
          <button class="refresh-btn" :title="t('common.reset')" @click="loadFileList">⟳</button>
        </div>
        <div class="file-list">
          <button
            v-for="f in logFiles"
            :key="f.name"
            class="file-item"
            :class="{ active: currentLogFile === f.name }"
            @click="selectFile(f.name)"
          >
            <div class="file-name">
              <span class="file-icon">📄</span>
              {{ f.name }}
            </div>
            <div class="file-meta">
              <span>{{ formatSize(f.size) }}</span>
              <span class="file-time">{{ f.mod_time }}</span>
            </div>
          </button>
          <div v-if="logFiles.length === 0" class="empty-files">{{ t('common.none') }}</div>
        </div>
      </aside>

      <!-- 右侧内容查看 -->
      <section class="card content-panel">
        <div class="content-head">
          <div class="head-left">
            <span class="file-title">{{ currentLogFile || '—' }}</span>
            <span class="file-size">{{ formatSize(logContentSize) }}</span>
          </div>
          <div class="head-right">
            <label class="auto-refresh" v-if="currentLogFile === 'latest.log'">
              <input v-model="autoRefresh" type="checkbox" />
              {{ t('audit.autoRefresh') }}
            </label>
            <button class="refresh-btn" :title="t('common.reset')" @click="loadFileContent">⟳</button>
          </div>
        </div>
        <div ref="logContentRef" class="log-content">
          <div v-if="filesLoading" class="loading">{{ t('common.loading') }}</div>
          <pre v-else-if="logLines.length > 0"><code
            v-for="(line, i) in logLines"
            :key="i"
            class="log-line"
            :class="'level-' + line.level"
          >{{ line.text }}
</code></pre>
          <div v-else class="empty-files">{{ t('common.none') }}</div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.logs-view {
  display: flex;
  flex-direction: column;
}
.page-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 16px;
}

/* Tabs */
.tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  border-bottom: 1px solid var(--color-border);
}
.tab-btn {
  background: transparent;
  border: 1px solid transparent;
  border-bottom: none;
  padding: 8px 16px;
  border-radius: 6px 6px 0 0;
  cursor: pointer;
  font-size: 14px;
  color: var(--color-text-muted);
  transition: all 0.15s;
}
.tab-btn:hover {
  color: var(--color-text);
  background: var(--color-bg-sunken);
}
.tab-btn.active {
  color: var(--color-primary);
  background: var(--color-bg-elevated);
  border-color: var(--color-border);
  border-bottom-color: var(--color-bg-elevated);
  margin-bottom: -1px;
  font-weight: 600;
}

.loading {
  text-align: center;
  padding: 32px;
  color: var(--color-text-muted);
}

/* 审计日志 */
.audit-card {
  padding: 0;
  overflow: hidden;
}
.table-wrap {
  overflow-x: auto;
}
.audit-card table {
  width: 100%;
  border-collapse: collapse;
}
.audit-card th,
.audit-card td {
  padding: 10px 14px;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
  font-size: 13px;
}
.audit-card th {
  background: var(--color-bg-sunken);
  font-weight: 600;
  color: var(--color-text-muted);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  position: sticky;
  top: 0;
}
.audit-row:hover {
  background: var(--color-bg-sunken);
}
.mono {
  font-family: 'SF Mono', Monaco, Consolas, 'Liberation Mono', monospace;
}
.time-cell {
  white-space: nowrap;
  color: var(--color-text-muted);
  font-size: 12px;
}
.user-cell {
  font-weight: 500;
}
.action-cell {
  white-space: nowrap;
}
.action-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  background: var(--color-primary);
  color: #fff;
  font-weight: 500;
}
.detail-cell {
  max-width: 360px;
  word-break: break-word;
  color: var(--color-text);
}
.ip-cell {
  white-space: nowrap;
  color: var(--color-text-muted);
  font-size: 12px;
}
.empty {
  text-align: center;
  color: var(--color-text-muted);
  padding: 32px;
}
.time-col {
  width: 160px;
}
.user-col {
  width: 110px;
}
.action-col {
  width: 160px;
}
.ip-col {
  width: 130px;
}

.pager {
  padding: 12px 16px;
  align-items: center;
  border-top: 1px solid var(--color-border);
  justify-content: center;
  gap: 8px;
}
.pager-btn {
  padding: 4px 12px;
}
.page-info {
  color: var(--color-text-muted);
  font-size: 13px;
}
.page-meta {
  padding: 6px 16px 12px;
  border-top: 1px dashed var(--color-border);
  font-size: 12px;
}
.pager-mini {
  padding: 2px 10px;
  font-size: 12px;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--color-text-muted);
}
.pager-mini:hover:not(:disabled) {
  border-color: var(--color-primary);
  color: var(--color-primary);
}
.pager-mini:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.page-numbers {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.page-num {
  min-width: 28px;
  height: 28px;
  padding: 0 6px;
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.page-num:hover:not(.current) {
  border-color: var(--color-primary);
  color: var(--color-primary);
}
.page-num.current {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: #fff;
  font-weight: 600;
  position: relative;
}
.jump-input {
  width: 36px;
  height: 22px;
  text-align: center;
  border: none;
  background: transparent;
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  outline: none;
  padding: 0;
}
.jump-input::placeholder {
  color: rgba(255, 255, 255, 0.7);
}
.ellipsis {
  color: var(--color-text-muted);
  padding: 0 4px;
  user-select: none;
}

/* 筛选卡片 */
.filter-card {
  padding: 14px 16px;
  margin-bottom: 12px;
}
.filter-title {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 10px;
  color: var(--color-text);
}
.filter-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 16px;
  margin-bottom: 12px;
}
.filter-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.filter-item > label {
  font-size: 12px;
  color: var(--color-text-muted);
  font-weight: 600;
}
.filter-item input[type="text"],
.filter-item input:not([type]) {
  padding: 6px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-sunken);
  color: var(--color-text);
  font-size: 13px;
}
.multi-input {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.multi-input .chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  background: var(--color-primary);
  color: #fff;
  border-radius: 10px;
  font-size: 12px;
}
.chip-x {
  background: transparent;
  border: none;
  color: #fff;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0;
}
.action-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  max-height: 120px;
  overflow-y: auto;
}
.action-opt {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  cursor: pointer;
  color: var(--color-text);
}
.action-opt input {
  width: auto;
  margin: 0;
}
.filter-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
.filter-actions button {
  padding: 6px 14px;
  font-size: 13px;
  border: 1px solid var(--color-border);
  background: transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.filter-actions button.primary {
  background: var(--color-primary);
  color: #fff;
  border-color: var(--color-primary);
}
.filter-actions button:hover {
  opacity: 0.9;
}

@media (max-width: 768px) {
  .filter-grid {
    grid-template-columns: 1fr;
  }
  .page-numbers {
    flex-wrap: wrap;
    justify-content: center;
  }
}

/* 日志文件 */
.files-layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 16px;
  align-items: start;
}
.file-list-panel {
  padding: 0;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 200px);
}
.file-list-head {
  padding: 12px 14px;
  border-bottom: 1px solid var(--color-border);
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  font-size: 13px;
}
.refresh-btn {
  padding: 2px 8px;
  font-size: 14px;
  background: transparent;
  border: 1px solid var(--color-border);
  cursor: pointer;
  border-radius: var(--radius-sm);
}
.refresh-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}
.file-list {
  flex: 1;
  overflow-y: auto;
  padding: 6px;
}
.file-item {
  display: block;
  width: 100%;
  text-align: left;
  padding: 8px 10px;
  margin-bottom: 4px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s;
}
.file-item:hover {
  background: var(--color-bg-sunken);
}
.file-item.active {
  background: var(--color-bg-sunken);
  border-color: var(--color-primary);
}
.file-name {
  font-size: 13px;
  font-weight: 500;
  word-break: break-all;
  color: var(--color-text);
}
.file-icon {
  margin-right: 4px;
}
.file-meta {
  margin-top: 4px;
  font-size: 11px;
  color: var(--color-text-muted);
  display: flex;
  justify-content: space-between;
}
.file-time {
  font-family: 'SF Mono', Monaco, Consolas, monospace;
}
.empty-files {
  padding: 24px;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 13px;
}

.content-panel {
  padding: 0;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 200px);
}
.content-head {
  padding: 10px 14px;
  border-bottom: 1px solid var(--color-border);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--color-bg-sunken);
}
.head-left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.file-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--color-text);
  font-family: 'SF Mono', Monaco, Consolas, monospace;
}
.file-size {
  font-size: 12px;
  color: var(--color-text-muted);
  padding: 2px 6px;
  background: var(--color-bg-elevated);
  border-radius: 4px;
}
.head-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.auto-refresh {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 0;
  font-size: 13px;
  color: var(--color-text-muted);
  cursor: pointer;
}
.auto-refresh input {
  width: auto;
  margin: 0;
}

.log-content {
  flex: 1;
  overflow-y: auto;
  padding: 10px 14px;
  background: var(--color-bg-sunken);
}
.log-content pre {
  margin: 0;
  font-family: 'SF Mono', Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}
.log-line {
  display: inline;
}
.log-line.level-info {
  color: var(--color-text);
}
.log-line.level-error {
  color: var(--color-danger);
  font-weight: 600;
}
.log-line.level-plain {
  color: var(--color-text-muted);
}

@media (max-width: 900px) {
  .files-layout {
    grid-template-columns: 1fr;
  }
  .file-list-panel {
    max-height: 220px;
  }
  .content-panel {
    max-height: calc(100vh - 420px);
  }
}

/* 任务4：移动端操作日志卡片布局
   详情列在窄屏下被挤压成竖条，改为卡片堆叠式布局 */
@media (max-width: 768px) {
  .audit-card .table-wrap {
    overflow-x: visible;
  }
  .audit-card table {
    display: block;
  }
  .audit-card thead {
    display: none;
  }
  .audit-card tbody {
    display: block;
  }
  .audit-card tr {
    display: block;
    padding: 10px 12px;
    border-bottom: 1px solid var(--color-border);
  }
  .audit-card tr.audit-row {
    background: var(--color-bg-elevated);
    margin-bottom: 8px;
    border-radius: 6px;
    border: 1px solid var(--color-border);
  }
  .audit-card td {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 4px 0;
    border: none;
    font-size: 13px;
    text-align: left;
    width: 100%;
    max-width: none;
    word-break: break-word;
    white-space: normal;
  }
  .audit-card td::before {
    content: attr(data-label);
    flex-shrink: 0;
    min-width: 84px;
    font-size: 11px;
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    padding-top: 1px;
  }
  /* 详情列在卡片布局中占满剩余空间 */
  .audit-card .detail-cell {
    max-width: none;
    flex: 1;
  }
  .audit-card .action-cell {
    white-space: normal;
  }
  .audit-card .time-cell,
  .audit-card .ip-cell {
    white-space: normal;
  }
  /* 卡片内 hover 背景不要覆盖整个行 */
  .audit-row:hover {
    background: var(--color-bg-elevated);
  }
  .pager {
    flex-wrap: wrap;
    gap: 8px;
    justify-content: center;
  }
}
</style>
