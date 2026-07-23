<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">微信提醒</p><h2>订阅消息</h2></div></section>
    <p v-if="error" class="error">{{ error }}</p>
    <form class="panel wide-panel" @submit.prevent="save">
      <ConfigRow label="学习开始提醒" v-model:template-id="form.study_start_template_id" v-model:enabled="form.study_start_enabled" />
      <ConfigRow label="完成提醒" v-model:template-id="form.completion_template_id" v-model:enabled="form.completion_enabled" />
      <ConfigRow label="23:30 决策提醒" v-model:template-id="form.decision_template_id" v-model:enabled="form.decision_enabled" />
      <ConfigRow label="未打卡提醒" v-model:template-id="form.missed_checkin_template_id" v-model:enabled="form.missed_checkin_enabled" />
      <button class="primary small-button">保存订阅配置</button>
    </form>
    <table class="data-table status-table">
      <thead><tr><th>类型</th><th>状态</th><th>消息</th><th>时间</th></tr></thead>
      <tbody><tr v-for="item in form.recent_status || []" :key="item.id"><td>{{ item.reminder_type }}</td><td>{{ item.status }}</td><td>{{ item.message || '-' }}</td><td>{{ item.created_at }}</td></tr></tbody>
    </table>
  </main>
</template>

<script setup lang="ts">
import { defineComponent, h, onMounted, reactive, ref } from 'vue'

import { AdminApi, type SubscriptionConfig } from '@/api'

const ConfigRow = defineComponent({
  props: { label: { type: String, required: true }, templateId: { type: String, required: true }, enabled: { type: Boolean, required: true } },
  emits: ['update:templateId', 'update:enabled'],
  setup(props, { emit }) {
    return () => h('div', { class: 'config-row' }, [
      h('label', { class: 'check-row' }, [h('input', { type: 'checkbox', checked: props.enabled, onChange: (event: Event) => emit('update:enabled', (event.target as HTMLInputElement).checked) }), props.label]),
      h('input', { value: props.templateId, placeholder: '模板 ID', onInput: (event: Event) => emit('update:templateId', (event.target as HTMLInputElement).value) }),
    ])
  },
})

const error = ref('')
const form = reactive<SubscriptionConfig>({
  study_start_template_id: '', completion_template_id: '', decision_template_id: '', missed_checkin_template_id: '',
  study_start_enabled: true, completion_enabled: true, decision_enabled: true, missed_checkin_enabled: true, recent_status: [],
})

onMounted(async () => {
  try {
    Object.assign(form, await AdminApi.subscriptionConfig())
  } catch (err) {
    error.value = err instanceof Error ? err.message : '订阅配置加载失败'
  }
})

async function save() {
  Object.assign(form, await AdminApi.saveSubscriptionConfig(form))
}
</script>
