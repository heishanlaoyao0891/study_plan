<template>
  <view class="page">
    <view class="panel" v-if="user">
      <view class="title">账号与数据</view>
      <view class="profile">
        <image class="avatar" v-if="user.avatar_url" :src="user.avatar_url" mode="aspectFill" />
        <view class="avatar placeholder" v-else>{{ (user.nickname || '学').slice(0, 1) }}</view>
        <view><view class="nickname">{{ user.nickname }}</view><view class="status">账号状态：{{ user.account_status || 'active' }}</view></view>
      </view>
      <button class="secondary" @click="goNickname">修改昵称</button>
    </view>

    <view class="panel">
      <view class="title">数据管理</view>
      <view class="desc">停用并保留会继续保存学习记录；删除并匿名化会清理个人与学习数据。</view>
      <button class="primary" @click="deactivate(true)">停用并保留数据</button>
      <button class="danger" @click="deactivate(false)">删除并匿名化数据</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { AuthApi, type User } from '@/api'
import { clearToken } from '@/api/request'

const user = ref<User | null>(null)

async function load() {
  try { user.value = await AuthApi.me() }
  catch (error: any) { uni.showToast({ title: error?.message || '加载失败', icon: 'none' }) }
}

function goNickname() { uni.navigateTo({ url: '/pages/nickname/nickname?mode=edit' }) }

async function deactivate(retain: boolean) {
  const confirmed = await new Promise<boolean>(resolve => uni.showModal({
    title: retain ? '停用并保留' : '删除并匿名化',
    content: retain ? '停用后可使用同一微信身份重新登录恢复数据。' : '删除后个人与学习数据将被清理，此操作不可恢复。',
    confirmColor: retain ? '#111827' : '#c95668',
    success: result => resolve(result.confirm),
  }))
  if (!confirmed) return
  await AuthApi.deactivate(retain, retain ? '用户选择保留数据' : '用户选择删除并匿名化数据')
  clearToken()
  uni.showToast({ title: retain ? '已停用' : '已删除', icon: 'success' })
  uni.reLaunch({ url: '/pages/login/login' })
}

onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; box-sizing: border-box; padding: 28rpx; background: linear-gradient(180deg,#fff0f7,#fffaf0 46%,#f7fbff); }
.panel { margin-bottom: 20rpx; padding: 30rpx; border: 1rpx solid #ffe0ea; border-radius: 28rpx; background: #fff; box-shadow: 0 14rpx 32rpx rgba(255,143,171,.1); }
.title { margin-bottom: 20rpx; color: #4b2b3f; font-size: 32rpx; font-weight: 900; }
.profile { display: flex; align-items: center; gap: 18rpx; }
.avatar { width: 82rpx; height: 82rpx; border-radius: 24rpx; background: #fff0f6; }
.placeholder { display: flex; align-items: center; justify-content: center; color: #ff6f91; font-size: 32rpx; font-weight: 900; }
.nickname { color: #111827; font-size: 29rpx; font-weight: 800; }
.status,.desc { margin-top: 7rpx; color: #7b8498; font-size: 23rpx; line-height: 1.6; }
.secondary,.primary,.danger { margin-top: 18rpx; border-radius: 999rpx; }
.secondary { background: #fff0f6; color: #ff6f91; }
.primary { background: #111827; color: #fff; }
.danger { background: #f7f4f4; color: #9a7379; }
</style>
