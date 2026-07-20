<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">Rewards</p><h2>Slack Configuration</h2></div></section>
    <p v-if="error" class="error">{{ error }}</p>
    <section class="panel-grid">
      <form class="panel" @submit.prevent="saveGlobal">
        <h3>Global default</h3>
        <NumberField label="Check-in minutes" v-model="global.checkin_minutes" />
        <NumberField label="Makeup cost ratio" v-model="global.makeup_cost_ratio" step="0.1" />
        <NumberField label="Streak bonus" v-model="global.streak_bonus" />
        <NumberField label="Quality bonus" v-model="global.quality_bonus" />
        <button class="primary small-button">Save global</button>
      </form>
      <form class="panel" @submit.prevent="saveUser">
        <h3>Per-user override</h3>
        <label class="field"><span>User ID</span><input v-model.number="targetUserId" type="number" min="1" /></label>
        <NumberField label="Check-in minutes" v-model="userConfig.checkin_minutes" />
        <NumberField label="Makeup cost ratio" v-model="userConfig.makeup_cost_ratio" step="0.1" />
        <NumberField label="Streak bonus" v-model="userConfig.streak_bonus" />
        <NumberField label="Quality bonus" v-model="userConfig.quality_bonus" />
        <button class="primary small-button">Save user</button>
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
    error.value = err instanceof Error ? err.message : 'Failed to load slack config'
  }
})

async function saveGlobal() {
  await AdminApi.saveGlobalSlackConfig(global)
}

async function saveUser() {
  if (!targetUserId.value) {
    error.value = 'User ID is required'
    return
  }
  await AdminApi.saveUserSlackConfig(targetUserId.value, userConfig)
}
</script>
