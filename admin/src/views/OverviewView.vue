<template>
  <main class="workspace">
    <section class="page-head">
      <div><p class="eyebrow">近 30 天运营</p><h2>总览仪表盘</h2><p v-if="data">{{ data.range.start }} 至 {{ data.range.end }}，用户分层以最近登录时间计算。</p></div>
      <button class="ghost small-button" :disabled="loading" @click="load">{{ loading && data ? '更新中…' : '刷新' }}</button>
    </section>
    <p v-if="error" class="error">{{ error }}</p>
    <div class="metric-grid" v-if="data">
      <article class="metric-card"><span>用户账号</span><strong>{{ data.summary.user_accounts }}</strong><small>不含管理员</small></article>
      <article class="metric-card"><span>7 日活跃</span><strong>{{ data.summary.active_users_7d }}</strong><small>{{ ratio(data.summary.active_users_7d, data.summary.user_accounts) }}%</small></article>
      <article class="metric-card"><span>进行中计划</span><strong>{{ data.summary.active_plans }}</strong><small>当前有效计划</small></article>
      <article class="metric-card"><span>今日学习</span><strong>{{ data.summary.study_minutes_today }}</strong><small>{{ data.summary.checkins_today }} 人打卡</small></article>
    </div>
    <div class="dashboard-grid" v-if="data">
      <section class="chart-panel"><header><div><p class="eyebrow">用户健康</p><h3>用户状态分布</h3></div><span>{{ data.summary.user_accounts }} 人</span></header><VChart class="chart" :option="segmentOption" autoresize /></section>
      <section class="chart-panel"><header><div><p class="eyebrow">用户增长</p><h3>每日新增用户</h3></div><span>共 {{ registrationTotal }} 人</span></header><VChart class="chart" :option="registrationOption" autoresize /></section>
      <section class="chart-panel"><header><div><p class="eyebrow">学习参与</p><h3>学习时长与打卡人数</h3></div><span>今日 {{ data.summary.study_minutes_today }} 分钟</span></header><VChart class="chart" :option="learningOption" autoresize /></section>
      <section class="chart-panel"><header><div><p class="eyebrow">计划存量</p><h3>计划状态分布</h3></div><span>{{ planTotal }} 个计划</span></header><VChart class="chart" :option="planOption" autoresize /></section>
    </div>
    <div class="skeleton-grid" v-else-if="loading"><div v-for="index in 4" :key="index" class="skeleton" /></div>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { use } from 'echarts/core'
