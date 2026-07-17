<template>
  <view class="checkin-page">
    <view class="header">
      <view class="date-text">{{ todayStr }}</view>
      <view class="streak" v-if="streak !== null">🔥 连续 {{ streak }} 天</view>
    </view>

    <view class="empty" v-if="!loading && checkins.length === 0">
      <view class="empty-icon">📝</view>
      <view>还没有学习计划，去创建一个吧～</view>
      <button class="link" @click="goPlans">去创建计划</button>
    </view>

    <view class="checkin-list" v-else>
      <view
        class="checkin-item"
        v-for="item in checkins"
        :key="item.plan_id"
        :class="{ done: item.completed, paused: item.status === 'paused' }"
        @click="toggle(item)"
      >
        <view class="check-box">{{ item.completed ? '✅' : '⚪' }}</view>
        <view class="info">
          <view class="title">{{ item.title }}</view>
          <view class="status" v-if="item.status === 'paused'">已暂停</view>
        </view>
      </view>

      <view class="hint small">点击勾选/取消打卡。当天所有计划打满即算完整打卡。</view>

      <view class="summary" v-if="doneCount > 0">
        今日已完成 {{ doneCount }}/{{ checkins.length }}
      </view>
    </view>

    <view class="loading" v-if="loading">加载中...</view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { CheckinApi, type CheckinInfo } from '@/api'

const todayStr = new Date().toISOString().slice(0, 10)
const checkins = ref<CheckinInfo[]>([])
const loading = ref(false)
const streak = ref<number | null>(null)
const doneCount = computed(() => checkins.value.filter(c => c.completed).length)

async function load() {
  loading.value = true
  try {
    const [list, s] = await Promise.all([
      CheckinApi.listByDate(todayStr),
      CheckinApi.streak().catch(() => null),
    ])
    checkins.value = list || []
    if (s) streak.value = s.streak
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

async function toggle(item: CheckinInfo) {
  if (item.status === 'paused') {
    uni.showToast({ title: '该计划已暂停', icon: 'none' })
    return
  }
  try {
    await CheckinApi.toggle({ plan_id: item.plan_id, date: todayStr, completed: !item.completed })
    item.completed = !item.completed
    // 刷新连续天数
    const s = await CheckinApi.streak().catch(() => null)
    if (s) streak.value = s.streak
    if (item.completed) uni.showToast({ title: '打卡成功 🎉', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '打卡失败', icon: 'none' })
  }
}

function goPlans() {
  uni.switchTab({ url: '/pages/plans/plans' })
}

onMounted(load)
// 从其他 tab 切回来时自动刷新
onShow(load)
</script>

<style lang="scss">
.checkin-page { min-height: 100vh; background: #f5f6fa; padding-bottom: 40rpx; }
.header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 30rpx 40rpx; background: linear-gradient(135deg, #4C8BF5, #6DA5F8);
  color: #fff;
}
.date-text { font-size: 36rpx; font-weight: 600; }
.streak { font-size: 26rpx; background: rgba(255,255,255,.25); padding: 6rpx 20rpx; border-radius: 40rpx; }

.empty { text-align: center; padding: 120rpx 40rpx; color: #888; }
.empty-icon { font-size: 120rpx; margin-bottom: 30rpx; }
.link { margin-top: 30rpx; color: #4C8BF5; background: transparent; font-size: 28rpx; }

.checkin-list { padding: 30rpx; }
.checkin-item {
  display: flex; align-items: center; padding: 32rpx;
  background: #fff; border-radius: 16rpx; margin-bottom: 20rpx;
  box-shadow: 0 2rpx 12rpx rgba(0,0,0,.04);
}
.checkin-item.done { background: #f6ffed; border: 1rpx solid #b7eb8f; }
.checkin-item.paused { opacity: .55; }
.check-box { font-size: 48rpx; margin-right: 24rpx; }
.info { flex: 1; }
.title { font-size: 32rpx; color: #333; }
.status { font-size: 22rpx; color: #ff9800; margin-top: 6rpx; }
.hint.small { color: #999; font-size: 24rpx; text-align: center; margin-top: 30rpx; }
.summary { text-align: center; color: #4C8BF5; font-size: 28rpx; margin-top: 20rpx; }
.loading { text-align: center; color: #999; margin-top: 40rpx; }
</style>