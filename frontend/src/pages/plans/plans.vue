<template>
  <view class="plans-page">
    <view class="summary">
      <view>
        <view class="summary-label">学习目标</view>
        <view class="summary-title">{{ activeCount }} 个进行中</view>
      </view>
      <button class="add-btn" @click="openCreate">新建</button>
    </view>

    <view class="stats-row">
      <view class="stat-box">
        <view class="stat-value">{{ plans.length }}</view>
        <view class="stat-label">目标计划</view>
      </view>
      <view class="stat-box">
        <view class="stat-value">{{ totalWeeklyHours }}</view>
        <view class="stat-label">周目标工时</view>
      </view>
      <view class="stat-box">
        <view class="stat-value">{{ pausedCount }}</view>
        <view class="stat-label">已暂停目标</view>
      </view>
    </view>

    <view class="quick-row">
      <button class="quick" @click="goSchedule">日程</button>
      <button class="quick" @click="goGroup">小组</button>
      <button class="quick" @click="goAI">AI 生成</button>
      <button class="quick" @click="goNotifications">提醒</button>
    </view>

    <view class="empty" v-if="!loading && plans.length === 0">
      <view class="empty-title">还没有计划</view>
      <view class="empty-desc">从一个明确的目标开始，系统会按天生成可执行任务，比如「Go 基础语法，每周 7 小时」。</view>
      <button class="primary-btn" @click="openCreate">创建第一个计划</button>
    </view>

    <view class="plan-list" v-else>
      <view class="plan-card" v-for="p in plans" :key="p.id" :class="{ paused: p.status === 'paused' }">
        <view class="card-head">
          <view class="plan-title">{{ p.title }}</view>
          <view class="status-pill" :class="p.status">{{ statusText(p.status) }}</view>
        </view>
        <view class="plan-desc" v-if="p.description">{{ p.description }}</view>
        <view class="metrics">
          <view class="metric">
            <view class="metric-num">{{ p.weekly_target_hours || 0 }}</view>
          <view class="metric-label">计划工时 / 周</view>
          </view>
          <view class="metric">
            <view class="metric-num">{{ p.ai_generated ? 'AI' : '手动' }}</view>
          <view class="metric-label">任务来源</view>
          </view>
        </view>
        <view class="actions">
          <button class="action" @click="togglePause(p)">{{ p.status === 'paused' ? '恢复' : '暂停' }}</button>
          <button class="action" @click="openEdit(p)">编辑目标</button>
          <button class="action" @click="shift(p)">平移任务</button>
          <button class="action" @click="batchShift(p)">批量平移</button>
          <button class="action" @click="invite(p)">邀请</button>
          <button class="action danger" @click="del(p)">删除</button>
        </view>
      </view>
    </view>

    <view class="modal" v-if="showModal" @click.self="closeModal">
      <view class="modal-body">
        <view class="modal-title">{{ editing ? '编辑计划' : '新建计划' }}</view>
        <view class="field">
          <text class="label">计划名称</text>
          <input class="input" v-model="form.title" placeholder="例如：学习 Go 语言" />
        </view>
        <view class="field">
          <text class="label">计划说明</text>
          <textarea class="textarea" v-model="form.description" placeholder="可选，写下阶段目标或范围" />
        </view>
        <view class="field">
          <text class="label">每周目标小时</text>
          <input class="input" v-model.number="form.weekly_target_hours" type="number" placeholder="例如：7" />
        </view>
        <view class="confirm-row" v-if="!editing">
          <label class="confirm-label">
            <checkbox :checked="!!form.confirm_overload" @click="form.confirm_overload = !form.confirm_overload" />
            <text>如系统提示压力过大，仍允许创建</text>
          </label>
        </view>
        <view class="modal-actions">
          <button class="cancel" @click="closeModal">取消</button>
          <button class="submit" @click="save">{{ editing ? '保存' : '创建' }}</button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { PlanApi, type Plan, type CreatePlanReq } from '@/api'

interface FormState extends CreatePlanReq { confirm_overload?: boolean }

const plans = ref<Plan[]>([])
const loading = ref(false)
const showModal = ref(false)
const editing = ref<Plan | null>(null)
const form = reactive<FormState>({ title: '', description: '', weekly_target_hours: 0, confirm_overload: false })
const activeCount = computed(() => plans.value.filter(p => p.status === 'active').length)
const pausedCount = computed(() => plans.value.filter(p => p.status === 'paused').length)
const totalWeeklyHours = computed(() => plans.value.filter(p => p.status === 'active').reduce((sum, p) => sum + (p.weekly_target_hours || 0), 0))

