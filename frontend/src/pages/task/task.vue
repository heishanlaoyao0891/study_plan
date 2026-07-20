<template>
  <view class="page">
    <view class="panel" v-if="detail">
      <view class="title">{{ detail.task.title }}</view>
      <view class="meta">{{ detail.plan.title }}</view>
      <view class="grid">
        <view><text>日期</text><view class="value">{{ detail.task.date }}</view></view>
        <view><text>计划时段</text><view class="value">{{ detail.task.planned_start || '--:--' }}-{{ detail.task.planned_end || '--:--' }}</view></view>
        <view><text>状态</text><view class="value">{{ statusText(detail.task.status) }}</view></view>
        <view><text>学习分钟</text><view class="value">{{ detail.task.study_minutes || 0 }}</view></view>
      </view>
      <view class="desc" v-if="detail.task.description">{{ detail.task.description }}</view>
      <view class="visibility-row">
        <view>小组可见</view>
        <button @click="togglePublic">{{ detail.task.public_to_group ? '已公开' : '未公开' }}</button>
      </view>
      <view class="actions">
        <button @click="start" v-if="detail.task.status !== 'in_progress' && detail.task.status !== 'completed'">开始</button>
        <button @click="complete" v-if="detail.task.status !== 'completed'">完成</button>
        <button @click="postpone" v-if="detail.task.status !== 'completed'">推迟</button>
        <button @click="makeup">补录</button>
      </view>
    </view>

    <view class="panel" v-if="detail?.history?.length">
      <view class="section-title">推迟记录</view>
      <view class="history" v-for="row in detail.history" :key="row.id">
        <view>{{ row.old_date }} -> {{ row.new_date }}</view>
        <text>{{ row.reason || '-' }}</text>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { StudyTaskApi } from '@/api'

const taskId = ref(0)
const detail = ref<any>(null)

onLoad((query: any) => { taskId.value = Number(query?.id || 0) })
onShow(load)

async function load() {
  if (!taskId.value) return
  detail.value = await StudyTaskApi.get(taskId.value)
}
async function start() { await StudyTaskApi.start(taskId.value); await load() }
async function complete() { await StudyTaskApi.complete(taskId.value); await load() }
async function postpone() {
  const task = detail.value.task
  const res = await modalInput('推迟任务', `${task.date} ${task.planned_start || '20:00'}-${task.planned_end || '21:00'}`)
  if (!res) return
  const parsed = parsePostpone(res)
  try {
    await StudyTaskApi.postpone(taskId.value, parsed.date, '手动推迟', parsed.start, parsed.end)
    await load()
  } catch (e: any) {
    if (e?.code === 409) {
      const ok = await confirmConflict()
      if (!ok) return
      await StudyTaskApi.postpone(taskId.value, parsed.date, '手动推迟', parsed.start, parsed.end, true)
      await load()
    } else {
      throw e
    }
  }
}
async function makeup() {
  const task = detail.value.task
  const res = await modalInput('补录结束时间', `${task.date} ${task.planned_start || '20:00'}-${task.planned_end || '21:00'}`)
  if (!res) return
  const parsed = parseMakeup(task, res)
  await StudyTaskApi.makeup(taskId.value, parsed.end, '手动补录', parsed.start)
  await load()
}
async function togglePublic() {
  const current = !!detail.value.task.public_to_group
  await StudyTaskApi.update(taskId.value, { public_to_group: !current })
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
function parseMakeup(task: any, value: string) {
  const raw = value.trim()
  if (raw.includes('-')) {
    const [date, range = task.planned_end || '21:00-22:00'] = raw.split(/\s+/)
    const [startTime = task.planned_start || '20:00', endTime = task.planned_end || '21:00'] = range.split('-')
    return { start: `${date} ${startTime}`, end: `${date} ${endTime}` }
  }
  return { start: `${task.date} ${task.planned_start || '20:00'}`, end: raw }
}
function confirmConflict() {
  return new Promise<boolean>(resolve => {
    uni.showModal({ title: '时间冲突', content: '目标时段已存在其他任务，仍然推迟吗？', success: res => resolve(!!res.confirm) })
  })
}
function statusText(status: string) { return status === 'completed' ? '已完成' : status === 'in_progress' ? '学习中' : '待执行' }
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel { margin-bottom: 20rpx; padding: 30rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.meta { margin-top: 8rpx; color: #606a80; font-size: 24rpx; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14rpx; margin-top: 24rpx; }
.grid view { padding: 18rpx; border-radius: 12rpx; background: #f8fafc; }
.grid text { display: block; color: #7b8498; font-size: 22rpx; }
.value { margin-top: 8rpx; color: #111827; font-size: 26rpx; font-weight: 800; }
.desc { margin-top: 22rpx; color: #606a80; font-size: 25rpx; line-height: 1.5; }
.visibility-row { display: flex; justify-content: space-between; align-items: center; margin-top: 22rpx; padding: 18rpx; border-radius: 12rpx; background: #f8fafc; color: #111827; font-size: 25rpx; font-weight: 700; }
.visibility-row button { margin: 0; height: 54rpx; line-height: 54rpx; border-radius: 10rpx; background: #eef4ff; color: #2264d1; font-size: 23rpx; }
.actions { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10rpx; margin-top: 24rpx; }
.actions button { margin: 0; height: 60rpx; line-height: 60rpx; border-radius: 10rpx; background: #eef4ff; color: #2264d1; font-size: 23rpx; }
.section-title { color: #111827; font-size: 28rpx; font-weight: 800; margin-bottom: 12rpx; }
.history { padding: 14rpx 0; border-top: 1rpx solid #eef2f7; color: #384257; font-size: 24rpx; }
.history text { display: block; margin-top: 6rpx; color: #7b8498; }
</style>
