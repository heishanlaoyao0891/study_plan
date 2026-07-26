<template>
  <view class="page">
    <view class="report-card">
      <view class="report-title">学习全景</view>
      <view class="report-grid">
        <view class="metric"><view class="num">{{ trend?.summary.study_minutes || 0 }}</view><view class="label">学习分钟</view></view>
        <view class="metric"><view class="num">{{ trend?.summary.completion_rate ?? 0 }}%</view><view class="label">任务完成率</view></view>
        <view class="metric"><view class="num">{{ balance }}</view><view class="label">躺平币</view></view>
      </view>
    </view>

    <view class="analysis">
      <view class="analysis-head"><view><view class="eyebrow">多维分析</view><view class="title">{{ periodLabel }} · {{ dimension === 'time' ? '时间趋势' : '计划对比' }}</view></view><view class="range">{{ trend?.start }} 至 {{ trend?.end }}</view></view>
      <view class="segments"><view v-for="item in periods" :key="item.value" :class="['segment', { active: period === item.value }]" @click="setPeriod(item.value)">{{ item.label }}</view></view>
      <view class="segments dimension"><view :class="['segment', { active: dimension === 'time' }]" @click="setDimension('time')">时间维度</view><view :class="['segment', { active: dimension === 'plan' }]" @click="setDimension('plan')">计划维度</view></view>

      <view class="loading" v-if="loading">正在整理学习数据...</view>
      <view class="error" v-else-if="error">{{ error }}<button @click="loadTrend">重新加载</button></view>
      <view class="empty" v-else-if="!trend?.series.length">这个范围内还没有学习计划</view>
      <template v-else>
        <view class="selected" v-if="selected"><text>{{ selected.plan_title || selected.label }}</text><text>学习 {{ selected.study_minutes }} 分 · 完成 {{ selected.completion_rate ?? 0 }}% · 超时 {{ selected.overtime_minutes }} 分</text></view>
        <scroll-view v-if="dimension === 'time'" class="chart-scroll" scroll-x>
          <view class="bars" :class="{ compact: period === '1m' }">
            <view v-for="(point, index) in trend.series" :key="point.key" :class="['bar-cell', { chosen: selected?.key === point.key }]" @click="selectedIndex = index">
              <view class="bar-track"><view class="bar" :style="{ height: `${barHeight(point.study_minutes)}%` }" /></view>
              <view class="axis">{{ showAxis(index) ? point.label : '' }}</view>
            </view>
          </view>
        </scroll-view>
        <view v-else class="plan-bars">
          <view v-for="(point, index) in trend.series" :key="point.key" :class="['plan-row', { chosen: selected?.key === point.key }]" @click="selectedIndex = index">
            <view class="plan-copy"><text>{{ point.plan_title || '未命名计划' }}</text><text>{{ point.study_minutes }} / {{ point.planned_minutes }} 分钟</text></view>
            <view class="plan-track"><view class="plan-fill" :style="{ width: `${barHeight(point.study_minutes)}%` }" /></view>
            <view class="plan-meta">完成 {{ point.completion_rate ?? 0 }}% · 超时 {{ point.overtime_minutes }} 分钟</view>
          </view>
        </view>
      </template>
    </view>

    <view class="summary-grid" v-if="trend">
      <view><text>计划时长</text><strong>{{ trend.summary.planned_minutes }}</strong><small>分钟</small></view>
      <view><text>超时时长</text><strong>{{ trend.summary.overtime_minutes }}</strong><small>分钟</small></view>
      <view><text>完成任务</text><strong>{{ trend.summary.completed_tasks }}/{{ trend.summary.total_tasks }}</strong><small>项</small></view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { SlackApi, StatsApi, type StatsPoint, type StatsTrend } from '@/api'

type Period = '7d' | '1m' | '1y'
type Dimension = 'time' | 'plan'
const periods: Array<{ value: Period; label: string }> = [{ value: '7d', label: '近7天' }, { value: '1m', label: '近1月' }, { value: '1y', label: '近1年' }]
const period = ref<Period>('7d'), dimension = ref<Dimension>('time'), trend = ref<StatsTrend | null>(null)
const balance = ref(0), loading = ref(false), error = ref(''), selectedIndex = ref(0)
const cache = new Map<string, StatsTrend>()
let requestSequence = 0
const periodLabel = computed(() => periods.find(item => item.value === period.value)?.label || '')
const selected = computed<StatsPoint | null>(() => trend.value?.series[selectedIndex.value] || null)
const maxStudy = computed(() => Math.max(1, ...(trend.value?.series || []).map(item => item.study_minutes)))

function setPeriod(value: Period) { if (period.value === value) return; period.value = value; loadTrend() }
function setDimension(value: Dimension) { if (dimension.value === value) return; dimension.value = value; loadTrend() }
function barHeight(value: number) { return value <= 0 ? 3 : Math.max(8, Math.round(value / maxStudy.value * 100)) }
function showAxis(index: number) { const length = trend.value?.series.length || 0; return length <= 12 || index === 0 || index === length - 1 || index % 5 === 0 }

