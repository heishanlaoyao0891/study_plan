<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">微信提醒</p><h2>订阅消息</h2><p>每个启用项必须填写微信模板 ID、小程序页面路径和 JSON 字段映射。映射格式为 {"thing1":"message"}，可用值：message、title、date、planned_start、planned_end、sender。</p></div></section>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="success" class="success">{{ success }}</p>
    <form class="panel wide-panel" @submit.prevent="save">
      <ConfigRow label="学习开始提醒" v-model:template-id="form.study_start_template_id" v-model:enabled="form.study_start_enabled" v-model:page="form.study_start_page" v-model:mapping="form.study_start_field_mapping" />
      <ConfigRow label="完成提醒" v-model:template-id="form.completion_template_id" v-model:enabled="form.completion_enabled" v-model:page="form.completion_page" v-model:mapping="form.completion_field_mapping" />
      <ConfigRow label="23:30 决策提醒" v-model:template-id="form.decision_template_id" v-model:enabled="form.decision_enabled" v-model:page="form.decision_page" v-model:mapping="form.decision_field_mapping" />
      <ConfigRow label="未打卡提醒" v-model:template-id="form.missed_checkin_template_id" v-model:enabled="form.missed_checkin_enabled" v-model:page="form.missed_checkin_page" v-model:mapping="form.missed_checkin_field_mapping" />
      <ConfigRow label="小组督学提醒" v-model:template-id="form.group_nudge_template_id" v-model:enabled="form.group_nudge_enabled" v-model:page="form.group_nudge_page" v-model:mapping="form.group_nudge_field_mapping" />
      <button class="primary small-button" :disabled="saving">{{ saving ? '保存中…' : '保存订阅配置' }}</button>
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
  props: { label: { type: String, required: true }, templateId: { type: String, required: true }, enabled: { type: Boolean, required: true }, page: { type: String, required: true }, mapping: { type: String, required: true } },
  emits: ['update:templateId', 'update:enabled', 'update:page', 'update:mapping'],
  setup(props, { emit }) {
    return () => h('div', { class: 'config-row' }, [
      h('label', { class: 'check-row' }, [h('input', { type: 'checkbox', checked: props.enabled, onChange: (event: Event) => emit('update:enabled', (event.target as HTMLInputElement).checked) }), props.label]),
      h('input', { value: props.templateId, placeholder: '模板 ID', onInput: (event: Event) => emit('update:templateId', (event.target as HTMLInputElement).value) }),
      h('input', { value: props.page, placeholder: '页面路径，例如 pages/checkin/checkin', onInput: (event: Event) => emit('update:page', (event.target as HTMLInputElement).value) }),
      h('input', { value: props.mapping, placeholder: 'JSON 字段映射，例如 {"thing1":"message"}', onInput: (event: Event) => emit('update:mapping', (event.target as HTMLInputElement).value) }),
    ])
  },
})

const error = ref('')
const success = ref('')
const saving = ref(false)
const form = reactive<SubscriptionConfig>({
  study_start_template_id: '', completion_template_id: '', decision_template_id: '', missed_checkin_template_id: '', group_nudge_template_id: '',
  study_start_enabled: false, completion_enabled: false, decision_enabled: false, missed_checkin_enabled: false, group_nudge_enabled: false,
  study_start_page: '', completion_page: '', decision_page: '', missed_checkin_page: '', group_nudge_page: '',
  study_start_field_mapping: '', completion_field_mapping: '', decision_field_mapping: '', missed_checkin_field_mapping: '', group_nudge_field_mapping: '', recent_status: [],
})

onMounted(async () => {
  try {
    Object.assign(form, await AdminApi.subscriptionConfig())
  } catch (err) {
    error.value = err instanceof Error ? err.message : '订阅配置加载失败'
  }
})

async function save() {
  error.value = ''
  success.value = ''
  saving.value = true
  try {
    Object.assign(form, await AdminApi.saveSubscriptionConfig(form))
    success.value = '订阅消息配置已保存并通过校验'
  } catch (err) {
    error.value = err instanceof Error ? err.message : '订阅配置保存失败'
  } finally {
    saving.value = false
  }
}
</script>
