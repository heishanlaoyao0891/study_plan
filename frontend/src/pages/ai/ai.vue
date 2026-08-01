<template>
  <view class="page" v-if="aiAvailable">
    <view class="panel">
      <view class="title">智能规划学习计划</view>
      <view class="desc">提交后由系统在后台规划并自动创建计划。离开页面不会中断，完成后可在计划列表中继续编辑。</view>
      <view class="field"><text>学习目标</text><input v-model="goal" :disabled="hasActiveJob" placeholder="例如：学习 Go 语言" /></view>
      <view class="grid">
        <view class="field"><text>每天小时</text><input v-model.number="hours" :disabled="hasActiveJob" type="number" /></view>
        <view class="field"><text>计划天数</text><input v-model.number="days" :disabled="hasActiveJob" type="number" /></view>
      </view>
      <view class="grid">
        <view class="field"><text>可用开始时间</text><picker mode="time" :disabled="hasActiveJob" :value="availableStart" @change="availableStart = $event.detail.value"><view class="picker-value">{{ availableStart }}</view></picker></view>
        <view class="field"><text>可用结束时间</text><picker mode="time" :disabled="hasActiveJob" :value="availableEnd" @change="availableEnd = $event.detail.value"><view class="picker-value">{{ availableEnd }}</view></picker></view>
      </view>
      <view class="field"><text>追加说明（可选）</text><textarea v-model="additionalInstructions" :disabled="hasActiveJob" maxlength="2000" placeholder="例如：周末少一点，多安排实践练习和阶段复盘" /></view>
      <button class="primary" :disabled="isSubmitting || isRestoring || hasActiveJob" :loading="isSubmitting" @click="handlePrimaryAction">
        {{ submitButtonText }}
      </button>
      <button class="secondary-generate" v-if="hasSavedPlan" :disabled="isSubmitting || isRestoring" @click="prepareNewPlan">生成另一个计划</button>
      <view class="error-panel" v-if="requestError">{{ requestError }}</view>
    </view>

    <view class="status-card" v-if="job">
      <view class="status-head">
        <view class="status-dot" :class="job.status" />
        <view>
          <view class="status-title">{{ statusTitle }}</view>
          <view class="status-note">{{ statusNote }}</view>
        </view>
      </view>
      <view class="status-meta" v-if="job.status === 'running'">正在执行第 {{ Math.max(job.attempt_count, 1) }} 次处理，请稍候。</view>
      <view class="status-meta" v-if="job.status === 'pending' && job.phase === 'retry_wait'">AI 返回暂未通过校验，后台 Agent 正在等待下一轮自动修复；失败尝试不会消耗今日生成次数。</view>
      <view class="status-meta" v-if="job.status === 'succeeded' && job.generation_source">{{ generationSummary }}</view>
      <view class="job-error" v-if="job.status === 'failed'">{{ job.error_message || '未能生成有效计划，请检查目标和可用时间后重试。' }}</view>
      <button class="confirm-overload" v-if="needsOverloadConfirmation" :loading="isSubmitting" :disabled="isSubmitting" @click="confirmOverload">确认负荷并继续生成</button>
      <button class="result-link" v-if="job.status === 'succeeded' && job.result_plan_id" @click="openResult">查看并编辑计划</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onHide, onShow, onUnload } from '@dcloudio/uni-app'
import { AIApi, ClientFeatureApi, type AIPlanJob } from '@/api'

