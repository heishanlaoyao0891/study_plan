<template>
  <main class="workspace">
    <section class="page-head">
      <div>
        <p class="eyebrow">用户详情</p>
        <h2>用户 #{{ userId }}</h2>
      </div>
      <router-link class="link" to="/users">返回用户列表</router-link>
    </section>

    <p v-if="error" class="error">{{ error }}</p>
    <section v-if="detail" class="panel-grid">
      <article class="panel">
        <h3>{{ detail.user.nickname || detail.user.openid || '学习用户' }}</h3>
        <dl class="detail-list">
          <dt>角色</dt><dd>{{ detail.user.role === 'admin' ? '管理员' : '用户' }}</dd>
          <dt>状态</dt><dd>{{ detail.user.banned_until ? '已封禁' : '正常' }}</dd>
          <dt>计划数</dt><dd>{{ detail.plan_count }}</dd>
          <dt>打卡数</dt><dd>{{ detail.checkin_count }}</dd>
          <dt>躺平分钟</dt><dd>{{ detail.slack_balance }}</dd>
          <dt>封禁原因</dt><dd>{{ detail.user.banned_reason || '-' }}</dd>
        </dl>
      </article>

      <article class="panel">
        <h3>封禁控制</h3>
        <label class="field"><span>封禁时长（小时，0 表示永久）</span><input v-model.number="durationHours" type="number" min="0" /></label>
        <label class="field"><span>原因</span><input v-model.trim="reason" /></label>
        <div class="button-row">
          <button class="primary small-button" type="button" @click="ban">封禁</button>
          <button class="ghost small-button" type="button" @click="unban">解除封禁</button>
        </div>
      </article>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'

import { AdminApi, type UserDetailResp } from '@/api'

const route = useRoute()
const userId = computed(() => Number(route.params.id))
const detail = ref<UserDetailResp | null>(null)
const durationHours = ref(24)
const reason = ref('')
const error = ref('')

async function load() {
  error.value = ''
  try {
    detail.value = await AdminApi.user(userId.value)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '用户详情加载失败'
  }
}

async function ban() {
  if (!reason.value) {
    error.value = '请填写封禁原因'
    return
  }
  await AdminApi.banUser(userId.value, { duration_hours: durationHours.value, reason: reason.value })
  await load()
}

async function unban() {
  await AdminApi.unbanUser(userId.value)
  await load()
}

onMounted(load)
</script>
