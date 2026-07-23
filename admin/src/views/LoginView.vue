<template>
  <main class="login-page">
    <form class="login-panel" @submit.prevent="submit">
      <div>
        <p class="eyebrow">学习打卡运营台</p>
        <h1>管理员登录</h1>
      </div>

      <label class="field">
        <span>账号</span>
        <input v-model.trim="username" autocomplete="username" required />
      </label>

      <label class="field">
        <span>密码</span>
        <input v-model="password" autocomplete="current-password" required type="password" />
      </label>

      <p v-if="error" class="error">{{ error }}</p>
      <button class="primary" :disabled="loading" type="submit">
        {{ loading ? '登录中...' : '登录' }}
      </button>
    </form>
  </main>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { AdminAuthApi } from '@/api'
import { saveSession } from '@/auth'

const route = useRoute()
const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const session = await AdminAuthApi.login(username.value, password.value)
    saveSession(session.token, session.user)
    await router.replace(String(route.query.redirect || '/'))
  } catch (err) {
    error.value = err instanceof Error ? err.message : '登录失败'
  } finally {
    loading.value = false
  }
}
</script>
