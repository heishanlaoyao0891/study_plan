<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">内容运营</p><h2>运营内容</h2></div></section>
    <p v-if="error" class="error">{{ error }}</p>
    <section class="panel-grid">
      <form class="panel" v-for="item in rows" :key="item.kind" @submit.prevent="save(item)">
        <h3>{{ kindLabel(item.kind) }}</h3>
        <label class="field"><span>标题</span><input v-model="item.title" /></label>
        <label class="field"><span>正文</span><textarea v-model="item.body" rows="8" /></label>
        <button class="primary small-button">保存</button>
      </form>
    </section>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { AdminApi, type OpsContent } from '@/api'

const rows = ref<OpsContent[]>([])
const error = ref('')

async function load() {
  error.value = ''
  try { rows.value = await AdminApi.opsContents() }
  catch (err) { error.value = err instanceof Error ? err.message : '运营内容加载失败' }
}

async function save(item: OpsContent) {
  await AdminApi.saveOpsContent(item.kind, { title: item.title, body: item.body })
}

function kindLabel(kind: string) {
  const labels: Record<string, string> = { privacy: '隐私政策', agreement: '用户协议', announcement: '公告', version: '版本说明' }
  return labels[kind] || kind
}

onMounted(load)
</script>

<style scoped>
textarea { width: 100%; padding: 12px 14px; border: 1px solid #f1ccd7; border-radius: 14px; color: #2f2430; background: #fffafd; font: inherit; resize: vertical; }
</style>
