<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">Operations</p><h2>Ops Content</h2></div></section>
    <p v-if="error" class="error">{{ error }}</p>
    <section class="panel-grid">
      <form class="panel" v-for="item in rows" :key="item.kind" @submit.prevent="save(item)">
        <h3>{{ item.kind }}</h3>
        <label class="field"><span>Title</span><input v-model="item.title" /></label>
        <label class="field"><span>Body</span><textarea v-model="item.body" rows="8" /></label>
        <button class="primary small-button">Save</button>
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
  catch (err) { error.value = err instanceof Error ? err.message : 'Failed to load content' }
}

async function save(item: OpsContent) {
  await AdminApi.saveOpsContent(item.kind, { title: item.title, body: item.body })
}

onMounted(load)
</script>

<style scoped>
textarea { width: 100%; padding: 10px 12px; border: 1px solid #cfd7e3; border-radius: 6px; color: #172033; background: #fbfcfe; font: inherit; resize: vertical; }
</style>
