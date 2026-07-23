<template>
  <view class="page">
    <view class="balance-card">
      <view>
        <view class="label">甜甜休息券</view>
        <view class="sub">努力学习后兑换的小休息，不代表现金或可交易价值</view>
      </view>
      <view class="balance">{{ balance }}<text>分钟</text></view>
    </view>

    <view class="action-card">
      <view class="title">记录一次安心休息</view>
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
      <view class="record-min">{{ recordAmount(r) }}</view>
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

function recordAmount(record: any) {
  if (record.delta_min) return `${record.delta_min > 0 ? '+' : ''}${record.delta_min}分`
  if (record.duration_min) return `${record.duration_min}分`
  return '进行中'
}

onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: linear-gradient(180deg, #fff0f7 0%, #fffaf0 42%, #f7fbff 100%); }
.balance-card, .action-card, .record { background: #fff; border: 1rpx solid #ffe0ea; border-radius: 28rpx; box-shadow: 0 12rpx 30rpx rgba(255, 143, 171, .10); }
.balance-card { display: flex; justify-content: space-between; align-items: center; padding: 34rpx; background: linear-gradient(135deg, #fff, #fff7fb); }
.label { color: #111827; font-size: 31rpx; font-weight: 800; }
.sub { margin-top: 8rpx; color: #7b8498; font-size: 23rpx; }
.balance { color: #ff6f91; font-size: 46rpx; font-weight: 900; }
.balance text { font-size: 22rpx; margin-left: 4rpx; }
.action-card { margin-top: 22rpx; padding: 28rpx; }
.title, .section-title { color: #111827; font-size: 30rpx; font-weight: 800; }
.input { margin-top: 20rpx; height: 78rpx; padding: 0 20rpx; border: 1rpx solid #ffe0ea; border-radius: 999rpx; background: #fffafd; }
.actions { display: grid; grid-template-columns: 1fr 1fr; gap: 16rpx; margin-top: 22rpx; }
.start, .stop { height: 76rpx; line-height: 76rpx; border-radius: 999rpx; font-size: 27rpx; }
.start { background: linear-gradient(135deg, #ff7aa2, #ffb45c); color: #fff; }
.stop { background: #fff0f6; color: #ff6f91; }
.section-title { margin: 34rpx 0 18rpx; }
.record { display: flex; justify-content: space-between; align-items: center; padding: 26rpx; margin-bottom: 16rpx; }
.record-title { color: #111827; font-size: 28rpx; font-weight: 700; }
.record-time { margin-top: 8rpx; color: #7b8498; font-size: 22rpx; }
.record-min { color: #ff6f91; font-size: 30rpx; font-weight: 900; }
.record-min text { font-size: 21rpx; }
</style>
