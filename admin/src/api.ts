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
  slack_balance?: number
  plan_count?: number
  checkin_count?: number
}

export interface OverviewMetrics {
  users: number
  active_plans: number
  checkins_today: number
  banned_users: number
}

export interface UserListResp {
  total: number
  page: number
  size: number
  users: AdminUser[]
}

export interface UserDetailResp {
  user: AdminUser
  plan_count: number
  checkin_count: number
  slack_balance: number
}

export interface SlackConfig {
  id: number
  user_id?: number | null
  checkin_minutes: number
  streak_bonus: number
  quality_bonus: number
  updated_at?: string
}

export interface AIConfig {
  provider: string
  model_name: string
  base_url: string
  request_timeout_seconds: number
  daily_generation_limit: number
  enabled: boolean
  api_key_masked?: string
  has_api_key?: boolean
}

export interface SubscriptionConfig {
  study_start_template_id: string
  completion_template_id: string
  decision_template_id: string
  missed_checkin_template_id: string
  study_start_enabled: boolean
  completion_enabled: boolean
  decision_enabled: boolean
  missed_checkin_enabled: boolean
  recent_status?: Array<{ id: number; reminder_type: string; status: string; message?: string; created_at: string }>
}

export interface AuditLog {
  id: number
  admin_user_id: number
  target_user_id?: number | null
  action_type: string
  reason?: string
  created_at: string
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

export const AdminApi = {
  overview() {
    return request<OverviewMetrics>('/api/admin/overview')
  },
  users(params: { page?: number; size?: number; search?: string; status?: string } = {}) {
    const query = new URLSearchParams()
    if (params.page) query.set('page', String(params.page))
    if (params.size) query.set('size', String(params.size))
    if (params.search) query.set('search', params.search)
    if (params.status) query.set('status', params.status)
    const qs = query.toString()
    return request<UserListResp>(`/api/admin/users${qs ? `?${qs}` : ''}`)
  },
  user(id: number) {
    return request<UserDetailResp>(`/api/admin/users/${id}`)
  },
  banUser(id: number, data: { duration_hours: number; reason: string }) {
    return request<AdminUser>(`/api/admin/users/${id}/ban`, { method: 'POST', body: JSON.stringify(data) })
  },
  unbanUser(id: number) {
    return request<AdminUser>(`/api/admin/users/${id}/unban`, { method: 'POST', body: JSON.stringify({}) })
  },
  slackConfigs() {
    return request<SlackConfig[]>('/api/admin/slack-config')
  },
  saveGlobalSlackConfig(data: Omit<SlackConfig, 'id' | 'user_id' | 'updated_at'>) {
    return request<SlackConfig>('/api/admin/slack-config', { method: 'PUT', body: JSON.stringify(data) })
  },
  saveUserSlackConfig(userId: number, data: Omit<SlackConfig, 'id' | 'user_id' | 'updated_at'>) {
    return request<SlackConfig>(`/api/admin/slack-config/${userId}`, { method: 'PUT', body: JSON.stringify(data) })
  },
  aiConfig() {
    return request<AIConfig>('/api/admin/ai-config')
  },
  saveAIConfig(data: AIConfig & { api_key?: string }) {
    return request<AIConfig>('/api/admin/ai-config', { method: 'PUT', body: JSON.stringify(data) })
  },
  testAIConfig(data: Partial<AIConfig> & { api_key?: string }) {
    return request<{ ok: boolean; message: string }>('/api/admin/ai-config/test', { method: 'POST', body: JSON.stringify(data) })
  },
  subscriptionConfig() {
    return request<SubscriptionConfig>('/api/admin/subscription-config')
  },
  saveSubscriptionConfig(data: SubscriptionConfig) {
    return request<SubscriptionConfig>('/api/admin/subscription-config', { method: 'PUT', body: JSON.stringify(data) })
  },
  auditLogs() {
    return request<{ logs: AuditLog[] }>('/api/admin/audit-logs')
  },
}