import type { EChartsCoreOption } from 'echarts/core'
import { BarChart, LineChart, PieChart } from 'echarts/charts'
import { CanvasRenderer } from 'echarts/renderers'
import { GridComponent, LegendComponent, TitleComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { AdminApi, type DashboardDay, type DashboardSlice, type OverviewMetrics } from '@/api'

use([CanvasRenderer, BarChart, LineChart, PieChart, GridComponent, LegendComponent, TitleComponent, TooltipComponent])

const colors = ['#16765a', '#d9a11e', '#7d8f87', '#c95268', '#5f75b8', '#a8b2ad']
const data = ref<OverviewMetrics | null>(null), error = ref(''), loading = ref(false)
const registrationTotal = computed(() => sum(data.value?.charts.registrations || [], 'count'))
const planTotal = computed(() => data.value?.charts.plan_statuses.reduce((total, item) => total + item.count, 0) || 0)
const baseAnimation = { animationDuration: 700, animationDurationUpdate: 450, animationEasing: 'cubicOut' as const, animationEasingUpdate: 'cubicOut' as const }
const textStyle = { color: '#52615a', fontFamily: 'Inter, "Microsoft YaHei", sans-serif' }

const segmentOption = computed<EChartsCoreOption>(() => donutOption(data.value?.charts.segments || [], '用户', data.value?.summary.user_accounts || 0))
const planOption = computed<EChartsCoreOption>(() => donutOption(data.value?.charts.plan_statuses || [], '计划', planTotal.value))
const registrationOption = computed<EChartsCoreOption>(() => {
  const rows = data.value?.charts.registrations || []
  return {
    ...baseAnimation,
    color: [colors[0]],
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, formatter: (items: any[]) => `${items[0].axisValue}<br/>新增用户：<b>${items[0].value}</b> 人` },
    grid: { left: 44, right: 18, top: 28, bottom: 42 },
    xAxis: { type: 'category', data: rows.map(row => row.date.slice(5)), axisTick: { show: false }, axisLabel: { ...textStyle, interval: 4 }, axisLine: { lineStyle: { color: '#dfe7e3' } } },
    yAxis: { type: 'value', minInterval: 1, axisLabel: textStyle, splitLine: { lineStyle: { color: '#edf2ef', type: 'dashed' } } },
    series: [{ name: '新增用户', type: 'bar', data: rows.map(row => row.count || 0), barMaxWidth: 18, itemStyle: { borderRadius: [5, 5, 0, 0], color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#2b9a78' }, { offset: 1, color: '#a7d9c6' }] } }, emphasis: { itemStyle: { shadowBlur: 12, shadowColor: '#16765a55' } } }],
  }
})
const learningOption = computed<EChartsCoreOption>(() => {
  const rows = data.value?.charts.learning_activity || []
  return {
    ...baseAnimation,
    color: [colors[0], colors[1]],
    tooltip: { trigger: 'axis', formatter: (items: any[]) => `${items[0].axisValue}<br/>${items.map(item => `${item.marker}${item.seriesName}：<b>${item.value}</b>${item.seriesIndex ? ' 人' : ' 分钟'}`).join('<br/>')}` },
    legend: { top: 0, right: 0, textStyle },
    grid: { left: 48, right: 48, top: 42, bottom: 42 },
    xAxis: { type: 'category', data: rows.map(row => row.date.slice(5)), axisTick: { show: false }, axisLabel: { ...textStyle, interval: 4 }, axisLine: { lineStyle: { color: '#dfe7e3' } } },
    yAxis: [{ type: 'value', name: '分钟', axisLabel: textStyle, splitLine: { lineStyle: { color: '#edf2ef', type: 'dashed' } } }, { type: 'value', name: '人数', minInterval: 1, axisLabel: textStyle, splitLine: { show: false } }],
    series: [
      { name: '学习分钟', type: 'bar', data: rows.map(row => row.study_minutes || 0), barMaxWidth: 16, itemStyle: { borderRadius: [5, 5, 0, 0] }, emphasis: { focus: 'series' } },
      { name: '打卡人数', type: 'line', yAxisIndex: 1, smooth: true, symbolSize: 6, data: rows.map(row => row.checkin_users || 0), lineStyle: { width: 3 }, areaStyle: { opacity: 0.08 }, emphasis: { focus: 'series' } },
    ],
  }
})

function donutOption(rows: DashboardSlice[], unit: string, total: number): EChartsCoreOption {
  return {
    ...baseAnimation,
    color: colors,
    tooltip: { trigger: 'item', formatter: (item: any) => `${item.name}<br/><b>${item.value}</b> ${unit}（${item.percent}%）` },
    legend: { orient: 'vertical', right: 10, top: 'middle', itemWidth: 10, itemHeight: 10, itemGap: 14, textStyle },
    title: { text: String(total), subtext: unit, left: '34%', top: '39%', textAlign: 'center', textStyle: { color: '#22312b', fontSize: 28, fontWeight: 800 }, subtextStyle: { color: '#78857f', fontSize: 12 } },
    series: [{ type: 'pie', radius: ['50%', '72%'], center: ['34%', '50%'], avoidLabelOverlap: true, padAngle: 2, itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 }, label: { show: false }, emphasis: { scale: true, scaleSize: 8, itemStyle: { shadowBlur: 16, shadowColor: '#24362f33' } }, data: rows.map(row => ({ name: row.label, value: row.count })) }],
  }
}
function ratio(value: number, total: number) { return total ? Math.round(value * 100 / total) : 0 }
function sum(rows: DashboardDay[], key: 'count') { return rows.reduce((total, row) => total + (row[key] || 0), 0) }
function validOverview(value: unknown): value is OverviewMetrics { const row = value as OverviewMetrics; return !!row?.range && !!row?.summary && !!row?.charts && Array.isArray(row.charts.segments) && Array.isArray(row.charts.registrations) && Array.isArray(row.charts.learning_activity) && Array.isArray(row.charts.plan_statuses) }
async function load() { loading.value = true; error.value = ''; try { const result = await AdminApi.overview(); if (!validOverview(result)) throw new Error('总览接口版本不匹配，请先更新后端服务'); data.value = result } catch (err) { if (!data.value) error.value = err instanceof Error ? err.message : '总览加载失败' } finally { loading.value = false } }
onMounted(load)
</script>

<style scoped>
.page-head p { margin: 6px 0 0; color: #6b7680; }
.metric-card { transition: transform .2s ease, box-shadow .2s ease; }
.metric-card:hover { transform: translateY(-3px); box-shadow: 0 12px 28px rgba(30, 55, 44, .1); }
.metric-card small { color: #7a8781; }
.dashboard-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; margin-top: 20px; }
.chart-panel { min-width: 0; padding: 22px; border: 1px solid #e1e7e4; border-radius: 8px; background: #fff; transition: box-shadow .2s ease; }
.chart-panel:hover { box-shadow: 0 14px 34px rgba(30, 55, 44, .08); }
.chart-panel header { display: flex; justify-content: space-between; align-items: flex-start; gap: 14px; }
.chart-panel h3 { margin: 3px 0 0; font-size: 18px; }
.chart-panel header > span { color: #6e7b75; font-size: 13px; }
.chart { width: 100%; height: 310px; margin-top: 8px; }
.skeleton-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 18px; margin-top: 20px; }
.skeleton { height: 350px; border-radius: 8px; background: linear-gradient(100deg, #edf2ef 20%, #f8faf9 45%, #edf2ef 70%); background-size: 300% 100%; animation: shimmer 1.3s infinite; }
@keyframes shimmer { from { background-position: 100% 0; } to { background-position: 0 0; } }
@media (max-width: 1000px) { .dashboard-grid, .skeleton-grid { grid-template-columns: 1fr; } }
</style>
