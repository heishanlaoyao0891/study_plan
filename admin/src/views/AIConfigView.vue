<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">智能生成</p><h2>模型任务拆解配置</h2><p class="page-note">本地 Agent 会立即生成可用基线；模型在后台负责拆解学习阶段和任务，后端再完成日期、时段与冲突修复。</p></div></section>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="status" class="status">{{ status }}</p>
    <section v-if="metrics" class="metrics-grid">
      <div><strong>{{ metrics.queue_depth }}</strong><span>当前队列</span></div>
      <div><strong>{{ percent(metrics.success_rate) }}</strong><span>AI 成功率</span></div>
      <div><strong>{{ percent(metrics.fallback_rate) }}</strong><span>回退率</span></div>
      <div><strong>{{ metrics.p50_latency_ms }} ms</strong><span>p50 延迟</span></div>
      <div><strong>{{ metrics.p95_latency_ms }} ms</strong><span>p95 延迟</span></div>
      <div><strong>{{ metrics.total_tokens }}</strong><span>Token 总量</span></div>
      <div><strong>{{ metrics.provider_attempts || 0 }}</strong><span>服务商尝试</span></div>
      <div><strong>{{ metrics.failed_provider_responses || 0 }}</strong><span>失败响应</span></div>
      <div><strong>{{ metrics.successful_generations || 0 }}</strong><span>成功生成计次</span></div>
    </section>
    <form class="panel wide-panel" @submit.prevent="save">
      <label class="check-row"><input v-model="form.enabled" type="checkbox" /> 启用模型任务拆解</label>
      <label class="field"><span>服务商</span><select v-model="form.provider" @change="applyPreset"><option value="siliconflow">SiliconFlow（推荐）</option><option value="openai_compatible">OpenAI 兼容服务</option><option value="mock">规则回退（不调用 AI）</option></select></label>
      <label class="field"><span>模型名称</span><input v-model.trim="form.model_name" /></label>
      <label class="field"><span>Base URL</span><input v-model.trim="form.base_url" /></label>
      <label class="field"><span>连接测试超时（秒）</span><input v-model.number="form.request_timeout_seconds" type="number" min="1" max="120" /></label>
      <label class="field"><span>交互基线目标（秒）</span><input v-model.number="form.interactive_target_seconds" type="number" min="1" max="5" /></label>
      <label class="field"><span>后台 Agent 预算</span><select v-model.number="form.background_job_timeout_seconds"><option :value="300">5 分钟</option><option :value="600">10 分钟</option></select></label>
      <label class="field"><span>每日生成上限</span><input v-model.number="form.daily_generation_limit" type="number" min="1" max="100" /></label>
      <label class="field"><span>API Key</span><input v-model="apiKey" :placeholder="form.api_key_masked || '留空则保留当前密钥'" type="password" /></label>
      <div class="config-summary">
        <span>当前模式：{{ modeLabel }}</span>
        <span>密钥存储：{{ keyStorageLabel }}</span>
        <span>AI 暂不可用时：后台保留任务并自动修复重试</span>
      </div>
      <div class="button-row">
        <button class="primary small-button">保存配置</button>
        <button class="ghost small-button" type="button" :disabled="!form.enabled || form.provider === 'mock'" @click="testProvider">测试结构化输出</button>
      </div>
    </form>
    <section class="panel wide-panel invocation-panel">
      <div class="invocation-head"><div><h3>AI 调用流水</h3><p class="page-note">每次真实模型 HTTP 请求均单独留痕；仅保存哈希、长度、耗时、Token 与安全错误分类。</p></div><button class="ghost small-button" type="button" @click="() => loadInvocations()">刷新</button></div>
      <div class="invocation-filters">
        <input v-model.trim="invocationFilter.user_id" placeholder="用户 ID" />
        <input v-model.trim="invocationFilter.job_id" placeholder="Job ID" />
        <select v-model="invocationFilter.status"><option value="">全部状态</option><option value="succeeded">成功</option><option value="failed">失败</option><option value="truncated">截断</option><option value="started">执行中</option></select>
        <button class="primary small-button" type="button" @click="queryInvocations">查询</button>
      </div>
      <div class="table-wrap">
        <table class="invocation-table"><thead><tr><th>时间 / Trace</th><th>用户 / Job</th><th>阶段 / 轮次</th><th>Provider / Model</th><th>结果</th><th>耗时 / Token</th><th>原因</th></tr></thead>
        <tbody><tr v-for="row in invocations" :key="row.trace_id"><td>{{ formatTime(row.started_at) }}<small>{{ row.trace_id.slice(0, 12) }}</small></td><td>{{ row.user_nickname || (row.user_id ? `用户 ${row.user_id}` : '系统') }}<small>{{ row.job_type }} {{ row.job_id || '' }}</small></td><td>{{ row.phase || '-' }}<small>批次 {{ row.batch_index || '-' }} · Agent {{ row.agent_attempt || '-' }} · HTTP {{ row.provider_attempt }}</small></td><td>{{ row.provider }}<small>{{ row.model }}</small></td><td><span class="invoke-status" :class="row.status">{{ statusLabel(row.status) }}</span><small>HTTP {{ row.http_status || '-' }} · {{ row.finish_reason || '-' }}</small></td><td>{{ row.duration_ms }} ms<small>{{ row.total_tokens || 0 }} tokens</small></td><td>{{ row.error_code || '-' }}<small>{{ row.error_message || '' }}</small></td></tr><tr v-if="!invocations.length"><td colspan="7">暂无调用记录</td></tr></tbody></table>
      </div>
      <div class="invocation-pagination">
        <span class="pagination-summary">共 {{ invocationTotal }} 条 · 第 {{ invocationPage }} / {{ invocationTotalPages }} 页</span>
        <label class="page-size"><span>每页</span><select v-model.number="invocationPageSize" @change="changeInvocationPageSize"><option :value="20">20</option><option :value="50">50</option><option :value="100">100</option></select></label>
        <div class="pagination-actions"><button class="ghost small-button" type="button" :disabled="invocationPage <= 1 || loadingInvocations" @click="loadInvocations(invocationPage - 1)">上一页</button><button class="ghost small-button" type="button" :disabled="invocationPage >= invocationTotalPages || loadingInvocations" @click="loadInvocations(invocationPage + 1)">下一页</button></div>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'

