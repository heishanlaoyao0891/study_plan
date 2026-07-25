// 业务接口封装：用户/计划/打卡
import { api } from './request'

// ---------- 类型 ----------
export interface User {
  id: number
  nickname: string
  avatar_url: string
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
  nickname_required: boolean
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
  default_planned_start: string
  default_planned_end: string
  study_weekdays: number[]
  study_dates: string[]
  schedule_overrides?: ScheduleOverride[]
  public_to_group: boolean
  ai_generated: boolean
  is_shared: boolean
  sort_order: number
  created_at: string
  updated_at: string
  total_tasks: number
  completed_tasks: number
  completion_rate?: number | null
}

export interface ScheduleOverride {
  id?: number
  plan_id?: number
  weekday?: number
  date?: string
  planned_start: string
  planned_end: string
}

export type TaskStatus = 'pending' | 'in_progress' | 'completed'
export type TimerState = 'pending' | 'running' | 'paused' | 'achieved' | 'completed'

export interface StudySession {
  id: number
  task_id: number
  user_id: number
  start_time: string
  end_time?: string
  duration_min: number
  duration_seconds: number
}

export interface DailyTask {
  id: number
  plan_id: number
  user_id: number
  date: string
  title: string
  description: string
  objective: string
  reflection?: string
  status: TaskStatus
  sort_order: number
  planned_start?: string
  planned_end?: string
  estimated_minutes: number
  difficulty?: string
  public_to_group: boolean
  needs_decision: boolean
  actual_start?: string
  actual_end?: string
  study_minutes: number
  study_seconds: number
  created_at: string
  updated_at: string
  decision_reason?: string
}

export interface TimerTask extends DailyTask {
  target_minutes: number
  accumulated_seconds: number
  active_session: StudySession | null
  timer_state: TimerState
  remaining_seconds: number
  overtime_seconds: number
}

export interface PostponeRecord {
  id: number
  task_id: number
  old_date: string
  new_date: string
  reason: string
  created_at: string
}

export interface TaskDetail {
  task: TimerTask
  plan: Plan
  history: PostponeRecord[]
}

export interface Motivation {
  id: number
  user_id: number
  date: string
  text: string
  source: string
  origin: 'ai' | 'library'
  created_at: string
}

export interface PostponeTaskReq {
  date: string
  planned_start?: string
  planned_end?: string
  reason?: string
  confirm_conflict?: boolean
}

export interface MakeupTaskReq {
  actual_date: string
  actual_start: string
  actual_end: string
  reason?: string
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
  objective?: string
  weekly_target_hours?: number
  start_date?: string
  end_date?: string
  public_to_group?: boolean
  confirm_overload?: boolean
  default_planned_start?: string
  default_planned_end?: string
  study_weekdays?: number[]
  study_dates?: string[]
  schedule_overrides?: ScheduleOverride[]
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
  eligible: boolean
  remaining_tasks: number
  task: TimerTask
}

export interface DailyCheckin {
  date: string
  completed_task_count: number
  eligible: boolean
  completed: boolean
  reward_awarded?: boolean
  reward_minutes?: number
}

export interface ConsecutiveCheckin {
  consecutive_checkin_days: number
  today_qualified: boolean
  display_text: string
  streak?: number
}

export type PlanActionId = 'toggle_status' | 'edit' | 'postpone' | 'invite' | 'delete'

export interface PlanActionLayout {
  direct: PlanActionId[]
  overflow: PlanActionId[]
}

export interface PlanDelayResult {
  plan?: Plan
  moved?: number
  moved_tasks?: number
  moved_task_count?: number
  old_start_date?: string
  old_end_date?: string
  new_start_date?: string
  new_end_date?: string
}

export interface UserSearchResult {
  invite_target_id: string
  nickname: string
  avatar_url?: string
}

export interface RecoveryAction {
  task_id: number
  title: string
  plan_id?: number
  plan_title?: string
  old_date: string
  new_date: string
  planned_start: string
  planned_end: string
  reason: string
  valid?: boolean
  validation_message?: string
}

export interface RecoveryPreview {
  mode: string
  missed_days: number
  overdue_tasks: number
  pending_decisions: number
  preview_token?: string
  token?: string
  version?: string
  actions: RecoveryAction[]
  occupancy?: RecoveryOccupancy[]
  occupied_intervals?: RecoveryOccupancy[]
}

export interface RecoveryOccupancy {
  task_id?: number
  id?: number | string
  title?: string
  date: string
  planned_start?: string
  planned_end?: string
  start?: string
  end?: string
}

export interface RecoveryApplyResult {
  applied: number
  skipped?: number
  moved?: number
}

