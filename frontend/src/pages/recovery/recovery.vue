<template>
  <view class="page">
    <view class="hero">
      <view class="kicker">{{ shiftMode ? '先检查延期后的日程' : '把落下的节奏轻轻接回来' }}</view>
      <view class="title">{{ shiftMode ? '延期预览' : '重新安排' }}</view>
      <view class="desc">先预览并调整日期时间，确认后才会移动任务。</view>
      <view class="meta">{{ shiftMode ? `${preview?.plan_title || '学习计划'} · 延期 ${preview?.days || shiftDays} 天` : `${preview?.overdue_tasks || 0} 个逾期任务` }}</view>
    </view>

    <view class="notice" v-if="preview && !previewToken">当前预览缺少版本 token，已禁止应用。请等待后端升级后刷新预览。</view>
    <view class="notice" v-else-if="preview && !hasServerOccupancy">当前 API 未提供现有日程占用，前端仅检查本次选择之间的重叠；应用时仍会由服务端完整校验。</view>
    <view class="conflict" v-if="conflictMessage"><view class="conflict-title">需要调整时间</view><view>{{ conflictMessage }}</view></view>

    <view class="empty" v-if="!loading && !rows.length">没有需要重新安排的任务</view>
    <view class="group" v-for="group in groupedRows" :key="group.name">
      <view class="group-title">{{ group.name }}</view>
      <view class="task" v-for="row in group.rows" :key="row.action.task_id" :class="{ deselected: !row.selected, 'has-conflict': rowHasConflict(row), 'is-ready': rowIsReady(row) }">
        <view class="task-head"><view><view class="task-title">{{ row.action.title }}</view><view class="old-date">原安排 {{ row.action.old_date }}</view></view><switch v-if="!shiftMode" color="#ff7aa2" :checked="row.selected" @change="setSelected(row, $event)" /></view>
        <view class="task-state" v-if="row.selected" :class="rowHasConflict(row) ? 'conflict-state' : 'ready-state'">{{ rowHasConflict(row) ? '存在冲突，请调整' : '可应用' }}</view>
        <view class="reason">{{ row.action.reason || '根据未来学习日与可用时段安排' }}</view>
        <view class="pickers">
          <view><text>新日期</text><picker mode="date" :value="row.action.new_date" :start="today" @change="setValue(row, 'new_date', $event)"><view>{{ row.action.new_date }}</view></picker></view>
          <view><text>开始</text><picker mode="time" :value="row.action.planned_start" @change="setValue(row, 'planned_start', $event)"><view>{{ row.action.planned_start }}</view></picker></view>
          <view><text>结束</text><picker mode="time" :value="row.action.planned_end" @change="setValue(row, 'planned_end', $event)"><view>{{ row.action.planned_end }}</view></picker></view>
        </view>
        <view class="row-warning" v-if="rowWarning(row)">{{ rowWarning(row) }}</view>
      </view>
    </view>

    <view class="footer" v-if="rows.length"><view class="footer-copy">{{ shiftMode ? `共 ${rows.length} 项` : `已选择 ${selectedRows.length}/${rows.length} 项` }}</view><button :disabled="!canApply || applying" :loading="applying" @click="apply">{{ shiftMode ? '确认延期' : '应用重新安排' }}</button></view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { PlanApi, RecoveryApi, type RecoveryAction, type RecoveryPreview } from '@/api'
import { localDateKey } from '@/utils/date'
import { formatScheduleConflicts, validateScheduleUnion } from '@/utils/schedule'

