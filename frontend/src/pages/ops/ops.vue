<template>
  <view class="page">
    <view class="panel">
      <view class="title">设置与说明</view>
      <view class="entry" @click="openContent('privacy')">隐私政策</view>
      <view class="entry" @click="openContent('agreement')">用户协议</view>
      <view class="entry" @click="openContent('version')">版本说明</view>
      <view class="entry" @click="goFeedback">反馈与问题报告</view>
      <view class="entry muted" @click="goAccount">账号与数据</view>
    </view>

    <view class="panel" v-if="content">
      <view class="title">{{ content.title }}</view>
      <view class="body">{{ content.body }}</view>
    </view>

  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { OpsApi } from '@/api'

const content = ref<any>(null)

async function openContent(kind: 'privacy' | 'agreement' | 'version') {
  content.value = await OpsApi.content(kind)
}

function goAccount() { uni.navigateTo({ url: '/pages/account/account' }) }
function goFeedback() { uni.navigateTo({ url: '/pages/feedback/feedback' }) }
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel { margin-bottom: 18rpx; padding: 30rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 32rpx; font-weight: 800; margin-bottom: 16rpx; }
.entry { padding: 24rpx 0; border-top: 1rpx solid #eef2f7; color: #2264d1; font-size: 27rpx; font-weight: 700; }
.entry.muted { color: #7b8498; }
.body { white-space: pre-wrap; color: #384257; font-size: 26rpx; line-height: 1.7; }
</style>
