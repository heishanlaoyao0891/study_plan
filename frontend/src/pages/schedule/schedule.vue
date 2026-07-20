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
          <view class="task-meta">{{ task.planned_start || '--:--' }}-{{ task.planned_end || '--:--' }} · {{ statusText(task.status) }} · {{ task.study_minutes || 0 }} 分钟</view>
          <view class="task-plan">{{ task.plan_title }}</view>
        </view>
        <view class="actions">
          <button @click="start(task)" v-if="task.status !== 'in_progress' && task.status !== 'completed'">开始</button>
          <button @click="complete(task)" v-if="task.status !== 'completed'">完成</button>
          <button @click="postpone(task)" v-if="task.status !== 'completed'">推迟</button>
          <button @click="makeup(task)">补录</button>
        </view>
      </view>
    </view>

    <view class="loading" v-if="loading">加载中...</view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { PlanApi, StudyTaskApi } from '@/api'

const loading = ref(false)
const tasks = ref<any[]>([])
const today = new Date()
const days = Array.from({ length: 7 }, (_, index) => dateKey(new Date(today.getTime() + index * 86400000)))
const tasksByDate = computed(() => {
  const map: Record<string, any[]> = {}
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
      return rows.map((task: any) => ({ ...task, plan_title: plan.title, plan_status: plan.status }))
    }))
    tasks.value = all.flat().filter(task => days.includes(task.date))
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

function openTask(task: any) { uni.navigateTo({ url: `/pages/task/task?id=${task.id}` }) }
async function start(task: any) { await StudyTaskApi.start(task.id); await load() }
async function complete(task: any) { await StudyTaskApi.complete(task.id); await load() }
async function postpone(task: any) {
  const tomorrow = dateKey(new Date(Date.now() + 86400000))
  const res = await modalInput('推迟任务', `${tomorrow} ${task.planned_start || '20:00'}-${task.planned_end || '21:00'}`)
  if (!res) return
  const parsed = parsePostpone(res)
  await StudyTaskApi.postpone(task.id, parsed.date, '手动推迟', parsed.start, parsed.end)
  await load()
}
async function makeup(task: any) {
  const res = await modalInput('补录结束时间', `${task.date} ${task.planned_end || '21:00'}`)
  if (!res) return
  await StudyTaskApi.makeup(task.id, res, '手动补录')
  await load()
}

function modalInput(title: string, placeholderText: string) {
  return new Promise<string | null>(resolve => {
    uni.showModal({ title, editable: true, placeholderText, success: res => resolve(res.confirm ? (res.content || placeholderText) : null) })
  })
}
function parsePostpone(value: string) {
  const [date, range = '20:00-21:00'] = value.trim().split(/\s+/)
  const [start = '20:00', end = '21:00'] = range.split('-')
  return { date, start, end }
}
function dateKey(date: Date) { return date.toISOString().slice(0, 10) }
function dayLabel(day: string) { return day === days[0] ? `${day} 今天` : day }
function statusText(status: string) { return status === 'completed' ? '已完成' : status === 'in_progress' ? '学习中' : '待执行' }
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
.actions { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10rpx; }
.actions button { margin: 0; height: 58rpx; line-height: 58rpx; border-radius: 10rpx; background: #eef4ff; color: #2264d1; font-size: 23rpx; }
.loading { margin-top: 24rpx; color: #7b8498; text-align: center; }
</style>
