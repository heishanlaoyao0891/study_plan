<template>
  <view class="page">
    <view class="hero">
      <view>
        <view class="title">学习小组</view>
        <view class="subtitle">{{ group ? group.name : '还没有加入小组' }}</view>
      </view>
      <view class="role" v-if="member">{{ member.role === 'leader' ? '组长' : '成员' }}</view>
    </view>

    <view class="panel" v-if="!group">
      <view class="panel-title">创建或加入</view>
      <input v-model="groupName" placeholder="小组名称" />
      <button class="primary" @click="createGroup">创建小组</button>
      <input v-model="joinCode" placeholder="输入邀请码" />
      <button class="secondary" @click="joinGroup">加入小组</button>
    </view>

    <view class="panel" v-if="history.length">
      <view class="panel-title">历史小组</view>
      <view class="rank" v-for="item in history" :key="item.id">
        <view>{{ item.name }}</view>
        <view>{{ item.ended_at ? item.ended_at.slice(0, 10) : '已结束' }}</view>
      </view>
    </view>

    <view v-else>
      <view class="panel">
        <view class="panel-title">邀请</view>
        <view class="invite-code" v-if="inviteCode">{{ inviteCode }}</view>
        <view class="hint" v-if="shareLink">{{ shareLink }}</view>
        <button class="primary" v-if="isLeader" @click="createInvite">生成 7 天邀请码</button>
        <button class="secondary" v-if="isLeader" @click="revokeInvite">作废邀请码</button>
      </view>

      <view class="panel">
        <view class="panel-title">成员状态</view>
        <view class="member" v-for="m in members" :key="m.user_id">
          <view>
            <view class="member-name">{{ m.nickname || `用户 #${m.user_id}` }} <text>Lv{{ m.level }}</text></view>
            <view class="member-meta">连续 {{ m.streak }} 天 · {{ m.study_minutes }} 分钟 · 完成率 {{ m.completion_rate }}%</view>
          </view>
          <view class="member-actions">
            <view class="done" :class="{ ok: m.today_completed }">{{ m.today_completed ? '今日完成' : '今日未完' }}</view>
            <button v-if="member && m.user_id !== member.user_id" @click="nudge(m.user_id)">提醒</button>
          </view>
        </view>
      </view>

      <view class="panel">
        <view class="panel-title">排行榜</view>
        <view class="tabs"><button @click="loadLeaderboard('weekly')">本周</button><button @click="loadLeaderboard('all')">全部</button></view>
        <view class="rank" v-for="(row, index) in leaderboard" :key="row.user_id">
          <view>#{{ index + 1 }} {{ row.nickname || `用户 #${row.user_id}` }}</view>
          <view>{{ row.streak }} 天 · {{ row.study_minutes }} 分</view>
        </view>
      </view>

      <view class="panel actions">
        <button class="secondary" v-if="isLeader" @click="renameGroup">改名</button>
        <button class="secondary" v-if="isLeader" @click="transferLeader">转让组长</button>
        <button class="danger" v-if="isLeader" @click="endGroup">结束小组</button>
        <button class="danger" v-else @click="leaveGroup">退出小组</button>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { GroupApi, type GroupMemberView, type StudyGroup, type StudyGroupMember } from '@/api'

const group = ref<StudyGroup | null>(null)
const member = ref<StudyGroupMember | null>(null)
const members = ref<GroupMemberView[]>([])
const leaderboard = ref<GroupMemberView[]>([])
const history = ref<StudyGroup[]>([])
const groupName = ref('学习小组')
const joinCode = ref('')
const inviteCode = ref('')
const shareLink = ref('')
const isLeader = computed(() => member.value?.role === 'leader')

async function load() {
  const current = await GroupApi.current().catch(() => null)
  group.value = current?.group || null
  member.value = current?.member || null
  if (group.value) {
    members.value = await GroupApi.members().catch(() => [])
    await loadLeaderboard('weekly')
  }
  history.value = await GroupApi.history().catch(() => [])
}

