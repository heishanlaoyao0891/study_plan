<template>
  <div class="shell">
    <aside class="sidebar">
      <div>
        <p class="eyebrow">学习打卡</p>
        <h1>运营管理台</h1>
      </div>
      <nav class="nav-list">
        <router-link to="/">运营总览</router-link>
        <router-link to="/users">用户管理</router-link>
        <router-link to="/slack-config">躺平币配置</router-link>
        <router-link to="/suspicious-records">异常记录</router-link>
        <router-link to="/ai-config">AI 配置</router-link>
        <router-link to="/subscription-config">订阅消息</router-link>
        <router-link to="/ops-content">运营内容</router-link>
        <router-link to="/feedback">用户反馈</router-link>
        <router-link to="/audit-logs">审计日志</router-link>
      </nav>
      <button class="ghost" type="button" @click="logout">退出登录</button>
    </aside>

    <section class="content">
      <header class="topbar">
        <div>
          <p class="eyebrow">当前管理员</p>
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
