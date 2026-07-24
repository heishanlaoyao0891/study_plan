<template>
  <view class="page">
    <view class="panel">
      <view class="title">设置与说明</view>
      <view class="entry" @click="openContent('privacy')">隐私政策</view>
      <view class="entry" @click="openContent('agreement')">用户协议</view>
      <view class="entry" @click="openContent('version')">版本说明</view>
      <view class="entry" @click="showFeedback = true">反馈与问题报告</view>
      <view class="entry muted" @click="goAccount">账号与数据</view>
    </view>

    <view class="panel" v-if="content">
      <view class="title">{{ content.title }}</view>
      <view class="body">{{ content.body }}</view>
    </view>

    <view class="panel" v-if="showFeedback">
      <view class="title">反馈</view>
      <input v-model="category" placeholder="类型，例如 bug / 建议" />
      <textarea v-model="feedback" placeholder="描述问题或建议" />
      <input v-model="contact" placeholder="联系方式，可选" />
      <button class="primary" @click="submitFeedback">提交</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { OpsApi } from '@/api'

const content = ref<any>(null)
const showFeedback = ref(false)
const category = ref('feedback')
const feedback = ref('')
const contact = ref('')

async function openContent(kind: 'privacy' | 'agreement' | 'version') {
  content.value = await OpsApi.content(kind)
  showFeedback.value = false
}

async function submitFeedback() {
  if (!feedback.value.trim()) {
    uni.showToast({ title: '请输入内容', icon: 'none' })
    return
  }
  await OpsApi.feedback({ category: category.value, content: feedback.value, contact: contact.value })
  feedback.value = ''
  contact.value = ''
  showFeedback.value = false
  uni.showToast({ title: '已提交', icon: 'success' })
}

function goAccount() { uni.navigateTo({ url: '/pages/account/account' }) }
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel { margin-bottom: 18rpx; padding: 30rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 32rpx; font-weight: 800; margin-bottom: 16rpx; }
.entry { padding: 24rpx 0; border-top: 1rpx solid #eef2f7; color: #2264d1; font-size: 27rpx; font-weight: 700; }
.entry.muted { color: #7b8498; }
.body { white-space: pre-wrap; color: #384257; font-size: 26rpx; line-height: 1.7; }
input, textarea { box-sizing: border-box; width: 100%; margin-bottom: 14rpx; padding: 0 18rpx; border-radius: 10rpx; border: 1rpx solid #dbe2ee; background: #f9fbff; font-size: 26rpx; }
input { height: 74rpx; }
textarea { height: 180rpx; padding-top: 18rpx; }
.primary { background: #111827; color: #fff; border-radius: 10rpx; }
</style>
