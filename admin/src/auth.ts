import { reactive } from 'vue'

import type { AdminUser } from './api'

const tokenKey = 'study_plan_admin_token'
const userKey = 'study_plan_admin_user'

function readUser(): AdminUser | null {
  const raw = localStorage.getItem(userKey)
  if (!raw) return null
  try {
    return JSON.parse(raw) as AdminUser
  } catch {
    localStorage.removeItem(userKey)
    return null
  }
}

export const authSession = reactive({
  token: localStorage.getItem(tokenKey) || '',
  user: readUser(),
  get isAdmin() {
    return this.user?.role === 'admin' && !!this.token
  },
})

export function saveSession(token: string, user: AdminUser) {
  authSession.token = token
  authSession.user = user
  localStorage.setItem(tokenKey, token)
  localStorage.setItem(userKey, JSON.stringify(user))
}

export function clearSession() {
  authSession.token = ''
  authSession.user = null
  localStorage.removeItem(tokenKey)
  localStorage.removeItem(userKey)
}
