<template>
  <main class="workspace">
    <section class="page-head">
      <div><p class="eyebrow">Risk</p><h2>Suspicious Records</h2></div>
      <button class="ghost small-button" type="button" @click="load">Refresh</button>
    </section>
    <p v-if="error" class="error">{{ error }}</p>

    <section class="panel-grid">
      <article class="panel">
        <h3>Flagged tasks</h3>
        <table class="data-table">
          <thead><tr><th>ID</th><th>Date</th><th>Title</th><th>Minutes</th><th>Reason</th></tr></thead>
          <tbody>
            <tr v-for="task in data.tasks" :key="task.id">
              <td>#{{ task.id }}</td><td>{{ task.date }}</td><td>{{ task.title }}</td><td>{{ task.study_minutes }}</td><td>{{ task.suspicious_reason || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </article>

      <article class="panel">
        <h3>Flagged sessions</h3>
        <table class="data-table">
          <thead><tr><th>ID</th><th>Task</th><th>Minutes</th><th>Note</th></tr></thead>
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
    error.value = err instanceof Error ? err.message : 'Failed to load suspicious records'
  }
}

onMounted(load)
</script>
