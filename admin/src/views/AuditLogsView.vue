<template>
  <main class="workspace">
    <section class="page-head">
      <div><p class="eyebrow">安全审计</p><h2>审计日志</h2></div>
      <button class="ghost small-button" type="button" @click="load">刷新</button>
    </section>
    <p v-if="error" class="error">{{ error }}</p>
    <table class="data-table">
      <thead><tr><th>时间</th><th>管理员</th><th>对象</th><th>操作</th><th>原因</th></tr></thead>
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
    error.value = err instanceof Error ? err.message : '审计日志加载失败'
  }
}

onMounted(load)
</script>
