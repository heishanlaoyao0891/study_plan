<template>
  <view class="page">
    <view class="panel">
      <view class="title">AI 生成学习计划</view>
      <view class="desc">当前是 mock 模式，会根据目标生成一组每日任务。</view>
      <view class="field"><text>学习目标</text><input v-model="goal" placeholder="例如：学习 Go 语言" /></view>
      <view class="grid">
        <view class="field"><text>每天小时</text><input v-model.number="hours" type="number" /></view>
        <view class="field"><text>计划天数</text><input v-model.number="days" type="number" /></view>
      </view>
      <button class="primary" @click="generate">生成计划</button>
    </view>

    <view class="result" v-if="result">
      <view class="result-title">{{ result.plan?.title }}</view>
      <view class="task" v-for="t in result.tasks" :key="t.id">
        <view class="task-title">{{ t.title }}</view>
        <view class="task-desc">{{ t.date }} · {{ t.description }}</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { AIApi } from '@/api'

const goal = ref('学习 Go 语言')
const hours = ref(1)
const days = ref(7)
const result = ref<any>(null)

async function generate() {
  if (!goal.value.trim()) {
    uni.showToast({ title: '请输入学习目标', icon: 'none' })
    return
  }
  try {
    result.value = await AIApi.generatePlan({ goal: goal.value, hours_per_day: hours.value, days: days.value })
    uni.showToast({ title: '已生成', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '生成失败', icon: 'none' })
  }
}
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel, .result { padding: 32rpx; border-radius: 18rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.desc { margin: 12rpx 0 28rpx; color: #7b8498; font-size: 25rpx; line-height: 1.5; }
.field { margin-bottom: 22rpx; }
.field text { display: block; margin-bottom: 10rpx; color: #606a80; font-size: 24rpx; }
.field input { height: 78rpx; padding: 0 20rpx; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 18rpx; }
.primary { background: #2264d1; color: #fff; border-radius: 12rpx; }
.result { margin-top: 22rpx; }
.result-title { color: #111827; font-size: 30rpx; font-weight: 800; margin-bottom: 18rpx; }
.task { padding: 20rpx 0; border-top: 1rpx solid #eef2f7; }
.task-title { color: #111827; font-size: 27rpx; font-weight: 700; }
.task-desc { margin-top: 8rpx; color: #7b8498; font-size: 23rpx; line-height: 1.5; }
</style>
