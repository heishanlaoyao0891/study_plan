<template>
  <view class="page">
    <view class="panel">
      <view class="title">提醒设置</view>
      <view class="desc">微信订阅消息需要用户主动授权。当前后端为占位接口，正式发布前接入模板 ID。</view>
      <button class="primary" @click="subscribe">订阅提醒</button>
      <button class="ghost" @click="unsubscribe">取消订阅</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { NotificationApi } from '@/api'
async function subscribe() {
  try { await NotificationApi.subscribe(); uni.showToast({ title: '已订阅', icon: 'success' }) }
  catch (e: any) { uni.showToast({ title: e?.message || '失败', icon: 'none' }) }
}
async function unsubscribe() {
  try { await NotificationApi.unsubscribe(); uni.showToast({ title: '已取消', icon: 'success' }) }
  catch (e: any) { uni.showToast({ title: e?.message || '失败', icon: 'none' }) }
}
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel { padding: 34rpx; border-radius: 18rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.desc { margin: 14rpx 0 32rpx; color: #7b8498; font-size: 26rpx; line-height: 1.6; }
.primary, .ghost { height: 82rpx; line-height: 82rpx; border-radius: 12rpx; font-size: 28rpx; }
.primary { background: #2264d1; color: #fff; }
.ghost { margin-top: 18rpx; background: #f3f6fb; color: #384257; }
</style>
