<template>
  <view class="page">
    <view class="panel">
      <view class="title">管理员</view>
      <view class="desc">首个注册用户为管理员。非管理员访问会返回无权访问。</view>
      <view class="config-row">
        <input v-model.number="checkinMinutes" type="number" placeholder="打卡奖励分钟" />
        <button class="small" @click="saveConfig">保存奖励</button>
      </view>
    </view>

    <view class="section-title">用户</view>
    <view class="user" v-for="u in users" :key="u.id">
      <view>
        <view class="name">#{{ u.id }} {{ u.nickname || u.openid }}</view>
        <view class="meta">{{ u.role }} · 躺平 {{ u.slack_balance || 0 }} 分</view>
        <view class="ban" v-if="u.banned_until">封禁至：{{ String(u.banned_until).slice(0, 16).replace('T', ' ') }}</view>
      </view>
      <view class="ops" v-if="u.role !== 'admin'">
        <button @click="ban(u, 24)">封1天</button>
        <button @click="unban(u)">解封</button>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { AdminApi } from '@/api'

const users = ref<any[]>([])
const checkinMinutes = ref(10)

async function load() {
  try {
    const [u, cfgs] = await Promise.all([AdminApi.users(), AdminApi.slackConfigs().catch(() => [])])
    users.value = u.users || []
    const global = (cfgs || []).find((c: any) => !c.user_id)
    if (global) checkinMinutes.value = global.checkin_minutes
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  }
}

async function saveConfig() {
  try {
    await AdminApi.saveSlackConfig({ checkin_minutes: checkinMinutes.value, streak_bonus: 0, quality_bonus: 0 })
    uni.showToast({ title: '已保存', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '保存失败', icon: 'none' })
  }
}

async function ban(u: any, hours: number) {
  try {
    await AdminApi.ban(u.id, hours, '管理员封禁')
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '封禁失败', icon: 'none' })
  }
}

async function unban(u: any) {
  try {
    await AdminApi.unban(u.id)
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '解封失败', icon: 'none' })
  }
}

onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel, .user { background: #fff; border: 1rpx solid #e9edf5; border-radius: 16rpx; }
.panel { padding: 32rpx; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.desc { margin: 12rpx 0 24rpx; color: #7b8498; font-size: 25rpx; line-height: 1.5; }
.config-row { display: grid; grid-template-columns: 1fr 180rpx; gap: 14rpx; }
.config-row input { height: 76rpx; padding: 0 20rpx; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; }
.small { margin: 0; height: 76rpx; line-height: 76rpx; background: #2264d1; color: #fff; border-radius: 12rpx; font-size: 24rpx; }
.section-title { margin: 34rpx 0 16rpx; color: #111827; font-size: 30rpx; font-weight: 800; }
.user { display: flex; justify-content: space-between; gap: 18rpx; padding: 24rpx; margin-bottom: 14rpx; }
.name { color: #111827; font-size: 27rpx; font-weight: 700; }
.meta, .ban { margin-top: 8rpx; color: #7b8498; font-size: 22rpx; }
.ban { color: #cf1322; }
.ops { display: flex; flex-direction: column; gap: 10rpx; width: 120rpx; }
.ops button { margin: 0; height: 48rpx; line-height: 48rpx; border-radius: 8rpx; font-size: 21rpx; background: #eef4ff; color: #2264d1; }
</style>
