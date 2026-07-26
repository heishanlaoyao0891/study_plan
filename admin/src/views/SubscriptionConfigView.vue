<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">微信提醒</p><h2>订阅消息</h2><p>配置微信公众平台已选用的订阅模板。启用后，小程序用户可在“提醒设置”主动授权；每次授权通常只能发送一次。</p></div></section>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="success" class="success">{{ success }}</p>
    <section class="panel guide-panel">
      <div class="guide-head"><div><p class="eyebrow">配置参照</p><h3>模板字段怎么填写</h3></div><button class="ghost small-button" type="button" @click="showGuide = !showGuide">{{ showGuide ? '收起示例' : '展开示例' }}</button></div>
      <div v-if="showGuide" class="guide-body">
        <ol class="steps">
          <li>在微信公众平台进入“小程序 → 订阅消息 → 公共模板库”，选用模板并复制模板 ID。</li>
          <li>查看模板详情中的字段名，例如 <code>thing1</code>、<code>date2</code>、<code>time3</code>，左侧字段名必须与微信模板完全一致。</li>
          <li>填写小程序页面路径，不要带开头斜杠，例如 <code>pages/checkin/checkin</code>。</li>
          <li>保存并启用后，让用户在小程序“提醒设置”找到对应提醒，点击“授权此提醒”并接受微信授权。</li>
        </ol>
        <div class="source-grid">
          <div><code>message</code><span>提醒文案</span></div><div><code>title</code><span>任务标题</span></div><div><code>date</code><span>任务日期</span></div>
          <div><code>planned_start</code><span>计划开始时间</span></div><div><code>planned_end</code><span>计划结束时间</span></div><div><code>sender</code><span>督学成员昵称</span></div>
        </div>
        <p class="guide-note">映射 JSON 的左侧是微信模板字段，右侧是上方系统数据源。字段名称取决于你在微信后台选用的模板，下面仅为格式示例。</p>
        <table class="data-table example-table">
          <thead><tr><th>提醒</th><th>推荐页面</th><th>字段映射示例</th><th>触发条件</th></tr></thead>
          <tbody>
            <tr v-for="example in examples" :key="example.label"><td>{{ example.label }}</td><td><code>{{ example.page }}</code></td><td><code>{{ example.mapping }}</code></td><td>{{ example.trigger }}</td></tr>
          </tbody>
        </table>
        <div class="flow"><strong>生效链路</strong><span>保存启用</span><b>→</b><span>小程序展示模板</span><b>→</b><span>用户微信授权</span><b>→</b><span>到期或督学触发</span><b>→</b><span>发送并记录结果</span></div>
      </div>
    </section>
    <form class="panel wide-panel" @submit.prevent="save">
      <ConfigRow label="学习开始提醒" v-model:template-id="form.study_start_template_id" v-model:enabled="form.study_start_enabled" v-model:page="form.study_start_page" v-model:mapping="form.study_start_field_mapping" />
      <ConfigRow label="完成提醒" v-model:template-id="form.completion_template_id" v-model:enabled="form.completion_enabled" v-model:page="form.completion_page" v-model:mapping="form.completion_field_mapping" />
      <ConfigRow label="23:30 决策提醒" v-model:template-id="form.decision_template_id" v-model:enabled="form.decision_enabled" v-model:page="form.decision_page" v-model:mapping="form.decision_field_mapping" />
      <ConfigRow label="未打卡提醒" v-model:template-id="form.missed_checkin_template_id" v-model:enabled="form.missed_checkin_enabled" v-model:page="form.missed_checkin_page" v-model:mapping="form.missed_checkin_field_mapping" />
      <ConfigRow label="小组督学提醒" v-model:template-id="form.group_nudge_template_id" v-model:enabled="form.group_nudge_enabled" v-model:page="form.group_nudge_page" v-model:mapping="form.group_nudge_field_mapping" />
			<ConfigRow label="躺平币余额提醒" v-model:template-id="form.slack_balance_template_id" v-model:enabled="form.slack_balance_enabled" v-model:page="form.slack_balance_page" v-model:mapping="form.slack_balance_field_mapping" />
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
const showGuide = ref(true)
const examples = [
  { label: '学习开始提醒', page: 'pages/checkin/checkin', mapping: '{"thing1":"title","date2":"date","time3":"planned_start"}', trigger: '任务计划开始时间到达' },
  { label: '完成提醒', page: 'pages/checkin/checkin', mapping: '{"thing1":"title","time2":"planned_end","thing3":"message"}', trigger: '进行中任务到达计划结束时间' },
  { label: '23:30 决策提醒', page: 'pages/checkin/checkin', mapping: '{"thing1":"title","thing2":"message"}', trigger: '23:30 仍有学习中或待处理任务' },
  { label: '未打卡提醒', page: 'pages/checkin/checkin', mapping: '{"thing1":"title","date2":"date","thing3":"message"}', trigger: '任务结束时间到达且当天未打卡' },
  { label: '小组督学提醒', page: 'pages/group/group', mapping: '{"name1":"sender","thing2":"message"}', trigger: '小组成员主动发起督学' },
	{ label: '躺平币余额提醒', page: 'pages/slack/slack', mapping: '{"thing1":"message"}', trigger: '余额从 10 分钟以上降至 10 分钟以内' },
]
const form = reactive<SubscriptionConfig>({
  study_start_template_id: '', completion_template_id: '', decision_template_id: '', missed_checkin_template_id: '', group_nudge_template_id: '', slack_balance_template_id: '',
  study_start_enabled: false, completion_enabled: false, decision_enabled: false, missed_checkin_enabled: false, group_nudge_enabled: false, slack_balance_enabled: false,
  study_start_page: '', completion_page: '', decision_page: '', missed_checkin_page: '', group_nudge_page: '', slack_balance_page: '',
  study_start_field_mapping: '', completion_field_mapping: '', decision_field_mapping: '', missed_checkin_field_mapping: '', group_nudge_field_mapping: '', slack_balance_field_mapping: '', recent_status: [],
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

<style scoped>
.guide-panel { margin-bottom: 20px; }
.guide-head { display: flex; align-items: center; justify-content: space-between; gap: 20px; }
.guide-head h3 { margin: 2px 0 0; }
.guide-body { margin-top: 20px; }
.steps { display: grid; gap: 8px; margin: 0; padding-left: 22px; color: #4b5563; line-height: 1.65; }
.source-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; margin-top: 18px; }
.source-grid div { display: flex; flex-direction: column; gap: 5px; padding: 12px 14px; border: 1px solid #e6e9ef; border-radius: 10px; background: #f8fafc; }
.source-grid span, .guide-note { color: #687386; font-size: 13px; }
.guide-note { margin: 16px 0 10px; }
.example-table code { white-space: normal; word-break: break-all; }
.flow { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; margin-top: 18px; padding: 14px; border-radius: 10px; background: #eef5ff; color: #31527d; font-size: 13px; }
.flow strong { color: #163a67; }
.flow b { color: #8aa0bc; }
code { color: #174a7e; font-family: Consolas, monospace; }
@media (max-width: 760px) { .source-grid { grid-template-columns: 1fr; } .guide-head { align-items: flex-start; } }
</style>
