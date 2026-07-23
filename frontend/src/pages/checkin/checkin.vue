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
          <button @click="makeup(task)">补录</button>
          <button @click="postpone(task)">推迟</button>
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
        :key="item.plan_id"
        :class="{ done: item.completed, disabled: item.status === 'paused' }"
      >
        <view class="state-dot"><view class="state-inner" /></view>
        <view class="item-main">
          <view class="item-title">{{ item.title }}</view>
          <view class="item-meta">{{ item.status === 'paused' ? '计划已暂停' : item.completed ? '今日已完成' : statusText(item) }}</view>
          <view class="item-meta">已学习 {{ item.study_minutes || 0 }} 分钟</view>
        </view>
        <view class="button-stack" v-if="item.status !== 'paused'">
          <button class="task-btn" v-if="item.task_status !== 'in_progress' && !item.completed" @click.stop="start(item)">开始</button>
          <button class="task-btn warn" v-if="item.task_status === 'in_progress'" @click.stop="stop(item)">结束</button>
          <button class="task-btn done-btn" v-if="!item.completed" @click.stop="complete(item)">完成</button>
        </view>
        <view class="state-text" v-else>暂停</view>
      </view>
    </view>

    <view class="loading" v-if="loading">加载中...</view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { CheckinApi, SlackApi, StudyTaskApi, type CheckinInfo } from '@/api'

const today = new Date()
const todayStr = today.toISOString().slice(0, 10)
const todayText = `${today.getMonth() + 1}月${today.getDate()}日`
const checkins = ref<CheckinInfo[]>([])
const loading = ref(false)
const streak = ref<number | null>(null)
const slackBalance = ref(0)
const pendingTasks = ref<any[]>([])
const doneCount = computed(() => checkins.value.filter(c => c.completed).length)
const totalCount = computed(() => checkins.value.length)
const percent = computed(() => totalCount.value === 0 ? 0 : Math.round((doneCount.value / totalCount.value) * 100))

async function load() {
  loading.value = true
  try {
    const [list, s, slack] = await Promise.all([
      CheckinApi.listByDate(todayStr),
      CheckinApi.streak().catch(() => null),
      SlackApi.balance().catch(() => null),
    ])
    checkins.value = list || []
    pendingTasks.value = await StudyTaskApi.pendingDecision(todayStr).catch(() => [])
    if (s) streak.value = s.streak
    if (slack) slackBalance.value = slack.balance
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

async function toggle(item: CheckinInfo) {
  if (item.status === 'paused') {
    uni.showToast({ title: '该计划已暂停', icon: 'none' })
    return
  }
  try {
    await CheckinApi.toggle({ plan_id: item.plan_id, date: todayStr, completed: !item.completed })
    item.completed = !item.completed
    const s = await CheckinApi.streak().catch(() => null)
    if (s) streak.value = s.streak
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

async function stop(item: CheckinInfo) {
  try {
    await StudyTaskApi.stop(item.task_id)
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '结束失败', icon: 'none' })
  }
}

async function complete(item: CheckinInfo) {
  try {
    await StudyTaskApi.complete(item.task_id)
    await load()
    uni.showToast({ title: '完成并累积躺平时间', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '完成失败', icon: 'none' })
  }
}

async function makeup(task: any) {
  const res = await new Promise<any>(resolve => {
    uni.showModal({ title: '补录结束时间', editable: true, placeholderText: `${todayStr} ${task.planned_start || '20:00'}-${task.planned_end || '21:00'}`, success: resolve })
  })
  if (!res.confirm) return
  try {
    const parsed = parseMakeup(task, res.content || `${todayStr} ${task.planned_start || '20:00'}-${task.planned_end || '21:00'}`)
    await StudyTaskApi.makeup(task.id, parsed.end, '手动补录', parsed.start)
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '补录失败', icon: 'none' })
  }
}

async function postpone(task: any) {
  const tomorrow = new Date(Date.now() + 86400000).toISOString().slice(0, 10)
  const res = await new Promise<any>(resolve => {
    uni.showModal({ title: '推迟任务', editable: true, placeholderText: tomorrow, success: resolve })
  })
  if (!res.confirm) return
  try {
    await StudyTaskApi.postpone(task.id, res.content || tomorrow, '手动推迟')
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '推迟失败', icon: 'none' })
  }
}

function statusText(item: CheckinInfo) {
  return item.task_status === 'in_progress' ? '学习中' : item.study_minutes > 0 ? '已记录学习' : '等待开始'
}

function parseMakeup(task: any, value: string) {
  const raw = value.trim()
  if (raw.includes('-')) {
    const [date, range = task.planned_end || '21:00-22:00'] = raw.split(/\s+/)
    const [startTime = task.planned_start || '20:00', endTime = task.planned_end || '21:00'] = range.split('-')
    return { start: `${date} ${startTime}`, end: `${date} ${endTime}` }
  }
  return { start: `${task.date} ${task.planned_start || '20:00'}`, end: raw }
}

function goPlans() {
  uni.switchTab({ url: '/pages/plans/plans' })
}

onMounted(load)
onShow(load)
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
.empty { margin-top: 22rpx; padding: 56rpx 34rpx; border-radius: 28rpx; background: #fff; border: 1rpx solid #ffe0ea; box-shadow: 0 14rpx 32rpx rgba(255, 143, 171, .10); }
.empty-title { color: #111827; font-size: 32rpx; font-weight: 800; }
.empty-desc { margin-top: 12rpx; color: #7b8498; font-size: 26rpx; line-height: 1.5; }
.primary-btn { margin-top: 32rpx; background: linear-gradient(135deg, #ff7aa2, #ffb45c); color: #fff; border-radius: 999rpx; }
.loading { text-align: center; color: #7b8498; margin-top: 34rpx; }
</style>
