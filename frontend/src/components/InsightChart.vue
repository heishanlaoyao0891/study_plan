<template>
  <view class="chart-shell">
    <qiun-data-charts
      :canvas-id="canvasId"
      :type="chartType"
      :chart-data="chartData"
      :opts="chartOptions"
      :animation="true"
      :ontouch="true"
      :error-show="false"
      @getIndex="selectPoint"
    />
  </view>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { StatsPoint } from '@/api'

const props = defineProps<{
  points: StatsPoint[]
  dimension: 'time' | 'plan'
}>()
const emit = defineEmits<{ select: [index: number] }>()

const canvasId = `insight-${Math.random().toString(36).slice(2, 9)}`
const chartType = computed(() => props.dimension === 'time' ? 'mix' : 'bar')
const chartData = computed(() => ({
  categories: props.points.map(point => point.plan_title || point.label),
  series: props.dimension === 'time'
    ? [
        { name: '学习分钟', type: 'column', data: props.points.map(point => point.study_minutes), color: '#20745a' },
        { name: '计划分钟', type: 'line', data: props.points.map(point => point.planned_minutes), color: '#d59a20' },
        { name: '超时分钟', type: 'area', data: props.points.map(point => point.overtime_minutes), color: '#d35f72' },
      ]
    : [
        { name: '学习分钟', data: props.points.map(point => point.study_minutes), color: '#20745a' },
        { name: '计划分钟', data: props.points.map(point => point.planned_minutes), color: '#d59a20' },
      ],
}))
const chartOptions = computed(() => ({
  color: ['#20745a', '#d59a20', '#d35f72'],
  padding: props.dimension === 'time' ? [18, 12, 4, 8] : [16, 18, 4, 78],
  enableScroll: props.dimension === 'time' && props.points.length > 12,
  dataLabel: false,
  dataPointShape: true,
  fontSize: 10,
  legend: { show: true, position: 'top', float: 'center', fontSize: 11, itemGap: 18 },
  xAxis: props.dimension === 'time'
    ? { disableGrid: true, itemCount: props.points.length > 12 ? 8 : props.points.length, scrollShow: props.points.length > 12, scrollAlign: 'right', fontSize: 9 }
    : { min: 0, gridType: 'dash', dashLength: 3, fontSize: 9 },
  yAxis: props.dimension === 'time'
    ? { gridType: 'dash', dashLength: 3, data: [{ min: 0, fontSize: 9 }] }
    : { disableGrid: true, fontSize: 9 },
  extra: props.dimension === 'time'
    ? {
        mix: { column: { width: 14 }, line: { type: 'curve', width: 2 }, area: { type: 'curve', opacity: 0.12, width: 2 } },
        tooltip: { showBox: true, showArrow: true, borderRadius: 6, bgColor: '#24362f', fontColor: '#ffffff', gridType: 'dash' },
      }
    : {
        bar: { type: 'group', width: 12, seriesGap: 3, categoryGap: 4 },
        tooltip: { showBox: true, showArrow: true, borderRadius: 6, bgColor: '#24362f', fontColor: '#ffffff' },
      },
}))

function selectPoint(event: any) {
  const index = Number(event?.currentIndex ?? event?.detail?.currentIndex)
  if (Number.isInteger(index) && index >= 0 && index < props.points.length) emit('select', index)
}
</script>

<style scoped>
.chart-shell { width: 100%; height: 620rpx; }
</style>
