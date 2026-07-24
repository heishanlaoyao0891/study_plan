<template>
  <view class="page">
    <view class="panel" v-if="detail">
      <view class="title">{{ detail.task.title }}</view>
      <view class="meta">{{ detail.plan.title }}</view>
      <view class="content-block">
        <view class="content-label">任务目标</view>
        <view class="content-text">{{ detail.task.objective || '暂未填写任务目标' }}</view>
      </view>
      <view class="grid">
        <view><text>日期</text><view class="value">{{ detail.task.date }}</view></view>
        <view><text>计划时段</text><view class="value">{{ detail.task.planned_start || '--:--' }}-{{ detail.task.planned_end || '--:--' }}</view></view>
        <view><text>状态</text><view class="value">{{ statusText(detail.task.timer_state) }}</view></view>
        <view><text>累计学习</text><view class="value">{{ durationText(detail.task.accumulated_seconds) }}</view></view>
      </view>
      <view class="desc" v-if="detail.task.description">{{ detail.task.description }}</view>
      <view class="content-block reflection-block" v-if="detail.task.timer_state === 'completed'">
        <view class="content-head"><view class="content-label">完成心得</view><button @click="openReflection">{{ detail.task.reflection ? '编辑' : '补充' }}</button></view>
        <view class="content-text muted" v-if="!detail.task.reflection">还没有记录心得</view>
        <view class="content-text" v-else>{{ detail.task.reflection }}</view>
      </view>
      <view class="visibility-row">
        <view>小组可见</view>
        <button @click="togglePublic">{{ detail.task.public_to_group ? '已公开' : '未公开' }}</button>
      </view>
      <view class="actions">
        <button @click="start" v-if="detail.task.timer_state === 'pending'">开始</button>
        <button @click="pause" v-if="detail.task.timer_state === 'running'">暂停</button>
        <button @click="resume" v-if="detail.task.timer_state === 'paused'">继续</button>
        <button @click="complete" v-if="detail.task.timer_state === 'achieved'">完成</button>
        <button @click="stop" v-if="detail.task.timer_state !== 'completed'">提前结束</button>
        <button @click="openCorrection('postpone')" v-if="detail.task.timer_state !== 'completed'">推迟</button>
        <button @click="openCorrection('makeup')">补录</button>
      </view>
    </view>

    <view class="panel" v-if="detail?.history?.length">
      <view class="section-title">推迟记录</view>
      <view class="history" v-for="row in detail.history" :key="row.id">
        <view>{{ row.old_date }} -> {{ row.new_date }}</view>
        <text>{{ row.reason || '-' }}</text>
      </view>
    </view>

    <view class="modal" v-if="showCorrection" @click.self="showCorrection = false">
      <view class="modal-body">
        <view class="section-title">{{ correctionMode === 'makeup' ? '补录学习' : '推迟任务' }}</view>
        <view class="picker-grid">
          <view class="picker-field"><text>日期</text><picker mode="date" :value="correction.date" @change="setCorrection('date', $event)"><view>{{ correction.date }}</view></picker></view>
          <view class="picker-field"><text>开始</text><picker mode="time" :value="correction.start" @change="setCorrection('start', $event)"><view>{{ correction.start }}</view></picker></view>
          <view class="picker-field"><text>结束</text><picker mode="time" :value="correction.end" @change="setCorrection('end', $event)"><view>{{ correction.end }}</view></picker></view>
        </view>
        <button class="primary" @click="submitCorrection()">确认</button>
        <button class="secondary" @click="showCorrection = false">取消</button>
      </view>
    </view>

    <view class="modal" v-if="showReflection" @click.self="showReflection = false">
      <view class="modal-body">
        <view class="section-title">编辑完成心得</view>
        <textarea class="reflection-input" v-model="reflectionDraft" maxlength="500" placeholder="记录收获或下一步" />
        <view class="counter">{{ reflectionDraft.length }}/500</view>
        <button class="primary" @click="saveReflection">保存</button>
        <button class="secondary" @click="showReflection = false">取消</button>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { StudyTaskApi, type TaskDetail } from '@/api'
