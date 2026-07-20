<template>
  <div class="shell">
    <aside class="sidebar">
      <div>
        <p class="eyebrow">Study Plan</p>
        <h1>Admin Console</h1>
      </div>
      <nav class="nav-list">
        <router-link to="/">Overview</router-link>
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