interface EditableRow { action: RecoveryAction; selected: boolean }
const preview = ref<RecoveryPreview | null>(null), rows = ref<EditableRow[]>([]), loading = ref(false), applying = ref(false)
const mode = ref('recovery'), shiftPlanId = ref(0), shiftDays = ref(0)
const shiftMode = computed(() => mode.value === 'plan_shift')
const today = localDateKey()
const previewToken = computed(() => preview.value?.preview_token || preview.value?.token || preview.value?.version || '')
const selectedRows = computed(() => rows.value.filter(row => row.selected))
const occupancy = computed(() => preview.value?.occupancy || preview.value?.occupied_intervals || [])
const hasServerOccupancy = computed(() => !!(preview.value?.occupancy || preview.value?.occupied_intervals))
const conflicts = computed(() => validateScheduleUnion([
  ...selectedRows.value.map(row => ({ id: row.action.task_id, title: row.action.title, date: row.action.new_date, start: row.action.planned_start, end: row.action.planned_end })),
  ...occupancy.value.map((row, index) => ({ id: row.task_id ?? row.id ?? `occupancy-${index}`, title: row.title || '现有任务', date: row.date, start: row.planned_start || row.start || '', end: row.planned_end || row.end || '' })),
]).filter(conflict => selectedRows.value.some(row => row.action.task_id === conflict.id)))
const conflictingTaskIDs = computed(() => {
  const ids = new Set<string>()
  conflicts.value.forEach(conflict => ids.add(String(conflict.id)))
  if (!shiftMode.value) return ids
  const occupiedDates = new Set(occupancy.value.filter(row => Number(row.plan_id) === shiftPlanId.value).map(row => row.date))
  const dateCounts = new Map<string, number>()
  selectedRows.value.forEach(row => dateCounts.set(row.action.new_date, (dateCounts.get(row.action.new_date) || 0) + 1))
  selectedRows.value.forEach(row => {
    if (occupiedDates.has(row.action.new_date) || (dateCounts.get(row.action.new_date) || 0) > 1) ids.add(String(row.action.task_id))
  })
  return ids
})
const duplicateDateMessage = computed(() => {
  if (!shiftMode.value) return ''
  const occupied = new Set(occupancy.value.filter(row => row.plan_id === shiftPlanId.value).map(row => row.date))
  const seen = new Set<string>()
  const duplicate = selectedRows.value.find(row => occupied.has(row.action.new_date) || seen.has(row.action.new_date) ? true : (seen.add(row.action.new_date), false))
  return duplicate ? `${duplicate.action.new_date} 在当前计划中已有任务，请为冲突任务选择其他日期。` : ''
})
const conflictMessage = computed(() => [duplicateDateMessage.value, formatScheduleConflicts(conflicts.value)].filter(Boolean).join('\n'))
const invalidRange = computed(() => selectedRows.value.some(row => !row.action.new_date || !row.action.planned_start || !row.action.planned_end || row.action.planned_start >= row.action.planned_end || row.action.valid === false))
const canApply = computed(() => !!previewToken.value && selectedRows.value.length > 0 && !invalidRange.value && !conflicts.value.length && !duplicateDateMessage.value)
const groupedRows = computed(() => { const groups = new Map<string, EditableRow[]>(); rows.value.forEach(row => { const name = row.action.plan_title || '学习计划'; groups.set(name, [...(groups.get(name) || []), row]) }); return Array.from(groups, ([name, grouped]) => ({ name, rows: grouped })) })

function rowHasConflict(row: EditableRow) { return row.selected && (row.action.valid === false || conflictingTaskIDs.value.has(String(row.action.task_id))) }
function rowIsReady(row: EditableRow) { return row.selected && !rowHasConflict(row) }
function rowWarning(row: EditableRow) {
  if (row.action.validation_message) return row.action.validation_message
  if (!rowHasConflict(row)) return ''
  return shiftMode.value ? '该日期与当前计划中的其他任务重复，请选择其他日期。' : '该时间与现有日程重叠，请调整日期或时间。'
}

async function load() {
  loading.value = true
  try {
    preview.value = shiftMode.value ? await PlanApi.shiftPreview(shiftPlanId.value, shiftDays.value) : await RecoveryApi.preview()
    rows.value = (preview.value.actions || []).map(action => ({ selected: action.valid !== false, action: { ...action, planned_start: action.planned_start || '20:00', planned_end: action.planned_end || '21:00', reason: action.reason || '根据未来学习日与可用时段安排' } }))
  } catch (error: any) { uni.showToast({ title: error?.message || '预览加载失败', icon: 'none' }) }
  finally { loading.value = false }
}
function setValue(row: EditableRow, field: 'new_date' | 'planned_start' | 'planned_end', event: any) { row.action[field] = event.detail.value; row.action.valid = !!row.action.new_date && !!row.action.planned_start && !!row.action.planned_end && row.action.planned_start < row.action.planned_end; row.action.validation_message = row.action.valid ? '' : '结束时间须晚于开始时间' }
function setSelected(row: EditableRow, event: any) { row.selected = !!event.detail.value }
async function apply() {
  if (!selectedRows.value.length) return
  if (!canApply.value) return
  const confirmed = await new Promise<boolean>(resolve => uni.showModal({ title: '应用重新安排', content: `确认移动已选择的 ${selectedRows.value.length} 个任务？`, success: result => resolve(result.confirm) }))
  if (!confirmed) return
  applying.value = true
  try {
    const actions = selectedRows.value.map(row => ({ ...row.action }))
    const result = shiftMode.value ? await PlanApi.applyShift(shiftPlanId.value, String(previewToken.value), actions) : await RecoveryApi.apply(String(previewToken.value), actions)
    uni.showToast({ title: `已调整 ${result.moved ?? result.applied} 项${result.skipped ? `，跳过 ${result.skipped} 项` : ''}`, icon: 'none' })
    if (shiftMode.value) { setTimeout(() => uni.navigateBack(), 500); return }
    await load()
  } catch (error: any) { const stale = error?.code === 409 && error?.raw?.stale; uni.showModal({ title: stale ? '预览已失效' : '应用失败', content: error?.message || (stale ? '正在刷新预览，请重新选择' : '请调整后重试'), showCancel: false }); if (stale) await load() }
  finally { applying.value = false }
}
onLoad((options: any) => { mode.value = options?.mode || 'recovery'; shiftPlanId.value = Number(options?.plan_id || 0); shiftDays.value = Number(options?.days || 0) })
onShow(load)
</script>

