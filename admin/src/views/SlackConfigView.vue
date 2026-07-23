<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">奖励规则</p><h2>躺平币配置</h2></div></section>
    <p v-if="error" class="error">{{ error }}</p>
    <section class="panel-grid">
      <form class="panel" @submit.prevent="saveGlobal">
        <h3>全局默认规则</h3>
        <NumberField label="打卡奖励分钟" v-model="global.checkin_minutes" />
        <NumberField label="补录消耗倍率" v-model="global.makeup_cost_ratio" step="0.1" />
        <NumberField label="连续打卡奖励" v-model="global.streak_bonus" />
        <NumberField label="质量奖励" v-model="global.quality_bonus" />
        <button class="primary small-button">保存全局规则</button>
      </form>
      <form class="panel" @submit.prevent="saveUser">
        <h3>单用户覆盖规则</h3>
        <label class="field"><span>用户 ID</span><input v-model.number="targetUserId" type="number" min="1" /></label>
        <NumberField label="打卡奖励分钟" v-model="userConfig.checkin_minutes" />
        <NumberField label="补录消耗倍率" v-model="userConfig.makeup_cost_ratio" step="0.1" />
        <NumberField label="连续打卡奖励" v-model="userConfig.streak_bonus" />
        <NumberField label="质量奖励" v-model="userConfig.quality_bonus" />
        <button class="primary small-button">保存用户规则</button>
      </form>
    </section>
  </main>
</template>

<script setup lang="ts">
import { defineComponent, h, onMounted, reactive, ref } from 'vue'

import { AdminApi } from '@/api'

const NumberField = defineComponent({
  props: { label: { type: String, required: true }, modelValue: { type: Number, required: true }, step: { type: String, default: '1' } },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('label', { class: 'field' }, [
      h('span', props.label),
      h('input', { type: 'number', min: '0', step: props.step, value: props.modelValue, onInput: (event: Event) => emit('update:modelValue', Number((event.target as HTMLInputElement).value)) }),
    ])
  },
})

const error = ref('')
const targetUserId = ref<number | null>(null)
const global = reactive({ checkin_minutes: 10, makeup_cost_ratio: 1, streak_bonus: 0, quality_bonus: 0 })
const userConfig = reactive({ checkin_minutes: 10, makeup_cost_ratio: 1, streak_bonus: 0, quality_bonus: 0 })

onMounted(async () => {
  try {
    const configs = await AdminApi.slackConfigs()
    const current = configs.find(item => !item.user_id)
    if (current) Object.assign(global, current)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '躺平币配置加载失败'
  }
})

async function saveGlobal() {
  await AdminApi.saveGlobalSlackConfig(global)
}

async function saveUser() {
  if (!targetUserId.value) {
    error.value = '请填写用户 ID'
    return
  }
  await AdminApi.saveUserSlackConfig(targetUserId.value, userConfig)
}
</script>
