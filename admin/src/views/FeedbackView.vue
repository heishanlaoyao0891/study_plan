<template>
  <main class="workspace">
    <section class="page-head">
      <div><p class="eyebrow">用户声音</p><h2>反馈收件箱</h2></div>
      <button class="ghost small-button" type="button" @click="load">刷新</button>
    </section>
    <p v-if="error" class="error">{{ error }}</p>
    <table class="data-table">
      <thead><tr><th>时间</th><th>用户</th><th>分类</th><th>内容</th><th>联系方式</th><th>状态</th></tr></thead>
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
  catch (err) { error.value = err instanceof Error ? err.message : '反馈加载失败' }
}

onMounted(load)
</script>