const goal = ref('学习 Go 语言')
const hours = ref(1)
const days = ref(7)
const availableStart = ref('20:00')
const availableEnd = ref('21:00')
const additionalInstructions = ref('')
const job = ref<AIPlanJob | null>(null)
let miniProgramBuild = false
// #ifdef MP-WEIXIN
miniProgramBuild = true
// #endif
const aiAvailable = ref(!miniProgramBuild)
const requestError = ref('')
const isSubmitting = ref(false)
const isRestoring = ref(false)
const hasActiveJob = computed(() => job.value?.status === 'pending' || job.value?.status === 'running')
const hasSavedPlan = computed(() => job.value?.status === 'succeeded' && !!job.value.result_plan_id)
const needsOverloadConfirmation = computed(() => job.value?.status === 'failed' && job.value.error_code === 'overload_confirmation_required')
const generationSummary = computed(() => {
  if (!job.value) return ''
  if ((job.value.generation_source === 'local_enriched' || job.value.generation_source === 'ai_decomposed') && job.value.enrichment_status === 'success') {
    const model = [job.value.provider, job.value.model].filter(Boolean).join(' / ')
    return model ? `AI 调用成功：${model}` : 'AI 调用成功并优化了计划内容'
  }
  const reasons: Record<string, string> = {
    provider_disabled: 'AI 未启用',
    provider_configuration_unavailable: 'AI 配置不可用',
    invalid_provider_configuration: 'AI 配置校验失败',
    daily_enrichment_limit_reached: '今日 AI 调用额度已用完',
    provider_quota_limited: 'AI 服务额度不足',
    quota_check_failed: 'AI 额度检查失败',
    provider_request_failed: 'AI 服务请求失败',
    enrichment_deadline_exceeded: 'AI 服务响应超时',
    request_cancelled: 'AI 请求已取消',
    invalid_provider_output: 'AI 返回内容未通过校验',
  }
  const reason = reasons[job.value.enrichment_reason || ''] || 'AI 未产生可用结果'
  return `本次未使用 AI 结果，已按本地规则生成：${reason}`
})
const submitButtonText = computed(() => {
  if (isRestoring.value) return '正在恢复生成状态…'
  if (isSubmitting.value) return '正在提交…'
  if (job.value?.status === 'pending') return '已提交，等待处理'
  if (job.value?.status === 'running') return '正在生成计划'
  if (hasSavedPlan.value) return '查看已保存计划'
  if (job.value?.status === 'failed') return '重新尝试生成'
  return job.value ? '生成另一个计划' : '生成智能计划'
})
const statusTitle = computed(() => ({
  pending: job.value?.phase === 'retry_wait' ? '后台 Agent 将继续修复' : '计划已进入队列',
  running: '正在生成计划',
  failed: '计划生成失败',
  succeeded: '计划已生成',
}[job.value?.status || 'pending']))
const statusNote = computed(() => ({
  pending: job.value?.phase === 'retry_wait' ? '无需重新提交或停留在页面，任务会从已保存状态继续。' : '后台即将开始处理，可以放心离开此页面。',
  running: '系统正在校验日程并生成任务，可以放心离开此页面。',
  failed: '本次任务已经结束，你可以调整条件后重新提交。',
  succeeded: '计划和任务已自动保存，可以直接查看和编辑。',
}[job.value?.status || 'pending']))

let visible = false
let pollTimer: ReturnType<typeof setTimeout> | undefined
let pollSequence = 0

function handlePrimaryAction() {
  if (hasSavedPlan.value) return openResult()
  return submit(false)
}

function prepareNewPlan() {
  stopTimer()
  requestError.value = ''
  job.value = null
}

async function submit(confirmOverload: boolean) {
  if (!(await ensureAIAvailable())) return
  if (isSubmitting.value || isRestoring.value || hasActiveJob.value) return
  requestError.value = ''
  if (!goal.value.trim()) return void uni.showToast({ title: '请输入学习目标', icon: 'none' })
  if (hours.value <= 0 || days.value <= 0) return void uni.showToast({ title: '每天小时和计划天数须大于 0', icon: 'none' })
  if (availableStart.value >= availableEnd.value) return void uni.showToast({ title: '可用结束时间须晚于开始时间', icon: 'none' })
  isSubmitting.value = true
  try {
    job.value = await AIApi.submitPlanJob({
      goal: goal.value.trim(),
      hours_per_day: hours.value,
      days: days.value,
      available_time_slot: `${availableStart.value}-${availableEnd.value}`,
      additional_instructions: additionalInstructions.value.trim() || undefined,
      idempotency_key: createSubmissionKey(),
      confirm_overload: confirmOverload,
    })
    uni.showToast({ title: '已提交后台生成', icon: 'success' })
    schedulePoll()
  } catch (error: any) {
    requestError.value = error?.message || '提交失败，请稍后重试'
  } finally {
    isSubmitting.value = false
  }
}

async function confirmOverload() {
  const confirmed = await new Promise<boolean>((resolve) => uni.showModal({
    title: '计划负荷提醒',
    content: '当前活动计划较多或每周学习负荷较高，仍要生成并创建这个计划吗？',
    confirmText: '继续生成',
    success: result => resolve(result.confirm),
    fail: () => resolve(false),
  }))
  if (confirmed) await submit(true)
}

