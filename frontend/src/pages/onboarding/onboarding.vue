<template>
  <view class="page">
    <view class="hero">
      <view class="title">开始使用</view>
      <view class="subtitle">先把手机号、提醒、计划和今日任务串起来。</view>
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
      <view class="step-desc">先建一个明确目标，系统会按天生成任务。</view>
      <button class="secondary" @click="goPlans">去创建</button>
    </view>

    <view class="step">
      <view class="step-title">第二步：今天任务</view>
      <view class="step-desc">创建完成后，直接回到今日任务开始执行。</view>
      <button class="secondary" @click="goCheckin">去查看</button>
    </view>

    <view class="footer-actions">
      <button class="ghost" @click="complete">完成引导</button>
      <button class="ghost" @click="goBind">返回绑定</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'

const reminderDismissed = ref(false)

onLoad(() => { reminderDismissed.value = !!uni.getStorageSync('onboarding_reminder_dismissed') })

function goNotifications() { uni.navigateTo({ url: '/pages/notifications/notifications' }) }
function goPlans() { uni.switchTab({ url: '/pages/plans/plans' }) }
function goCheckin() { uni.switchTab({ url: '/pages/checkin/checkin' }) }
function goBind() { uni.reLaunch({ url: '/pages/bind/bind' }) }
function dismissReminder() { uni.setStorageSync('onboarding_reminder_dismissed', true); reminderDismissed.value = true }
function complete() { uni.setStorageSync('onboarding_completed', true); uni.reLaunch({ url: '/pages/checkin/checkin' }) }
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
.footer-actions { margin-top: 20rpx; }
</style>
