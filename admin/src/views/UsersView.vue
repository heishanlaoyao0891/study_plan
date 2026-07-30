<template>
  <main class="workspace">
    <section class="page-head">
      <div>
        <p class="eyebrow">用户目录</p>
        <h2>用户管理</h2>
      </div>
      <div class="page-actions">
        <button class="primary small-button add-user" type="button" @click="openCreate">添加用户</button>
        <button class="ghost small-button" type="button" @click="load">刷新</button>
      </div>
    </section>

    <div class="toolbar">
      <input v-model.trim="search" placeholder="搜索登录名、昵称或 OpenID" @keyup.enter="load" />
      <select v-model="status" @change="load">
        <option value="all">全部展示</option>
        <option value="active">正常</option>
        <option value="banned">已封禁</option>
        <option value="deleted">已删除</option>
      </select>
      <button class="primary small-button" type="button" @click="load">筛选</button>
    </div>

    <section v-if="creating || initialPassword" class="panel create-user-panel">
      <div>
        <h3>创建用户</h3>
        <p>填写登录名和昵称后，系统会生成随机初始密码，仅在本次创建成功后显示一次。</p>
      </div>
      <form v-if="!initialPassword" class="create-user-form" @submit.prevent="createUser">
        <label class="field"><span>登录名</span><input v-model.trim="createForm.username" autocomplete="off" maxlength="24" placeholder="微信号或 11 位手机号" /></label>
        <label class="field"><span>昵称</span><input v-model.trim="createForm.nickname" autocomplete="off" maxlength="20" placeholder="用户展示名称" /></label>
        <div class="button-row"><button class="primary small-button" :disabled="submitting" type="submit">{{ submitting ? '创建中...' : '创建并生成密码' }}</button><button class="ghost small-button" :disabled="submitting" type="button" @click="closeCreate">取消</button></div>
      </form>
      <div v-else class="initial-password"><span>初始密码</span><code>{{ initialPassword }}</code><button class="ghost small-button" type="button" @click="copyInitialPassword">复制</button><button class="ghost small-button" type="button" @click="closeCreate">完成</button></div>
    </section>

    <p v-if="error" class="error">{{ error }}</p>
    <table class="data-table">
      <thead>
        <tr><th>ID</th><th>登录名</th><th>昵称</th><th>OpenID</th><th>最近登录</th><th>角色</th><th>状态</th><th>躺平分钟</th><th>操作</th></tr>
      </thead>
      <tbody>
        <tr v-for="user in users" :key="user.id">
          <td>#{{ user.id }}</td>
          <td>{{ user.username || '-' }}</td>
          <td>{{ user.nickname || '-' }}</td>
          <td><code>{{ user.openid || '-' }}</code></td>
					<td><span>{{ formatDate(user.last_login_at) }}</span><small v-if="user.last_login_method">{{ loginMethod(user.last_login_method) }}</small></td>
          <td><span class="pill">{{ user.role === 'admin' ? '管理员' : '用户' }}</span></td>
					<td>{{ userStatus(user) }}</td>
          <td>{{ user.slack_balance || 0 }}</td>
          <td class="actions"><router-link class="link" :to="`/users/${user.id}`">查看</router-link><button v-if="canDelete(user)" class="delete-button" type="button" :disabled="deletingId === user.id" @click="remove(user)">{{ deletingId === user.id ? '删除中...' : '删除' }}</button></td>
        </tr>
      </tbody>
    </table>
  </main>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'

import { AdminApi, type AdminUser } from '@/api'

const users = ref<AdminUser[]>([])
const search = ref('')
const status = ref('active')
const error = ref('')
const deletingId = ref<number | null>(null)
const creating = ref(false)
const submitting = ref(false)
const initialPassword = ref('')
const createForm = reactive({ username: '', nickname: '' })

async function load() {
  error.value = ''
  try {
    const res = await AdminApi.users({ page: 1, size: 50, search: search.value, status: status.value })
    users.value = res.users
  } catch (err) {
    error.value = err instanceof Error ? err.message : '用户列表加载失败'
  }
}

function openCreate() {
  createForm.username = ''
  createForm.nickname = ''
  initialPassword.value = ''
  creating.value = true
  error.value = ''
}

function closeCreate() {
  creating.value = false
  initialPassword.value = ''
}

async function createUser() {
  if (!/^[A-Za-z0-9_]{4,24}$/.test(createForm.username)) {
    error.value = '登录名需为 4-24 位字母、数字或下划线'
    return
  }
  if (createForm.nickname.length < 2 || createForm.nickname.length > 20) {
    error.value = '昵称长度需要为 2-20 个字符'
    return
  }
  submitting.value = true
  error.value = ''
  try {
    const result = await AdminApi.createUser(createForm)
    initialPassword.value = result.initial_password
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '创建用户失败'
  } finally {
    submitting.value = false
  }
}

async function copyInitialPassword() {
  try {
    await navigator.clipboard.writeText(initialPassword.value)
  } catch {
    error.value = '复制失败，请手动记录初始密码'
  }
}

function canDelete(user: AdminUser) {
  return user.role !== 'admin' && user.account_status !== 'deleted'
}

async function remove(user: AdminUser) {
  const label = user.username || user.nickname || `用户 #${user.id}`
  if (!window.confirm(`确认删除 ${label}？该操作会清理学习数据并无法恢复。`)) return
  deletingId.value = user.id
  error.value = ''
  try {
    await AdminApi.deleteUser(user.id)
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除用户失败'
  } finally {
    deletingId.value = null
  }
}

onMounted(load)
function formatDate(value?: string){if(!value)return '从未登录';const date=new Date(value);return Number.isNaN(date.getTime())?value:date.toLocaleString('zh-CN',{hour12:false})}
function loginMethod(value:string){const labels:Record<string,string>={h5_password:'H5 密码',h5_register:'H5 注册',wechat:'微信',wechat_register:'微信注册',wechat_link:'微信绑定'};return labels[value]||value}
function userStatus(user:AdminUser){if(user.account_status==='deleted')return '已删除';if(user.account_status==='inactive')return '已停用';if(user.banned_until&&new Date(user.banned_until).getTime()>Date.now())return '已封禁';return '正常'}
</script>

<style scoped>
.page-actions,.actions,.initial-password{display:flex;align-items:center;gap:10px}.add-user{display:inline-flex;align-items:center}.create-user-panel{display:grid;gap:18px;margin-bottom:18px}.create-user-panel p{margin:7px 0 0;color:#7a6773}.create-user-form{display:grid;grid-template-columns:repeat(2,minmax(0,1fr)) auto;align-items:end;gap:12px}.initial-password{padding:14px;border:1px solid #f3d6d0;border-radius:14px;background:#fff6da}.initial-password span{font-weight:900}.initial-password code{flex:1;overflow-wrap:anywhere;color:#4b3040;font-weight:900}.delete-button{border:0;background:transparent;color:#b42318;cursor:pointer;font-weight:900}.delete-button:disabled{cursor:default;opacity:.55}td code{display:block;max-width:210px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}td small{display:block;margin-top:4px;color:#7a8790}
</style>
