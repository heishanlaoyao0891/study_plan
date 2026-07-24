<template>
  <view class="page">
    <view class="panel">
      <view class="title">AI 生成学习计划</view>
      <view class="desc">生成的是可编辑建议，确认后再保存为计划。</view>
      <view class="field"><text>学习目标</text><input v-model="goal" placeholder="例如：学习 Go 语言" /></view>
      <view class="grid">
        <view class="field"><text>每天小时</text><input v-model.number="hours" type="number" /></view>
        <view class="field"><text>计划天数</text><input v-model.number="days" type="number" /></view>
      </view>
      <view class="grid">
        <view class="field"><text>可用开始时间</text><picker mode="time" :value="availableStart" @change="availableStart = $event.detail.value"><view class="picker-value">{{ availableStart }}</view></picker></view>
        <view class="field"><text>可用结束时间</text><picker mode="time" :value="availableEnd" @change="availableEnd = $event.detail.value"><view class="picker-value">{{ availableEnd }}</view></picker></view>
      </view>
      <view class="field"><text>追加说明</text><textarea v-model="refinement" placeholder="例如：周末少一点，工作日晚间安排" /></view>
      <button class="primary" @click="generate">生成建议</button>
      <button class="secondary" v-if="preview" @click="commit">确认保存</button>
      <view class="error-panel" v-if="errorMessage">{{ errorMessage }}</view>
    </view>

    <view class="result" v-if="preview">
      <view class="result-title">{{ preview.title }}</view>
      <view class="preview-summary">{{ preview.summary }}</view>
      <view class="preview-rationale">{{ preview.rationale }}</view>
      <view class="task" v-for="(t, index) in preview.tasks" :key="`${t.date}-${index}`">
        <view class="task-head">
          <view class="task-title">第 {{ index + 1 }} 天</view>
          <button class="link-btn" @click="removeTask(index)">删除</button>
        </view>
        <view class="field"><text>日期</text><input v-model="t.date" /></view>
        <view class="field"><text>开始时间</text><input v-model="t.planned_start" placeholder="20:00" /></view>
        <view class="field"><text>结束时间</text><input v-model="t.planned_end" placeholder="21:00" /></view>
        <view class="field"><text>标题</text><input v-model="t.title" /></view>
        <view class="field"><text>任务目标</text><textarea v-model="t.objective" maxlength="500" placeholder="描述当天具体要完成什么" /></view>
        <view class="field"><text>描述</text><textarea v-model="t.description" /></view>
        <view class="grid">
          <view class="field"><text>预计分钟</text><input v-model.number="t.estimated_minutes" type="number" /></view>
          <view class="field"><text>难度</text><input v-model="t.difficulty" placeholder="easy/medium/hard" /></view>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { AIApi } from '@/api'
import { formatScheduleConflicts, validateScheduleUnion } from '@/utils/schedule'

const goal = ref('学习 Go 语言')
const hours = ref(1)
const days = ref(7)
const availableStart = ref('20:00')
const availableEnd = ref('21:00')
const refinement = ref('')
const preview = ref<any>(null)
const errorMessage = ref('')

async function generate() {
  errorMessage.value = ''
  if (!goal.value.trim()) {
    uni.showToast({ title: '请输入学习目标', icon: 'none' })
    return
  }
  if (availableStart.value >= availableEnd.value) {
    uni.showToast({ title: '可用结束时间须晚于开始时间', icon: 'none' })
    return
  }
  try {
    const resp = await AIApi.generatePlan({ goal: goal.value, hours_per_day: hours.value, days: days.value, available_time_slot: `${availableStart.value}-${availableEnd.value}`, refinement: refinement.value })
    preview.value = resp.preview || resp.data?.preview || resp
    uni.showToast({ title: '已生成预览', icon: 'success' })
  } catch (e: any) {
    errorMessage.value = apiScheduleError(e) || e?.message || '生成失败'
  }
}

function removeTask(index: number) {
  preview.value.tasks.splice(index, 1)
}

async function commit() {
  errorMessage.value = ''
  const invalid = preview.value?.tasks?.find((task: any) => !task.objective?.trim() || task.objective.trim().toLowerCase() === task.title?.trim().toLowerCase())
  if (invalid) {
    uni.showToast({ title: '每个任务需填写比标题更具体的目标', icon: 'none' })
    return
  }
  const conflicts = validateScheduleUnion((preview.value?.tasks || []).map((task: any, index: number) => ({ id: task.id || index, title: task.title || `任务 ${index + 1}`, date: task.date, start: task.planned_start, end: task.planned_end })))
  if (conflicts.length) {
    errorMessage.value = `时间安排需要调整：\n${formatScheduleConflicts(conflicts)}`
    return
  }
  try {
    await AIApi.commitPlan(preview.value)
    uni.showToast({ title: '已保存', icon: 'success' })
    uni.reLaunch({ url: '/pages/plans/plans' })
  } catch (e: any) {
    errorMessage.value = apiScheduleError(e) || e?.message || '保存失败'
  }
}
function apiScheduleError(error: any) {
  const rows = error?.raw?.invalid_tasks
  if (!Array.isArray(rows)) return ''
  return `时间安排需要调整：\n${rows.map((row: any) => `${row.date ? `${row.date} ` : ''}「${row.title}」重叠覆盖 ${row.covered_minutes} 分钟${row.covered_intervals?.length ? `（${row.covered_intervals.map((range: any) => `${range.start}-${range.end}`).join('、')}）` : ''}`).join('\n')}`
}
</script>

<style lang="scss">
.page { min-height: 100vh; padding: 28rpx; box-sizing: border-box; background: #f6f7fb; }
.panel, .result { padding: 32rpx; border-radius: 18rpx; background: #fff; border: 1rpx solid #e9edf5; }
.title { color: #111827; font-size: 34rpx; font-weight: 800; }
.desc { margin: 12rpx 0 28rpx; color: #7b8498; font-size: 25rpx; line-height: 1.5; }
.field { margin-bottom: 22rpx; }
.field text { display: block; margin-bottom: 10rpx; color: #606a80; font-size: 24rpx; }
.field input, .field textarea { width: 100%; box-sizing: border-box; height: 78rpx; padding: 0 20rpx; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; }
.picker-value { box-sizing: border-box; height: 78rpx; padding: 20rpx; border: 1rpx solid #dbe2ee; border-radius: 12rpx; background: #f9fbff; color: #111827; }
.field textarea { height: 140rpx; padding-top: 16rpx; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 18rpx; }
.primary, .secondary { background: #2264d1; color: #fff; border-radius: 12rpx; }
.secondary { margin-top: 18rpx; background: #eef4ff; color: #2264d1; }
.result { margin-top: 22rpx; }
.result-title { color: #111827; font-size: 30rpx; font-weight: 800; margin-bottom: 18rpx; }
.preview-summary, .preview-rationale { color: #606a80; font-size: 24rpx; line-height: 1.5; margin-bottom: 16rpx; }
.task { padding: 20rpx 0; border-top: 1rpx solid #eef2f7; }
.task-head { display: flex; align-items: center; justify-content: space-between; }
.task-title { color: #111827; font-size: 27rpx; font-weight: 700; }
.link-btn { margin: 0; padding: 0; background: transparent; color: #cf1322; font-size: 24rpx; }
.error-panel { margin-top: 18rpx; padding: 18rpx; border-radius: 12rpx; background: #fff1f3; color: #b4455b; font-size: 23rpx; line-height: 1.55; white-space: pre-line; }
</style>
