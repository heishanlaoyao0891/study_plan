<template>
  <view class="page">
    <view class="topbar">
      <view class="brand-mark">学</view>
      <view>
        <view class="brand-name">Study Plan</view>
        <view class="brand-sub">计划、打卡、奖励闭环</view>
      </view>
    </view>

    <view class="panel hero">
      <view class="hero-kicker">Personal Learning Console</view>
      <view class="hero-title">把学习计划变成每天能完成的动作</view>
      <view class="hero-copy">先记录，后优化。MVP 当前支持手动计划、今日打卡和基础登录。</view>
      <button class="primary-btn" @click="onLogin">微信登录</button>
      <button class="secondary-btn" @click="showDev = !showDev">{{ showDev ? '收起调试' : '本地调试' }}</button>
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
const apiBase = ref(getApiBase())
const mockCode = ref('test_user_' + Math.floor(Math.random() * 10000))

function saveApiBase() {
  setApiBase(apiBase.value.trim())
  uni.showToast({ title: '已保存', icon: 'success' })
}

function afterLogin(resp: LoginResp) {
  setToken(resp.token)
  uni.showToast({ title: '登录成功', icon: 'success' })
  setTimeout(() => uni.reLaunch({ url: '/pages/checkin/checkin' }), 300)
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
.page {
  min-height: 100vh;
  box-sizing: border-box;
  padding: 44rpx 32rpx;
  background: #f6f7fb;
}
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
  border-radius: 18rpx;
  background: #111827;
  color: #fff;
  font-size: 34rpx;
  font-weight: 700;
}
.brand-name { color: #111827; font-size: 34rpx; font-weight: 700; }
.brand-sub { margin-top: 4rpx; color: #7b8498; font-size: 23rpx; }
.panel {
  background: #fff;
  border: 1rpx solid #e9edf5;
  border-radius: 16rpx;
  box-shadow: 0 10rpx 30rpx rgba(19, 35, 78, 0.06);
}
.hero { padding: 44rpx 36rpx 36rpx; }
.hero-kicker {
  color: #2264d1;
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
.primary-btn { background: #2264d1; color: #fff; }
.secondary-btn { margin-top: 20rpx; background: #eef4ff; color: #2264d1; }
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
