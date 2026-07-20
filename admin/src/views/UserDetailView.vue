<template>
  <main class="workspace">
    <section class="page-head">
      <div>
        <p class="eyebrow">User</p>
        <h2>#{{ userId }}</h2>
      </div>
      <router-link class="link" to="/users">Back to users</router-link>
    </section>

    <p v-if="error" class="error">{{ error }}</p>
    <section v-if="detail" class="panel-grid">
      <article class="panel">
        <h3>{{ detail.user.nickname || detail.user.openid || 'User' }}</h3>
        <dl class="detail-list">
          <dt>Role</dt><dd>{{ detail.user.role }}</dd>
          <dt>Status</dt><dd>{{ detail.user.banned_until ? 'Banned' : 'Active' }}</dd>
          <dt>Plans</dt><dd>{{ detail.plan_count }}</dd>
          <dt>Check-ins</dt><dd>{{ detail.checkin_count }}</dd>
          <dt>Slack balance</dt><dd>{{ detail.slack_balance }}</dd>
          <dt>Ban reason</dt><dd>{{ detail.user.banned_reason || '-' }}</dd>
        </dl>
      </article>

      <article class="panel">
        <h3>Ban controls</h3>
        <label class="field"><span>Duration hours, 0 for permanent</span><input v-model.number="durationHours" type="number" min="0" /></label>
        <label class="field"><span>Reason</span><input v-model.trim="reason" /></label>
        <div class="button-row">
          <button class="primary small-button" type="button" @click="ban">Ban</button>
          <button class="ghost small-button" type="button" @click="unban">Unban</button>
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
    error.value = err instanceof Error ? err.message : 'Failed to load user'
  }
}

async function ban() {
  if (!reason.value) {
    error.value = 'Reason is required'
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
