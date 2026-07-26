<template>
  <view class="page">
    <view class="hero">
      <view>
        <view class="title">学习小组</view>
        <view class="subtitle">{{ group ? group.name : '还没有加入小组' }}</view>
      </view>
      <view class="role" v-if="member">{{ member.role === 'leader' ? '组长' : '成员' }}</view>
    </view>
	<view class="feedback" :class="feedbackType" v-if="feedback">{{ feedback }}</view>
	<view class="panel loading" v-if="loading">正在读取小组信息...</view>

    <view class="panel" v-if="!group">
      <view class="panel-title">创建或加入</view>
      <input v-model="groupName" placeholder="小组名称" />
      <button class="primary" @click="createGroup">创建小组</button>
      <input v-model="joinCode" placeholder="输入邀请码" />
      <button class="secondary" @click="joinGroup">加入小组</button>
    </view>

    <view class="panel" v-if="!group && history.length">
      <view class="panel-title">历史小组</view>
      <view class="rank" v-for="item in history" :key="item.id">
        <view>{{ item.name }}</view>
        <view>{{ item.ended_at ? item.ended_at.slice(0, 10) : '已结束' }}</view>
      </view>
    </view>

    <view v-if="group">
      <view class="panel">
        <view class="panel-title">邀请</view>
        <view class="invite-code" v-if="inviteCode">{{ inviteCode }}</view>
        <view class="hint" v-if="shareLink">{{ shareLink }}</view>
        <button class="primary" @click="createInvite">生成 7 天邀请码</button>
        <button class="secondary" @click="revokeInvite">作废邀请码</button>
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
			<button v-if="isLeader && m.role !== 'leader'" @click="transferLeader(m)">转让</button>
			<button class="remove" v-if="isLeader && m.role !== 'leader'" @click="removeMember(m)">移除</button>
          </view>
        </view>
				<view class="empty" v-if="!members.length">暂时没有可展示的成员数据</view>
      </view>

      <view class="panel">
        <view class="panel-title">排行榜</view>
        <view class="hint">用于小组内互相鼓励，不代表能力评价。</view>
        <view class="tabs"><button @click="loadLeaderboard('weekly')">本周</button><button @click="loadLeaderboard('all')">全部</button></view>
        <view class="rank" v-for="(row, index) in leaderboard" :key="row.user_id">
          <view>#{{ index + 1 }} {{ row.nickname || `用户 #${row.user_id}` }}</view>
          <view>{{ row.streak }} 天 · {{ row.study_minutes }} 分</view>
        </view>
				<view class="empty" v-if="!leaderboard.length">当前周期还没有排行数据</view>
      </view>

      <view class="panel actions">
        <button class="secondary" v-if="isLeader" @click="renameGroup">改名</button>
        <button class="danger" v-if="isLeader" @click="endGroup">结束小组</button>
        <button class="danger" v-else @click="leaveGroup">退出小组</button>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad, onShareAppMessage, onShow } from '@dcloudio/uni-app'
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
const feedback = ref('')
const feedbackType = ref<'success' | 'error'>('success')
const loading = ref(false)
const isLeader = computed(() => member.value?.role === 'leader')

async function load() {
	loading.value = true
	try {
		const current = await GroupApi.current()
		group.value = current.group || null
		member.value = current.member || null
		members.value = []
		leaderboard.value = []
		const requests: Promise<unknown>[] = group.value
			? [GroupApi.members(), GroupApi.leaderboard('weekly'), GroupApi.history()]
			: [Promise.resolve([]), Promise.resolve({ rows: [] }), GroupApi.history()]
		const [memberResult, rankResult, historyResult] = await Promise.all(requests.map(settle))
		if (memberResult.status === 'fulfilled') members.value = Array.isArray(memberResult.value) ? memberResult.value as GroupMemberView[] : []
		if (rankResult.status === 'fulfilled') leaderboard.value = Array.isArray((rankResult.value as any)?.rows) ? (rankResult.value as any).rows : []
		history.value = historyResult.status === 'fulfilled' && Array.isArray(historyResult.value) ? historyResult.value as StudyGroup[] : []
		const failures = [memberResult, rankResult, historyResult].filter(result => result.status === 'rejected').length
		if (failures) showFeedback(`${failures} 个小组区域加载失败，可稍后重试`, 'error')
	} catch (error: any) {
		group.value = null
		member.value = null
		showFeedback(error?.message || '小组信息加载失败', 'error')
	} finally { loading.value = false }
}

function settle<T>(request: Promise<T>): Promise<{ status: 'fulfilled'; value: T } | { status: 'rejected'; reason: unknown }> {
	return request.then(value => ({ status: 'fulfilled' as const, value }), reason => ({ status: 'rejected' as const, reason }))
}

async function createGroup() {
  try {
    await GroupApi.create({ name: groupName.value || '学习小组' })
    await load()
	showFeedback('小组已创建')
  } catch (e: any) {
	showFeedback(e?.message || '创建失败', 'error')
  }
}

async function joinGroup() {
  if (!joinCode.value.trim()) return
  try {
    await GroupApi.join(joinCode.value.trim())
    await load()
	showFeedback('已加入小组')
  } catch (e: any) {
	showFeedback(e?.message || '加入失败', 'error')
  }
}

