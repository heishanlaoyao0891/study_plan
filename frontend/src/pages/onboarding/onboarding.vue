<template>
  <view class="page">
    <view class="hero">
      <view class="title">欢迎来到学习花园</view>
      <view class="subtitle">从一个目标开始，把它变成今天能完成的小步。</view>
    </view>

    <view class="step" v-if="!reminderDismissed">
      <view class="step-title">提醒</view>
      <view class="step-desc">开启后，系统会在学习开始、完成和未打卡时发出提示。</view>
      <view class="actions">
        <button class="primary" @click="goNotifications">去开启</button>
        <button class="secondary" @click="dismissReminder">暂不需要</button>
      </view>
    </view>

    <view class="step">
      <view class="step-title">第一步：创建计划</view>
      <view class="step-desc">告诉 AI 你的学习目标，它会拆成有日期和时段的任务。</view>
      <button class="primary wide" @click="goAI">AI 生成第一个计划</button>
      <button class="secondary wide" @click="goPlans">手动创建</button>
    </view>

    <view class="step">
      <view class="step-title">第二步：今天任务</view>
      <view class="step-desc">创建完成后，直接回到今日任务开始执行。</view>
      <button class="secondary" @click="goCheckin">去查看</button>
    </view>

    <view class="actions finish"><button class="ghost" @click="skip">跳过引导</button><button class="primary" @click="complete">完成引导</button></view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { AuthApi } from '@/api'

const reminderDismissed = ref(false)

function goNotifications() { uni.navigateTo({ url: '/pages/notifications/notifications' }) }
function goPlans() { uni.switchTab({ url: '/pages/plans/plans' }) }
function goAI() { uni.navigateTo({ url: '/pages/ai/ai' }) }
function goCheckin() { uni.switchTab({ url: '/pages/checkin/checkin' }) }
function dismissReminder() { reminderDismissed.value = true }
async function finish(status: 'completed' | 'skipped') {
  await AuthApi.updateOnboarding(status)
  uni.reLaunch({ url: '/pages/checkin/checkin' })
}
function complete() { finish('completed') }
function skip() { finish('skipped') }
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 32rpx; box-sizing: border-box; background: #f6f7fb; }
.hero, .step { padding: 30rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.subtitle, .step-desc { margin-top: 10rpx; color: #7b8498; font-size: 24rpx; line-height: 1.6; }
.title, .step-title { color: #111827; font-size: 34rpx; font-weight: 800; }
.step { margin-top: 18rpx; }
.actions, .footer-actions { display: grid; grid-template-columns: 1fr 1fr; gap: 12rpx; margin-top: 18rpx; }
.primary, .secondary, .ghost { height: 76rpx; line-height: 76rpx; border-radius: 10rpx; font-size: 26rpx; }
.primary { background: #111827; color: #fff; }
.secondary { background: #eef4ff; color: #2264d1; }
.ghost { background: #f3f6fb; color: #384257; }
.wide { margin-top: 16rpx; }
.finish { margin-top: 20rpx; }
</style>
