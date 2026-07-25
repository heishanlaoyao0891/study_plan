<template>
  <main class="workspace">
    <section class="page-head">
      <div><p class="eyebrow">用户声音</p><h2>反馈收件箱</h2></div>
      <button class="ghost small-button" type="button" :disabled="loading" @click="load">刷新</button>
    </section>

    <div class="toolbar">
      <select v-model="category" @change="load">
        <option value="">全部分类</option><option value="issue">问题故障</option><option value="suggestion">功能建议</option>
        <option value="content">内容问题</option><option value="account">账号问题</option><option value="other">其他</option>
      </select>
      <select v-model="status" @change="load">
        <option value="">全部状态</option><option value="open">待处理</option><option value="processing">处理中</option>
        <option value="resolved">已解决</option><option value="closed">已关闭</option>
      </select>
      <button class="primary small-button" type="button" :disabled="loading" @click="load">筛选</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="loading" class="state">正在加载反馈…</p>
    <p v-else-if="!rows.length" class="state">没有符合条件的反馈</p>
    <div v-else class="feedback-layout">
      <div class="inbox">
        <button v-for="row in rows" :key="row.id" class="feedback-item" :class="{ active: selected?.id === row.id }" type="button" @click="select(row)">
          <span class="item-head"><strong>#{{ row.id }} · {{ categoryLabel(row.category) }}</strong><span class="pill">{{ statusLabel(row.status) }}</span></span>
          <span class="item-user">{{ row.user_nickname || `用户 #${row.user_id}` }} · {{ formatTime(row.created_at) }}</span>
          <span class="item-content">{{ row.content }}</span>
        </button>
      </div>

      <section v-if="selected" class="panel detail">
        <div class="detail-head"><div><p class="eyebrow">反馈 #{{ selected.id }}</p><h3>{{ categoryLabel(selected.category) }}</h3></div><span class="pill">{{ statusLabel(selected.status) }}</span></div>
        <dl><dt>用户</dt><dd>{{ selected.user_nickname || '-' }}（#{{ selected.user_id }}）</dd><dt>提交时间</dt><dd>{{ formatTime(selected.created_at) }}</dd><dt>联系方式</dt><dd>{{ selected.contact || '-' }}</dd></dl>
        <div class="content-block">{{ selected.content }}</div>
        <label class="field"><span>处理状态</span><select v-model="editStatus"><option value="open">待处理</option><option value="processing">处理中</option><option value="resolved">已解决</option><option value="closed">已关闭</option></select></label>
        <label class="field"><span>给用户的公开回复（最多 1000 字）</span><textarea v-model="response" maxlength="1000" rows="7" placeholder="仅填写可公开给该用户的内容" /></label>
        <p v-if="saveError" class="error">{{ saveError }}</p>
        <p v-if="success" class="success">{{ success }}</p>
        <button class="primary small-button" type="button" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存处理结果' }}</button>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { AdminApi, type FeedbackReport } from '@/api'

const rows = ref<FeedbackReport[]>([])
const selected = ref<FeedbackReport | null>(null)
const category = ref('')
const status = ref('')
const editStatus = ref<FeedbackReport['status']>('open')
const response = ref('')
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const saveError = ref('')
const success = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    rows.value = await AdminApi.feedback({ category: category.value, status: status.value })
    if (selected.value) {
      const refreshed = rows.value.find(row => row.id === selected.value?.id)
      if (refreshed) select(refreshed)
      else selected.value = null
    }
  } catch (err) { error.value = err instanceof Error ? err.message : '反馈加载失败' }
  finally { loading.value = false }
}

function select(row: FeedbackReport) {
  selected.value = row
  editStatus.value = row.status
  response.value = row.public_response || ''
  saveError.value = ''
  success.value = ''
}

async function save() {
  if (!selected.value || saving.value) return
  saveError.value = ''
  success.value = ''
  if (Array.from(response.value.trim()).length > 1000) { saveError.value = '公开回复不能超过 1000 个字符'; return }
  saving.value = true
  try {
    const originalResponse = selected.value.public_response || ''
    const trimmedResponse = response.value.trim()
    const responseUpdate = trimmedResponse === originalResponse
      ? {}
      : trimmedResponse
        ? { public_response: trimmedResponse }
        : { clear_public_response: true }
    const updated = await AdminApi.updateFeedback(selected.value.id, { status: editStatus.value, ...responseUpdate })
    const index = rows.value.findIndex(row => row.id === updated.id)
    if (index >= 0) rows.value[index] = { ...rows.value[index], ...updated }
    select(rows.value[index])
    success.value = '处理结果已保存'
  } catch (err) { saveError.value = err instanceof Error ? err.message : '保存失败' }
  finally { saving.value = false }
}

function categoryLabel(value: string) { return ({ issue: '问题故障', suggestion: '功能建议', content: '内容问题', account: '账号问题', other: '其他' } as Record<string, string>)[value] || value }
function statusLabel(value: string) { return ({ open: '待处理', processing: '处理中', resolved: '已解决', closed: '已关闭' } as Record<string, string>)[value] || value }
function formatTime(value: string) { return value ? new Date(value).toLocaleString() : '-' }

onMounted(load)
</script>

<style scoped>
.state { padding: 32px; border: 1px dashed #e3d8de; border-radius: 16px; color: #786b74; background: #fff; text-align: center; }
.feedback-layout { display: grid; grid-template-columns: minmax(300px, .8fr) minmax(420px, 1.2fr); gap: 20px; align-items: start; }
.inbox { display: grid; gap: 10px; }
.feedback-item { display: grid; gap: 8px; width: 100%; padding: 16px; border: 1px solid #eadde3; border-radius: 14px; background: #fff; color: #332a30; text-align: left; cursor: pointer; }
.feedback-item.active { border-color: #d96187; box-shadow: 0 0 0 2px rgba(217,97,135,.1); }
.item-head { display: flex; justify-content: space-between; gap: 12px; align-items: center; }
.item-user { color: #8a7a84; font-size: 13px; }
.item-content { overflow: hidden; color: #554952; text-overflow: ellipsis; white-space: nowrap; }
.detail { position: sticky; top: 20px; }
.detail-head { display: flex; align-items: start; justify-content: space-between; }
.detail-head h3 { margin: 3px 0 0; }
dl { display: grid; grid-template-columns: 90px 1fr; gap: 8px 12px; margin: 20px 0; font-size: 14px; }
dt { color: #8a7a84; } dd { margin: 0; }
.content-block { margin-bottom: 20px; padding: 16px; border-radius: 12px; background: #faf6f8; line-height: 1.7; white-space: pre-wrap; }
textarea { width: 100%; box-sizing: border-box; padding: 12px 14px; border: 1px solid #f1ccd7; border-radius: 14px; color: #2f2430; background: #fffafd; font: inherit; resize: vertical; }
.success { color: #267052; }
@media (max-width: 900px) { .feedback-layout { grid-template-columns: 1fr; } .detail { position: static; } }
</style>
