<template>
  <view class="checkin-page">
    <view class="hero">
      <view class="bubble one" />
      <view class="bubble two" />
      <view>
        <view class="kicker">今天也要轻轻闪光</view>
        <view class="date-text">{{ todayText }}</view>
        <view class="subtitle">完成一点点，也很了不起 · {{ todayStr }}</view>
      </view>
      <view class="streak-box">
        <view class="streak-num">{{ streak ?? 0 }}</view>
        <view class="streak-label">连胜天数</view>
      </view>
    </view>

    <view class="wallet-panel">
      <view>
        <view class="wallet-label">甜甜休息券</view>
        <view class="wallet-desc">完成学习后自动兑换，放心休息也有仪式感</view>
      </view>
      <view class="wallet-value">{{ slackBalance }}<text> 分钟</text></view>
    </view>

    <view class="motivation-panel" v-if="motivation">
      <view class="motivation-label">每日寄语</view>
      <view class="motivation-text">{{ motivation.text }}</view>
      <view class="motivation-source">{{ motivation.source }}</view>
    </view>

    <view class="progress-panel">
      <view class="progress-head">
        <view>
          <view class="progress-title">今日进度</view>
          <view class="progress-desc">{{ doneCount }}/{{ totalCount }} 个计划已完成</view>
        </view>
        <view class="progress-percent">{{ percent }}%</view>
      </view>
      <view class="progress-track"><view class="progress-fill" :style="{ width: percent + '%' }" /></view>
    </view>

    <view class="decision-panel" v-if="pendingTasks.length">
      <view class="decision-title">待处理学习</view>
      <view class="decision" v-for="task in pendingTasks" :key="task.id">
        <view>
          <view class="decision-name">{{ task.title }}</view>
          <view class="decision-meta">仍处于学习中，需结束或推迟</view>
        </view>
        <view class="decision-actions">
          <button @click="openCorrection(task, 'makeup')">补录</button>
          <button @click="openCorrection(task, 'postpone')">推迟</button>
        </view>
      </view>
    </view>

    <view class="toolbar">
      <view class="toolbar-title">今日小任务</view>
      <button class="mini-btn" @click="load">刷新</button>
    </view>

    <view class="empty" v-if="!loading && checkins.length === 0">
      <view class="empty-title">今天还没有小任务</view>
      <view class="empty-desc">先许下一个学习愿望，我来帮你把它拆成每天能完成的小步。</view>
      <button class="primary-btn" @click="goPlans">去种下计划</button>
    </view>

    <view class="list" v-else>
      <view
        class="item"
        v-for="item in checkins"
        :key="item.task_id"
        :class="{ done: item.completed, disabled: item.status === 'paused' }"
        @click="openDetail(item)"
      >
        <view class="state-dot"><view class="state-inner" /></view>
        <view class="item-main">
          <view class="item-title">{{ item.task.title || item.title }}</view>
          <view
            v-if="item.task.timer_state !== 'completed'"
            class="objective"
            :class="{ collapsed: !expandedTasks.has(item.task_id) }"
            @click.stop="toggleObjective(item.task_id)"
          >{{ item.task.objective || '暂未填写任务目标' }}</view>
          <view class="expand-link" v-if="item.task.timer_state !== 'completed' && item.task.objective" @click.stop="toggleObjective(item.task_id)">{{ expandedTasks.has(item.task_id) ? '收起' : '展开目标' }}</view>
          <view class="item-meta">{{ item.status === 'paused' ? '计划已暂停' : item.completed ? '今日已打卡' : statusText(item) }}</view>
          <view class="item-meta" v-if="item.task.timer_state !== 'completed'">计划 {{ item.task.planned_start || '--:--' }}-{{ item.task.planned_end || '--:--' }} · {{ timerText(item.task) }}</view>
          <view class="item-meta">累计 {{ durationText(accumulatedSeconds(item.task)) }}</view>
        </view>
        <view class="button-stack">
          <button class="task-btn" v-if="item.status !== 'paused' && item.task.timer_state === 'pending'" @click.stop="start(item)">开始</button>
          <button class="task-btn warn" v-if="item.status !== 'paused' && item.task.timer_state === 'running'" @click.stop="pause(item)">暂停</button>
          <button class="task-btn" v-if="item.status !== 'paused' && item.task.timer_state === 'paused'" @click.stop="resume(item)">继续</button>
          <button class="task-btn done-btn" v-if="item.status !== 'paused' && item.task.timer_state === 'achieved'" @click.stop="openCompletion(item, false)">完成</button>
          <button class="task-btn more" v-if="item.status !== 'paused' && item.task.timer_state !== 'completed'" @click.stop="openMore(item)">更多</button>
          <button class="task-btn checkin-btn" :disabled="!item.eligible || item.completed || item.status === 'paused'" @click.stop="checkin(item)">{{ checkinLabel(item) }}</button>
        </view>
      </view>
    </view>

    <view class="loading" v-if="loading">加载中...</view>

    <view class="modal" v-if="completionItem" @click.self="cancelCompletion">
      <view class="modal-body">
        <view class="modal-title">{{ completionEarly ? '结束本次学习' : '完成任务' }}</view>
        <view class="completion-summary">{{ completionItem.task.objective || '暂未填写任务目标' }}</view>
        <view class="completion-time">本次累计 {{ durationText(accumulatedSeconds(completionItem.task)) }}</view>
        <textarea class="reflection-input" v-model="reflection" maxlength="500" placeholder="写下今天的收获或下一步（可选）" />
        <view class="char-count">{{ reflection.length }}/500</view>
        <button class="submit" @click="submitCompletion(true)">保存心得并完成</button>
        <button class="skip" @click="submitCompletion(false)">跳过，直接完成</button>
        <button class="cancel" @click="cancelCompletion">取消</button>
      </view>
    </view>

    <view class="modal" v-if="correctionItem" @click.self="closeCorrection">
      <view class="modal-body">
        <view class="modal-title">{{ correctionMode === 'makeup' ? '补录学习' : '推迟任务' }}</view>
        <view class="picker-grid">
          <view class="picker-field"><text>日期</text><picker mode="date" :value="correction.date" @change="setCorrection('date', $event)"><view class="picker-value">{{ correction.date }}</view></picker></view>
          <view class="picker-field"><text>开始</text><picker mode="time" :value="correction.start" @change="setCorrection('start', $event)"><view class="picker-value">{{ correction.start }}</view></picker></view>
          <view class="picker-field"><text>结束</text><picker mode="time" :value="correction.end" @change="setCorrection('end', $event)"><view class="picker-value">{{ correction.end }}</view></picker></view>
        </view>
        <view class="correction-summary">{{ correction.date }} {{ correction.start }}-{{ correction.end }}</view>
        <button class="submit" @click="submitCorrection()">确认{{ correctionMode === 'makeup' ? '补录' : '推迟' }}</button>
        <button class="cancel" @click="closeCorrection">取消</button>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { onHide, onShow } from '@dcloudio/uni-app'
