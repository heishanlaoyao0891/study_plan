<template>
  <view class="checkin-page">
    <view class="hero">
      <view>
        <view class="date-text">{{ todayText }}</view>
        <view class="subtitle">{{ todayStr }}</view>
      </view>
      <view class="streak-box">
        <view class="streak-num">{{ streak ?? 0 }}</view>
        <view class="streak-label">连续天数</view>
      </view>
    </view>

    <view class="progress-panel">
      <view class="progress-head">
        <view>
          <view class="progress-title">今日进度</view>
          <view class="progress-desc">{{ doneCount }}/{{ totalCount }} 个计划已完成</view>
        </view>
        <view class="progress-percent">{{ percent }}%</view>
      </view>
      <view class="progress-track"><view class="progress-fill" :style="{ width: percent + '%' }" /></view>
    </view>

    <view class="toolbar">
      <view class="toolbar-title">今日计划</view>
      <button class="mini-btn" @click="load">刷新</button>
    </view>

    <view class="empty" v-if="!loading && checkins.length === 0">
      <view class="empty-title">还没有学习计划</view>
      <view class="empty-desc">先创建一个计划，明天的打卡就不再靠记忆。</view>
      <button class="primary-btn" @click="goPlans">创建计划</button>
    </view>

    <view class="list" v-else>
      <view
        class="item"
        v-for="item in checkins"
        :key="item.plan_id"
        :class="{ done: item.completed, disabled: item.status === 'paused' }"
        @click="toggle(item)"
      >
        <view class="state-dot"><view class="state-inner" /></view>
        <view class="item-main">
          <view class="item-title">{{ item.title }}</view>
          <view class="item-meta">{{ item.status === 'paused' ? '计划已暂停' : item.completed ? '今日已完成' : '等待打卡' }}</view>
        </view>
        <view class="state-text">{{ item.completed ? '完成' : '打卡' }}</view>
      </view>
    </view>

    <view class="loading" v-if="loading">加载中...</view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { CheckinApi, type CheckinInfo } from '@/api'

const today = new Date()
const todayStr = today.toISOString().slice(0, 10)
const todayText = `${today.getMonth() + 1}月${today.getDate()}日`
const checkins = ref<CheckinInfo[]>([])
const loading = ref(false)
const streak = ref<number | null>(null)
const doneCount = computed(() => checkins.value.filter(c => c.completed).length)
const totalCount = computed(() => checkins.value.length)
const percent = computed(() => totalCount.value === 0 ? 0 : Math.round((doneCount.value / totalCount.value) * 100))

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
    const s = await CheckinApi.streak().catch(() => null)
    if (s) streak.value = s.streak
  } catch (e: any) {
    uni.showToast({ title: e?.message || '打卡失败', icon: 'none' })
  }
}

function goPlans() {
  uni.switchTab({ url: '/pages/plans/plans' })
}

onMounted(load)
onShow(load)
</script>

<style lang="scss">
.checkin-page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 28rpx 28rpx 60rpx;
  background: #f6f7fb;
}
.hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 34rpx 34rpx;
  border-radius: 18rpx;
  background: #111827;
  color: #fff;
}
.date-text { font-size: 44rpx; font-weight: 800; }
.subtitle { margin-top: 8rpx; color: #aeb7c8; font-size: 24rpx; }
.streak-box {
  width: 150rpx;
  padding: 18rpx 0;
  border-radius: 14rpx;
  background: rgba(255, 255, 255, 0.08);
  text-align: center;
}
.streak-num { font-size: 44rpx; font-weight: 800; }
.streak-label { color: #aeb7c8; font-size: 22rpx; }
.progress-panel {
  margin-top: 24rpx;
  padding: 30rpx;
  border-radius: 16rpx;
  background: #fff;
  border: 1rpx solid #e9edf5;
}
.progress-head { display: flex; align-items: center; justify-content: space-between; }
.progress-title { color: #111827; font-size: 31rpx; font-weight: 700; }
.progress-desc { margin-top: 8rpx; color: #7b8498; font-size: 24rpx; }
.progress-percent { color: #2264d1; font-size: 42rpx; font-weight: 800; }
.progress-track { margin-top: 28rpx; height: 16rpx; border-radius: 99rpx; background: #edf2f8; overflow: hidden; }
.progress-fill { height: 100%; border-radius: 99rpx; background: #2264d1; transition: width .2s ease; }
.toolbar { display: flex; align-items: center; justify-content: space-between; margin: 34rpx 4rpx 18rpx; }
.toolbar-title { color: #111827; font-size: 32rpx; font-weight: 800; }
.mini-btn { margin: 0; width: 120rpx; height: 56rpx; line-height: 56rpx; border-radius: 10rpx; background: #eef4ff; color: #2264d1; font-size: 24rpx; }
.list { display: flex; flex-direction: column; gap: 18rpx; }
.item {
  display: flex;
  align-items: center;
  min-height: 118rpx;
  padding: 0 26rpx;
  border-radius: 16rpx;
  background: #fff;
  border: 1rpx solid #e9edf5;
}
.item.done { border-color: #bdd7ff; background: #f7fbff; }
.item.disabled { opacity: .55; }
.state-dot {
  width: 38rpx;
  height: 38rpx;
  border-radius: 50%;
  border: 3rpx solid #cbd5e1;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 24rpx;
}
.done .state-dot { border-color: #2264d1; }
.state-inner { width: 18rpx; height: 18rpx; border-radius: 50%; background: transparent; }
.done .state-inner { background: #2264d1; }
.item-main { flex: 1; min-width: 0; }
.item-title { color: #111827; font-size: 30rpx; font-weight: 700; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.item-meta { margin-top: 8rpx; color: #7b8498; font-size: 23rpx; }
.state-text { color: #2264d1; font-size: 25rpx; font-weight: 700; }
.empty { margin-top: 22rpx; padding: 56rpx 34rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.empty-title { color: #111827; font-size: 32rpx; font-weight: 800; }
.empty-desc { margin-top: 12rpx; color: #7b8498; font-size: 26rpx; line-height: 1.5; }
.primary-btn { margin-top: 32rpx; background: #2264d1; color: #fff; border-radius: 12rpx; }
.loading { text-align: center; color: #7b8498; margin-top: 34rpx; }
</style>
