<template>
  <view class="page">
    <view class="shell">
      <view class="status-mark" aria-hidden="true">
        <view class="pause-line" />
        <view class="pause-line" />
      </view>

      <view class="eyebrow">账号状态</view>
      <view class="title">访问已暂停</view>
      <view class="intro">你的学习数据和计划仍会保留，恢复访问后可以继续原来的进度。</view>

      <view class="card">
        <view class="label">暂停原因</view>
        <view class="reason">{{ state?.reason || '账号当前需要暂停访问，如有疑问请联系管理员。' }}</view>
        <view class="divider" />
        <template v-if="state?.permanent">
          <view class="status permanent">长期暂停</view>
          <view class="hint">此状态没有自动恢复时间，请联系管理员了解详情。</view>
        </template>
        <template v-else>
          <view class="label">预计恢复时间</view>
          <view class="unlock">{{ unlockText }}</view>
          <view class="countdown">
            <view v-for="item in countdownParts" :key="item.label" class="time-part">
              <text class="number">{{ item.value }}</text>
              <text class="unit">{{ item.label }}</text>
            </view>
          </view>
        </template>
      </view>

      <view class="now-row" aria-live="polite">
        <text class="pulse" />
        <text>{{ statusText }}</text>
        <text class="now">当前 {{ nowText }}</text>
      </view>

      <button v-if="canRefresh" class="primary" :loading="refreshing" :disabled="refreshing" @click="refreshStatus">检查是否已恢复</button>
      <button v-else class="primary" @click="returnToLogin">{{ expired ? '返回登录并重试' : '返回登录' }}</button>
      <button class="secondary" @click="clearSession">退出并清除本机登录</button>
      <view class="support">需要帮助？请联系向你提供账号或邀请码的管理员，并说明“账号访问暂停”。</view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onHide, onLoad, onShow, onUnload } from '@dcloudio/uni-app'
import { AuthApi } from '@/api'
import { clearToken, getToken } from '@/api/request'
import { routeForUser } from '@/utils/auth-routing'
import { clearBanState, getBanState, serverTime, type BanState } from '@/utils/ban-state'

const state = ref<BanState | null>(null)
const now = ref(Date.now())
const refreshing = ref(false)
let timer: ReturnType<typeof setInterval> | undefined
let lastAutoRefresh = 0

const remaining = computed(() => state.value ? Math.max(0, Date.parse(state.value.banned_until) - now.value) : 0)
const expired = computed(() => !!state.value && !state.value.permanent && remaining.value <= 0)
const canRefresh = computed(() => !!getToken() && state.value?.token_retained === true)
const unlockText = computed(() => state.value ? formatDate(Date.parse(state.value.banned_until)) : '--')
const nowText = computed(() => formatDate(now.value))
const statusText = computed(() => refreshing.value ? '正在向服务器确认状态' : expired.value ? '暂停时间已结束，可检查恢复状态' : '访问暂停中')
const countdownParts = computed(() => {
  let seconds = Math.floor(remaining.value / 1000)
  const days = Math.floor(seconds / 86400)
  seconds %= 86400
  const hours = Math.floor(seconds / 3600)
  seconds %= 3600
  const minutes = Math.floor(seconds / 60)
  return [
    { label: '天', value: pad(days) },
    { label: '时', value: pad(hours) },
    { label: '分', value: pad(minutes) },
    { label: '秒', value: pad(seconds % 60) },
  ]
})

function pad(value: number) {
  return String(value).padStart(2, '0')
}