// ---------- 接口 ----------
export const AuthApi = {
  login(code: string, nickname = '', avatar_url = '') {
    return api.post<LoginResp>('/api/auth/login', { code, nickname, avatar_url }, { auth: false })
  },
  me() {
    return api.get<User>('/api/auth/me')
  },
  setNickname(nickname: string) {
    return api.put<User>('/api/auth/nickname', { nickname })
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
  validateSchedule(data: CreatePlanReq) {
    return api.post<{ valid: boolean }>('/api/plans/validate-schedule', data)
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
  delay(id: number, days: number) {
    return api.put<PlanDelayResult>(`/api/plans/${id}/shift`, { days })
  },
  shift(id: number, days: number, start_date = '') {
    return api.put<Plan>(`/api/plans/${id}/shift`, { days, start_date })
  },
  invite(id: number, invite_target_id: string) {
    return api.post<any>(`/api/plans/${id}/invite`, { invite_target_id })
  },
  tasks(id: number) {
    return api.get<TimerTask[]>(`/api/plans/${id}/tasks`)
  },
  createTask(id: number, data: { date: string; title: string; objective: string; description?: string; planned_start?: string; planned_end?: string; estimated_minutes?: number; difficulty?: string; public_to_group?: boolean; sort_order?: number }) {
    return api.post<TimerTask>(`/api/plans/${id}/tasks`, data)
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
    return api.get<RecoveryPreview>('/api/recovery/preview')
  },
  apply(preview_token: string, actions: RecoveryAction[]) {
    return api.post<RecoveryApplyResult>('/api/recovery/apply', { token: preview_token, actions })
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
  daily(date: string) {
    return api.get<DailyCheckin>(`/api/checkins/daily?date=${date}`)
  },
  completeDaily(date: string) {
    return api.post<DailyCheckin>('/api/checkins/daily', { date, completed: true })
  },
  consecutive() {
    return api.get<ConsecutiveCheckin>('/api/checkins/consecutive')
  },
}

export const StudyTaskApi = {
  get(id: number) {
    return api.get<TaskDetail>(`/api/tasks/${id}`)
  },
  start(id: number) {
    return api.put<{ task: TimerTask; session: StudySession }>(`/api/tasks/${id}/start`)
  },
  resume(id: number) {
    return api.put<{ task: TimerTask; session: StudySession }>(`/api/tasks/${id}/resume`)
  },
  pause(id: number) {
    return api.put<{ task: TimerTask; session: StudySession | null }>(`/api/tasks/${id}/pause`)
  },
  stop(id: number, reflection?: string) {
    return api.put<TimerTask>(`/api/tasks/${id}/stop`, reflection === undefined ? {} : { reflection })
  },
  complete(id: number, reflection?: string) {
    return api.put<TimerTask>(`/api/tasks/${id}/complete`, reflection === undefined ? {} : { reflection })
  },
  reflection(id: number, reflection: string) {
    return api.put<DailyTask>(`/api/tasks/${id}/reflection`, { reflection })
  },
  update(id: number, data: Partial<Pick<DailyTask, 'date' | 'title' | 'description' | 'objective' | 'sort_order' | 'planned_start' | 'planned_end' | 'estimated_minutes' | 'difficulty' | 'public_to_group'>>) {
    return api.put<DailyTask>(`/api/tasks/${id}`, data)
  },
  visibility(id: number, public_to_group: boolean) {
    return api.put<DailyTask>(`/api/tasks/${id}/visibility`, { public_to_group })
  },
  remove(id: number) {
    return api.delete<any>(`/api/tasks/${id}`)
  },
  postpone(id: number, data: PostponeTaskReq) {
    return api.put<{ task: DailyTask; record: PostponeRecord }>(`/api/tasks/${id}/postpone`, data)
  },
  makeup(id: number, data: MakeupTaskReq) {
    return api.put<{ task: DailyTask; makeup_cost_minutes: number }>(`/api/tasks/${id}/makeup`, data)
  },
  pendingDecision(date?: string) {
    return api.get<DailyTask[]>(`/api/tasks/pending-decision${date ? `?date=${date}` : ''}`)
  },
  compensateMidnight() {
    return api.post<any>('/api/tasks/midnight-compensate', {})
  },
}

export const MotivationApi = {
  daily(date?: string) {
    return api.get<Motivation>(`/api/motivation/daily${date ? `?date=${date}` : ''}`)
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
  generatePlan(data: { goal: string; hours_per_day?: number; days?: number; start_date?: string; available_time_slot?: string; skip_dates?: string[]; refinement?: string }) {
    return api.post<any>('/api/ai/generate-plan', data)
  },
  regeneratePlan(data: { goal: string; hours_per_day?: number; days?: number; start_date?: string; available_time_slot?: string; skip_dates?: string[]; refinement?: string }) {
    return api.post<any>('/api/ai/regenerate', data)
  },
  commitPlan(preview: any) {
    return api.post<any>('/api/ai/commit-plan', { preview })
  },
}

export const UserApi = {
  search(query: string) {
    return api.get<UserSearchResult[]>(`/api/users/search?q=${encodeURIComponent(query)}`)
  },
  planActionLayout() {
    return api.get<PlanActionLayout>('/api/users/me/plan-action-layout')
  },
  savePlanActionLayout(layout: PlanActionLayout) {
    return api.put<PlanActionLayout>('/api/users/me/plan-action-layout', layout)
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
