<template>
  <main class="workspace">
    <section class="page-head">
      <div><p class="eyebrow">风险巡检</p><h2>异常记录</h2></div>
      <button class="ghost small-button" type="button" @click="load">刷新</button>
    </section>
    <p v-if="error" class="error">{{ error }}</p>

    <section class="panel-grid">
      <article class="panel">
        <h3>异常任务</h3>
        <table class="data-table">
          <thead><tr><th>ID</th><th>日期</th><th>任务</th><th>分钟</th><th>原因</th></tr></thead>
          <tbody>
            <tr v-for="task in data.tasks" :key="task.id">
              <td>#{{ task.id }}</td><td>{{ task.date }}</td><td>{{ task.title }}</td><td>{{ task.study_minutes }}</td><td>{{ task.suspicious_reason || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </article>

      <article class="panel">
        <h3>异常学习会话</h3>
        <table class="data-table">
          <thead><tr><th>ID</th><th>任务</th><th>分钟</th><th>备注</th></tr></thead>
          <tbody>
            <tr v-for="session in data.sessions" :key="session.id">
              <td>#{{ session.id }}</td><td>#{{ session.task_id }}</td><td>{{ session.duration_min }}</td><td>{{ session.review_note || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </article>
    </section>
  </main>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'

import { AdminApi, type SuspiciousRecordsResp } from '@/api'

const error = ref('')
const data = reactive<SuspiciousRecordsResp>({ tasks: [], sessions: [] })

async function load() {
  error.value = ''
  try {
    const res = await AdminApi.suspiciousRecords()
    data.tasks = res.tasks || []
    data.sessions = res.sessions || []
  } catch (err) {
    error.value = err instanceof Error ? err.message : '异常记录加载失败'
  }
}

onMounted(load)
</script>
