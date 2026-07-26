<template>
  <view class="page">
    <view class="panel">
      <view class="title">智能规划学习计划</view>
      <view class="desc">系统会先立即生成可用基线，AI 再在后台拆解学习阶段和任务；日期、时段和冲突仍由系统统一校验。</view>
      <view class="field"><text>学习目标</text><input v-model="goal" placeholder="例如：学习 Go 语言" /></view>
      <view class="grid">
        <view class="field"><text>每天小时</text><input v-model.number="hours" type="number" /></view>
        <view class="field"><text>计划天数</text><input v-model.number="days" type="number" /></view>
      </view>
      <view class="field"><text>开始日期</text><picker mode="date" :value="startDate" @change="startDate = $event.detail.value"><view class="picker-value">{{ startDate }}</view></picker></view>
      <view class="field"><text>跳过日期</text><input v-model="skipDatesText" placeholder="多个日期用逗号分隔，例如 2026-08-02,2026-08-03" /></view>
      <view class="grid">
        <view class="field"><text>可用开始时间</text><picker mode="time" :value="availableStart" @change="availableStart = $event.detail.value"><view class="picker-value">{{ availableStart }}</view></picker></view>
        <view class="field"><text>可用结束时间</text><picker mode="time" :value="availableEnd" @change="availableEnd = $event.detail.value"><view class="picker-value">{{ availableEnd }}</view></picker></view>
      </view>
      <view class="field"><text>追加说明</text><textarea v-model="refinement" placeholder="例如：周末少一点，工作日晚间安排" /></view>
      <button class="primary" :disabled="isGenerating || isCommitting" :loading="isGenerating" @click="generate">{{ isGenerating ? '正在生成本地基线…' : '生成智能计划' }}</button>
      <button class="secondary" v-if="preview" :disabled="isCommitting || isGenerating" :loading="isCommitting" @click="commit">{{ isCommitting ? '正在保存…' : '确认保存' }}</button>
      <view class="error-panel" v-if="errorMessage">{{ errorMessage }}</view>
    </view>

    <view class="result" v-if="preview">
      <view class="source-card">
        <view class="source-title">{{ source === 'ai_decomposed' ? 'AI 拆解计划' : '本地智能计划' }}</view>
        <view class="source-note">{{ enrichmentMessage }}</view>
        <button class="review-button" v-if="aiCandidate" @click="reviewAICandidate">查看 AI 新版本</button>
      </view>
      <view class="warning" v-for="warning in warnings" :key="warning">{{ warning }}</view>
      <view class="result-title">{{ preview.title }}</view>
      <view class="preview-summary">{{ preview.summary }}</view>
      <view class="preview-rationale">{{ preview.rationale }}</view>
      <view class="task" v-for="(t, index) in preview.tasks" :key="`${t.date}-${index}`">
        <view class="task-head"><view class="task-title">任务 {{ index + 1 }}</view><view class="task-actions"><button size="mini" :disabled="isMutating || index === 0" @click="moveTask(index, -1)">上移</button><button size="mini" :disabled="isMutating || index === preview.tasks.length - 1" @click="moveTask(index, 1)">下移</button><button size="mini" :disabled="isMutating || t.estimated_minutes < 30" @click="splitTask(t)">拆分</button><button size="mini" :disabled="isMutating || preview.tasks.length === 1" @click="removeTask(t)">删除</button></view></view>
        <view class="field"><text>日期</text><input v-model="t.date" /></view>
        <view class="field"><text>开始时间</text><input v-model="t.planned_start" placeholder="20:00" @input="syncEstimatedMinutes(t)" /></view>
        <view class="field"><text>结束时间</text><input v-model="t.planned_end" placeholder="21:00" @input="syncEstimatedMinutes(t)" /></view>
        <view class="field"><text>标题</text><input v-model="t.title" /></view>
        <view class="field"><text>任务目标</text><textarea v-model="t.objective" maxlength="500" placeholder="描述当天具体要完成什么" /></view>
        <view class="field"><text>描述</text><textarea v-model="t.description" /></view>
        <view class="grid">
          <view class="field"><text>预计分钟</text><input :value="t.estimated_minutes" type="number" disabled /></view>
          <view class="field"><text>难度</text><picker :range="difficultyOptions" range-key="label" :value="difficultyIndex(t.difficulty)" @change="setDifficulty(t, $event.detail.value)"><view class="picker-value">{{ difficultyLabel(t.difficulty) }}</view></picker></view>
        </view>
      </view>
      <button class="secondary" :disabled="isMutating" :loading="isMutating" @click="addTask">新增任务</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onUnload } from '@dcloudio/uni-app'
