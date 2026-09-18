<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as api from '@/api'
import type { ShareLink, TagGrant, Tag } from '@/api/types'
import { ApiError } from '@/api/client'
import { useI18nStore } from '@/stores/i18n'
import { useToast } from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'

const i18n = useI18nStore()
const t = i18n.t
const toast = useToast()
const auth = useAuthStore()

const links = ref<ShareLink[]>([])
const grantsOwned = ref<TagGrant[]>([])
const grantsReceived = ref<TagGrant[]>([])
const tags = ref<Tag[]>([])

// new share link form
const linkName = ref('')
const linkScope = ref<'all' | 'tags'>('all')
const linkTagIds = ref<number[]>([])
const linkExpiry = ref('') // date input YYYY-MM-DD
const linkSlug = ref('') // 可选自定义短址，空则使用随机 token
const creatingLink = ref(false)

// slug inline edit state (per link row)
const editingSlugId = ref<number | null>(null)
const editingSlugValue = ref('')

// link name + tags inline edit state (per link row)
const editingLinkId = ref<number | null>(null)
const editingName = ref('')
const editingScope = ref<'all' | 'tags'>('all')
const editingTagIds = ref<number[]>([])
const savingLinkEdit = ref(false)

// new grant form
const grantUser = ref('')
const grantTagId = ref<number | null>(null)
const grantPerm = ref<'view' | 'edit'>('edit')
const creatingGrant = ref(false)

