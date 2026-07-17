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
  title: string
  status: string
  date: string
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