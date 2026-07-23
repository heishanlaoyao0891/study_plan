<template>
  <main class="workspace">
    <section class="page-head">
      <div>
        <p class="eyebrow">用户目录</p>
        <h2>用户管理</h2>
      </div>
      <button class="ghost small-button" type="button" @click="load">刷新</button>
    </section>

    <div class="toolbar">
      <input v-model.trim="search" placeholder="搜索昵称或 openid" @keyup.enter="load" />
      <select v-model="status" @change="load">
        <option value="">全部状态</option>
        <option value="active">正常</option>
        <option value="banned">已封禁</option>
      </select>
      <button class="primary small-button" type="button" @click="load">筛选</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
    <table class="data-table">
      <thead>
        <tr><th>ID</th><th>用户</th><th>角色</th><th>状态</th><th>躺平分钟</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="user in users" :key="user.id">
          <td>#{{ user.id }}</td>
          <td>{{ user.nickname || user.openid || '-' }}</td>
          <td><span class="pill">{{ user.role === 'admin' ? '管理员' : '用户' }}</span></td>
          <td>{{ user.banned_until ? '已封禁' : '正常' }}</td>
          <td>{{ user.slack_balance || 0 }}</td>
          <td><router-link class="link" :to="`/users/${user.id}`">查看</router-link></td>
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
    error.value = err instanceof Error ? err.message : '用户列表加载失败'
  }
}

onMounted(load)
</script>
