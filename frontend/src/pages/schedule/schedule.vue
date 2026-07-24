<template>
  <view class="page">
    <view class="hero">
      <view class="title">任务日程</view>
      <view class="subtitle">{{ days[0] }} - {{ days[days.length - 1] }}</view>
    </view>

    <view class="day" v-for="day in days" :key="day">
      <view class="day-title">{{ dayLabel(day) }}</view>
      <view class="empty" v-if="!tasksByDate[day]?.length">暂无任务</view>
      <view class="task" v-for="task in tasksByDate[day] || []" :key="task.id">
        <view class="task-main" @click="openTask(task)">
          <view class="task-title">{{ task.title }}</view>
          <view class="task-meta">{{ task.planned_start || '--:--' }}-{{ task.planned_end || '--:--' }} · {{ statusText(task.timer_state) }} · {{ task.study_minutes || 0 }} 分钟</view>
          <view class="task-plan">{{ task.plan_title }}</view>
        </view>
        <view class="actions">
          <button @click="start(task)" v-if="task.timer_state === 'pending'">开始</button>
          <button @click="pause(task)" v-if="task.timer_state === 'running'">暂停</button>
          <button @click="resume(task)" v-if="task.timer_state === 'paused'">继续</button>
          <button @click="complete(task)" v-if="task.timer_state === 'achieved'">完成</button>
          <button @click="openTask(task)">更多</button>
        </view>
      </view>
    </view>

    <view class="loading" v-if="loading">加载中...</view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { PlanApi, StudyTaskApi, type TimerTask } from '@/api'
import { addLocalDays, localDateKey } from '@/utils/date'

const loading = ref(false)
type ScheduleTask = TimerTask & { plan_title: string; plan_status: string }
const tasks = ref<ScheduleTask[]>([])
const today = new Date()
const days = Array.from({ length: 7 }, (_, index) => localDateKey(addLocalDays(today, index)))
const tasksByDate = computed(() => {
  const map: Record<string, ScheduleTask[]> = {}
  for (const day of days) map[day] = []
  for (const task of tasks.value) {
    if (map[task.date]) map[task.date].push(task)
  }
  for (const day of days) map[day].sort((a, b) => `${a.planned_start || ''}${a.id}`.localeCompare(`${b.planned_start || ''}${b.id}`))
  return map
})

async function load() {
  loading.value = true
  try {
    const plans = await PlanApi.list()
    const all = await Promise.all(plans.map(async plan => {
      const rows = await PlanApi.tasks(plan.id).catch(() => [])
      return rows.map(task => ({ ...task, plan_title: plan.title, plan_status: plan.status }))
    }))
    tasks.value = all.flat().filter(task => days.includes(task.date))
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

function openTask(task: TimerTask) { uni.navigateTo({ url: `/pages/task/task?id=${task.id}` }) }
async function start(task: TimerTask) {
  try { await StudyTaskApi.start(task.id); await load() }
  catch (e: any) { uni.showToast({ title: e?.message || '开始失败', icon: 'none' }) }
}
async function pause(task: TimerTask) {
  try { await StudyTaskApi.pause(task.id); await load() }
  catch (e: any) { uni.showToast({ title: e?.message || '暂停失败', icon: 'none' }) }
}
async function resume(task: TimerTask) {
  try { await StudyTaskApi.resume(task.id); await load() }
  catch (e: any) { uni.showToast({ title: e?.message || '继续失败', icon: 'none' }) }
}
async function complete(task: TimerTask) {
  try { await StudyTaskApi.complete(task.id); await load() }
  catch (e: any) { uni.showToast({ title: e?.message || '完成失败', icon: 'none' }) }
}
function dayLabel(day: string) { return day === days[0] ? `${day} 今天` : day }
function statusText(status: string) { return ({ pending: '待执行', running: '学习中', paused: '已暂停', achieved: '目标已达成', completed: '已完成' } as Record<string, string>)[status] || status }
onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.hero, .day { padding: 30rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 36rpx; font-weight: 800; }
.subtitle { margin-top: 8rpx; color: #7b8498; font-size: 24rpx; }
.day { margin-top: 20rpx; }
.day-title { color: #111827; font-size: 30rpx; font-weight: 800; margin-bottom: 14rpx; }
.empty { color: #8a92a6; font-size: 24rpx; padding: 12rpx 0; }
.task { padding: 20rpx 0; border-top: 1rpx solid #eef2f7; }
.task-main { margin-bottom: 16rpx; }
.task-title { color: #111827; font-size: 28rpx; font-weight: 800; }
.task-meta, .task-plan { margin-top: 8rpx; color: #606a80; font-size: 23rpx; }
.actions { display: flex; gap: 10rpx; }
.actions button { flex: 1; }
.actions button { margin: 0; height: 58rpx; line-height: 58rpx; border-radius: 10rpx; background: #eef4ff; color: #2264d1; font-size: 23rpx; }
.loading { margin-top: 24rpx; color: #7b8498; text-align: center; }
</style>
