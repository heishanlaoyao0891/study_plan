<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">智能生成</p><h2>AI 配置</h2></div></section>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="status" class="status">{{ status }}</p>
    <form class="panel wide-panel" @submit.prevent="save">
      <label class="check-row"><input v-model="form.enabled" type="checkbox" /> 启用 AI 生成</label>
      <label class="field"><span>服务商</span><input v-model.trim="form.provider" /></label>
      <label class="field"><span>模型名称</span><input v-model.trim="form.model_name" /></label>
      <label class="field"><span>Base URL</span><input v-model.trim="form.base_url" /></label>
      <label class="field"><span>请求超时（秒）</span><input v-model.number="form.request_timeout_seconds" type="number" min="1" /></label>
      <label class="field"><span>每日生成上限</span><input v-model.number="form.daily_generation_limit" type="number" min="0" /></label>
      <label class="field"><span>API Key</span><input v-model="apiKey" :placeholder="form.api_key_masked || '留空则保留当前密钥'" type="password" /></label>
      <div class="button-row">
        <button class="primary small-button">保存配置</button>
        <button class="ghost small-button" type="button" @click="testProvider">测试连接</button>
      </div>
    </form>
  </main>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'

import { AdminApi, type AIConfig } from '@/api'

const error = ref('')
const status = ref('')
const apiKey = ref('')
const form = reactive<AIConfig>({ provider: 'mock', model_name: '', base_url: '', request_timeout_seconds: 30, daily_generation_limit: 5, enabled: true })

onMounted(async () => {
  try {
    Object.assign(form, await AdminApi.aiConfig())
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'AI 配置加载失败'
  }
})

async function save() {
  await AdminApi.saveAIConfig({ ...form, api_key: apiKey.value || undefined })
  apiKey.value = ''
  status.value = 'AI 配置已保存'
}

async function testProvider() {
  const result = await AdminApi.testAIConfig({ ...form, api_key: apiKey.value || undefined })
  status.value = result.ok ? '连接测试通过' : result.message
}
</script>
