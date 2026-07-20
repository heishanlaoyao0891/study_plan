<template>
  <view class="page">
    <view class="panel">
      <view class="title">绑定手机号</view>
      <view class="desc">手机号验证仅在已开通微信手机号能力时使用。头像可选，不影响使用。</view>

      <view class="profile-row" v-if="user">
        <image class="avatar" v-if="user.avatar_url" :src="user.avatar_url" mode="aspectFill" />
        <view class="avatar placeholder" v-else>{{ (user.nickname || '学').slice(0, 1) }}</view>
        <view>
          <view class="name">{{ user.nickname || '学习用户' }}</view>
          <view class="status">{{ user.phone_verified_at ? '手机号已绑定' : '等待绑定手机号' }}</view>
        </view>
      </view>

      <button class="secondary-btn" open-type="chooseAvatar" @chooseavatar="onChooseAvatar">选择头像</button>
      <button class="primary-btn" open-type="getPhoneNumber" @getphonenumber="onGetPhoneNumber">授权手机号</button>
      <button class="ghost-btn" @click="enterApp">进入应用</button>
    </view>

    <view class="panel dev">
      <view class="section-title">本地调试</view>
      <input class="field-input" v-model="devPhone" placeholder="13800000000" />
      <button class="dark-btn" @click="bindDevPhone">绑定调试手机号</button>
    </view>

    <view class="error" v-if="errMsg">{{ errMsg }}</view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { AuthApi, type User } from '@/api'

const user = ref<User | null>(null)
const errMsg = ref('')
const devPhone = ref('13800000000')

async function load() {
  try {
    user.value = await AuthApi.me()
  } catch (e: any) {
    errMsg.value = e?.message || '加载用户失败'
  }
}

async function onChooseAvatar(event: any) {
  const avatarUrl = event?.detail?.avatarUrl || ''
  if (!avatarUrl) return
  try {
    user.value = await AuthApi.updateAvatar(avatarUrl)
    uni.showToast({ title: '头像已更新', icon: 'success' })
  } catch (e: any) {
    errMsg.value = e?.message || '头像更新失败'
  }
}

async function onGetPhoneNumber(event: any) {
  errMsg.value = ''
  const code = event?.detail?.code || ''
  if (!code) {
    errMsg.value = '未获得手机号授权，请重试。'
    return
  }
  try {
    user.value = await AuthApi.bindPhone(code)
    uni.showToast({ title: '已绑定', icon: 'success' })
    enterApp()
  } catch (e: any) {
    errMsg.value = e?.message || '手机号绑定失败'
  }
}

async function bindDevPhone() {
  try {
    user.value = await AuthApi.bindPhone('', devPhone.value)
    uni.showToast({ title: '已绑定', icon: 'success' })
    enterApp()
  } catch (e: any) {
    errMsg.value = e?.message || '手机号绑定失败'
  }
}

function enterApp() {
  const onboardingDone = !!uni.getStorageSync('onboarding_completed')
  uni.reLaunch({ url: onboardingDone ? '/pages/checkin/checkin' : '/pages/onboarding/onboarding' })
}

onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; box-sizing: border-box; padding: 44rpx 32rpx; background: #f6f7fb; }
.panel { padding: 34rpx; border: 1rpx solid #e9edf5; border-radius: 16rpx; background: #fff; }
.title { color: #111827; font-size: 38rpx; font-weight: 800; }
.desc { margin-top: 14rpx; color: #606a80; font-size: 26rpx; line-height: 1.55; }
.profile-row { display: flex; align-items: center; gap: 18rpx; margin: 30rpx 0; }
.avatar { width: 88rpx; height: 88rpx; border-radius: 16rpx; background: #eef4ff; }
.placeholder { display: flex; align-items: center; justify-content: center; color: #2264d1; font-size: 34rpx; font-weight: 800; }
.name { color: #111827; font-size: 30rpx; font-weight: 800; }
.status { margin-top: 6rpx; color: #7b8498; font-size: 24rpx; }
.primary-btn, .secondary-btn, .ghost-btn, .dark-btn { height: 88rpx; line-height: 88rpx; border-radius: 12rpx; font-size: 28rpx; }
.primary-btn { margin-top: 18rpx; background: #2264d1; color: #fff; }
.secondary-btn { background: #eef4ff; color: #2264d1; }
.ghost-btn { margin-top: 18rpx; background: #f3f6fb; color: #384257; }
.dev { margin-top: 22rpx; }
.section-title { color: #111827; font-size: 28rpx; font-weight: 800; margin-bottom: 18rpx; }
.field-input { box-sizing: border-box; width: 100%; height: 80rpx; padding: 0 22rpx; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; font-size: 27rpx; }
.dark-btn { margin-top: 18rpx; background: #111827; color: #fff; }
.error { margin-top: 24rpx; padding: 22rpx; border-radius: 12rpx; background: #fff1f0; color: #cf1322; font-size: 25rpx; }
</style>
