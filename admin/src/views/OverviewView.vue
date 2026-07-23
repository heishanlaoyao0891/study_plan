<template>
  <main class="workspace">
    <section class="page-head">
      <div>
        <p class="eyebrow">今日运营概览</p>
        <h2>总览仪表盘</h2>
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
  { label: '注册用户', value: data.value.users },
  { label: '进行中计划', value: data.value.active_plans },
  { label: '今日打卡', value: data.value.checkins_today },
  { label: '封禁用户', value: data.value.banned_users },
])

onMounted(async () => {
  try {
    data.value = await AdminApi.overview()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '总览加载失败'
  }
})
</script>
