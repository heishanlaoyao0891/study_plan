// 业务接口封装：用户/计划/打卡
import { api } from './request'

// ---------- 类型 ----------
export interface User {
  id: number
  openid: string
  nickname: string
  avatar_url: string
  weekly_hours: number
  slack_balance: number
  role: 'user' | 'admin'
  banned_until?: string
  banned_reason?: string
}

export interface LoginResp {
  token: string
  user: User
}

export interface Plan {
  id: number
  user_id: number
  title: string
  description: string
  status: 'active' | 'paused' | 'archived'
  weekly_target_hours: number
  start_date?: string
  end_date?: string
  ai_generated: boolean
  is_shared: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface CreatePlanReq {
  title: string
  description?: string
  weekly_target_hours?: number
  start_date?: string
  end_date?: string
  confirm_overload?: boolean
}

export interface CheckinInfo {
  plan_id: number
  task_id: number
  title: string
  status: string
  task_status: 'pending' | 'in_progress' | 'completed'
  date: string
  study_minutes: number
  completed: boolean
}

export interface CreateCheckinReq {
  plan_id: number
  date: string
  completed?: boolean | null
}

// ---------- 接口 ----------
export const AuthApi = {
  login(code: string, nickname = '', avatar_url = '') {
    return api.post<LoginResp>('/api/auth/login', { code, nickname, avatar_url }, { auth: false })
  },
}

export const PlanApi = {
  list(status?: string) {
    const q = status ? `?status=${status}` : ''
    return api.get<Plan[]>(`/api/plans${q}`)
  },
  get(id: number) {
    return api.get<Plan>(`/api/plans/${id}`)
  },
  create(data: CreatePlanReq) {
    return api.post<Plan>('/api/plans', data)
  },
  update(id: number, data: Partial<Plan>) {
    return api.put<Plan>(`/api/plans/${id}`, data)
  },
  remove(id: number) {
    return api.delete<{ deleted: number }>(`/api/plans/${id}`)
  },
  pause(id: number) {
    return api.put<Plan>(`/api/plans/${id}/pause`)
  },
  resume(id: number) {
    return api.put<Plan>(`/api/plans/${id}/resume`)
  },
  shift(id: number, days: number) {
    return api.put<Plan>(`/api/plans/${id}/shift`, { days })
  },
}

export const CheckinApi = {
  listByDate(date?: string) {
    const q = date ? `?date=${date}` : ''
    return api.get<CheckinInfo[]>(`/api/checkins${q}`)
  },
  toggle(data: CreateCheckinReq) {
    return api.post<any>(`/api/checkins`, data)
  },
  streak() {
    return api.get<{ streak: number; streak_str: string }>(`/api/checkins/streak`)
  },
}

export const StudyTaskApi = {
  start(id: number) {
    return api.put<any>(`/api/tasks/${id}/start`)
  },
  stop(id: number) {
    return api.put<any>(`/api/tasks/${id}/stop`)
  },
  complete(id: number) {
    return api.put<any>(`/api/tasks/${id}/complete`)
  },
  makeup(id: number, end_time: string, reason = '') {
    return api.put<any>(`/api/tasks/${id}/makeup`, { end_time, reason })
  },
  pendingDecision(date?: string) {
    return api.get<any[]>(`/api/tasks/pending-decision${date ? `?date=${date}` : ''}`)
  },
}

export const SlackApi = {
  balance() {
    return api.get<{ balance: number; unit: string }>('/api/slack/balance')
  },
  start(activity: string) {
    return api.post<any>('/api/slack/start', { activity })
  },
  stop() {
    return api.put<any>('/api/slack/stop')
  },
  records() {
    return api.get<any[]>('/api/slack/records')
  },
}

export const StatsApi = {
  calendar(month?: string) {
    return api.get<any[]>(`/api/stats/calendar${month ? `?month=${month}` : ''}`)
  },
  dailyDistribution(date?: string) {
    return api.get<any[]>(`/api/stats/daily-distribution${date ? `?date=${date}` : ''}`)
  },
  weeklyReport() {
    return api.get<any>('/api/stats/weekly-report')
  },
  monthlyReport() {
    return api.get<any>('/api/stats/monthly-report')
  },
  slackDistribution(month?: string) {
    return api.get<any[]>(`/api/stats/slack-distribution${month ? `?month=${month}` : ''}`)
  },
}

export const AIApi = {
  generatePlan(data: { goal: string; hours_per_day?: number; days?: number; start_date?: string; skip_dates?: string[] }) {
    return api.post<any>('/api/ai/generate-plan', data)
  },
}

export const NotificationApi = {
  list() {
    return api.get<any[]>('/api/notifications/subscriptions')
  },
  subscribe() {
    return api.post<any>('/api/notifications/subscribe', {})
  },
  unsubscribe() {
    return api.delete<any>('/api/notifications/subscribe')
  },
}

export const AdminApi = {
  users() {
    return api.get<any>('/api/admin/users')
  },
  ban(id: number, duration_hours: number, reason: string) {
    return api.post<any>(`/api/admin/users/${id}/ban`, { duration_hours, reason })
  },
  unban(id: number) {
    return api.post<any>(`/api/admin/users/${id}/unban`, {})
  },
  slackConfigs() {
    return api.get<any[]>('/api/admin/slack-config')
  },
  saveSlackConfig(data: { checkin_minutes: number; streak_bonus?: number; quality_bonus?: number }) {
    return api.put<any>('/api/admin/slack-config', data)
  },
}
