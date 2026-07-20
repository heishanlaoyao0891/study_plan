<template>
  <main class="workspace">
    <section class="page-head">
      <div>
        <p class="eyebrow">Directory</p>
        <h2>Users</h2>
      </div>
      <button class="ghost small-button" type="button" @click="load">Refresh</button>
    </section>

    <div class="toolbar">
      <input v-model.trim="search" placeholder="Search nickname or openid" @keyup.enter="load" />
      <select v-model="status" @change="load">
        <option value="">All status</option>
        <option value="active">Active</option>
        <option value="banned">Banned</option>
      </select>
      <button class="primary small-button" type="button" @click="load">Apply</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
    <table class="data-table">
      <thead>
        <tr><th>ID</th><th>Name</th><th>Role</th><th>Status</th><th>Slack</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="user in users" :key="user.id">
          <td>#{{ user.id }}</td>
          <td>{{ user.nickname || user.openid || '-' }}</td>
          <td><span class="pill">{{ user.role }}</span></td>
          <td>{{ user.banned_until ? 'Banned' : 'Active' }}</td>
          <td>{{ user.slack_balance || 0 }}</td>
          <td><router-link class="link" :to="`/users/${user.id}`">Open</router-link></td>
        </tr>
      </tbody>
    </table>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'

import { AdminApi, type AdminUser } from '@/api'

const users = ref<AdminUser[]>([])
const search = ref('')
const status = ref('')
const error = ref('')

async function load() {
  error.value = ''
  try {
    const res = await AdminApi.users({ page: 1, size: 50, search: search.value, status: status.value })
    users.value = res.users
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load users'
  }
}

onMounted(load)
</script>