function formatDate(timestamp: number) {
  if (!Number.isFinite(timestamp)) return '--'
  const date = new Date(timestamp)
  return `${date.getFullYear()}年${pad(date.getMonth() + 1)}月${pad(date.getDate())}日 ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function syncClock() {
  const latest = getBanState()
  if (!latest) {
    clearInterval(timer)
    uni.reLaunch({ url: getToken() ? '/pages/checkin/checkin' : '/pages/login/login' })
    return
  }
  state.value = latest
  now.value = serverTime(latest)
  if (canRefresh.value && expired.value && Date.now() - lastAutoRefresh > 10000) refreshStatus()
}

async function refreshStatus() {
  if (!canRefresh.value || refreshing.value) return
  refreshing.value = true
  lastAutoRefresh = Date.now()
  try {
    const user = await AuthApi.me()
    clearBanState()
    uni.reLaunch({ url: routeForUser(user) })
  } catch {
    state.value = getBanState()
  } finally {
    refreshing.value = false
  }
}

function returnToLogin() {
  clearToken()
  clearBanState()
  uni.reLaunch({ url: '/pages/login/login' })
}

function clearSession() {
  clearToken()
  clearBanState()
  uni.reLaunch({ url: '/pages/login/login' })
}

function startTimer() {
  clearInterval(timer)
  syncClock()
  timer = setInterval(syncClock, 1000)
}

onLoad(() => {
  state.value = getBanState()
  if (!state.value) {
    uni.reLaunch({ url: getToken() ? '/pages/checkin/checkin' : '/pages/login/login' })
  }
})
onShow(() => {
  startTimer()
  if (canRefresh.value && Date.now() - lastAutoRefresh > 30000) refreshStatus()
})
onHide(() => clearInterval(timer))
onUnload(() => clearInterval(timer))
</script>

<style lang="scss">
.page { min-height: 100vh; box-sizing: border-box; padding: 64rpx 30rpx; background: #f5f7f6; color: #24302b; }
.shell { width: 100%; max-width: 680px; margin: 0 auto; text-align: center; }
.status-mark { display: flex; width: 104rpx; height: 104rpx; margin: 0 auto 28rpx; align-items: center; justify-content: center; gap: 12rpx; border: 2rpx solid #d9a11e; border-radius: 50%; background: #fff9e8; }
.pause-line { width: 10rpx; height: 38rpx; border-radius: 3rpx; background: #bd7f00; }
.eyebrow { color: #28705a; font-size: 22rpx; font-weight: 800; }
.title { margin-top: 12rpx; font-size: 44rpx; line-height: 1.25; font-weight: 900; }
.intro { max-width: 610rpx; margin: 18rpx auto 34rpx; color: #66716d; font-size: 25rpx; line-height: 1.7; }
.card { box-sizing: border-box; padding: 34rpx 32rpx; border: 1rpx solid #dfe5e2; border-radius: 12rpx; background: #fff; box-shadow: 0 10rpx 28rpx rgba(36, 48, 43, .07); text-align: left; }
.label { color: #75807b; font-size: 22rpx; font-weight: 700; }
.reason { margin-top: 12rpx; color: #29352f; font-size: 29rpx; line-height: 1.6; font-weight: 700; word-break: break-word; }
.divider { height: 1rpx; margin: 28rpx 0; background: #e7ebe9; }
.unlock { margin-top: 10rpx; color: #4f5e57; font-size: 25rpx; }
.countdown { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12rpx; margin-top: 26rpx; }
.time-part { min-width: 0; padding: 18rpx 4rpx 15rpx; border-radius: 8rpx; background: #f1f5f3; text-align: center; }
.number { display: block; color: #176449; font-size: 34rpx; font-weight: 900; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
.unit { display: block; margin-top: 4rpx; color: #687a72; font-size: 19rpx; }
.status.permanent { display: inline-block; padding: 10rpx 18rpx; border-radius: 8rpx; background: #fff4d8; color: #8a5c00; font-size: 25rpx; font-weight: 800; }
.hint { margin-top: 17rpx; color: #66716d; font-size: 24rpx; line-height: 1.6; }
.now-row { display: flex; flex-wrap: wrap; justify-content: center; align-items: center; gap: 10rpx; margin: 25rpx 0; color: #5e6c66; font-size: 21rpx; }
.pulse { width: 12rpx; height: 12rpx; border-radius: 50%; background: #d49a12; box-shadow: 0 0 0 7rpx rgba(212, 154, 18, .12); }
.now { margin-left: 7rpx; color: #7a8781; }
button { margin: 0; }
button::after { border: 0; }
.primary, .secondary { width: 100%; height: 88rpx; line-height: 88rpx; border-radius: 8rpx; font-size: 27rpx; font-weight: 800; }
.primary { background: #176449; color: #fff; }
.secondary { margin-top: 16rpx; background: transparent; color: #44534c; border: 1rpx solid #ccd5d1; }
.support { margin: 24rpx auto 0; max-width: 620rpx; color: #75807b; font-size: 21rpx; line-height: 1.65; }
@media (max-width: 340px) {
  .countdown { gap: 6rpx; }
  .number { font-size: 28rpx; }
  .now { width: 100%; margin-left: 0; }
}
@media (min-width: 800px) {
  .page { display: flex; align-items: center; padding: 48px; }
  .shell { max-width: 560px; }
  .card { padding: 30px 34px; }
  .title { font-size: 38px; }
  .intro { font-size: 17px; }
  .primary, .secondary { max-width: 460px; }
}
</style>
