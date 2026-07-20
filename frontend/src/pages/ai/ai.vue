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
      <view class="field"><text>追加说明</text><textarea v-model="refinement" placeholder="例如：周末少一点，工作日晚间安排" /></view>
      <button class="primary" @click="generate">生成建议</button>
      <button class="secondary" v-if="preview" @click="commit">确认保存</button>
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

const goal = ref('学习 Go 语言')
const hours = ref(1)
const days = ref(7)
const refinement = ref('')
const preview = ref<any>(null)

async function generate() {
  if (!goal.value.trim()) {
    uni.showToast({ title: '请输入学习目标', icon: 'none' })
    return
  }
  try {
    const resp = await AIApi.generatePlan({ goal: goal.value, hours_per_day: hours.value, days: days.value, refinement: refinement.value })
    preview.value = resp.preview || resp.data?.preview || resp
    uni.showToast({ title: '已生成预览', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '生成失败', icon: 'none' })
  }
}

function removeTask(index: number) {
  preview.value.tasks.splice(index, 1)
}

async function commit() {
  try {
    await AIApi.commitPlan(preview.value)
    uni.showToast({ title: '已保存', icon: 'success' })
    uni.reLaunch({ url: '/pages/plans/plans' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '保存失败', icon: 'none' })
  }
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
</style>
