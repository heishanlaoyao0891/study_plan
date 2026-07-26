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
    </section>
    <form class="panel wide-panel" @submit.prevent="save">
      <label class="check-row"><input v-model="form.enabled" type="checkbox" /> 启用模型任务拆解</label>
      <label class="field"><span>服务商</span><select v-model="form.provider" @change="applyPreset"><option value="siliconflow">SiliconFlow（推荐）</option><option value="openai_compatible">OpenAI 兼容服务</option><option value="mock">规则回退（不调用 AI）</option></select></label>
      <label class="field"><span>模型名称</span><input v-model.trim="form.model_name" /></label>
      <label class="field"><span>Base URL</span><input v-model.trim="form.base_url" /></label>
      <label class="field"><span>请求超时（秒）</span><input v-model.number="form.request_timeout_seconds" type="number" min="1" /></label>
      <label class="field"><span>交互基线目标（秒）</span><input v-model.number="form.interactive_target_seconds" type="number" min="1" max="5" /></label>
      <label class="field"><span>后台拆解预算（秒）</span><input v-model.number="form.background_job_timeout_seconds" type="number" min="15" max="120" /></label>
      <label class="field"><span>每日生成上限</span><input v-model.number="form.daily_generation_limit" type="number" min="1" max="100" /></label>
      <label class="field"><span>API Key</span><input v-model="apiKey" :placeholder="form.api_key_masked || '留空则保留当前密钥'" type="password" /></label>
      <div class="config-summary">
        <span>当前模式：{{ modeLabel }}</span>
        <span>密钥存储：{{ keyStorageLabel }}</span>
        <span>AI 不可用时：明确返回规则回退模式</span>
      </div>
      <div class="button-row">
        <button class="primary small-button">保存配置</button>
        <button class="ghost small-button" type="button" :disabled="!form.enabled || form.provider === 'mock'" @click="testProvider">测试结构化输出</button>
      </div>
    </form>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'

import { AdminApi, type AIConfig, type AIPlanningMetrics } from '@/api'

const error = ref('')
const status = ref('')
const apiKey = ref('')
const metrics = ref<AIPlanningMetrics | null>(null)
const form = reactive<AIConfig>({ provider: 'mock', model_name: '', base_url: '', request_timeout_seconds: 30, interactive_target_seconds: 2, background_job_timeout_seconds: 60, daily_generation_limit: 5, enabled: true })
const modeLabel = computed(() => ({ ai: 'AI 生成', fallback: '规则回退', disabled: '已停用' }[form.effective_mode || (form.enabled ? (form.provider === 'mock' ? 'fallback' : 'ai') : 'disabled')]))
const keyStorageLabel = computed(() => ({ encrypted: '已加密', plaintext: '明文，需重新保存密钥', missing: '未配置' }[form.key_storage || 'missing']))

onMounted(async () => {
  try {
    const [config, planningMetrics] = await Promise.all([AdminApi.aiConfig(), AdminApi.aiMetrics()])
    Object.assign(form, config)
    metrics.value = planningMetrics
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'AI 配置加载失败'
  }
})

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
</script>

<style scoped>
.metrics-grid { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:12px; margin-bottom:18px; }
.metrics-grid div { display:flex; flex-direction:column; gap:3px; padding:14px; border:1px solid #e4e8f0; border-radius:10px; background:#fff; }
.metrics-grid strong { font-size:20px; color:#1f4f9a; }
.metrics-grid span { color:#788397; font-size:13px; }
@media (max-width:760px) { .metrics-grid { grid-template-columns:repeat(2,minmax(0,1fr)); } }
</style>