async function createGroup() {
  await GroupApi.create({ name: groupName.value || '学习小组' })
  await load()
}

async function joinGroup() {
  if (!joinCode.value.trim()) return
  await GroupApi.join(joinCode.value.trim())
  await load()
}

async function createInvite() {
  if (!group.value) return
  const inv = await GroupApi.invite(group.value.id)
  inviteCode.value = inv.code
  shareLink.value = inv.share_link
}

async function revokeInvite() {
  if (!group.value) return
  await GroupApi.revokeInvite(group.value.id)
  inviteCode.value = ''
  shareLink.value = ''
}

async function loadLeaderboard(scope: 'weekly' | 'all') {
  const res = await GroupApi.leaderboard(scope).catch(() => ({ rows: [] }))
  leaderboard.value = res.rows || []
}

async function renameGroup() {
  if (!group.value) return
  const res = await modalInput('小组名称', group.value.name)
  if (!res) return
  await GroupApi.update(group.value.id, { name: res })
  await load()
}

async function transferLeader() {
  if (!group.value) return
  const res = await modalInput('转让给用户 ID', '')
  const userId = Number(res || 0)
  if (!userId) return
  await GroupApi.transfer(group.value.id, userId)
  await load()
}

async function endGroup() {
  if (!group.value) return
  await GroupApi.end(group.value.id)
  await load()
}

async function leaveGroup() {
  if (!group.value) return
  await GroupApi.leave(group.value.id)
  await load()
}

async function nudge(userId: number) {
  if (!group.value) return
  try {
    await GroupApi.nudge(group.value.id, userId)
    uni.showToast({ title: '已提醒', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '提醒失败', icon: 'none' })
  }
}

function modalInput(title: string, placeholderText: string) {
  return new Promise<string | null>(resolve => {
    uni.showModal({ title, editable: true, placeholderText, success: res => resolve(res.confirm ? (res.content || placeholderText) : null) })
  })
}

onLoad((query: any) => {
  if (query?.code) joinCode.value = String(query.code)
})
onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.hero, .panel { padding: 30rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.hero { display: flex; justify-content: space-between; align-items: center; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.subtitle, .hint, .member-meta { margin-top: 8rpx; color: #7b8498; font-size: 23rpx; }
.role, .done { color: #2264d1; font-size: 24rpx; font-weight: 800; }
.done.ok { color: #0f766e; }
.panel { margin-top: 20rpx; }
.panel-title { color: #111827; font-size: 29rpx; font-weight: 800; margin-bottom: 16rpx; }
input { height: 74rpx; margin-bottom: 14rpx; padding: 0 18rpx; border-radius: 10rpx; border: 1rpx solid #dbe2ee; background: #f9fbff; }
button { margin: 0 0 14rpx; border-radius: 10rpx; }
.primary { background: #111827; color: #fff; }
.secondary { background: #eef4ff; color: #2264d1; }
.danger { background: #fff1f2; color: #be123c; }
.invite-code { color: #111827; font-size: 38rpx; font-weight: 800; letter-spacing: 0; margin-bottom: 10rpx; }
.member, .rank { display: flex; justify-content: space-between; align-items: center; padding: 18rpx 0; border-top: 1rpx solid #eef2f7; }
.member-name { color: #111827; font-size: 27rpx; font-weight: 800; }
.member-name text { color: #0f766e; font-size: 22rpx; }
.member-actions { display: flex; align-items: center; gap: 12rpx; }
.member-actions button { margin: 0; height: 54rpx; line-height: 54rpx; border-radius: 10rpx; background: #eef4ff; color: #2264d1; font-size: 22rpx; }
.tabs { display: grid; grid-template-columns: 1fr 1fr; gap: 12rpx; }
.actions { display: grid; grid-template-columns: 1fr 1fr; gap: 12rpx; }
</style>
