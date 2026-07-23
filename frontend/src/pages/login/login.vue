<template>
  <view class="page">
    <view class="topbar">
      <view class="brand-mark">学</view>
      <view>
        <view class="brand-name">学习花园</view>
        <view class="brand-sub">把每天的小努力养成会发光的习惯</view>
      </view>
    </view>

    <view class="panel hero">
      <view class="hero-kicker">今日份学习魔法</view>
      <view class="hero-title">把大目标养成每天会发芽的小任务</view>
      <view class="hero-copy">计划、打卡、AI 拆解、小组陪伴和甜甜休息券，都放在一个轻盈的小花园里。</view>
      <button class="primary-btn" @click="onLogin">微信登录</button>
      <button class="secondary-btn" v-if="isDev" @click="showDev = !showDev">{{ showDev ? '收起调试' : '本地调试' }}</button>
    </view>

    <view class="panel dev" v-if="showDev">
      <view class="section-title">本地调试</view>
      <view class="field">
        <text class="field-label">API 地址</text>
        <input class="field-input" v-model="apiBase" placeholder="http://localhost:8080" />
      </view>
      <view class="field">
        <text class="field-label">Mock Code</text>
        <input class="field-input" v-model="mockCode" placeholder="test_user" />
      </view>
      <view class="dev-actions">
        <button class="ghost-btn" @click="saveApiBase">保存地址</button>
        <button class="dark-btn" @click="mockLogin">Mock 登录</button>
      </view>
    </view>

    <view class="error" v-if="errMsg">{{ errMsg }}</view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { AuthApi, type LoginResp } from '@/api'
import { setToken, setApiBase, getApiBase } from '@/api/request'

const errMsg = ref('')
const showDev = ref(false)
const isDev = import.meta.env.DEV
const apiBase = ref(getApiBase())
const mockCode = ref('test_user_' + Math.floor(Math.random() * 10000))

function saveApiBase() {
  setApiBase(apiBase.value.trim())
  uni.showToast({ title: '已保存', icon: 'success' })
}

function afterLogin(resp: LoginResp) {
  setToken(resp.token)
  uni.showToast({ title: '登录成功', icon: 'success' })
  const onboardingDone = !!uni.getStorageSync('onboarding_completed')
  const url = onboardingDone ? '/pages/checkin/checkin' : '/pages/onboarding/onboarding'
  setTimeout(() => uni.reLaunch({ url }), 300)
}

async function onLogin() {
  errMsg.value = ''
  try {
    const loginRes: any = await new Promise((resolve, reject) => {
      uni.login({ provider: 'weixin', success: resolve, fail: reject })
    })
    const code = loginRes?.code || ''
    if (!code) {
      errMsg.value = '获取微信 code 失败，请使用本地调试模式。'
      return
    }
    const resp = await AuthApi.login(code, '', '')
    afterLogin(resp)
  } catch (e: any) {
    errMsg.value = e?.message || '登录失败'
  }
}

async function mockLogin() {
  errMsg.value = ''
  try {
    setApiBase(apiBase.value.trim())
    const resp = await AuthApi.login(mockCode.value, '测试用户', '')
    afterLogin(resp)
  } catch (e: any) {
    errMsg.value = e?.message || 'mock 登录失败'
  }
}
</script>

<style lang="scss">
.page { min-height: 100vh; box-sizing: border-box; padding: 44rpx 32rpx; background: linear-gradient(180deg, #fff0f7 0%, #fffaf0 48%, #f7fbff 100%); }
.topbar {
  display: flex;
  align-items: center;
  gap: 20rpx;
  margin-bottom: 40rpx;
}
.brand-mark {
  display: flex;
  width: 72rpx;
  height: 72rpx;
  align-items: center;
  justify-content: center;
  border-radius: 24rpx;
  background: linear-gradient(135deg, #ff8fab, #ffc36a);
  color: #fff;
  font-size: 34rpx;
  font-weight: 700;
}
.brand-name { color: #111827; font-size: 34rpx; font-weight: 700; }
.brand-sub { margin-top: 4rpx; color: #7b8498; font-size: 23rpx; }
.panel {
  background: #fff;
  border: 1rpx solid #e9edf5;
  border-radius: 30rpx;
  box-shadow: 0 18rpx 42rpx rgba(255, 143, 171, 0.16);
}
.hero { padding: 44rpx 36rpx 36rpx; }
.hero-kicker {
  color: #ff6f91;
  font-size: 22rpx;
  font-weight: 700;
  text-transform: uppercase;
}
.hero-title {
  margin-top: 20rpx;
  color: #111827;
  font-size: 46rpx;
  line-height: 1.22;
  font-weight: 800;
}
.hero-copy {
  margin: 22rpx 0 38rpx;
  color: #606a80;
  font-size: 27rpx;
  line-height: 1.6;
}
.primary-btn,
.secondary-btn,
.dark-btn,
.ghost-btn {
  height: 88rpx;
  line-height: 88rpx;
  border-radius: 12rpx;
  font-size: 29rpx;
}
.primary-btn { background: linear-gradient(135deg, #ff7aa2, #ffb45c); color: #fff; }
.secondary-btn { margin-top: 20rpx; background: #fff0f6; color: #ff6f91; }
.dev { margin-top: 24rpx; padding: 30rpx; }
.section-title { color: #111827; font-size: 30rpx; font-weight: 700; margin-bottom: 24rpx; }
.field { margin-bottom: 22rpx; }
.field-label { display: block; color: #606a80; font-size: 24rpx; margin-bottom: 10rpx; }
.field-input {
  box-sizing: border-box;
  width: 100%;
  height: 80rpx;
  padding: 0 22rpx;
  border: 1rpx solid #dbe2ee;
  border-radius: 12rpx;
  background: #f9fbff;
  color: #111827;
  font-size: 27rpx;
}
.dev-actions { display: grid; grid-template-columns: 1fr 1fr; gap: 18rpx; }
.ghost-btn { background: #f3f6fb; color: #384257; }
.dark-btn { background: #111827; color: #fff; }
.error {
  margin-top: 24rpx;
  padding: 22rpx;
  border-radius: 12rpx;
  background: #fff1f0;
  color: #cf1322;
  font-size: 25rpx;
}
</style>