async function loadTrend() {
  const key = `${period.value}:${dimension.value}`
  const sequence = ++requestSequence
  error.value = ''
  if (cache.has(key)) { trend.value = cache.get(key)!; selectedIndex.value = 0; return }
  loading.value = true
  try {
    const result = await StatsApi.trend(period.value, dimension.value)
    if (sequence !== requestSequence) return
    cache.set(key, result); trend.value = result; selectedIndex.value = 0
  } catch (cause: any) { if (sequence === requestSequence) error.value = cause?.message || '统计数据加载失败' }
  finally { if (sequence === requestSequence) loading.value = false }
}

onShow(() => { loadTrend(); SlackApi.balance().then(result => { balance.value = result.balance }).catch(() => {}) })
</script>

<style lang="scss">
.page{min-height:100vh;box-sizing:border-box;padding:28rpx 28rpx 70rpx;background:#f4f7f5;color:#22312b}.report-card{padding:34rpx;border-radius:12rpx;background:#176449;color:#fff}.report-title{font-size:32rpx;font-weight:900}.report-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:12rpx;margin-top:26rpx}.metric{padding:20rpx 8rpx;border-radius:8rpx;background:#ffffff18;text-align:center}.num{font-size:34rpx;font-weight:900}.label{margin-top:6rpx;color:#d3e9df;font-size:20rpx}.analysis{margin-top:18rpx;padding:28rpx;border:1rpx solid #dce5e0;border-radius:12rpx;background:#fff}.analysis-head{display:flex;justify-content:space-between;align-items:flex-end;gap:16rpx}.eyebrow{color:#28705a;font-size:20rpx;font-weight:900}.title{margin-top:6rpx;font-size:30rpx;font-weight:900}.range{color:#7b8781;font-size:20rpx}.segments{display:grid;grid-template-columns:repeat(3,1fr);gap:6rpx;margin-top:24rpx;padding:6rpx;border-radius:8rpx;background:#edf2ef}.segments.dimension{grid-template-columns:repeat(2,1fr);margin-top:10rpx}.segment{padding:14rpx 6rpx;border-radius:6rpx;color:#637169;font-size:23rpx;text-align:center}.segment.active{background:#fff;color:#176449;font-weight:900;box-shadow:0 3rpx 10rpx #193c2d14}.selected{display:flex;flex-wrap:wrap;justify-content:space-between;gap:8rpx;margin-top:22rpx;padding:18rpx;border-radius:8rpx;background:#f5f8f6;color:#53625a;font-size:21rpx}.selected text:first-child{color:#26372f;font-weight:800}.chart-scroll{width:100%;margin-top:18rpx}.bars{display:flex;align-items:flex-end;gap:12rpx;min-width:100%;height:330rpx}.bars.compact{width:1200rpx}.bar-cell{display:flex;flex:1;min-width:34rpx;height:100%;flex-direction:column;justify-content:flex-end}.bar-track{display:flex;height:260rpx;align-items:flex-end;border-bottom:1rpx solid #d9e2dd}.bar{width:70%;min-height:6rpx;margin:0 auto;border-radius:5rpx 5rpx 0 0;background:#78b99f}.bar-cell.chosen .bar{background:#d99b1d}.axis{height:34rpx;padding-top:8rpx;color:#809088;font-size:18rpx;text-align:center}.plan-bars{margin-top:18rpx}.plan-row{padding:20rpx 0;border-top:1rpx solid #edf1ef}.plan-copy{display:flex;justify-content:space-between;gap:12rpx;font-size:23rpx}.plan-copy text:first-child{font-weight:800}.plan-track{height:16rpx;margin-top:12rpx;border-radius:5rpx;background:#edf2ef;overflow:hidden}.plan-fill{height:100%;background:#78b99f}.plan-row.chosen .plan-fill{background:#d99b1d}.plan-meta{margin-top:8rpx;color:#718078;font-size:20rpx}.summary-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:10rpx;margin-top:18rpx}.summary-grid>view{padding:24rpx 8rpx;border:1rpx solid #dce5e0;border-radius:10rpx;background:#fff;text-align:center}.summary-grid text,.summary-grid small{display:block;color:#77847e;font-size:20rpx}.summary-grid strong{display:block;margin:8rpx 0 2rpx;font-size:31rpx}.loading,.empty,.error{padding:48rpx 0;color:#76847d;text-align:center}.error{color:#a23e4d}.error button{margin-top:14rpx;background:#eef5f1;color:#176449;font-size:22rpx}@media(max-width:340px){.analysis-head{align-items:flex-start;flex-direction:column}.report-grid,.summary-grid{gap:6rpx}.label{font-size:18rpx}}
</style>
