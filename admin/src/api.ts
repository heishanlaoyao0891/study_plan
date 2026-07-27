import { authSession, clearSession } from './auth'

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
  username?: string
  account_status?: 'active' | 'inactive' | 'deleted'
	last_login_at?: string
	last_login_method?: string
	created_at?: string
  plan_count?: number
  checkin_count?: number
}

export interface DashboardSlice { key: string; label: string; count: number }
export interface DashboardDay { date: string; count?: number; study_minutes?: number; checkin_users?: number }
export interface OverviewMetrics {
  as_of: string
  range: { start: string; end: string; days: number }
  summary: { user_accounts: number; active_users_7d: number; active_plans: number; checkins_today: number; study_minutes_today: number }
  charts: { segments: DashboardSlice[]; registrations: DashboardDay[]; learning_activity: DashboardDay[]; plan_statuses: DashboardSlice[] }
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
  makeup_cost_ratio: number
  streak_bonus: number
  quality_bonus: number
  updated_at?: string
}

export interface AIConfig {
  provider: string
  model_name: string
  base_url: string
  request_timeout_seconds: number
  interactive_target_seconds: number
  background_job_timeout_seconds: number
  daily_generation_limit: number
  enabled: boolean
  api_key_masked?: string
  has_api_key?: boolean
  key_storage?: 'missing' | 'plaintext' | 'encrypted'
  effective_mode?: 'disabled' | 'fallback' | 'ai'
  fallback_enabled?: boolean
}

export interface AIPlanningMetrics {
  window_days: number
  queue_depth: number
  status_counts: Record<string, number>
  success_rate: number
  fallback_rate: number
  p50_latency_ms: number
  p95_latency_ms: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  fallback_reasons: Record<string, number>
  provider_models: Record<string, number>
  provider_attempts: number
  successful_generations: number
  prompt_playbook_patterns: Array<{ pattern_key: string; version: number; count: number; guidance: string }>
}

export interface SubscriptionConfig {
  study_start_template_id: string
  completion_template_id: string
  decision_template_id: string
  missed_checkin_template_id: string
  group_nudge_template_id: string
	slack_balance_template_id: string
  study_start_enabled: boolean
  completion_enabled: boolean
  decision_enabled: boolean
  missed_checkin_enabled: boolean
  group_nudge_enabled: boolean
	slack_balance_enabled: boolean
  study_start_page: string
  completion_page: string
  decision_page: string
  missed_checkin_page: string
  group_nudge_page: string
	slack_balance_page: string
  study_start_field_mapping: string
  completion_field_mapping: string
  decision_field_mapping: string
  missed_checkin_field_mapping: string
  group_nudge_field_mapping: string
	slack_balance_field_mapping: string
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

export interface SuspiciousRecordsResp {
  tasks: Array<{ id: number; title: string; date: string; study_minutes: number; suspicious_reason?: string }>
  sessions: Array<{ id: number; task_id: number; duration_min: number; review_note?: string }>
}

export interface OpsContent {
  id: number
  kind: string
  title: string
  body: string
  updated_at: string
}

export interface FeedbackReport {
  id: number
  user_id: number
  category: string
  content: string
  contact?: string
  user_nickname?: string
  status: 'open' | 'processing' | 'resolved' | 'closed'
  public_response?: string
  responded_at?: string
  created_at: string
  updated_at: string
}

export interface AdminLoginResp {
  token: string
  user: AdminUser
}

export interface RegistrationInvitation {
  id: number
  code_prefix: string
  status: 'active' | 'used' | 'expired' | 'disabled'
  created_at: string
  expires_at: string
  used_at?: string | null
  user_id?: number | null
  used_by_user?: AdminUser | null
  used_by?: AdminUser | null
  used_user?: AdminUser | null
}

export interface InvitationListResp {
  invitations: RegistrationInvitation[]
}

export interface InvitationGenerateResp {
  codes: string[]
  invitations?: RegistrationInvitation[]
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
  const body = (await res.json()) as ApiEnvelope<T>
  if (!res.ok) {
    if (res.status === 401 || res.status === 403) {
      clearSession()
      const loginPath = `${import.meta.env.BASE_URL}login`
      if (window.location.pathname !== loginPath) {
        window.location.replace(`${loginPath}?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`)
      }
      throw new Error('登录已过期，请重新登录')
    }
    throw new Error(body.message || `请求失败：HTTP ${res.status}`)
  }

  if (body.code !== 0) {
    throw new Error(body.message || '请求失败')
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
  createPasswordReset(id: number) {
    return request<{ code: string; expires_at: string }>(`/api/admin/users/${id}/password-reset`, { method: 'POST', body: JSON.stringify({}) })
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
  aiMetrics() {
    return request<AIPlanningMetrics>('/api/admin/ai-metrics')
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
  suspiciousRecords() {
    return request<SuspiciousRecordsResp>('/api/admin/suspicious-records')
  },
  opsContents() {
    return request<OpsContent[]>('/api/admin/ops-contents')
  },
  saveOpsContent(kind: string, data: { title: string; body: string }) {
    return request<OpsContent>(`/api/admin/ops-contents/${kind}`, { method: 'PUT', body: JSON.stringify(data) })
  },
  feedback(params: { category?: string; status?: string } = {}) {
    const query = new URLSearchParams()
    if (params.category) query.set('category', params.category)
    if (params.status) query.set('status', params.status)
    const qs = query.toString()
    return request<FeedbackReport[]>(`/api/admin/feedback${qs ? `?${qs}` : ''}`)
  },
  updateFeedback(id: number, data: { status: FeedbackReport['status']; public_response?: string; clear_public_response?: boolean }) {
    return request<FeedbackReport>(`/api/admin/feedback/${id}`, { method: 'PUT', body: JSON.stringify(data) })
  },
  invitations() {
    return request<InvitationListResp | RegistrationInvitation[]>('/api/admin/invitations')
  },
  generateInvitations(count: number) {
    return request<InvitationGenerateResp | string[]>('/api/admin/invitations', {
      method: 'POST',
      body: JSON.stringify({ count }),
    })
  },
  disableInvitation(id: number) {
    return request<RegistrationInvitation | { disabled: boolean }>(`/api/admin/invitations/${id}`, { method: 'DELETE' })
  },
}
