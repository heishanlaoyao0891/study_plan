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
				<tr><th>ID</th><th>用户</th><th>OpenID</th><th>最近登录</th><th>角色</th><th>状态</th><th>躺平分钟</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="user in users" :key="user.id">
          <td>#{{ user.id }}</td>
          <td>{{ user.nickname || user.openid || '-' }}</td>
					<td><code>{{ user.openid || '-' }}</code></td>
					<td><span>{{ formatDate(user.last_login_at) }}</span><small v-if="user.last_login_method">{{ loginMethod(user.last_login_method) }}</small></td>
          <td><span class="pill">{{ user.role === 'admin' ? '管理员' : '用户' }}</span></td>
					<td>{{ userStatus(user) }}</td>
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
function formatDate(value?: string){if(!value)return '从未登录';const date=new Date(value);return Number.isNaN(date.getTime())?value:date.toLocaleString('zh-CN',{hour12:false})}
function loginMethod(value:string){const labels:Record<string,string>={h5_password:'H5 密码',h5_register:'H5 注册',wechat:'微信',wechat_register:'微信注册',wechat_link:'微信绑定'};return labels[value]||value}
function userStatus(user:AdminUser){if(user.account_status==='deleted')return '已删除';if(user.account_status==='inactive')return '已停用';if(user.banned_until&&new Date(user.banned_until).getTime()>Date.now())return '已封禁';return '正常'}
</script>

<style scoped>
td code{display:block;max-width:210px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}td small{display:block;margin-top:4px;color:#7a8790}
</style>
