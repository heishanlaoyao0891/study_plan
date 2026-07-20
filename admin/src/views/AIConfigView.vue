<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">Generation</p><h2>AI Configuration</h2></div></section>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="status" class="status">{{ status }}</p>
    <form class="panel wide-panel" @submit.prevent="save">
      <label class="check-row"><input v-model="form.enabled" type="checkbox" /> Enabled</label>
      <label class="field"><span>Provider</span><input v-model.trim="form.provider" /></label>
      <label class="field"><span>Model name</span><input v-model.trim="form.model_name" /></label>
      <label class="field"><span>Base URL</span><input v-model.trim="form.base_url" /></label>
      <label class="field"><span>Request timeout seconds</span><input v-model.number="form.request_timeout_seconds" type="number" min="1" /></label>
      <label class="field"><span>Daily generation limit</span><input v-model.number="form.daily_generation_limit" type="number" min="0" /></label>
      <label class="field"><span>API key</span><input v-model="apiKey" :placeholder="form.api_key_masked || 'Leave blank to keep current key'" type="password" /></label>
      <div class="button-row">
        <button class="primary small-button">Save AI config</button>
        <button class="ghost small-button" type="button" @click="testProvider">Test provider</button>
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
const form = reactive<AIConfig>({ provider: 'mock', model_name: '', base_url: '', request_timeout_seconds: 30, daily_generation_limit: 20, enabled: true })

onMounted(async () => {
  try {
    Object.assign(form, await AdminApi.aiConfig())
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load AI config'
  }
})

async function save() {
  await AdminApi.saveAIConfig({ ...form, api_key: apiKey.value || undefined })
  apiKey.value = ''
}

async function testProvider() {
  const result = await AdminApi.testAIConfig({ ...form, api_key: apiKey.value || undefined })
  status.value = result.ok ? 'Provider test passed' : result.message
}
</script>