import { AIApi, type PlanningPreview, type PlanningPreviewTask, type PlanningResponse } from '@/api'
import { formatScheduleConflicts, validateScheduleUnion } from '@/utils/schedule'

const goal = ref('学习 Go 语言')
const hours = ref(1)
const days = ref(7)
const availableStart = ref('20:00')
const availableEnd = ref('21:00')
const refinement = ref('')
const startDate = ref(todayString())
const skipDatesText = ref('')
const preview = ref<PlanningPreview | null>(null)
const errorMessage = ref('')
const isGenerating = ref(false)
const isCommitting = ref(false)
const isMutating = ref(false)
const source = ref<PlanningResponse['source']>('local')
const provenanceToken = ref('')
const previewID = ref('')
const idempotencyKey = ref('')
const warnings = ref<string[]>([])
const enrichmentMessage = ref('计划由本地规则生成，未使用外部模型。')
const activeJobID = ref('')
const previewVersion = ref(0)
const aiCandidate = ref<PlanningResponse | null>(null)
const difficultyOptions = [
  { value: 'easy', label: '简单' },
  { value: 'medium', label: '适中' },
  { value: 'hard', label: '困难' },
]
let generationSequence = 0
let pollingTimer: ReturnType<typeof setTimeout> | null = null
let baselineSnapshot = ''

const enrichmentMessages: Record<PlanningResponse['enrichment']['status'], string> = {
  success: 'AI 已在不改变日期、时段和任务数量的前提下优化内容。',
  queued: '本地基线已可用，AI 任务拆解正在排队。',
  decomposing: 'AI 正在拆解学习阶段和具体任务。',
  scheduling: 'AI 拆解已完成，系统正在安排日期和时段。',
  ready: 'AI 拆解版本已生成。',
  fallback: 'AI 拆解未完成，当前本地基线仍可正常保存。',
  expired: 'AI 拆解任务已过期，当前本地基线仍可正常保存。',
  disabled: 'AI 增强未启用，本地计划仍然完整可用。',
  configuration_error: 'AI 配置暂不可用，已返回本地计划。',
  quota_limited: 'AI 增强额度暂不可用，已返回本地计划。',
  timeout: 'AI 增强未在约 12 秒内完成，已返回本地计划。',
  cancelled: 'AI 增强已取消，已保留本地计划。',
  invalid_output: 'AI 返回内容未通过规则校验，已使用本地计划。',
  provider_error: 'AI 服务暂不可用，已返回本地计划。',
}

onUnload(() => stopPolling(true))

function difficultyIndex(value: string) {
  const index = difficultyOptions.findIndex(option => option.value === value)
  return index < 0 ? 0 : index
}

function difficultyLabel(value: string) {
  return difficultyOptions[difficultyIndex(value)].label
}

function setDifficulty(task: PlanningPreviewTask, index: string | number) {
  task.difficulty = difficultyOptions[Number(index)]?.value || 'easy'
}