import { AdminApi, type AIConfig, type AIInvocationLog, type AIPlanningMetrics } from '@/api'

const error = ref('')
const status = ref('')
const apiKey = ref('')
const metrics = ref<AIPlanningMetrics | null>(null)
const invocations = ref<AIInvocationLog[]>([])
const invocationFilter = reactive({ user_id: '', job_id: '', status: '' })
const invocationPage = ref(1)
const invocationPageSize = ref(20)
const invocationTotal = ref(0)
const loadingInvocations = ref(false)
const invocationTotalPages = computed(() => Math.max(1, Math.ceil(invocationTotal.value / invocationPageSize.value)))
const form = reactive<AIConfig>({ provider: 'mock', model_name: '', base_url: '', request_timeout_seconds: 30, interactive_target_seconds: 2, background_job_timeout_seconds: 300, daily_generation_limit: 5, enabled: true })
const modeLabel = computed(() => ({ ai: 'AI 生成', fallback: '规则回退', disabled: '已停用' }[form.effective_mode || (form.enabled ? (form.provider === 'mock' ? 'fallback' : 'ai') : 'disabled')]))
const keyStorageLabel = computed(() => ({ encrypted: '已加密', plaintext: '明文，需重新保存密钥', missing: '未配置' }[form.key_storage || 'missing']))

onMounted(async () => {
  try {
    const [config, planningMetrics, invocationResult] = await Promise.all([AdminApi.aiConfig(), AdminApi.aiMetrics(), AdminApi.aiInvocations({ page: invocationPage.value, size: invocationPageSize.value })])
    Object.assign(form, config)
    metrics.value = planningMetrics
    invocations.value = invocationResult.items
    invocationPage.value = invocationResult.page
    invocationTotal.value = invocationResult.total
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'AI 配置加载失败'
  }
})

async function loadInvocations(page = invocationPage.value) {
  error.value = ''
  loadingInvocations.value = true
  try {
    const [result, planningMetrics] = await Promise.all([
      AdminApi.aiInvocations({ page, size: invocationPageSize.value, ...invocationFilter }),
      AdminApi.aiMetrics(),
    ])
    invocations.value = result.items
    invocationPage.value = result.page
    invocationTotal.value = result.total
    metrics.value = planningMetrics
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'AI 调用流水加载失败'
  } finally {
    loadingInvocations.value = false
  }
}

