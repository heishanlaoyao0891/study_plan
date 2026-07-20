<template>
  <main class="workspace">
    <section class="page-head">
      <div><p class="eyebrow">Operations</p><h2>Feedback</h2></div>
      <button class="ghost small-button" type="button" @click="load">Refresh</button>
    </section>
    <p v-if="error" class="error">{{ error }}</p>
    <table class="data-table">
      <thead><tr><th>Time</th><th>User</th><th>Category</th><th>Content</th><th>Contact</th><th>Status</th></tr></thead>
      <tbody>
        <tr v-for="row in rows" :key="row.id">
          <td>{{ row.created_at }}</td><td>#{{ row.user_id }}</td><td>{{ row.category }}</td><td>{{ row.content }}</td><td>{{ row.contact || '-' }}</td><td>{{ row.status }}</td>
        </tr>
      </tbody>
    </table>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { AdminApi, type FeedbackReport } from '@/api'

const rows = ref<FeedbackReport[]>([])
const error = ref('')

async function load() {
  error.value = ''
  try { rows.value = await AdminApi.feedback() }
  catch (err) { error.value = err instanceof Error ? err.message : 'Failed to load feedback' }
}

onMounted(load)
</script>
