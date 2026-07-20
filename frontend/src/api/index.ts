// 业务接口封装：用户/计划/打卡
import { api } from './request'

// ---------- 类型 ----------
export interface User {
  id: number
  openid: string
  nickname: string
  avatar_url: string
  phone_number?: string
  phone_verified_at?: string
  weekly_hours: number
  slack_balance: number
  role: 'user' | 'admin'
  banned_until?: string
  banned_reason?: string
  account_status?: 'active' | 'inactive' | 'deleted'
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
  public_to_group: boolean
  ai_generated: boolean
  is_shared: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface StudyGroup {
  id: number
  name: string
  leader_user_id: number
  status: 'active' | 'ended'
  end_date?: string
  ended_at?: string
}

export interface StudyGroupMember {
  id: number
  group_id: number
  user_id: number
  role: 'leader' | 'member'
  status: 'active' | 'left' | 'removed'
  joined_at: string
  nickname?: string
  avatar_url?: string
}

export interface GroupMemberView {
  user_id: number
  nickname: string
  avatar_url?: string
  role: 'leader' | 'member'
  level: number
  streak: number
  study_minutes: number
  completion_rate: number
  today_completed: boolean
}

export interface CreatePlanReq {
  title: string
  description?: string
  weekly_target_hours?: number
  start_date?: string
  end_date?: string
  public_to_group?: boolean
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
  planned_start?: string
  planned_end?: string
  estimated_minutes?: number
  needs_decision?: boolean
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
  me() {
    return api.get<User>('/api/auth/me')
  },
  bindPhone(code: string, phone_number = '') {
    return api.post<User>('/api/auth/phone', { code, phone_number })
  },
  updateAvatar(avatar_url: string) {
    return api.put<User>('/api/auth/avatar', { avatar_url })
  },
  deactivate(retain: boolean, note = '') {
    return api.post<User>('/api/auth/deactivate', { retain, note })
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
  shift(id: number, days: number, start_date = '') {
    return api.put<Plan>(`/api/plans/${id}/shift`, { days, start_date })
  },
  invite(id: number, user_id: number) {
    return api.post<any>(`/api/plans/${id}/invite`, { user_id })
  },
  tasks(id: number) {
    return api.get<any[]>(`/api/plans/${id}/tasks`)
  },
  createTask(id: number, data: { date: string; title: string; description?: string; sort_order?: number }) {
    return api.post<any>(`/api/plans/${id}/tasks`, data)
  },
  reorderTasks(id: number, task_ids: number[]) {
    return api.put<any>(`/api/plans/${id}/tasks/reorder`, { task_ids })
  },
}

export const GroupApi = {
  current() {
    return api.get<{ group: StudyGroup | null; member: StudyGroupMember | null }>('/api/groups/current')
  },
  history() {
    return api.get<StudyGroup[]>('/api/groups/history')
  },
  create(data: { name: string; end_date?: string }) {
    return api.post<{ group: StudyGroup; member: StudyGroupMember }>('/api/groups', data)
  },
  update(id: number, data: { name?: string; end_date?: string }) {
    return api.put<StudyGroup>(`/api/groups/${id}`, data)
  },
  leave(id: number) {
    return api.post<any>(`/api/groups/${id}/leave`, {})
  },
  end(id: number) {
    return api.post<any>(`/api/groups/${id}/end`, {})
  },
  transfer(id: number, user_id: number) {
    return api.post<any>(`/api/groups/${id}/transfer`, { user_id })
  },
  remove(id: number, userId: number) {
    return api.post<any>(`/api/groups/${id}/members/${userId}/remove`, {})
  },
  nudge(id: number, userId: number) {
    return api.post<any>(`/api/groups/${id}/members/${userId}/nudge`, {})
  },
  join(code: string) {
    return api.post<any>('/api/groups/join', { code })
  },
  invite(id: number, days = 7) {
    return api.post<any>(`/api/groups/${id}/invitations?days=${days}`, {})
  },
  revokeInvite(id: number) {
    return api.post<any>(`/api/groups/${id}/invitations/revoke`, {})
  },
  members() {
    return api.get<GroupMemberView[]>('/api/groups/current/members')
  },
  leaderboard(scope: 'weekly' | 'all' = 'weekly') {
    return api.get<{ scope: string; rows: GroupMemberView[] }>(`/api/groups/current/leaderboard?scope=${scope}`)
  },
}

export const RecoveryApi = {
  preview() {
    return api.get<{ mode: string; missed_days: number; overdue_tasks: number; pending_decisions: number; actions: Array<{ task_id: number; title: string; old_date: string; new_date: string }> }>('/api/recovery/preview')
  },
  apply(actions: Array<{ task_id: number; title: string; old_date: string; new_date: string }>) {
    return api.post<{ applied: number }>('/api/recovery/apply', { actions })
  },
}

export const OpsApi = {
  content(kind: 'privacy' | 'agreement' | 'announcement' | 'version') {
    return api.get<{ kind: string; title: string; body: string; updated_at: string }>(`/api/ops/content/${kind}`)
  },
  feedback(data: { category?: string; content: string; contact?: string }) {
    return api.post<any>('/api/feedback', data)
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
  get(id: number) {
    return api.get<any>(`/api/tasks/${id}`)
  },
  start(id: number) {
    return api.put<any>(`/api/tasks/${id}/start`)
  },
  stop(id: number) {
    return api.put<any>(`/api/tasks/${id}/stop`)
  },
  complete(id: number) {
    return api.put<any>(`/api/tasks/${id}/complete`)
  },
  update(id: number, data: any) {
    return api.put<any>(`/api/tasks/${id}`, data)
  },
  remove(id: number) {
    return api.delete<any>(`/api/tasks/${id}`)
  },
  postpone(id: number, date: string, reason = '', planned_start = '', planned_end = '', confirm_conflict = false) {
    return api.put<any>(`/api/tasks/${id}/postpone`, { date, reason, planned_start, planned_end, confirm_conflict })
  },
  makeup(id: number, actual_end: string, reason = '', actual_start = '') {
    return api.put<any>(`/api/tasks/${id}/makeup`, { actual_start, actual_end, reason })
  },
  pendingDecision(date?: string) {
    return api.get<any[]>(`/api/tasks/pending-decision${date ? `?date=${date}` : ''}`)
  },
  compensateMidnight() {
    return api.post<any>('/api/tasks/midnight-compensate', {})
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
  efficiency(days = 30) {
    return api.get<any>(`/api/stats/efficiency?days=${days}`)
  },
}

export const AIApi = {
  generatePlan(data: { goal: string; hours_per_day?: number; days?: number; start_date?: string; skip_dates?: string[]; refinement?: string }) {
    return api.post<any>('/api/ai/generate-plan', data)
  },
  regeneratePlan(data: { goal: string; hours_per_day?: number; days?: number; start_date?: string; skip_dates?: string[]; refinement?: string }) {
    return api.post<any>('/api/ai/regenerate', data)
  },
  commitPlan(preview: any) {
    return api.post<any>('/api/ai/commit-plan', { preview })
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
  due(date?: string) {
    return api.get<any>(`/api/notifications/due${date ? `?date=${date}` : ''}`)
  },
}