import { CheckinApi, MotivationApi, SlackApi, StudyTaskApi, type CheckinInfo, type DailyTask, type Motivation, type TimerTask } from '@/api'
import { addLocalDays, localDateKey } from '@/utils/date'

const todayStr = ref('')
const todayText = ref('')
const checkins = ref<CheckinInfo[]>([])
const loading = ref(false)
const streak = ref<number | null>(null)
const slackBalance = ref(0)
const pendingTasks = ref<DailyTask[]>([])
const motivation = ref<Motivation | null>(null)
const now = ref(Date.now())
const snapshotAt = ref(Date.now())
const expandedTasks = ref(new Set<number>())
const completionItem = ref<CheckinInfo | null>(null)
const completionEarly = ref(false)
const reflection = ref('')
const correctionItem = ref<DailyTask | null>(null)
const correctionMode = ref<'makeup' | 'postpone'>('postpone')
const correction = ref({ date: '', start: '20:00', end: '21:00' })
let ticker: ReturnType<typeof setInterval> | null = null
const doneCount = computed(() => checkins.value.filter(c => c.completed).length)
const totalCount = computed(() => checkins.value.length)
const percent = computed(() => totalCount.value === 0 ? 0 : Math.round((doneCount.value / totalCount.value) * 100))

async function load() {
  loading.value = true
  try {
    const [list, s, slack, message, pending] = await Promise.all([
      CheckinApi.listByDate(todayStr.value),
      CheckinApi.streak().catch(() => null),
      SlackApi.balance().catch(() => null),
      MotivationApi.daily(todayStr.value).catch(() => null),
      StudyTaskApi.pendingDecision(todayStr.value).catch(() => []),
    ])
    checkins.value = list || []
    pendingTasks.value = pending
    motivation.value = message
    snapshotAt.value = Date.now()
    expandedTasks.value = new Set(checkins.value.filter(item => item.task.timer_state === 'running').map(item => item.task_id))
    if (s) streak.value = s.streak
    if (slack) slackBalance.value = slack.balance
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

async function checkin(item: CheckinInfo) {
  if (item.status === 'paused') {
    uni.showToast({ title: '该计划已暂停', icon: 'none' })
    return
  }
  try {
    await CheckinApi.toggle({ plan_id: item.plan_id, date: todayStr.value, completed: true })
    await load()
    uni.showToast({ title: '今日打卡完成', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '打卡失败', icon: 'none' })
  }
}

async function start(item: CheckinInfo) {
  try {
    await StudyTaskApi.start(item.task_id)
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '开始失败', icon: 'none' })
  }
}