import { addLocalDays, localDateKey } from '@/utils/date'

const taskId = ref(0)
const detail = ref<TaskDetail | null>(null)
const showCorrection = ref(false)
const correctionMode = ref<'makeup' | 'postpone'>('postpone')
const correction = ref({ date: localDateKey(), start: '20:00', end: '21:00' })
const showReflection = ref(false)
const reflectionDraft = ref('')

onLoad((query: any) => { taskId.value = Number(query?.id || 0) })
onShow(load)

async function load() {
  if (!taskId.value) return
  try { detail.value = await StudyTaskApi.get(taskId.value) }
  catch (e: any) { uni.showToast({ title: e?.message || '加载失败', icon: 'none' }) }
}
async function start() {
  try { await StudyTaskApi.start(taskId.value); await load() }
  catch (e: any) { uni.showToast({ title: e?.message || '开始失败', icon: 'none' }) }
}
async function pause() {
  try { await StudyTaskApi.pause(taskId.value); await load() }
  catch (e: any) { uni.showToast({ title: e?.message || '暂停失败', icon: 'none' }) }
}
async function resume() {
  try { await StudyTaskApi.resume(taskId.value); await load() }
  catch (e: any) { uni.showToast({ title: e?.message || '继续失败', icon: 'none' }) }
}
async function complete() {
  try { await StudyTaskApi.complete(taskId.value); await load() }
  catch (e: any) { uni.showToast({ title: e?.message || '完成失败', icon: 'none' }) }
}
async function stop() {
  const ok = await new Promise<boolean>(resolve => uni.showModal({ title: '提前结束', content: '当前累计时长未达到目标，仍要完成任务吗？', success: result => resolve(result.confirm) }))
  if (!ok) return
  try { await StudyTaskApi.stop(taskId.value); await load() }
  catch (e: any) { uni.showToast({ title: e?.message || '结束失败', icon: 'none' }) }
}
async function togglePublic() {
  if (!detail.value) return
  const current = !!detail.value.task.public_to_group
  try { await StudyTaskApi.update(taskId.value, { public_to_group: !current }); await load() }
  catch (e: any) { uni.showToast({ title: e?.message || '更新失败', icon: 'none' }) }
}
function openCorrection(mode: 'makeup' | 'postpone') {
  if (!detail.value) return
  const task = detail.value.task
  correctionMode.value = mode
  correction.value = { date: mode === 'postpone' ? localDateKey(addLocalDays(new Date(), 1)) : task.date, start: task.planned_start || '20:00', end: task.planned_end || '21:00' }
  showCorrection.value = true
}
function setCorrection(field: 'date' | 'start' | 'end', event: any) { correction.value[field] = event.detail.value }
async function submitCorrection(confirmConflict = false) {
  const value = correction.value
  try {
    if (correctionMode.value === 'makeup') await StudyTaskApi.makeup(taskId.value, { actual_date: value.date, actual_start: `${value.date} ${value.start}`, actual_end: `${value.date} ${value.end}`, reason: '手动补录' })
    else await StudyTaskApi.postpone(taskId.value, { date: value.date, planned_start: value.start, planned_end: value.end, reason: '手动推迟', confirm_conflict: confirmConflict })
    showCorrection.value = false
    await load()
  } catch (e: any) {
    if (correctionMode.value === 'postpone' && e?.code === 409 && !confirmConflict) {
      uni.showModal({ title: '时间冲突', content: '目标时段已有任务，仍要推迟吗？', success: result => { if (result.confirm) submitCorrection(true) } })
    } else uni.showToast({ title: e?.message || '操作失败', icon: 'none' })
  }
}
function openReflection() { if (!detail.value) return; reflectionDraft.value = detail.value.task.reflection || ''; showReflection.value = true }
async function saveReflection() {
  try { await StudyTaskApi.reflection(taskId.value, reflectionDraft.value); showReflection.value = false; await load(); uni.showToast({ title: '心得已保存', icon: 'success' }) }
  catch (e: any) { uni.showToast({ title: e?.message || '保存失败', icon: 'none' }) }
}
function durationText(seconds: number) { const minutes = Math.floor(seconds / 60); return minutes >= 60 ? `${Math.floor(minutes / 60)} 小时 ${minutes % 60} 分` : `${minutes} 分 ${seconds % 60} 秒` }
function statusText(status: string) { return ({ pending: '待执行', running: '学习中', paused: '已暂停', achieved: '目标已达成', completed: '已完成' } as Record<string, string>)[status] || status }
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel { margin-bottom: 20rpx; padding: 30rpx; border-radius: 16rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.meta { margin-top: 8rpx; color: #606a80; font-size: 24rpx; }
.content-block { margin-top: 22rpx; padding: 20rpx; border-radius: 12rpx; background: #fff9f2; }
.content-label { color: #9a5b00; font-size: 22rpx; font-weight: 800; }
.content-text { margin-top: 8rpx; color: #384257; font-size: 25rpx; line-height: 1.55; }
.content-text.muted { color: #8a92a6; }
.content-head { display: flex; align-items: center; justify-content: space-between; }
.content-head button { margin: 0; padding: 0 20rpx; height: 50rpx; line-height: 50rpx; background: #fff0f6; color: #ff6f91; font-size: 22rpx; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14rpx; margin-top: 24rpx; }
.grid view { padding: 18rpx; border-radius: 12rpx; background: #f8fafc; }
.grid text { display: block; color: #7b8498; font-size: 22rpx; }
.value { margin-top: 8rpx; color: #111827; font-size: 26rpx; font-weight: 800; }
.desc { margin-top: 22rpx; color: #606a80; font-size: 25rpx; line-height: 1.5; }
.visibility-row { display: flex; justify-content: space-between; align-items: center; margin-top: 22rpx; padding: 18rpx; border-radius: 12rpx; background: #f8fafc; color: #111827; font-size: 25rpx; font-weight: 700; }
.visibility-row button { margin: 0; height: 54rpx; line-height: 54rpx; border-radius: 10rpx; background: #eef4ff; color: #2264d1; font-size: 23rpx; }
.actions { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10rpx; margin-top: 24rpx; }
.actions button { margin: 0; height: 60rpx; line-height: 60rpx; border-radius: 10rpx; background: #eef4ff; color: #2264d1; font-size: 23rpx; }
.section-title { color: #111827; font-size: 28rpx; font-weight: 800; margin-bottom: 12rpx; }
.history { padding: 14rpx 0; border-top: 1rpx solid #eef2f7; color: #384257; font-size: 24rpx; }
.history text { display: block; margin-top: 6rpx; color: #7b8498; }
.modal { position: fixed; inset: 0; z-index: 99; display: flex; align-items: flex-end; background: rgba(17,24,39,.46); }
.modal-body { width: 100%; box-sizing: border-box; padding: 34rpx 30rpx 42rpx; border-radius: 24rpx 24rpx 0 0; background: #fff; }
.picker-grid { display: grid; grid-template-columns: 1.4fr 1fr 1fr; gap: 12rpx; }
.picker-field text { display: block; margin-bottom: 8rpx; color: #7b8498; font-size: 22rpx; }
.picker-field picker view { padding: 20rpx 8rpx; border-radius: 10rpx; background: #f8fafc; color: #111827; font-size: 24rpx; text-align: center; }
.primary, .secondary { margin-top: 18rpx; border-radius: 999rpx; }
.primary { background: #2264d1; color: #fff; }
.secondary { background: #f3f6fb; color: #606a80; }
.reflection-input { width: 100%; height: 180rpx; box-sizing: border-box; padding: 18rpx; border-radius: 12rpx; background: #f8fafc; font-size: 25rpx; }
.counter { margin-top: 8rpx; color: #8a92a6; font-size: 21rpx; text-align: right; }
</style>
