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
  await StudyTaskApi.postpone(taskId.value, parsed.date, '手动推迟', parsed.start, parsed.end)
  await load()
}
async function makeup() {
  const task = detail.value.task
  const res = await modalInput('补录结束时间', `${task.date} ${task.planned_end || '21:00'}`)
  if (!res) return
  await StudyTaskApi.makeup(taskId.value, res, '手动补录')
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
.actions { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10rpx; margin-top: 24rpx; }
.actions button { margin: 0; height: 60rpx; line-height: 60rpx; border-radius: 10rpx; background: #eef4ff; color: #2264d1; font-size: 23rpx; }
.section-title { color: #111827; font-size: 28rpx; font-weight: 800; margin-bottom: 12rpx; }
.history { padding: 14rpx 0; border-top: 1rpx solid #eef2f7; color: #384257; font-size: 24rpx; }
.history text { display: block; margin-top: 6rpx; color: #7b8498; }
</style>
