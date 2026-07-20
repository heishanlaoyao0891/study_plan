<template>
  <main class="workspace">
    <section class="page-head">
      <div><p class="eyebrow">Security</p><h2>Audit Logs</h2></div>
      <button class="ghost small-button" type="button" @click="load">Refresh</button>
    </section>
    <p v-if="error" class="error">{{ error }}</p>
    <table class="data-table">
      <thead><tr><th>Time</th><th>Admin</th><th>Target</th><th>Action</th><th>Reason</th></tr></thead>
      <tbody>
        <tr v-for="log in logs" :key="log.id">
          <td>{{ log.created_at }}</td><td>#{{ log.admin_user_id }}</td><td>{{ log.target_user_id ? `#${log.target_user_id}` : '-' }}</td><td>{{ log.action_type }}</td><td>{{ log.reason || '-' }}</td>
        </tr>
      </tbody>
    </table>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'

import { AdminApi, type AuditLog } from '@/api'

const logs = ref<AuditLog[]>([])
const error = ref('')

async function load() {
  error.value = ''
  try {
    logs.value = (await AdminApi.auditLogs()).logs
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load audit logs'
  }
}

onMounted(load)
</script>
