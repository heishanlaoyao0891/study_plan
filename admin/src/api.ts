import { authSession } from './auth'

const apiBase = (import.meta.env.VITE_ADMIN_API_BASE || '').replace(/\/$/, '')

interface ApiEnvelope<T> {
  code: number
  data?: T
  message?: string
}

export interface AdminUser {
  id: number
  openid?: string
  nickname?: string
  avatar_url?: string
  role: 'user' | 'admin'
  banned_until?: string
  banned_reason?: string
}

export interface AdminLoginResp {
  token: string
  user: AdminUser
}

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  headers.set('Content-Type', 'application/json')
  if (authSession.token) {
    headers.set('Authorization', `Bearer ${authSession.token}`)
  }

  const res = await fetch(`${apiBase}${path}`, {
    ...options,
    headers,
  })
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`)
  }

  const body = (await res.json()) as ApiEnvelope<T>
  if (body.code !== 0) {
    throw new Error(body.message || 'Request failed')
  }
  return body.data as T
}

export const AdminAuthApi = {
  login(username: string, password: string) {
    return request<AdminLoginResp>('/api/admin/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
  },
}