<style lang="scss">
.page{min-height:100vh;box-sizing:border-box;padding:28rpx 28rpx 180rpx;background:linear-gradient(180deg,#fff0f7,#fffaf0 46%,#f7fbff)}.hero{padding:34rpx;border-radius:32rpx;background:linear-gradient(135deg,#ff8fab,#ffc36a);color:#fff;box-shadow:0 18rpx 40rpx rgba(255,143,171,.24)}.kicker{font-size:22rpx;font-weight:800}.title{margin-top:10rpx;font-size:43rpx;font-weight:900}.desc{margin-top:10rpx;color:rgba(255,255,255,.9);font-size:24rpx;line-height:1.5}.meta{margin-top:18rpx;padding:10rpx 16rpx;border-radius:99rpx;background:rgba(255,255,255,.2);font-size:22rpx;display:inline-block}.notice,.conflict{margin-top:18rpx;padding:22rpx;border-radius:18rpx;font-size:23rpx;line-height:1.55}.notice{background:#fff5df;color:#8b601f;border:1rpx solid #ffe0a8}.conflict{background:#fff1f3;color:#b4455b;border:1rpx solid #ffcbd5;white-space:pre-line}.conflict-title{margin-bottom:6rpx;font-weight:900}.group{margin-top:24rpx}.group-title{margin:0 6rpx 12rpx;color:#4b2b3f;font-size:29rpx;font-weight:900}.task{margin-bottom:14rpx;padding:26rpx;border-radius:26rpx;background:#fff;border:1rpx solid #ffe0ea;box-shadow:0 10rpx 26rpx rgba(255,143,171,.09)}.task.is-ready{background:#f2fff8;border-color:#91dfb3;box-shadow:0 10rpx 26rpx rgba(55,164,104,.12)}.task.has-conflict{background:#fff3f4;border-color:#ff9caa;box-shadow:0 10rpx 26rpx rgba(207,72,94,.12)}.task.deselected{opacity:.52}.task-head{display:flex;align-items:center;justify-content:space-between}.task-title{color:#382333;font-size:28rpx;font-weight:900}.old-date{margin-top:5rpx;color:#8a92a6;font-size:21rpx}.task-state{display:inline-block;margin-top:12rpx;padding:5rpx 12rpx;border-radius:99rpx;font-size:20rpx;font-weight:800}.ready-state{background:#d9f8e7;color:#27774a}.conflict-state{background:#ffe0e4;color:#bb3f54}.reason{margin-top:14rpx;color:#706274;font-size:23rpx}.pickers{display:grid;grid-template-columns:1.35fr 1fr 1fr;gap:10rpx;margin-top:18rpx}.pickers text{display:block;margin-bottom:7rpx;color:#8a92a6;font-size:20rpx}.pickers picker view{padding:18rpx 6rpx;border-radius:12rpx;background:#f8fafc;color:#382333;font-size:22rpx;text-align:center}.row-warning{margin-top:12rpx;color:#c44b61;font-size:22rpx}.empty{margin-top:24rpx;padding:48rpx;border-radius:24rpx;background:#fff;color:#7b8498;text-align:center}.footer{position:fixed;left:0;right:0;bottom:0;z-index:20;display:flex;align-items:center;gap:20rpx;padding:20rpx 28rpx 34rpx;background:rgba(255,255,255,.96);box-shadow:0 -10rpx 32rpx rgba(75,43,63,.1)}.footer-copy{color:#606a80;font-size:23rpx}.footer button{flex:1;margin:0;border-radius:99rpx;background:linear-gradient(135deg,#ff7aa2,#ffb45c);color:#fff;font-weight:800}.footer button[disabled]{background:#e8ebf0;color:#8a92a6}
</style>