async function generate() {
	if (isGenerating.value || isCommitting.value) return
  errorMessage.value = ''
  if (!goal.value.trim()) {
    uni.showToast({ title: '请输入学习目标', icon: 'none' })
    return
  }
  if (availableStart.value >= availableEnd.value) {
    uni.showToast({ title: '可用结束时间须晚于开始时间', icon: 'none' })
    return
  }
	const sequence = ++generationSequence
	stopPolling(true)
	isGenerating.value = true
	preview.value = null
	provenanceToken.value = ''
	idempotencyKey.value = ''
	aiCandidate.value = null
	uni.removeStorageSync('ai_plan_pending_commit')
	try {
		const resp = await AIApi.generatePlan({ goal: goal.value, hours_per_day: hours.value, days: days.value, start_date: startDate.value, skip_dates: parsedSkipDates(), available_time_slot: `${availableStart.value}-${availableEnd.value}`, refinement: refinement.value })
		if (sequence !== generationSequence) return
		preview.value = resp.preview
    source.value = resp.source
    provenanceToken.value = resp.provenance_token
    previewID.value = resp.preview_id
    previewVersion.value = resp.preview_version
    baselineSnapshot = JSON.stringify(resp.preview)
    idempotencyKey.value = createCommitKey()
    uni.setStorageSync('ai_plan_pending_commit', { provenanceToken: provenanceToken.value, idempotencyKey: idempotencyKey.value })
    warnings.value = resp.warnings || []
    enrichmentMessage.value = enrichmentMessages[resp.enrichment.status]
    if (resp.job?.id) {
      activeJobID.value = resp.job.id
      scheduleJobPoll(sequence, 800)
    }
    uni.showToast({ title: '已生成预览', icon: 'success' })
	} catch (e: any) {
		if (sequence !== generationSequence) return
		errorMessage.value = apiScheduleError(e) || e?.message || '生成失败'
	} finally {
		if (sequence === generationSequence) isGenerating.value = false
	}
}

async function mutatePreview(mutation: Parameters<typeof AIApi.mutatePreview>[2]) {
  if (!preview.value || !previewID.value || !previewVersion.value || isMutating.value) return
  isMutating.value = true
  errorMessage.value = ''
  try {
    const result = await AIApi.mutatePreview(previewID.value, previewVersion.value, mutation)
    preview.value = result.preview
    previewID.value = result.preview_id
    previewVersion.value = result.preview_version
    source.value = result.source
    provenanceToken.value = result.provenance_token
    idempotencyKey.value = createCommitKey()
    uni.setStorageSync('ai_plan_pending_commit', { provenanceToken: provenanceToken.value, idempotencyKey: idempotencyKey.value })
  } catch (e: any) {
    errorMessage.value = e?.message || '更新预览失败'
  } finally {
    isMutating.value = false
  }
}

function addTask() {
  if (!preview.value?.tasks.length) return
  const last = preview.value.tasks[preview.value.tasks.length - 1]
  const date = nextEligibleDate(last.date)
  void mutatePreview({ operation: 'add', insert_after_identity: last.identity, task: { identity: '', date, planned_start: last.planned_start, planned_end: last.planned_end, title: '新增学习任务', objective: '完成新增任务的具体学习产出', description: '', estimated_minutes: last.estimated_minutes, difficulty: 'easy' } })
}

function removeTask(task: PlanningPreviewTask) {
  void mutatePreview({ operation: 'remove', task_identity: task.identity })
}

function splitTask(task: PlanningPreviewTask) {
  void mutatePreview({ operation: 'split', task_identity: task.identity, first_part_minutes: Math.floor(task.estimated_minutes / 2) })
}

function moveTask(index: number, direction: number) {
  if (!preview.value) return
  const identities = preview.value.tasks.map(task => task.identity)
  const target = index + direction
  ;[identities[index], identities[target]] = [identities[target], identities[index]]
  void mutatePreview({ operation: 'reorder', ordered_identities: identities })
}

function parsedSkipDates() {
  return skipDatesText.value.split(/[,，\s]+/).map(value => value.trim()).filter(Boolean)
}

function nextEligibleDate(value: string) {
  const skipped = new Set(parsedSkipDates())
  const date = new Date(`${value}T00:00:00`)
  for (let offset = 0; offset < 120; offset++) {
    date.setDate(date.getDate() + 1)
    const candidate = localDateString(date)
    if (!skipped.has(candidate)) return candidate
  }
  return value
}

function todayString() {
  return localDateString(new Date())
}

