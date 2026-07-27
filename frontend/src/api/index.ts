// 业务接口封装：用户/计划/打卡
import { api } from './request'

// ---------- 类型 ----------
export interface User {
  id: number
  username?: string
  nickname: string
  avatar_url: string
  weekly_hours: number
  slack_balance: number
  role: 'user' | 'admin'
  banned_until?: string
  banned_reason?: string
  account_status?: 'active' | 'inactive' | 'deleted'
  onboarding_status: 'not_started' | 'completed' | 'skipped'
  onboarding_version: number
  onboarding_completed_at?: string
}

export interface LoginResp {
  token: string
  user: User
  nickname_required: boolean
}

export interface RegistrationRequiredResp {
  registration_required: true
  registration_token: string
}

export interface RegistrationReq {
  invite_code: string
  username: string
  nickname: string
  password: string
}

export interface NotificationDeliveryResult {
  status: 'sent' | 'failed' | 'processing' | `skipped_${string}`
  message?: string
}

export interface NotificationTemplateMetadata {
  platform: string
  templates: NotificationTemplate[]
}

export interface NotificationTemplate {
  reminder_type: string
  template_id: string
}

export interface NotificationSubscription {
  id: number
  reminder_type: string
  template_id: string
  subscribed: boolean
  created_at: string
  updated_at: string
}

export type WechatLoginResp = LoginResp | RegistrationRequiredResp

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
  generation_source: '' | 'local' | 'local_enriched' | 'ai_decomposed'
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
  task_drafts?: TaskDraft[]
}

export interface TaskDraft {
  date: string
  title: string
  objective: string
  description: string
  planned_start: string
  planned_end: string
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

export interface NextTaskInfo {
  task: DailyTask
  plan: Plan
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
	version?: number
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
	plan_id?: number
	plan_title?: string
	days?: number
  occupancy?: RecoveryOccupancy[]
  occupied_intervals?: RecoveryOccupancy[]
}

export interface RecoveryOccupancy {
  task_id?: number
  id?: number | string
  title?: string
	plan_id?: number
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

export type AIPlanJobStatus = 'pending' | 'running' | 'succeeded' | 'failed'

export interface AIPlanJob {
  id: number
  status: AIPlanJobStatus
  attempt_count: number
  result_plan_id?: number
  error_code?: string
  error_message?: string
  generation_source?: 'local' | 'local_enriched' | 'ai_decomposed'
  provider?: string
  model?: string
  enrichment_status?: string
  enrichment_reason?: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export interface SubmitAIPlanJobReq {
  goal: string
  hours_per_day?: number
  days?: number
  start_date?: string
  available_time_slot?: string
  skip_dates?: string[]
  additional_instructions?: string
  idempotency_key?: string
  confirm_overload?: boolean
}

export type FeedbackCategory = 'issue' | 'suggestion' | 'content' | 'account' | 'other'
export type FeedbackStatus = 'open' | 'processing' | 'resolved' | 'closed'

export interface FeedbackReport {
  id: number
  category: FeedbackCategory
  content: string
  status: FeedbackStatus
  public_response?: string
  responded_at?: string
  created_at: string
  updated_at: string
}

// ---------- 接口 ----------
export const AuthApi = {
  login(code: string) {
    return api.post<WechatLoginResp>('/api/auth/login', { code }, { auth: false })
  },
  h5Register(data: RegistrationReq) {
    return api.post<LoginResp>('/api/auth/h5/register', data, { auth: false })
  },
  h5Login(username: string, password: string) {
    return api.post<LoginResp>('/api/auth/h5/login', { username, password }, { auth: false })
  },
  wechatRegister(data: RegistrationReq & { registration_token: string }) {
    return api.post<LoginResp>('/api/auth/wechat/register', data, { auth: false })
  },
  wechatLink(data: { registration_token: string; username: string; password: string }) {
    return api.post<LoginResp>('/api/auth/wechat/link', data, { auth: false })
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
  updateOnboarding(status: 'completed' | 'skipped') {
    return api.post<User>('/api/auth/onboarding', { status })
  },
  changePassword(current_password: string, new_password: string) {
    return api.post<{ token: string; user: User }>('/api/auth/password/change', { current_password, new_password })
  },
  resetPassword(username: string, code: string, new_password: string) {
    return api.post<{ reset: boolean }>('/api/auth/password/reset', { username, code, new_password }, { auth: false })
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
	shiftPreview(id: number, days: number) {
		return api.get<RecoveryPreview>(`/api/plans/${id}/shift-preview?days=${days}`)
	},
	applyShift(id: number, preview_token: string, actions: RecoveryAction[]) {
		return api.post<RecoveryApplyResult>(`/api/plans/${id}/shift-apply`, { token: preview_token, actions })
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
    return api.post<NotificationDeliveryResult>(`/api/groups/${id}/members/${userId}/nudge`, {})
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
  feedback(data: { category: FeedbackCategory; content: string; contact?: string }) {
    return api.post<FeedbackReport>('/api/feedback', data)
  },
  feedbackHistory() {
    return api.get<FeedbackReport[]>('/api/feedback')
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
  next(after?: string) {
    return api.get<NextTaskInfo | null>(`/api/tasks/next${after ? `?after=${after}` : ''}`)
  },
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
		return api.get<{ balance: number; unit: string; can_start: boolean; blocked_reason: string; low_balance: boolean; active_session?: any }>('/api/slack/balance')
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
	trend(period: '7d' | '1m' | '1y', dimension: 'time' | 'plan') {
		return api.get<StatsTrend>(`/api/stats/trend?period=${period}&dimension=${dimension}`)
	},
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

export interface StatsMetrics { study_minutes: number; planned_minutes: number; overtime_minutes: number; completed_tasks: number; total_tasks: number; completion_rate: number | null }
export interface StatsPoint extends StatsMetrics { key: string; label: string; start: string; end: string; plan_id?: number; plan_title?: string }
export interface StatsTrend { period: '7d' | '1m' | '1y'; dimension: 'time' | 'plan'; start: string; end: string; bucket_unit: 'day' | 'month' | 'plan'; summary: StatsMetrics; series: StatsPoint[] }

export const AIApi = {
  submitPlanJob(data: SubmitAIPlanJobReq) {
    return api.post<AIPlanJob>('/api/ai/plan-jobs', data)
  },
  currentPlanJob() {
    return api.get<AIPlanJob | null>('/api/ai/plan-jobs/current')
  },
  getPlanJob(id: number) {
    return api.get<AIPlanJob>(`/api/ai/plan-jobs/${id}`)
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
    return api.get<NotificationSubscription[]>('/api/notifications/subscriptions')
  },
  templates() {
    return api.get<NotificationTemplateMetadata>('/api/notifications/templates')
  },
  subscribe(reminderType: string, templateId: string, result: string) {
    return api.post<{ accepted: string[] }>('/api/notifications/subscribe', { reminder_type: reminderType, template_id: templateId, result })
  },
  unsubscribe() {
    return api.delete<any>('/api/notifications/subscribe')
  },
  due(date?: string) {
    return api.get<any>(`/api/notifications/due${date ? `?date=${date}` : ''}`)
  },
}
