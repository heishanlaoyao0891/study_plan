<template>
  <view class="page">
    <view class="report-card">
      <view class="report-title">本周概览</view>
      <view class="report-grid">
        <view class="metric"><view class="num">{{ weekly.total_study_minutes || 0 }}</view><view class="label">学习分钟</view></view>
        <view class="metric"><view class="num">{{ weekly.completed_checkins || 0 }}</view><view class="label">完成打卡</view></view>
        <view class="metric"><view class="num">{{ weekly.slack_minutes || 0 }}</view><view class="label">躺平分钟</view></view>
      </view>
    </view>

    <view class="section-title">今日分布</view>
    <view class="row" v-for="r in daily" :key="r.plan_id">
      <view>
        <view class="row-title">{{ r.title }}</view>
        <view class="row-sub">{{ r.status }}</view>
      </view>
      <view class="row-num">{{ r.study_minutes }} 分</view>
    </view>

    <view class="section-title">躺平分布</view>
    <view class="row" v-for="r in slackDist" :key="r.activity">
      <view class="row-title">{{ r.activity || '未填写' }}</view>
      <view class="row-num">{{ r.minutes }} 分</view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { StatsApi } from '@/api'

const weekly = ref<any>({})
const daily = ref<any[]>([])
const slackDist = ref<any[]>([])

async function load() {
  try {
    const today = new Date().toISOString().slice(0, 10)
    const month = today.slice(0, 7)
    const [w, d, s] = await Promise.all([
      StatsApi.weeklyReport(),
      StatsApi.dailyDistribution(today),
      StatsApi.slackDistribution(month),
    ])
    weekly.value = w || {}
    daily.value = d || []
    slackDist.value = s || []
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  }
}

onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.report-card { padding: 34rpx; border-radius: 18rpx; background: #111827; color: #fff; }
.report-title { font-size: 32rpx; font-weight: 800; }
.report-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16rpx; margin-top: 28rpx; }
.metric { padding: 22rpx 10rpx; border-radius: 14rpx; background: rgba(255,255,255,.08); text-align: center; }
.num { font-size: 34rpx; font-weight: 800; }
.label { margin-top: 8rpx; color: #aeb7c8; font-size: 21rpx; }
.section-title { margin: 34rpx 0 16rpx; color: #111827; font-size: 30rpx; font-weight: 800; }
.row { display: flex; justify-content: space-between; align-items: center; padding: 26rpx; margin-bottom: 14rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.row-title { color: #111827; font-size: 28rpx; font-weight: 700; }
.row-sub { margin-top: 6rpx; color: #7b8498; font-size: 22rpx; }
.row-num { color: #2264d1; font-size: 28rpx; font-weight: 800; }
</style>