function changeInvocationPageSize() {
  invocationPage.value = 1
  loadInvocations(1)
}

function queryInvocations() {
  invocationPage.value = 1
  loadInvocations(1)
}

async function save() {
  error.value = ''
  try {
    Object.assign(form, await AdminApi.saveAIConfig({ ...form, api_key: apiKey.value || undefined }))
    apiKey.value = ''
    status.value = 'AI 配置已保存'
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'AI 配置保存失败'
  }
}

async function testProvider() {
  error.value = ''
  try {
    const result = await AdminApi.testAIConfig({ ...form, api_key: apiKey.value || undefined })
    status.value = result.ok ? '结构化计划测试通过' : `测试失败：${result.message}`
  } catch (err) {
    error.value = err instanceof Error ? err.message : '连接测试失败'
  }
}

function applyPreset() {
  if (form.provider === 'siliconflow') {
    form.base_url = 'https://api.siliconflow.cn/v1'
    form.model_name = 'deepseek-ai/DeepSeek-V3.2'
  }
}

function percent(value: number) {
  return `${Math.round(value * 100)}%`
}

function formatTime(value: string) { return new Date(value).toLocaleString('zh-CN') }
function statusLabel(value: AIInvocationLog['status']) { return ({ started: '执行中', succeeded: '成功', failed: '失败', truncated: '截断' } as const)[value] }
</script>

<style scoped>
.metrics-grid { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:12px; margin-bottom:18px; }
.metrics-grid div { display:flex; flex-direction:column; gap:3px; padding:14px; border:1px solid #e4e8f0; border-radius:10px; background:#fff; }
.metrics-grid strong { font-size:20px; color:#1f4f9a; }
.metrics-grid span { color:#788397; font-size:13px; }
.invocation-panel { width:100%; max-width:none; margin-top:18px; }
.invocation-head { display:flex; justify-content:space-between; gap:16px; align-items:flex-start; }
.invocation-filters { display:grid; grid-template-columns:1fr 1fr 1fr auto; gap:10px; margin:16px 0; }
.invocation-filters input,.invocation-filters select { min-height:38px; padding:8px 10px; border:1px solid #d9dfeb; border-radius:8px; }
.table-wrap { width:100%; overflow-x:auto; }
table { width:100%; border-collapse:collapse; }
.invocation-table { table-layout:fixed; }
.invocation-table th:nth-child(1) { width:14%; }.invocation-table th:nth-child(2) { width:16%; }.invocation-table th:nth-child(3) { width:16%; }.invocation-table th:nth-child(4) { width:14%; }.invocation-table th:nth-child(5) { width:14%; }.invocation-table th:nth-child(6) { width:11%; }.invocation-table th:nth-child(7) { width:15%; }
th,td { padding:10px; border-bottom:1px solid #e6eaf1; text-align:left; vertical-align:top; overflow-wrap:anywhere; }
td small { display:block; max-width:260px; margin-top:4px; color:#7b8496; overflow-wrap:anywhere; }
.invoke-status { display:inline-block; padding:2px 8px; border-radius:999px; background:#eef2f7; }
.invoke-status.succeeded { color:#18794e; background:#e8f7ef; }.invoke-status.failed { color:#b4233b; background:#fff0f2; }.invoke-status.truncated { color:#9a5a10; background:#fff4df; }.invoke-status.started { color:#2459a9; background:#eaf2ff; }
.invocation-pagination { display:flex; align-items:center; justify-content:flex-end; gap:14px; flex-wrap:wrap; margin-top:14px; color:#788397; font-size:13px; }
.pagination-summary { margin-right:auto; }.page-size,.pagination-actions { display:flex; align-items:center; gap:8px; }.page-size select { min-height:34px; padding:4px 8px; border:1px solid #d9dfeb; border-radius:7px; background:#fff; color:#334155; }.pagination-actions button:disabled { opacity:.45; cursor:not-allowed; }
@media (max-width:760px) { .metrics-grid { grid-template-columns:repeat(2,minmax(0,1fr)); } }
</style>