async function loadAll() {
  try {
    const [l, go, gr, tg] = await Promise.all([
      api.listShareLinks(),
      api.listGrants('owned'),
      api.listGrants('received'),
      api.listTags(),
    ])
    links.value = l
    grantsOwned.value = go
    grantsReceived.value = gr
    tags.value = tg
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

onMounted(loadAll)

function toggleLinkTag(id: number) {
  const i = linkTagIds.value.indexOf(id)
  if (i >= 0) linkTagIds.value.splice(i, 1)
  else linkTagIds.value.push(id)
}

async function createLink() {
  if (linkScope.value === 'tags' && linkTagIds.value.length === 0) {
    toast.error(t('common.required'))
    return
  }
  creatingLink.value = true
  try {
    const expires_at = linkExpiry.value ? new Date(linkExpiry.value + 'T23:59:59').toISOString() : null
    const link = await api.createShareLink({
      name: linkName.value.trim(),
      scope_mode: linkScope.value,
      tag_ids: linkTagIds.value,
      expires_at,
      slug: linkSlug.value.trim(),
    })
    links.value.unshift(link)
    linkName.value = ''
    linkTagIds.value = []
    linkExpiry.value = ''
    linkSlug.value = ''
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    creatingLink.value = false
  }
}

async function revokeLink(id: number) {
  try {
    await api.deleteShareLink(id)
    links.value = links.value.filter((l) => l.id !== id)
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

async function rotateLink(id: number) {
  try {
    const updated = await api.rotateShareLink(id)
    links.value = links.value.map((l) => (l.id === id ? updated : l))
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

function startSlugEdit(l: ShareLink) {
  cancelLinkEdit()
  editingSlugId.value = l.id
  editingSlugValue.value = l.slug || ''
}

function cancelSlugEdit() {
  editingSlugId.value = null
  editingSlugValue.value = ''
}

async function saveSlug(l: ShareLink) {
  try {
    const updated = await api.updateShareLink(l.id, { slug: editingSlugValue.value.trim() })
    links.value = links.value.map((x) => (x.id === l.id ? updated : x))
    cancelSlugEdit()
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

function startLinkEdit(l: ShareLink) {
  cancelSlugEdit()
  editingLinkId.value = l.id
  editingName.value = l.name || ''
  editingScope.value = l.scope_mode
  editingTagIds.value = [...(l.tag_ids || [])]
}

function cancelLinkEdit() {
  editingLinkId.value = null
  editingName.value = ''
  editingTagIds.value = []
}

function toggleEditingTag(id: number) {
  const i = editingTagIds.value.indexOf(id)
  if (i >= 0) editingTagIds.value.splice(i, 1)
  else editingTagIds.value.push(id)
}

function linkTagNames(l: ShareLink): string {
  if (l.scope_mode === 'all') return ''
  return l.tag_ids
    .map((id) => tags.value.find((tg) => tg.id === id)?.name ?? `#${id}`)
    .join(', ')
}

async function saveLinkEdit(l: ShareLink) {
  if (editingScope.value === 'tags' && editingTagIds.value.length === 0) {
    toast.error(t('common.required'))
    return
  }
  savingLinkEdit.value = true
  try {
    const updated = await api.updateShareLink(l.id, {
      name: editingName.value.trim(),
      scope_mode: editingScope.value,
      tag_ids: editingScope.value === 'tags' ? [...editingTagIds.value] : [],
    })
    links.value = links.value.map((x) => (x.id === l.id ? updated : x))
    cancelLinkEdit()
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    savingLinkEdit.value = false
  }
}

function copyLink(url: string) {
  void navigator.clipboard?.writeText(url).then(
    () => toast.success(t('common.success')),
    () => toast.error(t('common.failed')),
  )
}

async function createGrantFn() {
  if (!grantUser.value.trim() || grantTagId.value == null) {
    toast.error(t('common.required'))
    return
  }
  creatingGrant.value = true
  try {
    const g = await api.createGrant(grantUser.value.trim(), grantTagId.value, grantPerm.value)
    grantsOwned.value.unshift(g)
    grantUser.value = ''
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  } finally {
    creatingGrant.value = false
  }
}

async function removeGrant(id: number, mine: 'owned' | 'received') {
  try {
    await api.deleteGrant(id)
    if (mine === 'owned') grantsOwned.value = grantsOwned.value.filter((g) => g.id !== id)
    else grantsReceived.value = grantsReceived.value.filter((g) => g.id !== id)
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}

async function setGrantPerm(g: TagGrant, perm: 'view' | 'edit') {
  try {
    const updated = await api.updateGrant(g.id, perm)
    grantsOwned.value = grantsOwned.value.map((x) => (x.id === g.id ? updated : x))
    toast.success(t('common.success'))
  } catch (e) {
    toast.error((e as ApiError).message)
  }
}
</script>

<template>
  <section v-if="!auth.isReadOnly" class="card mt-16">
    <h3 class="section-title">{{ t('share.title') }}</h3>
    <span class="hint">{{ t('share.hint') }}</span>

    <div class="col mt-8">
      <label>{{ t('share.linkName') }}</label>
      <input v-model="linkName" type="text" :placeholder="t('share.linkName')" />
    </div>
    <div class="row gap-8 mt-8">
      <label class="radio-opt">
        <input v-model="linkScope" type="radio" value="all" />{{ t('share.scopeAll') }}
      </label>
      <label class="radio-opt">
        <input v-model="linkScope" type="radio" value="tags" />{{ t('share.scopeTags') }}
      </label>
    </div>
    <div v-if="linkScope === 'tags'" class="tag-check-grid mt-8">
      <label
        v-for="tg in tags"
        :key="tg.id"
        class="tag-check"
        :class="{ checked: linkTagIds.includes(tg.id) }"
      >
        <input
          type="checkbox"
          :checked="linkTagIds.includes(tg.id)"
          @change="toggleLinkTag(tg.id)"
        />
        <span class="tag-dot" :style="{ background: tg.color }"></span>{{ tg.name }}
      </label>
    </div>
    <div class="col mt-8">
      <label>{{ t('share.expiry') }} ({{ t('common.optional') }})</label>
      <input v-model="linkExpiry" type="date" />
    </div>
    <div class="col mt-8">
      <label>{{ t('share.slug') }} ({{ t('common.optional') }})</label>
      <input v-model="linkSlug" type="text" :placeholder="t('share.slugHint')" maxlength="32" />
    </div>
    <div class="row mt-8">
      <button class="primary" :disabled="creatingLink" @click="createLink">
        {{ creatingLink ? t('common.loading') : t('share.create') }}
      </button>
    </div>

    <div v-if="links.length > 0" class="link-list mt-16">
      <div v-for="l in links" :key="l.id" class="link-row">
        <div class="grow">
          <template v-if="editingLinkId === l.id">
            <div class="col gap-8">
              <input
                v-model="editingName"
                type="text"
                :placeholder="t('share.linkName')"
                maxlength="50"
              />
              <div class="row gap-8">
                <label class="radio-opt">
                  <input v-model="editingScope" type="radio" value="all" />{{ t('share.scopeAll') }}
                </label>
                <label class="radio-opt">
                  <input v-model="editingScope" type="radio" value="tags" />{{ t('share.scopeTags') }}
                </label>
              </div>
              <div v-if="editingScope === 'tags'" class="tag-check-grid">
                <label
                  v-for="tg in tags"
                  :key="tg.id"
                  class="tag-check"
                  :class="{ checked: editingTagIds.includes(tg.id) }"
                >
                  <input
                    type="checkbox"
                    :checked="editingTagIds.includes(tg.id)"
                    @change="toggleEditingTag(tg.id)"
                  />
                  <span class="tag-dot" :style="{ background: tg.color }"></span>{{ tg.name }}
                </label>
              </div>
              <div class="row gap-8">
                <button class="primary" :disabled="savingLinkEdit" @click="saveLinkEdit(l)">
                  {{ savingLinkEdit ? t('common.loading') : t('common.save') }}
                </button>
                <button @click="cancelLinkEdit">{{ t('common.cancel') }}</button>
              </div>
            </div>
          </template>
          <template v-else>
            <strong>{{ l.name || t('share.untitled') }}</strong>
            <span class="muted"> · {{ l.scope_mode }}</span>
            <span v-if="l.scope_mode === 'tags'" class="muted"> · 🏷️ {{ linkTagNames(l) || `${l.tag_ids.length} tags` }}</span>
            <span v-if="l.expires_at" class="muted"> · ⏳ {{ l.expires_at.slice(0, 10) }}</span>
            <span v-if="l.slug" class="slug-badge">/{{ l.slug }}</span>
            <div class="url">{{ l.url }}</div>
          </template>
          <div v-if="editingSlugId === l.id" class="row gap-8 mt-8">
            <input
              v-model="editingSlugValue"
              type="text"
              :placeholder="t('share.slugHint')"
              maxlength="32"
              class="grow"
            />
            <button class="primary" @click="saveSlug(l)">{{ t('common.save') }}</button>
            <button @click="cancelSlugEdit">{{ t('common.cancel') }}</button>
          </div>
        </div>
        <div class="row gap-8">
          <button @click="copyLink(l.url)">📋</button>
          <button @click="startLinkEdit(l)" :title="t('share.linkName')">📝</button>
          <button @click="startSlugEdit(l)" :title="t('share.slug')">🔗</button>
          <button @click="rotateLink(l.id)" :title="t('share.rotate')">🔄</button>
          <button class="danger" @click="revokeLink(l.id)" :title="t('share.revoke')">🗑</button>
        </div>
      </div>
    </div>
  </section>

  <section v-if="!auth.isReadOnly" class="card mt-16">
    <h3 class="section-title">{{ t('grant.title') }}</h3>
    <span class="hint">{{ t('grant.hint') }}</span>

    <div class="row gap-8 mt-8">
      <input v-model="grantUser" type="text" :placeholder="t('grant.username')" class="grow" />
      <select v-model.number="grantTagId">
        <option :value="null">-- {{ t('birthday.tag') }} --</option>
        <option v-for="tg in tags" :key="tg.id" :value="tg.id">{{ tg.name }}</option>
      </select>
      <select v-model="grantPerm">
        <option value="edit">{{ t('grant.edit') }}</option>
        <option value="view">{{ t('grant.view') }}</option>
      </select>
      <button class="primary" :disabled="creatingGrant" @click="createGrantFn">{{ t('common.add') }}</button>
    </div>

    <div v-if="grantsOwned.length > 0" class="mt-8">
      <h4>{{ t('grant.owned') }}</h4>
      <div v-for="g in grantsOwned" :key="g.id" class="link-row">
        <span class="grow">🏷️ {{ g.tag_name }} → 👤 {{ g.grantee_username }}</span>
        <select :value="g.permission" @change="setGrantPerm(g, ($event.target as HTMLSelectElement).value as 'view' | 'edit')">
          <option value="edit">{{ t('grant.edit') }}</option>
          <option value="view">{{ t('grant.view') }}</option>
        </select>
        <button class="danger" @click="removeGrant(g.id, 'owned')">🗑</button>
      </div>
    </div>

    <div v-if="grantsReceived.length > 0" class="mt-8">
      <h4>{{ t('grant.received') }}</h4>
      <div v-for="g in grantsReceived" :key="g.id" class="link-row">
        <span class="grow">🏷️ {{ g.tag_name }} ← 👤 {{ g.owner_username }} ({{ g.permission }})</span>
        <button @click="removeGrant(g.id, 'received')" :title="t('grant.leave')">🚪</button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.section-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}
.hint {
  font-size: 12px;
  color: var(--color-text-muted);
}
.radio-opt {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  cursor: pointer;
}
.radio-opt input {
  width: auto;
}
.tag-check-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.tag-check {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}
.tag-check.checked {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: #fff;
}
.tag-check input {
  display: none;
}
.link-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.link-row {
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 8px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
}
.url {
  font-size: 12px;
  color: var(--color-text-muted);
  word-break: break-all;
}
.muted {
  color: var(--color-text-muted);
  font-size: 12px;
}
.slug-badge {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  border-radius: 4px;
  padding: 1px 6px;
  margin-left: 4px;
  white-space: nowrap;
}
button.danger {
  color: var(--color-danger);
}
</style>