async function load() {
  loading.value = true
  try { plans.value = await PlanApi.list() }
  catch (e: any) { uni.showToast({ title: e?.message || '加载失败', icon: 'none' }) }
  finally { loading.value = false }
}

function resetForm() {
  form.title = ''
  form.description = ''
  form.weekly_target_hours = 0
  form.confirm_overload = false
}

function openCreate() {
  editing.value = null
  resetForm()
  showModal.value = true
}

function openEdit(p: Plan) {
  editing.value = p
  form.title = p.title
  form.description = p.description || ''
  form.weekly_target_hours = p.weekly_target_hours || 0
  form.confirm_overload = false
  showModal.value = true
}

function closeModal() { showModal.value = false }

async function save() {
  if (!form.title?.trim()) {
    uni.showToast({ title: '请输入计划名称', icon: 'none' })
    return
  }
  try {
    if (editing.value) {
      await PlanApi.update(editing.value.id, {
        title: form.title,
        description: form.description,
        weekly_target_hours: form.weekly_target_hours,
      })
      uni.showToast({ title: '已保存', icon: 'success' })
    } else {
      await PlanApi.create({ ...form })
      uni.showToast({ title: '已创建', icon: 'success' })
    }
    showModal.value = false
    await load()
  } catch (e: any) {
    const msg = e?.message || '保存失败'
    if (/overload/i.test(msg)) {
      uni.showModal({ title: '创建压力提示', content: '当前计划可能过多或周目标过高。勾选「仍允许创建」后可继续。', showCancel: false })
    } else {
      uni.showToast({ title: msg, icon: 'none' })
    }
  }
}

async function togglePause(p: Plan) {
  try {
    const updated = p.status === 'paused' ? await PlanApi.resume(p.id) : await PlanApi.pause(p.id)
    Object.assign(p, updated)
  } catch (e: any) {
    uni.showToast({ title: e?.message || '操作失败', icon: 'none' })
  }
}

async function del(p: Plan) {
  const ok = await new Promise<boolean>(resolve => {
    uni.showModal({ title: '删除计划', content: `删除「${p.title}」及相关打卡记录？`, success: r => resolve(r.confirm) })
  })
  if (!ok) return
  try {
    await PlanApi.remove(p.id)
    plans.value = plans.value.filter(x => x.id !== p.id)
  } catch (e: any) {
    uni.showToast({ title: e?.message || '删除失败', icon: 'none' })
  }
}

async function shift(p: Plan) {
  const res = await new Promise<any>(resolve => {
    uni.showModal({ title: '平移计划', editable: true, placeholderText: '输入天数，例如 1 或 -1', success: resolve })
  })
  if (!res.confirm) return
  const days = Number(res.content || 0)
  if (!days) {
    uni.showToast({ title: '请输入非 0 天数', icon: 'none' })
    return
  }
  try {
    await PlanApi.shift(p.id, days)
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '平移失败', icon: 'none' })
  }
}

async function batchShift(p: Plan) {
  const daysRes = await new Promise<any>(resolve => {
    uni.showModal({ title: '批量平移', editable: true, placeholderText: '输入平移天数，例如 3 或 -1', success: resolve })
  })
  if (!daysRes.confirm) return
  const days = Number(daysRes.content || 0)
  if (!days) {
    uni.showToast({ title: '请输入非 0 天数', icon: 'none' })
    return
  }
  const startRes = await new Promise<any>(resolve => {
    const tomorrow = new Date(Date.now() + 86400000).toISOString().slice(0, 10)
    uni.showModal({ title: '起始日期', editable: true, placeholderText: tomorrow, success: resolve })
  })
  if (!startRes.confirm) return
  try {
    await PlanApi.shift(p.id, days, startRes.content || new Date(Date.now() + 86400000).toISOString().slice(0, 10))
    uni.showToast({ title: '已批量平移', icon: 'success' })
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '平移失败', icon: 'none' })
  }
}

async function invite(p: Plan) {
  const res = await new Promise<any>(resolve => {
    uni.showModal({ title: '邀请用户', editable: true, placeholderText: '输入用户 ID', success: resolve })
  })
  if (!res.confirm) return
  const userId = Number(res.content || 0)
  if (!userId) {
    uni.showToast({ title: '请输入用户 ID', icon: 'none' })
    return
  }
  try {
    await PlanApi.invite(p.id, userId)
    uni.showToast({ title: '已邀请', icon: 'success' })
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '邀请失败', icon: 'none' })
  }
}

function statusText(s: string) { return s === 'paused' ? '暂停' : s === 'archived' ? '归档' : '进行' }
function goSchedule() { uni.navigateTo({ url: '/pages/schedule/schedule' }) }
function goGroup() { uni.navigateTo({ url: '/pages/group/group' }) }
function goAI() { uni.navigateTo({ url: '/pages/ai/ai' }) }
function goNotifications() { uni.navigateTo({ url: '/pages/notifications/notifications' }) }
onShow(load)
</script>