async function createInvite() {
  if (!group.value) return
	try {
		const inv = await GroupApi.invite(group.value.id)
		inviteCode.value = inv.code
		shareLink.value = inv.share_link
		showFeedback('新邀请码已生成，旧邀请码已失效')
	} catch (error: any) {
		showFeedback(error?.message || '邀请码生成失败', 'error')
	}
}

async function revokeInvite() {
  if (!group.value) return
	if (!await confirmAction('作废邀请码', '确认作废小组当前有效的邀请码？已分享的邀请码将无法再使用。', '确认作废')) return
	try {
		await GroupApi.revokeInvite(group.value.id)
		inviteCode.value = ''
		shareLink.value = ''
		showFeedback('邀请码已作废')
	} catch (error: any) {
		showFeedback(error?.message || '邀请码作废失败', 'error')
	}
}

async function loadLeaderboard(scope: 'weekly' | 'all') {
	try {
		const res = await GroupApi.leaderboard(scope)
		leaderboard.value = Array.isArray(res?.rows) ? res.rows : []
	} catch (error: any) {
		showFeedback(error?.message || '排行榜加载失败', 'error')
	}
}

async function renameGroup() {
  if (!group.value) return
  const res = await modalInput('小组名称', group.value.name)
  if (!res) return
	try {
		await GroupApi.update(group.value.id, { name: res })
		await load()
		showFeedback('小组名称已更新')
	} catch (error: any) {
		showFeedback(error?.message || '改名失败', 'error')
	}
}

async function transferLeader(target: GroupMemberView) {
  if (!group.value) return
	const name = memberName(target)
	if (!await confirmAction('转让组长', `确认将组长转让给「${name}」？转让后你将成为普通成员。`, '确认转让')) return
	try {
		await GroupApi.transfer(group.value.id, target.user_id)
		await load()
		showFeedback(`已将组长转让给「${name}」`)
	} catch (error: any) {
		showFeedback(error?.message || '转让失败', 'error')
	}
}

async function removeMember(target: GroupMemberView) {
	if (!group.value) return
	const name = memberName(target)
	if (!await confirmAction('移除成员', `确认将「${name}」移出小组？对方之后可通过新邀请码再次加入。`, '确认移除')) return
	try {
		await GroupApi.remove(group.value.id, target.user_id)
		await load()
		showFeedback(`已移除「${name}」`)
	} catch (error: any) {
		showFeedback(error?.message || '移除失败', 'error')
	}
}

async function endGroup() {
  if (!group.value) return
	if (!await confirmAction('结束小组', '确认结束小组？所有成员将退出，当前邀请码也会失效。', '确认结束')) return
	try {
		await GroupApi.end(group.value.id)
		await load()
		showFeedback('小组已结束')
	} catch (error: any) {
		showFeedback(error?.message || '结束小组失败', 'error')
	}
}

async function leaveGroup() {
  if (!group.value) return
	if (!await confirmAction('退出小组', '确认退出当前小组？之后需要新的有效邀请码才能重新加入。', '确认退出')) return
	try {
		await GroupApi.leave(group.value.id)
		await load()
		showFeedback('已退出小组')
	} catch (error: any) {
		showFeedback(error?.message || '退出小组失败', 'error')
	}
}

async function nudge(userId: number) {
  if (!group.value) return
  try {
    const result = await GroupApi.nudge(group.value.id, userId)
	if (result.status === 'sent') showFeedback('提醒已发送')
	else showFeedback(result.message || '提醒未发送，请确认对方已授权微信提醒', 'error')
  } catch (e: any) {
	showFeedback(e?.message || '提醒失败', 'error')
  }
}

function modalInput(title: string, placeholderText: string) {
  return new Promise<string | null>(resolve => {
    uni.showModal({ title, editable: true, placeholderText, success: res => resolve(res.confirm ? (res.content || placeholderText) : null) })
  })
}

function confirmAction(title: string, content: string, confirmText: string) {
	return new Promise<boolean>(resolve => {
		uni.showModal({ title, content, confirmText, confirmColor: '#be123c', success: result => resolve(result.confirm), fail: () => resolve(false) })
	})
}

function memberName(target: GroupMemberView) {
	return target.nickname || `用户 #${target.user_id}`
}

function showFeedback(message: string, type: 'success' | 'error' = 'success') {
	feedback.value = message
	feedbackType.value = type
	uni.showToast({ title: message, icon: type === 'success' ? 'success' : 'none', duration: 2500 })
}

onLoad((query: any) => {
  if (query?.code) joinCode.value = String(query.code)
})
onShareAppMessage(() => ({ title: group.value ? `加入「${group.value.name}」一起学习` : '加入学习小组', path: shareLink.value || '/pages/group/group' }))
onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.hero, .panel { padding: 30rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.hero { display: flex; justify-content: space-between; align-items: center; }
.feedback { margin-top: 20rpx; padding: 20rpx 24rpx; border-radius: 12rpx; color: #166534; background: #dcfce7; font-size: 24rpx; }
.feedback.error { color: #991b1b; background: #fee2e2; }
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
.member-actions button.remove { color: #be123c; background: #fff1f2; }
.tabs { display: grid; grid-template-columns: 1fr 1fr; gap: 12rpx; }
.actions { display: grid; grid-template-columns: 1fr 1fr; gap: 12rpx; }
.loading, .empty { color: #7b8498; font-size: 24rpx; text-align: center; }
</style>