async function restoreCurrentJob() {
  if (!aiAvailable.value) return
  const sequence = ++pollSequence
  isRestoring.value = true
  requestError.value = ''
  try {
    const current = await AIApi.currentPlanJob()
    if (!visible || sequence !== pollSequence) return
    job.value = current
    schedulePoll()
  } catch (error: any) {
    if (visible && sequence === pollSequence) requestError.value = error?.message || '生成状态加载失败'
  } finally {
    if (sequence === pollSequence) isRestoring.value = false
  }
}

async function pollJob() {
  const current = job.value
  if (!visible || !current || !hasActiveJob.value) return
  const sequence = ++pollSequence
  try {
    const updated = await AIApi.getPlanJob(current.id)
    if (!visible || sequence !== pollSequence) return
    job.value = updated
    requestError.value = ''
  } catch (error: any) {
    if (visible && sequence === pollSequence) requestError.value = error?.message || '生成状态更新失败'
  }
  if (visible && sequence === pollSequence && hasActiveJob.value) schedulePoll()
}

function schedulePoll() {
  stopTimer()
  if (visible && hasActiveJob.value) pollTimer = setTimeout(pollJob, 2000)
}

function stopTimer() {
  if (pollTimer) clearTimeout(pollTimer)
  pollTimer = undefined
}

function stopPolling() {
  visible = false
  pollSequence++
  stopTimer()
  isRestoring.value = false
}

function createSubmissionKey() {
  return `plan_${Date.now().toString(36)}_${Math.random().toString(36).slice(2)}_${Math.random().toString(36).slice(2)}`
}

function openResult() {
  if (job.value?.result_plan_id) uni.navigateTo({ url: `/pages/plan-detail/plan-detail?id=${job.value.result_plan_id}` })
}

async function ensureAIAvailable() {
  if (!miniProgramBuild) return true
  try {
    const features = await ClientFeatureApi.get()
    aiAvailable.value = features?.mini_program_ai_enabled === true
  } catch {
    aiAvailable.value = false
  }
  if (aiAvailable.value) return true
  stopPolling()
  uni.switchTab({ url: '/pages/plans/plans' })
  return false
}

onShow(async () => {
  visible = true
  await ensureAIAvailable()
  if (aiAvailable.value) restoreCurrentJob()
})
onHide(stopPolling)
onUnload(stopPolling)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel, .status-card { padding: 32rpx; border-radius: 18rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.desc { margin: 12rpx 0 28rpx; color: #7b8498; font-size: 25rpx; line-height: 1.5; }
.field { margin-bottom: 22rpx; }
.field text { display: block; margin-bottom: 10rpx; color: #606a80; font-size: 24rpx; }
.field input, .field textarea { width: 100%; box-sizing: border-box; height: 78rpx; padding: 0 20rpx; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; }
.field textarea { height: 170rpx; padding-top: 16rpx; }
.picker-value { box-sizing: border-box; height: 78rpx; padding: 20rpx; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; color: #111827; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 18rpx; }
.primary { background: #2264d1; color: #fff; border-radius: 12rpx; }
.primary[disabled] { opacity: .65; }
.secondary-generate { margin-top: 16rpx; border: 1rpx solid #cddbf4; border-radius: 12rpx; background: #fff; color: #2264d1; }
.status-card { margin-top: 22rpx; }
.status-head { display: flex; align-items: center; gap: 18rpx; }
.status-dot { width: 22rpx; height: 22rpx; flex: 0 0 auto; border-radius: 50%; background: #94a3b8; }
.status-dot.pending { background: #e0a12b; }
.status-dot.running { background: #2264d1; box-shadow: 0 0 0 8rpx #dce9ff; }
.status-dot.succeeded { background: #2d9b67; }
.status-dot.failed { background: #c84d64; }
.status-title { color: #111827; font-size: 29rpx; font-weight: 800; }
.status-note, .status-meta { margin-top: 8rpx; color: #687389; font-size: 24rpx; line-height: 1.5; }
.status-meta { margin-top: 22rpx; padding: 16rpx; border-radius: 10rpx; background: #f4f7fc; }
.job-error, .error-panel { margin-top: 18rpx; padding: 18rpx; border-radius: 12rpx; background: #fff1f3; color: #b4455b; font-size: 23rpx; line-height: 1.55; }
.confirm-overload, .result-link { margin-top: 24rpx; border-radius: 12rpx; }
.confirm-overload { background: #fff3df; color: #9a5a10; }
.result-link { background: #eaf2ff; color: #2264d1; }
</style>
