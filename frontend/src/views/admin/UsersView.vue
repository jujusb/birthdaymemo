<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api'
import type { User } from '@/api/types'
import { ApiError } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useI18nStore } from '@/stores/i18n'
import { useToast } from '@/composables/useToast'
import { showConfirm } from '@/composables/useConfirm'

const auth = useAuthStore()
const i18n = useI18nStore()
const t = i18n.t
const toast = useToast()

const users = ref<User[]>([])
const loading = ref(false)

const showCreate = ref(false)
const showReset = ref(false)
const resetTarget = ref<User | null>(null)
const createForm = ref({ username: '', password: '', role: 'user' as 'user' | 'guest' | 'admin' })
const resetForm = ref({ password: '' })
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    users.value = await api.listUsers()
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    loading.value = false
  }
}
onMounted(load)

function openCreate() {
  createForm.value = { username: '', password: '', role: 'user' }
  showCreate.value = true
}
function openReset(u: User) {
  resetTarget.value = u
  resetForm.value = { password: '' }
  showReset.value = true
}

async function doCreate() {
  if (!createForm.value.username || !createForm.value.password) {
    toast.error(t('common.required'))
    return
  }
  saving.value = true
  try {
    await api.createUser(createForm.value.username, createForm.value.password, createForm.value.role)
    toast.success(t('common.success'))
    showCreate.value = false
    load()
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    saving.value = false
  }
}

async function doReset() {
  if (!resetTarget.value || !resetForm.value.password) {
    toast.error(t('common.required'))
    return
  }
  saving.value = true
  try {
    await api.resetUserPassword(resetTarget.value.id, resetForm.value.password)
    toast.success(t('common.success'))
    showReset.value = false
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    saving.value = false
  }
}

async function remove(u: User, e: MouseEvent) {
  if (u.id === auth.user?.id) {
    toast.error(t('admin.cannotDeleteSelf'))
    return
  }
  // 任务3：使用自定义 ConfirmDialog 替代浏览器 confirm()
  // 弹窗定位到点击的删除按钮右下方
  const target = e.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()
  const ok = await showConfirm(t('admin.deleteConfirm') + ' "' + u.username + '"?', {
    x: rect.left,
    y: rect.bottom + 4,
  })
  if (!ok) return
  try {
    await api.deleteUser(u.id)
    toast.success(t('common.success'))
    load()
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

function formatTime(s: string): string {
  try {
    return new Date(s).toLocaleString()
  } catch {
    return s
  }
}
</script>

<template>
  <div class="users-view">
    <div class="row between mb-16">
      <h2 class="page-title">{{ t('admin.users') }}</h2>
      <button class="primary" @click="openCreate">+ {{ t('admin.newUser') }}</button>
    </div>

    <div v-if="loading" class="loading">{{ t('common.loading') }}</div>

    <div v-else class="card table-wrap">
      <table>
        <thead>
          <tr>
            <th>{{ t('admin.username') }}</th>
            <th>{{ t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td>
              <div class="user-cell">
                <span class="uname">{{ u.username }}</span>
                <span class="badge" :class="u.role">{{ t('role.' + u.role) }}</span>
                <span class="muted created">{{ formatTime(u.created_at) }}</span>
              </div>
            </td>
            <td>
              <div class="row gap-8">
                <button @click="openReset(u)">{{ t('admin.resetPassword') }}</button>
                <button class="danger" :disabled="u.id === auth.user?.id" @click="remove(u, $event)">
                  {{ t('common.delete') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create user modal -->
    <div v-if="showCreate" class="modal-backdrop" @click.self="showCreate = false">
      <div class="modal" role="dialog" aria-modal="true">
        <h3 class="modal-title">{{ t('admin.newUser') }}</h3>
        <div class="col mt-16">
          <label>{{ t('admin.username') }}</label>
          <input v-model="createForm.username" type="text" />
        </div>
        <div class="col mt-16">
          <label>{{ t('admin.initialPassword') }}</label>
          <input v-model="createForm.password" type="password" autocomplete="new-password" />
        </div>
        <div class="col mt-16">
          <label>{{ t('role.title') }}</label>
          <select v-model="createForm.role">
            <option value="user">{{ t('role.user') }}</option>
            <option value="guest">{{ t('role.guest') }}</option>
            <option value="admin">{{ t('role.admin') }}</option>
          </select>
          <span class="hint">{{ t('role.guestHint') }}</span>
        </div>
        <div class="row between mt-24 actions">
          <button @click="showCreate = false">{{ t('common.cancel') }}</button>
          <button class="primary" :disabled="saving" @click="doCreate">{{ t('common.save') }}</button>
        </div>
      </div>
    </div>

    <!-- Reset password modal -->
    <div v-if="showReset" class="modal-backdrop" @click.self="showReset = false">
      <div class="modal" role="dialog" aria-modal="true">
        <h3 class="modal-title">{{ t('admin.resetPassword') }} - {{ resetTarget?.username }}</h3>
        <div class="col mt-16">
          <label>{{ t('login.newPassword') }}</label>
          <input v-model="resetForm.password" type="password" autocomplete="new-password" />
        </div>
        <div class="row between mt-24 actions">
          <button @click="showReset = false">{{ t('common.cancel') }}</button>
          <button class="primary" :disabled="saving" @click="doReset">{{ t('common.save') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.users-view {
  display: flex;
  flex-direction: column;
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
.table-wrap {
  padding: 0;
  overflow-x: auto;
}
.user-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.uname {
  font-weight: 600;
}
.badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 10px;
  font-size: 12px;
  align-self: flex-start;
}
.badge.admin {
  background: var(--color-primary);
  color: #fff;
}
.badge.user {
  background: var(--color-bg-sunken);
  color: var(--color-text-muted);
}
.created {
  font-size: 12px;
}
.muted {
  color: var(--color-text-muted);
}
.modal-title {
  font-size: 16px;
  font-weight: 600;
}
.actions {
  justify-content: flex-end;
}
</style>
