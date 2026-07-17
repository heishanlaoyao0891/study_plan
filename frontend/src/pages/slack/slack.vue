<template>
  <view class="page">
    <view class="balance-card">
      <view>
        <view class="label">可用躺平时间</view>
        <view class="sub">完成学习打卡后自动累积</view>
      </view>
      <view class="balance">{{ balance }}<text>分钟</text></view>
    </view>

    <view class="action-card">
      <view class="title">记录一次躺平</view>
      <input class="input" v-model="activity" placeholder="今天躺平主要在干什么？" />
      <view class="actions">
        <button class="start" @click="start">开始躺平</button>
        <button class="stop" @click="stop">结束躺平</button>
      </view>
    </view>

    <view class="section-title">最近记录</view>
    <view class="record" v-for="r in records" :key="r.id">
      <view>
        <view class="record-title">{{ r.activity || '未填写' }}</view>
        <view class="record-time">{{ formatTime(r.start_time) }}</view>
      </view>
      <view class="record-min">{{ r.duration_min || '进行中' }}<text v-if="r.duration_min">分</text></view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { SlackApi } from '@/api'

const balance = ref(0)
const activity = ref('')
const records = ref<any[]>([])

async function load() {
  try {
    const [b, rs] = await Promise.all([SlackApi.balance(), SlackApi.records()])
    balance.value = b.balance
    records.value = rs || []
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  }
}

async function start() {
  if (!activity.value.trim()) {
    uni.showToast({ title: '先写躺平内容', icon: 'none' })
    return
  }
  try {
    await SlackApi.start(activity.value.trim())
    activity.value = ''
    await load()
    uni.showToast({ title: '开始记录', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '开始失败', icon: 'none' })
  }
}

async function stop() {
  try {
    await SlackApi.stop()
    await load()
    uni.showToast({ title: '已结束', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '结束失败', icon: 'none' })
  }
}

function formatTime(v: string) {
  if (!v) return ''
  return v.slice(0, 16).replace('T', ' ')
}

onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.balance-card, .action-card, .record { background: #fff; border: 1rpx solid #e9edf5; border-radius: 16rpx; }
.balance-card { display: flex; justify-content: space-between; align-items: center; padding: 34rpx; }
.label { color: #111827; font-size: 31rpx; font-weight: 800; }
.sub { margin-top: 8rpx; color: #7b8498; font-size: 23rpx; }
.balance { color: #0f766e; font-size: 46rpx; font-weight: 800; }
.balance text { font-size: 22rpx; margin-left: 4rpx; }
.action-card { margin-top: 22rpx; padding: 28rpx; }
.title, .section-title { color: #111827; font-size: 30rpx; font-weight: 800; }
.input { margin-top: 20rpx; height: 78rpx; padding: 0 20rpx; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; }
.actions { display: grid; grid-template-columns: 1fr 1fr; gap: 16rpx; margin-top: 22rpx; }
.start, .stop { height: 76rpx; line-height: 76rpx; border-radius: 12rpx; font-size: 27rpx; }
.start { background: #111827; color: #fff; }
.stop { background: #eef4ff; color: #2264d1; }
.section-title { margin: 34rpx 0 18rpx; }
.record { display: flex; justify-content: space-between; align-items: center; padding: 26rpx; margin-bottom: 16rpx; }
.record-title { color: #111827; font-size: 28rpx; font-weight: 700; }
.record-time { margin-top: 8rpx; color: #7b8498; font-size: 22rpx; }
.record-min { color: #2264d1; font-size: 30rpx; font-weight: 800; }
.record-min text { font-size: 21rpx; }
</style>
