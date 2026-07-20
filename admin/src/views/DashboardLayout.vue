<template>
  <div class="shell">
    <aside class="sidebar">
      <div>
        <p class="eyebrow">Study Plan</p>
        <h1>Admin Console</h1>
      </div>
      <nav class="nav-list">
        <router-link to="/">Overview</router-link>
        <router-link to="/users">Users</router-link>
        <router-link to="/slack-config">Slack Config</router-link>
        <router-link to="/suspicious-records">Suspicious Records</router-link>
        <router-link to="/ai-config">AI Config</router-link>
        <router-link to="/subscription-config">Subscriptions</router-link>
        <router-link to="/ops-content">Ops Content</router-link>
        <router-link to="/feedback">Feedback</router-link>
        <router-link to="/audit-logs">Audit Logs</router-link>
      </nav>
      <button class="ghost" type="button" @click="logout">Sign out</button>
    </aside>

    <section class="content">
      <header class="topbar">
        <div>
          <p class="eyebrow">Signed in</p>
          <strong>{{ authSession.user?.nickname || authSession.user?.openid || 'Admin' }}</strong>
        </div>
      </header>
      <router-view />
    </section>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

import { authSession, clearSession } from '@/auth'

const router = useRouter()

function logout() {
  clearSession()
  router.replace('/login')
}
</script>
