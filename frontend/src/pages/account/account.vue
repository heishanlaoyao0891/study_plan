<template>
  <view class="page">
    <view class="panel" v-if="user">
      <view class="title">账户设置</view>
      <view class="row">
        <view>
          <view class="label">手机号</view>
          <view class="value">{{ maskedPhone }}</view>
        </view>
        <button class="secondary" @click="rebindPhone">更换手机号</button>
      </view>
      <view class="row">
        <view>
          <view class="label">账号状态</view>
          <view class="value">{{ user.account_status || 'active' }}</view>
        </view>
      </view>
    </view>

    <view class="panel">
      <view class="title">数据管理</view>
      <view class="desc">保留会继续保存你的学习记录；删除会清除个人与学习数据，但保留账号壳以便未来同身份恢复。</view>
      <button class="primary" @click="deactivate(true)">停用并保留数据</button>
      <button class="danger" @click="deactivate(false)">删除并匿名化数据</button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { AuthApi, type User } from '@/api'

const user = ref<User | null>(null)
const maskedPhone = computed(() => {
  const phone = user.value?.phone_number || ''
  if (!phone || phone.length < 7) return phone || '--'
  return `${phone.slice(0, 3)}****${phone.slice(-4)}`
})

async function load() {
  user.value = await AuthApi.me()
}

function rebindPhone() {
  uni.navigateTo({ url: '/pages/bind/bind' })
}

async function deactivate(retain: boolean) {
  const note = retain ? '用户选择保留数据' : '用户选择删除并匿名化数据'
  const ok = await new Promise<boolean>(resolve => {
    uni.showModal({ title: retain ? '停用并保留' : '删除并匿名化', content: retain ? '停用后未来可重新登录恢复数据。' : '删除后个人与学习数据将被清理。', success: res => resolve(!!res.confirm) })
  })
  if (!ok) return
  user.value = await AuthApi.deactivate(retain, note)
  uni.showToast({ title: retain ? '已停用' : '已删除', icon: 'success' })
}

onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel { padding: 30rpx; margin-bottom: 18rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; margin-bottom: 18rpx; }
.row { display: flex; align-items: center; justify-content: space-between; gap: 16rpx; padding: 18rpx 0; border-top: 1rpx solid #eef2f7; }
.label { color: #7b8498; font-size: 23rpx; }
.value { margin-top: 8rpx; color: #111827; font-size: 28rpx; font-weight: 800; }
.desc { color: #606a80; font-size: 24rpx; line-height: 1.6; margin-bottom: 18rpx; }
.secondary, .primary, .danger { margin: 0; height: 76rpx; line-height: 76rpx; border-radius: 10rpx; font-size: 25rpx; }
.secondary { background: #eef4ff; color: #2264d1; }
.primary { background: #111827; color: #fff; margin-top: 12rpx; }
.danger { background: #fff1f0; color: #cf1322; margin-top: 12rpx; }
</style>
