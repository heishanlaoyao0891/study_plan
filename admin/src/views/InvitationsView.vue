<template>
  <main class="workspace">
    <section class="page-head">
      <div>
        <p class="eyebrow">早期访问</p>
        <h2>注册邀请</h2>
      </div>
      <button class="ghost small-button" type="button" :disabled="loading" @click="load">刷新</button>
    </section>

    <section class="panel generation-panel">
      <div>
        <h3>生成邀请码</h3>
        <p>每个邀请码仅可使用一次，并在生成七天后过期。</p>
      </div>
      <form class="generate-form" @submit.prevent="generate">
        <label class="field">
          <span>生成数量</span>
          <input v-model.number="count" type="number" min="1" max="100" />
        </label>
        <button class="primary small-button" :disabled="generating" type="submit">
          {{ generating ? '生成中...' : '生成邀请码' }}
        </button>
      </form>
    </section>

    <section v-if="newCodes.length" class="panel new-codes">
      <div class="codes-heading">
        <div>
          <p class="eyebrow">仅本次显示</p>
          <h3>请妥善保存新邀请码</h3>
        </div>
        <button class="ghost small-button" type="button" @click="copyAll">复制全部</button>
      </div>
      <p class="codes-warning">离开或刷新页面后无法再次查看完整邀请码。</p>
      <div class="code-list">
        <div v-for="code in newCodes" :key="code" class="code-row">
          <code>{{ code }}</code>
          <button class="copy-button" type="button" @click="copy(code)">{{ copied === code ? '已复制' : '复制' }}</button>
        </div>
      </div>
    </section>

    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="notice" class="status">{{ notice }}</p>

    <table class="data-table invitation-table">
      <thead>
        <tr><th>前缀</th><th>状态</th><th>创建时间</th><th>过期时间</th><th>使用用户</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="invitation in invitations" :key="invitation.id">
          <td><code>{{ invitation.code_prefix }}...</code></td>
          <td><span class="pill" :class="`invitation-${invitation.status}`">{{ statusText[invitation.status] }}</span></td>
          <td>{{ formatTime(invitation.created_at) }}</td>
          <td>{{ formatTime(invitation.expires_at) }}</td>
          <td>{{ usedUser(invitation) }}</td>
          <td>
            <button
              v-if="invitation.status === 'active'"
              class="disable-button"
              type="button"
              :disabled="disablingId === invitation.id"
              @click="disable(invitation)"
            >
              {{ disablingId === invitation.id ? '停用中...' : '停用' }}
            </button>
            <span v-else class="muted">-</span>
          </td>
        </tr>
        <tr v-if="!loading && invitations.length === 0">
          <td class="empty-cell" colspan="6">还没有注册邀请码</td>
        </tr>
      </tbody>
    </table>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'

import { AdminApi, type RegistrationInvitation } from '@/api'

const invitations = ref<RegistrationInvitation[]>([])
const newCodes = ref<string[]>([])
const count = ref(1)
const loading = ref(false)
const generating = ref(false)
const disablingId = ref<number | null>(null)
const copied = ref('')
const error = ref('')
const notice = ref('')
const statusText: Record<RegistrationInvitation['status'], string> = {
  active: '可使用',
  used: '已使用',
  expired: '已过期',
  disabled: '已停用',
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const response = await AdminApi.invitations()
    invitations.value = Array.isArray(response) ? response : response.invitations
  } catch (err) {
    error.value = err instanceof Error ? err.message : '邀请码列表加载失败'
  } finally {
    loading.value = false
  }
}

async function generate() {
  error.value = ''
  notice.value = ''
  if (!Number.isInteger(count.value) || count.value < 1 || count.value > 100) {
    error.value = '生成数量需要是 1 到 100 的整数'
    return
  }
  generating.value = true
  try {
    const response = await AdminApi.generateInvitations(count.value)
    newCodes.value = Array.isArray(response) ? response : response.codes
    notice.value = `已生成 ${newCodes.value.length} 个邀请码`
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '邀请码生成失败'
  } finally {
    generating.value = false
  }
}

async function disable(invitation: RegistrationInvitation) {
  if (!window.confirm(`确认停用邀请码 ${invitation.code_prefix}...？停用后无法恢复。`)) return
  disablingId.value = invitation.id
  error.value = ''
  notice.value = ''
  try {
    await AdminApi.disableInvitation(invitation.id)
    notice.value = '邀请码已停用'
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '邀请码停用失败'
  } finally {
    disablingId.value = null
  }
}

async function copy(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    copied.value = value
    window.setTimeout(() => {
      if (copied.value === value) copied.value = ''
    }, 1600)
  } catch {
    error.value = '复制失败，请手动选择邀请码'
  }
}

function copyAll() {
  copy(newCodes.value.join('\n'))
}

function formatTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function usedUser(invitation: RegistrationInvitation) {
  const user = invitation.used_by_user || invitation.used_by || invitation.used_user
  if (!user) return invitation.user_id ? `#${invitation.user_id}` : '-'
  return `${user.nickname || user.openid || '用户'} (#${user.id})`
}

onMounted(load)
</script>

<style scoped>
.generation-panel { display: flex; align-items: end; justify-content: space-between; gap: 32px; margin-bottom: 18px; }
.generation-panel p, .codes-warning { margin: 8px 0 0; color: #7a6773; font-size: 14px; }
.generate-form { display: flex; align-items: end; gap: 12px; }
.generate-form .field { width: 150px; }
.new-codes { margin-bottom: 18px; border-color: #ffc1d2; background: linear-gradient(135deg, #fff, #fff6e8); }
.codes-heading { display: flex; align-items: center; justify-content: space-between; }
.codes-warning { color: #a34e2c; font-weight: 800; }
.code-list { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin-top: 18px; }
.code-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 11px 12px; border: 1px solid #f3d6d0; border-radius: 13px; background: rgba(255, 255, 255, .9); }
.code-row code { min-width: 0; overflow-wrap: anywhere; color: #4b3040; font-weight: 800; }
.copy-button, .disable-button { border: 0; background: transparent; color: #e65d87; cursor: pointer; font-weight: 900; }
.copy-button { flex: 0 0 auto; }
.disable-button:disabled { cursor: default; opacity: .55; }
.invitation-table { margin-top: 18px; }
.invitation-table code { color: #604758; font-weight: 800; }
.invitation-active { color: #166534; background: #dcfce7; }
.invitation-used { color: #1d4ed8; background: #dbeafe; }
.invitation-expired, .invitation-disabled { color: #6b7280; background: #f3f4f6; }
.muted { color: #a99ca4; }
.empty-cell { padding: 34px !important; color: #8b7b84; text-align: center !important; }
</style>
