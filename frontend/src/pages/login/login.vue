<template>
  <view class="login-page">
    <view class="logo">📚</view>
    <view class="title">学习打卡</view>
    <view class="subtitle">坚持每日学习 · 赚躺平币</view>

    <button class="btn-login" open-type="getUserInfo" @click="onLogin">微信一键登录</button>

    <view class="hint" v-if="errMsg">{{ errMsg }}</view>
    <view class="hint small">登录后即可创建学习计划并每日打卡</view>

    <view class="dev-panel" v-if="showDev">
      <view class="dev-title">本地调试</view>
      <view class="dev-row">
        <text>API 地址：</text>
        <input v-model="apiBase" placeholder="http://localhost:8080" />
      </view>
      <button size="mini" @click="saveApiBase">保存</button>
      <view class="dev-row">
        <text>mock code：</text>
        <input v-model="mockCode" placeholder="任意字符串" />
      </view>
      <button size="mini" @click="mockLogin">mock 登录</button>
    </view>
    <view class="dev-toggle" @click="showDev = !showDev">{{ showDev ? '收起' : '开发模式 →' }}</view>
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
  uni.showToast({ title: '已保存' })
}

function afterLogin(resp: LoginResp) {
  setToken(resp.token)
  uni.showToast({ title: '登录成功', icon: 'success' })
  setTimeout(() => uni.reLaunch({ url: '/pages/checkin/checkin' }), 300)
}

async function onLogin() {
  errMsg.value = ''
  try {
    const loginRes: any = await new Promise((resolve, reject) =>
      uni.login({
        provider: 'weixin',
        success: resolve,
        fail: reject,
      })
    )
    const code = loginRes?.code || ''
    if (!code) {
      errMsg.value = '获取微信 code 失败'
      return
    }
    const resp = await AuthApi.login(code, '', '')
    afterLogin(resp)
  } catch (e: any) {
    errMsg.value = e?.message || '登录失败'
  }
}

async function mockLogin() {
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
.login-page {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 120rpx;
  height: 100vh;
  background: linear-gradient(180deg, #eef4ff 0%, #f8f8f8 100%);
}
.logo { font-size: 160rpx; margin-bottom: 30rpx; }
.title { font-size: 48rpx; font-weight: 600; color: #333; }
.subtitle { font-size: 26rpx; color: #888; margin-top: 16rpx; margin-bottom: 100rpx; }
.btn-login {
  width: 560rpx;
  background: #07c160;
  color: #fff;
  border-radius: 48rpx;
  font-size: 32rpx;
  margin-bottom: 30rpx;
}
.hint { color: #e53935; font-size: 26rpx; margin-bottom: 20rpx; }
.hint.small { color: #999; }
.dev-panel {
  margin-top: 60rpx;
  width: 600rpx;
  padding: 30rpx;
  background: #fff;
  border-radius: 16rpx;
  border: 1rpx solid #eee;
}
.dev-title { font-size: 28rpx; color: #333; margin-bottom: 20rpx; font-weight: 600; }
.dev-row { display: flex; align-items: center; margin: 20rpx 0; font-size: 26rpx; color: #555; }
.dev-row input { flex: 1; margin-left: 12rpx; border: 1rpx solid #ddd; border-radius: 8rpx; padding: 8rpx 16rpx; }
.dev-toggle { margin-top: 40rpx; color: #4C8BF5; font-size: 24rpx; }
</style>