<style lang="scss">
.plans-page { min-height: 100vh; box-sizing: border-box; padding: 28rpx 28rpx 60rpx; background: #f6f7fb; }
.summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 34rpx;
  border-radius: 18rpx;
  background: #fff;
  border: 1rpx solid #e9edf5;
}
.summary-label { color: #2264d1; font-size: 22rpx; font-weight: 800; }
.summary-title { margin-top: 10rpx; color: #111827; font-size: 42rpx; font-weight: 800; }
.add-btn { margin: 0; width: 132rpx; height: 70rpx; line-height: 70rpx; background: #2264d1; color: #fff; border-radius: 12rpx; font-size: 27rpx; }
.stats-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14rpx; margin-top: 18rpx; }
.stat-box { padding: 24rpx 12rpx; border-radius: 14rpx; background: #fff; border: 1rpx solid #e9edf5; text-align: center; }
.stat-value { color: #111827; font-size: 34rpx; font-weight: 800; }
.stat-label { margin-top: 8rpx; color: #7b8498; font-size: 22rpx; }
.quick-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14rpx; margin-top: 18rpx; }
.quick { margin: 0; height: 68rpx; line-height: 68rpx; border-radius: 12rpx; background: #eef4ff; color: #2264d1; font-size: 25rpx; }
.plan-list { margin-top: 24rpx; display: flex; flex-direction: column; gap: 18rpx; }
.plan-card { padding: 28rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.plan-card.paused { opacity: .62; }
.card-head { display: flex; align-items: center; justify-content: space-between; gap: 18rpx; }
.plan-title { flex: 1; min-width: 0; color: #111827; font-size: 32rpx; font-weight: 800; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.status-pill { min-width: 76rpx; text-align: center; padding: 8rpx 0; border-radius: 99rpx; color: #2264d1; background: #eef4ff; font-size: 22rpx; font-weight: 700; }
.status-pill.paused { color: #9a5b00; background: #fff7e6; }
.plan-desc { margin-top: 14rpx; color: #606a80; font-size: 25rpx; line-height: 1.5; }
.metrics { display: grid; grid-template-columns: 1fr 1fr; gap: 14rpx; margin-top: 22rpx; }
.metric { padding: 20rpx; border-radius: 12rpx; background: #f8fafc; }
.metric-num { color: #111827; font-size: 30rpx; font-weight: 800; }
.metric-label { margin-top: 6rpx; color: #7b8498; font-size: 22rpx; }
.actions { display: grid; grid-template-columns: repeat(5, 1fr); gap: 10rpx; margin-top: 22rpx; }
.action { margin: 0; height: 64rpx; line-height: 64rpx; border-radius: 10rpx; background: #f3f6fb; color: #384257; font-size: 24rpx; }
.action.danger { color: #cf1322; background: #fff1f0; }
.empty { margin-top: 24rpx; padding: 54rpx 34rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.empty-title { color: #111827; font-size: 32rpx; font-weight: 800; }
.empty-desc { margin-top: 12rpx; color: #7b8498; font-size: 26rpx; line-height: 1.5; }
.primary-btn { margin-top: 32rpx; background: #2264d1; color: #fff; border-radius: 12rpx; }
.modal { position: fixed; inset: 0; z-index: 99; background: rgba(17,24,39,.46); display: flex; align-items: flex-end; }
.modal-body { width: 100%; box-sizing: border-box; padding: 34rpx 30rpx 42rpx; border-radius: 24rpx 24rpx 0 0; background: #fff; }
.modal-title { color: #111827; font-size: 34rpx; font-weight: 800; margin-bottom: 28rpx; }
.field { margin-bottom: 22rpx; }
.label { display: block; color: #606a80; font-size: 24rpx; margin-bottom: 10rpx; }
.input, .textarea { box-sizing: border-box; width: 100%; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; color: #111827; font-size: 27rpx; }
.input { height: 80rpx; padding: 0 20rpx; }
.textarea { height: 138rpx; padding: 18rpx 20rpx; }
.confirm-row { margin: 8rpx 0 24rpx; color: #606a80; font-size: 24rpx; }
.confirm-label { display: flex; align-items: center; gap: 10rpx; }
.modal-actions { display: grid; grid-template-columns: 1fr 1fr; gap: 16rpx; }
.cancel, .submit { margin: 0; height: 82rpx; line-height: 82rpx; border-radius: 12rpx; font-size: 28rpx; }
.cancel { background: #f3f6fb; color: #384257; }
.submit { background: #2264d1; color: #fff; }
</style>
