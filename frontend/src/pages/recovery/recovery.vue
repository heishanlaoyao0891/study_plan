<template>
  <view class="page">
    <view class="panel">
      <view class="title">重新安排</view>
      <view class="desc">当你落后时，系统会把未完成任务排进未来几天，确认后才会生效。</view>
      <view class="meta">{{ preview?.mode || 'rule' }} · 落后 {{ preview?.missed_days || 0 }} 天 · {{ preview?.overdue_tasks || 0 }} 个任务待整理</view>
      <button class="primary" @click="load">刷新预览</button>
    </view>

    <view class="panel" v-if="preview?.actions?.length">
      <view class="section-title">调整计划</view>
      <view class="row" v-for="a in preview.actions" :key="a.task_id">
        <view>
          <view class="row-title">{{ a.title }}</view>
          <view class="row-sub">{{ a.old_date }} → {{ a.new_date }}</view>
        </view>
      </view>
      <button class="primary" @click="apply">确认应用</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { RecoveryApi } from '@/api'

const preview = ref<any>(null)

async function load() {
  preview.value = await RecoveryApi.preview().catch(() => null)
}

async function apply() {
  if (!preview.value?.actions?.length) return
  const ok = await new Promise<boolean>(resolve => {
    uni.showModal({ title: '应用恢复', content: '确认按建议重新安排未完成任务吗？', success: res => resolve(!!res.confirm) })
  })
  if (!ok) return
  const res = await RecoveryApi.apply(preview.value.actions)
  uni.showToast({ title: `已调整 ${res.applied} 个任务`, icon: 'success' })
  await load()
}

onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel { padding: 30rpx; margin-bottom: 18rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.desc, .meta, .row-sub { margin-top: 10rpx; color: #7b8498; font-size: 24rpx; line-height: 1.5; }
.section-title { color: #111827; font-size: 28rpx; font-weight: 800; margin-bottom: 14rpx; }
.row { padding: 16rpx 0; border-top: 1rpx solid #eef2f7; }
.row-title { color: #111827; font-size: 26rpx; font-weight: 700; }
.primary { margin-top: 18rpx; background: #111827; color: #fff; border-radius: 10rpx; }
</style>