async function pause(item: CheckinInfo) {
  try {
    await StudyTaskApi.pause(item.task_id)
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '暂停失败', icon: 'none' })
  }
}

async function resume(item: CheckinInfo) {
  try {
    await StudyTaskApi.resume(item.task_id)
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '继续失败', icon: 'none' })
  }
}

function openCompletion(item: CheckinInfo, early: boolean) {
  completionItem.value = item
  completionEarly.value = early
  reflection.value = ''
}

function cancelCompletion() {
  completionItem.value = null
  reflection.value = ''
}

async function submitCompletion(saveReflection: boolean) {
  const item = completionItem.value
  if (!item) return
  try {
    const note = saveReflection ? reflection.value.trim() : undefined
    if (completionEarly.value) await StudyTaskApi.stop(item.task_id, note)
    else await StudyTaskApi.complete(item.task_id, note)
    cancelCompletion()
    await load()
    uni.showToast({ title: '任务已完成', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '完成失败', icon: 'none' })
  }
}

function statusText(item: CheckinInfo) {
  const labels = { pending: '等待开始', running: '学习中', paused: '已暂停', achieved: '目标已达成', completed: '已完成' }
  return labels[item.task.timer_state]
}

function checkinLabel(item: CheckinInfo) {
  if (item.completed) return '今日已打卡'
  if (item.remaining_tasks > 0) return `还剩 ${item.remaining_tasks} 项`
  return '完成今日打卡'
}

function accumulatedSeconds(task: TimerTask) {
  return task.accumulated_seconds + (task.timer_state === 'running' ? Math.max(0, Math.floor((now.value - snapshotAt.value) / 1000)) : 0)
}

function timerText(task: TimerTask) {
  const remaining = task.target_minutes * 60 - accumulatedSeconds(task)
  return remaining >= 0 ? `剩余 ${durationText(remaining)}` : `超时 ${durationText(-remaining)}`
}

function durationText(seconds: number) {
  const safe = Math.max(0, Math.floor(seconds))
  const hours = Math.floor(safe / 3600)
  const minutes = Math.floor((safe % 3600) / 60)
  const secs = safe % 60
  return hours ? `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}` : `${minutes}:${String(secs).padStart(2, '0')}`
}

function toggleObjective(id: number) {
  const next = new Set(expandedTasks.value)
  next.has(id) ? next.delete(id) : next.add(id)
  expandedTasks.value = next
}

