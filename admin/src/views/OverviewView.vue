<template>
  <main class="workspace">
    <section class="page-head">
      <div>
        <p class="eyebrow">Operations</p>
        <h2>Overview</h2>
      </div>
    </section>

    <div class="metric-grid">
      <article class="metric-card" v-for="item in metrics" :key="item.label">
        <span>{{ item.label }}</span>
        <strong>{{ item.value }}</strong>
      </article>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import { AdminApi, type OverviewMetrics } from '@/api'

const data = ref<OverviewMetrics>({ users: 0, active_plans: 0, checkins_today: 0, banned_users: 0 })
const error = ref('')
const metrics = computed(() => [
  { label: 'Users', value: data.value.users },
  { label: 'Active plans', value: data.value.active_plans },
  { label: 'Check-ins today', value: data.value.checkins_today },
  { label: 'Banned users', value: data.value.banned_users },
])

onMounted(async () => {
  try {
    data.value = await AdminApi.overview()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load overview'
  }
})
</script>
