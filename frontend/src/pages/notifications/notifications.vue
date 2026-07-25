<template>
  <view class="page">
    <view class="panel">
      <view class="title">提醒设置</view>
      <!-- #ifdef MP-WEIXIN -->
      <view class="desc">微信订阅消息需要逐项授权。微信每次授权通常只允许发送一次，后续可再次订阅。</view>
      <button class="primary" @click="subscribe">订阅提醒</button>
      <button class="ghost" @click="unsubscribe">取消订阅</button>
      <!-- #endif -->
      <!-- #ifndef MP-WEIXIN -->
      <view class="desc unsupported">H5 不支持微信订阅消息授权。请使用已绑定同一账号的微信小程序，在“提醒设置”中完成订阅。</view>
      <!-- #endif -->
    </view>

    <view class="events">
      <view class="events-title">今日提醒队列</view>
      <view class="event" v-for="e in events" :key="`${e.type}-${e.task?.id}`">
        <view>
          <view class="event-type">{{ typeText(e.type) }}</view>
          <view class="event-task">{{ e.task?.title }}</view>
        </view>
        <view class="event-date">{{ e.task?.date }}</view>
      </view>
      <view class="empty" v-if="!events.length">暂无待发送提醒</view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { NotificationApi } from '@/api'
const events = ref<any[]>([])

async function load() {
  const now = new Date()
  const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
  try {
    const data = await NotificationApi.due(today)
    events.value = data.events || []
  } catch {
    events.value = []
  }
}

async function subscribe() {
  // #ifdef MP-WEIXIN
  try {
    const metadata = await NotificationApi.templates()
    const templateIds = metadata.templates.map((item) => item.template_id)
    if (!templateIds.length) {
      uni.showToast({ title: '暂无已启用的提醒模板', icon: 'none' })
      return
    }
    const results = await new Promise<Record<string, string>>((resolve, reject) => {
      uni.requestSubscribeMessage({ tmplIds: templateIds, success: (result) => resolve(result as unknown as Record<string, string>), fail: reject })
    })
    const saved = await NotificationApi.subscribe(results)
    uni.showToast({ title: saved.accepted.length ? `已订阅 ${saved.accepted.length} 项` : '未接受任何提醒', icon: 'none' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '授权失败', icon: 'none' })
  }
  // #endif
}
async function unsubscribe() {
  try { await NotificationApi.unsubscribe(); uni.showToast({ title: '已取消', icon: 'success' }) }
  catch (e: any) { uni.showToast({ title: e?.message || '失败', icon: 'none' }) }
}

function typeText(type: string) {
  const map: Record<string, string> = {
    study_start: '到点学习',
    completion: '完成提醒',
    decision_2330: '23:30 决策',
    missed_checkin: '未打卡',
    group_nudge: '小组督学',
  }
  return map[type] || type
}

onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel { padding: 34rpx; border-radius: 18rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.desc { margin: 14rpx 0 32rpx; color: #7b8498; font-size: 26rpx; line-height: 1.6; }
.unsupported { padding: 24rpx; border-radius: 12rpx; background: #f3f6fb; color: #4b5568; }
.primary, .ghost { height: 82rpx; line-height: 82rpx; border-radius: 12rpx; font-size: 28rpx; }
.primary { background: #2264d1; color: #fff; }
.ghost { margin-top: 18rpx; background: #f3f6fb; color: #384257; }
.events { margin-top: 22rpx; padding: 30rpx; border-radius: 18rpx; background: #fff; border: 1rpx solid #e9edf5; }
.events-title { color: #111827; font-size: 30rpx; font-weight: 800; margin-bottom: 16rpx; }
.event { display: flex; align-items: center; justify-content: space-between; gap: 18rpx; padding: 18rpx 0; border-top: 1rpx solid #eef2f7; }
.event-type { color: #2264d1; font-size: 25rpx; font-weight: 800; }
.event-task { margin-top: 6rpx; color: #606a80; font-size: 23rpx; }
.event-date { color: #7b8498; font-size: 22rpx; }
.empty { color: #7b8498; font-size: 24rpx; padding: 18rpx 0; }
</style>