function localDateString(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function scheduleJobPoll(sequence: number, delay: number) {
  if (!activeJobID.value || sequence !== generationSequence) return
  pollingTimer = setTimeout(async () => {
    if (!activeJobID.value || sequence !== generationSequence) return
    try {
      const result = await AIApi.planningJob(activeJobID.value)
      if (sequence !== generationSequence) return
      enrichmentMessage.value = enrichmentMessages[result.job.status]
      if (result.job.status === 'ready' && result.preview && result.provenance_token && result.preview_version) {
        const candidate: PlanningResponse = {
          preview: result.preview,
          mode: 'ai',
          source: result.source || 'ai_decomposed',
          provenance_token: result.provenance_token,
          preview_id: result.preview_id || result.job.preview_id,
          preview_version: result.preview_version,
          expires_at: result.expires_at || result.job.expires_at,
          context_fingerprint: result.context_fingerprint || '',
          job: result.job,
          warnings: [],
          enrichment: { status: 'ready', reason: '', provider: result.job.provider || '', model: result.job.model || '' },
          request_budget_ms: 0,
          phase_timings_ms: { context: 0, local_planning: 0, enrichment: 0, total: 0 },
        }
        if (preview.value && JSON.stringify(preview.value) === baselineSnapshot) applyPlanningVersion(candidate)
        else aiCandidate.value = candidate
        stopPolling(false)
        return
      }
      if (['fallback', 'cancelled', 'expired'].includes(result.job.status)) {
        stopPolling(false)
        return
      }
      scheduleJobPoll(sequence, 1200)
    } catch {
      scheduleJobPoll(sequence, Math.min(delay * 2, 5000))
    }
  }, delay)
}

function applyPlanningVersion(result: PlanningResponse) {
  preview.value = result.preview
  source.value = result.source
  provenanceToken.value = result.provenance_token
  previewID.value = result.preview_id
  previewVersion.value = result.preview_version
  idempotencyKey.value = createCommitKey()
  baselineSnapshot = JSON.stringify(result.preview)
  aiCandidate.value = null
  enrichmentMessage.value = 'AI 已完成任务拆解，系统已重新排程并校验。'
  uni.setStorageSync('ai_plan_pending_commit', { provenanceToken: provenanceToken.value, idempotencyKey: idempotencyKey.value })
}

function reviewAICandidate() {
  if (aiCandidate.value) applyPlanningVersion(aiCandidate.value)
}

function stopPolling(cancelRemote: boolean) {
  if (pollingTimer) clearTimeout(pollingTimer)
  pollingTimer = null
  const jobID = activeJobID.value
  activeJobID.value = ''
  if (cancelRemote && jobID) void AIApi.cancelPlanningJob(jobID).catch(() => undefined)
}

async function commit() {
	if (!preview.value || isCommitting.value || isGenerating.value) return
  errorMessage.value = ''
  const invalid = preview.value?.tasks?.find((task) => !task.objective?.trim() || task.objective.trim().toLowerCase() === task.title?.trim().toLowerCase())
  if (invalid) {
    uni.showToast({ title: '每个任务需填写比标题更具体的目标', icon: 'none' })
    return
  }
  const conflicts = validateScheduleUnion((preview.value?.tasks || []).map((task, index) => ({ id: index, title: task.title || `任务 ${index + 1}`, date: task.date, start: task.planned_start, end: task.planned_end })))
  if (conflicts.length) {
    errorMessage.value = `时间安排需要调整：\n${formatScheduleConflicts(conflicts)}`
    return
  }
  isCommitting.value = true
  try {
    await commitPreview(false)
    uni.showToast({ title: '已保存', icon: 'success' })
    uni.removeStorageSync('ai_plan_pending_commit')
    uni.reLaunch({ url: '/pages/plans/plans' })
  } catch (e: any) {
    if (String(e?.message || '').includes('confirm_overload required')) {
      const confirmed = await new Promise<boolean>((resolve) => uni.showModal({ title: '计划负荷提醒', content: '当前活跃计划数量或每周总学时较高，仍要保存吗？', success: (result) => resolve(result.confirm), fail: () => resolve(false) }))
      if (confirmed) {
        try {
          await commitPreview(true)
          uni.removeStorageSync('ai_plan_pending_commit')
          uni.showToast({ title: '已保存', icon: 'success' })
          uni.reLaunch({ url: '/pages/plans/plans' })
          return
        } catch (retryError: any) {
          e = retryError
        }
      }
    }
    errorMessage.value = apiScheduleError(e) || e?.message || '保存失败'
  } finally {
    isCommitting.value = false
  }
}

function commitPreview(confirmOverload: boolean) {
	if (!preview.value) return Promise.reject(new Error('没有可保存的计划'))
  const stored = uni.getStorageSync('ai_plan_pending_commit') || {}
  provenanceToken.value ||= stored.provenanceToken || ''
  idempotencyKey.value ||= stored.idempotencyKey || createCommitKey()
  uni.setStorageSync('ai_plan_pending_commit', { provenanceToken: provenanceToken.value, idempotencyKey: idempotencyKey.value })
	return AIApi.commitPlan(preview.value, previewID.value, previewVersion.value, provenanceToken.value, idempotencyKey.value, confirmOverload)
}

function createCommitKey() {
  return `plan_${Date.now().toString(36)}_${Math.random().toString(36).slice(2)}_${Math.random().toString(36).slice(2)}`
}

function syncEstimatedMinutes(task: PlanningPreviewTask) {
  const match = /^(\d{2}):(\d{2})$/.exec(task.planned_start || '')
  const endMatch = /^(\d{2}):(\d{2})$/.exec(task.planned_end || '')
  if (!match || !endMatch) return
  const start = Number(match[1]) * 60 + Number(match[2])
  const end = Number(endMatch[1]) * 60 + Number(endMatch[2])
  if (start >= 0 && end > start && end <= 24 * 60) task.estimated_minutes = end - start
}
function apiScheduleError(error: any) {
  const rows = error?.raw?.invalid_tasks
  if (!Array.isArray(rows)) return ''
  return `时间安排需要调整：\n${rows.map((row: any) => `${row.date ? `${row.date} ` : ''}「${row.title}」重叠覆盖 ${row.covered_minutes} 分钟${row.covered_intervals?.length ? `（${row.covered_intervals.map((range: any) => `${range.start}-${range.end}`).join('、')}）` : ''}`).join('\n')}`
}
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel, .result { padding: 32rpx; border-radius: 18rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.desc { margin: 12rpx 0 28rpx; color: #7b8498; font-size: 25rpx; line-height: 1.5; }
.field { margin-bottom: 22rpx; }
.field text { display: block; margin-bottom: 10rpx; color: #606a80; font-size: 24rpx; }
.field input, .field textarea { width: 100%; box-sizing: border-box; height: 78rpx; padding: 0 20rpx; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; }
.picker-value { box-sizing: border-box; height: 78rpx; padding: 20rpx; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; color: #111827; }
.field textarea { height: 140rpx; padding-top: 16rpx; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 18rpx; }
.primary, .secondary { background: #2264d1; color: #fff; border-radius: 12rpx; }
.primary[disabled] { opacity: .65; }
.secondary { margin-top: 18rpx; background: #eef4ff; color: #2264d1; }
.result { margin-top: 22rpx; }
.result-title { color: #111827; font-size: 30rpx; font-weight: 800; margin-bottom: 18rpx; }
.preview-summary, .preview-rationale { color: #606a80; font-size: 24rpx; line-height: 1.5; margin-bottom: 16rpx; }
.source-card { margin-bottom: 18rpx; padding: 18rpx; border-radius: 12rpx; background: #eef6ff; color: #365574; }
.source-title { margin-bottom: 6rpx; color: #174f8a; font-size: 25rpx; font-weight: 700; }
.source-note, .warning { font-size: 23rpx; line-height: 1.5; }
.review-button { margin-top: 14rpx; background: #fff; color: #2264d1; font-size: 23rpx; }
.warning { margin-bottom: 12rpx; padding: 16rpx; border-radius: 10rpx; background: #fff7e6; color: #8a5a10; }
.task { padding: 20rpx 0; border-top: 1rpx solid #eef2f7; }
.task-head { display: flex; align-items: center; justify-content: space-between; }
.task-actions { display:flex; gap:6rpx; flex-wrap:wrap; justify-content:flex-end; }
.task-actions button { margin:0; padding:0 12rpx; font-size:20rpx; }
.task-title { color: #111827; font-size: 27rpx; font-weight: 700; }
.error-panel { margin-top: 18rpx; padding: 18rpx; border-radius: 12rpx; background: #fff1f3; color: #b4455b; font-size: 23rpx; line-height: 1.55; white-space: pre-line; }
</style>