function openMore(item: CheckinInfo) {
  uni.showActionSheet({
    itemList: ['结束本次学习', '推迟任务', '补录学习', '查看详情'],
    success: ({ tapIndex }) => {
      if (tapIndex === 0) openCompletion(item, item.task.timer_state !== 'achieved')
      if (tapIndex === 1) openCorrection(item.task, 'postpone')
      if (tapIndex === 2) openCorrection(item.task, 'makeup')
      if (tapIndex === 3) openDetail(item)
    },
  })
}

function openCorrection(task: DailyTask, mode: 'makeup' | 'postpone') {
  correctionItem.value = task
  correctionMode.value = mode
  correction.value = {
    date: mode === 'postpone' ? localDateKey(addLocalDays(new Date(), 1)) : task.date,
    start: task.planned_start || '20:00',
    end: task.planned_end || '21:00',
  }
}

function setCorrection(field: 'date' | 'start' | 'end', event: any) {
  correction.value[field] = event.detail.value
}

function closeCorrection() { correctionItem.value = null }

async function submitCorrection(confirmConflict = false) {
  const task = correctionItem.value
  if (!task) return
  const value = correction.value
  try {
    if (correctionMode.value === 'makeup') {
      await StudyTaskApi.makeup(task.id, { actual_date: value.date, actual_start: `${value.date} ${value.start}`, actual_end: `${value.date} ${value.end}`, reason: '手动补录' })
    } else {
      await StudyTaskApi.postpone(task.id, { date: value.date, planned_start: value.start, planned_end: value.end, reason: '手动推迟', confirm_conflict: confirmConflict })
    }
    closeCorrection()
    await load()
  } catch (e: any) {
    if (correctionMode.value === 'postpone' && e?.code === 409 && !confirmConflict) {
      uni.showModal({ title: '时间冲突', content: '目标时段已有任务，仍要推迟吗？', success: result => { if (result.confirm) submitCorrection(true) } })
      return
    }
    uni.showToast({ title: e?.message || '操作失败', icon: 'none' })
  }
}

function goPlans() {
  uni.switchTab({ url: '/pages/plans/plans' })
}

function openDetail(item: CheckinInfo) { uni.navigateTo({ url: `/pages/task/task?id=${item.task_id}` }) }
function startTicker() {
  if (ticker) clearInterval(ticker)
  ticker = setInterval(() => { now.value = Date.now() }, 1000)
}
function stopTicker() { if (ticker) clearInterval(ticker); ticker = null }
function refreshToday() {
  const today = new Date()
  todayStr.value = localDateKey(today)
  todayText.value = `${today.getMonth() + 1}月${today.getDate()}日`
}
onShow(() => { refreshToday(); now.value = Date.now(); startTicker(); load() })
onHide(stopTicker)
onUnmounted(stopTicker)
</script>

<style lang="scss">
.checkin-page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 28rpx 28rpx 60rpx;
  background: linear-gradient(180deg, #fff0f7 0%, #fffaf0 42%, #f7fbff 100%);
}
.hero {
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 34rpx 34rpx;
  border-radius: 34rpx;
  background: linear-gradient(135deg, #ff8fab 0%, #ffc36a 100%);
  color: #fff;
  box-shadow: 0 20rpx 44rpx rgba(255, 143, 171, .28);
}
.bubble { position: absolute; border-radius: 999rpx; background: rgba(255,255,255,.28); }
.bubble.one { width: 170rpx; height: 170rpx; right: -52rpx; top: -60rpx; }
.bubble.two { width: 96rpx; height: 96rpx; left: 300rpx; bottom: -42rpx; }
.kicker { position: relative; display: inline-flex; padding: 8rpx 18rpx; border-radius: 999rpx; background: rgba(255,255,255,.24); font-size: 22rpx; font-weight: 800; }
.date-text { font-size: 44rpx; font-weight: 800; }
.subtitle { margin-top: 8rpx; color: rgba(255,255,255,.88); font-size: 24rpx; }
.streak-box {
  width: 150rpx;
  padding: 18rpx 0;
  border-radius: 28rpx;
  background: rgba(255, 255, 255, 0.26);
  text-align: center;
}
.streak-num { font-size: 44rpx; font-weight: 800; }
.streak-label { color: rgba(255,255,255,.86); font-size: 22rpx; }
.wallet-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 18rpx;
  padding: 26rpx 30rpx;
  border-radius: 28rpx;
  background: #fff;
  border: 1rpx solid #ffe0ea;
  box-shadow: 0 14rpx 32rpx rgba(255, 143, 171, .12);
}
.wallet-label { color: #4b2b3f; font-size: 29rpx; font-weight: 800; }
.wallet-desc { margin-top: 8rpx; color: #7b8498; font-size: 22rpx; }
.wallet-value { color: #ff6f91; font-size: 38rpx; font-weight: 900; }
.wallet-value text { font-size: 22rpx; font-weight: 600; }
.motivation-panel { position: relative; margin-top: 18rpx; padding: 24rpx 30rpx; border-radius: 28rpx; background: #fffdf7; border: 1rpx solid #ffe4af; }
.motivation-label { color: #c87818; font-size: 22rpx; font-weight: 800; }
.motivation-text { margin-top: 8rpx; color: #4b2b3f; font-size: 27rpx; line-height: 1.5; display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; }
.motivation-source { margin-top: 6rpx; color: #9a7a55; font-size: 21rpx; text-align: right; }
.progress-panel {
  margin-top: 24rpx;
  padding: 30rpx;
  border-radius: 28rpx;
  background: #fff;
  border: 1rpx solid #ffe0ea;
  box-shadow: 0 14rpx 32rpx rgba(255, 180, 92, .12);
}
.progress-head { display: flex; align-items: center; justify-content: space-between; }
.progress-title { color: #4b2b3f; font-size: 31rpx; font-weight: 800; }
.progress-desc { margin-top: 8rpx; color: #7b8498; font-size: 24rpx; }
.progress-percent { color: #ff7aa2; font-size: 42rpx; font-weight: 900; }
.progress-track { margin-top: 28rpx; height: 16rpx; border-radius: 99rpx; background: #edf2f8; overflow: hidden; }
.progress-fill { height: 100%; border-radius: 99rpx; background: linear-gradient(90deg, #ff8fab, #ffc36a); transition: width .2s ease; }
.decision-panel { margin-top: 20rpx; padding: 26rpx; border-radius: 16rpx; background: #fff7e6; border: 1rpx solid #ffe1a6; }
.decision-title { color: #7a4b00; font-size: 29rpx; font-weight: 800; margin-bottom: 14rpx; }
.decision { display: flex; align-items: center; justify-content: space-between; gap: 16rpx; padding: 14rpx 0; }
.decision-name { color: #111827; font-size: 26rpx; font-weight: 700; }
.decision-meta { margin-top: 6rpx; color: #9a5b00; font-size: 22rpx; }
.decision-actions { display: flex; gap: 8rpx; width: 170rpx; }
.decision-actions button { margin: 0; flex: 1; height: 52rpx; line-height: 52rpx; border-radius: 8rpx; background: #fff; color: #9a5b00; font-size: 21rpx; padding: 0; }
.toolbar { display: flex; align-items: center; justify-content: space-between; margin: 34rpx 4rpx 18rpx; }
.toolbar-title { color: #4b2b3f; font-size: 32rpx; font-weight: 900; }
.mini-btn { margin: 0; width: 120rpx; height: 56rpx; line-height: 56rpx; border-radius: 999rpx; background: #fff0f6; color: #ff6f91; font-size: 24rpx; }
.list { display: flex; flex-direction: column; gap: 18rpx; }
.item {
  display: flex;
  align-items: center;
  min-height: 118rpx;
  padding: 0 26rpx;
  border-radius: 26rpx;
  background: #fff;
  border: 1rpx solid #ffe0ea;
  box-shadow: 0 10rpx 26rpx rgba(255, 143, 171, .10);
}
.item.done { border-color: #ffc7d8; background: #fff7fb; }
.item.disabled { opacity: .55; }
.state-dot {
  width: 38rpx;
  height: 38rpx;
  border-radius: 50%;
  border: 3rpx solid #cbd5e1;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 24rpx;
}
.done .state-dot { border-color: #ff7aa2; }
.state-inner { width: 18rpx; height: 18rpx; border-radius: 50%; background: transparent; }
.done .state-inner { background: #ff7aa2; }
.item-main { flex: 1; min-width: 0; }
.item-title { color: #111827; font-size: 30rpx; font-weight: 700; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.objective { margin-top: 10rpx; color: #4b5568; font-size: 24rpx; line-height: 1.45; }
.objective.collapsed { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; }
.expand-link { margin-top: 4rpx; color: #ff6f91; font-size: 21rpx; }
.item-meta { margin-top: 8rpx; color: #7b8498; font-size: 23rpx; }
.state-text { color: #ff6f91; font-size: 25rpx; font-weight: 700; }
.button-stack { display: flex; flex-direction: column; gap: 8rpx; width: 112rpx; }
.task-btn {
  margin: 0;
  height: 48rpx;
  line-height: 48rpx;
  border-radius: 999rpx;
  background: #fff0f6;
  color: #ff6f91;
  font-size: 22rpx;
  padding: 0;
}
.task-btn.warn { background: #fff7e6; color: #9a5b00; }
.task-btn.done-btn { background: linear-gradient(135deg, #ff7aa2, #ffb45c); color: #fff; }
.task-btn.more { background: #f3f6fb; color: #606a80; }
.task-btn.checkin-btn { background: #4da982; color: #fff; }
.empty { margin-top: 22rpx; padding: 56rpx 34rpx; border-radius: 28rpx; background: #fff; border: 1rpx solid #ffe0ea; box-shadow: 0 14rpx 32rpx rgba(255, 143, 171, .10); }
.empty-title { color: #111827; font-size: 32rpx; font-weight: 800; }
.empty-desc { margin-top: 12rpx; color: #7b8498; font-size: 26rpx; line-height: 1.5; }
.primary-btn { margin-top: 32rpx; background: linear-gradient(135deg, #ff7aa2, #ffb45c); color: #fff; border-radius: 999rpx; }
.loading { text-align: center; color: #7b8498; margin-top: 34rpx; }
.modal { position: fixed; inset: 0; z-index: 99; display: flex; align-items: flex-end; background: rgba(17,24,39,.46); }
.modal-body { width: 100%; box-sizing: border-box; padding: 34rpx 30rpx 42rpx; border-radius: 28rpx 28rpx 0 0; background: #fff; }
.modal-title { color: #111827; font-size: 34rpx; font-weight: 800; }
.completion-summary, .completion-time, .correction-summary { margin-top: 14rpx; color: #606a80; font-size: 24rpx; line-height: 1.5; }
.reflection-input { width: 100%; height: 180rpx; box-sizing: border-box; margin-top: 22rpx; padding: 18rpx; border-radius: 16rpx; background: #f8fafc; font-size: 26rpx; }
.char-count { margin: 8rpx 0 18rpx; color: #8a92a6; font-size: 22rpx; text-align: right; }
.submit, .skip, .cancel { margin-top: 14rpx; border-radius: 999rpx; font-size: 27rpx; }
.submit { background: linear-gradient(135deg, #ff7aa2, #ffb45c); color: #fff; }
.skip { background: #fff0f6; color: #ff6f91; }
.cancel { background: #f3f6fb; color: #606a80; }
.picker-grid { display: grid; grid-template-columns: 1.4fr 1fr 1fr; gap: 12rpx; margin-top: 24rpx; }
.picker-field text { display: block; margin-bottom: 8rpx; color: #7b8498; font-size: 22rpx; }
.picker-value { padding: 20rpx 10rpx; border-radius: 12rpx; background: #f8fafc; color: #111827; font-size: 24rpx; text-align: center; }
</style>
