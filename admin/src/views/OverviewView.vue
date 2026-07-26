<template>
  <main class="workspace">
    <section class="page-head"><div><p class="eyebrow">近 30 天运营</p><h2>总览仪表盘</h2><p>{{ data?.range.start }} 至 {{ data?.range.end }}，用户分层以最近登录时间计算。</p></div><button class="ghost small-button" :disabled="loading" @click="load">刷新</button></section>
    <p v-if="error" class="error">{{ error }}</p>
    <div class="metric-grid" v-if="data">
      <article class="metric-card"><span>用户账号</span><strong>{{ data.summary.user_accounts }}</strong><small>不含管理员</small></article>
      <article class="metric-card"><span>7 日活跃</span><strong>{{ data.summary.active_users_7d }}</strong><small>{{ ratio(data.summary.active_users_7d, data.summary.user_accounts) }}%</small></article>
      <article class="metric-card"><span>进行中计划</span><strong>{{ data.summary.active_plans }}</strong><small>当前有效计划</small></article>
      <article class="metric-card"><span>今日学习</span><strong>{{ data.summary.study_minutes_today }}</strong><small>{{ data.summary.checkins_today }} 人打卡</small></article>
    </div>
    <div class="dashboard-grid" v-if="data">
      <section class="chart-panel"><header><div><p class="eyebrow">用户健康</p><h3>用户状态分布</h3></div><span>{{ data.summary.user_accounts }} 人</span></header><div class="donut-layout"><div class="donut" :style="donutStyle(data.charts.segments)"><div><strong>{{ data.summary.active_users_7d }}</strong><span>7日活跃</span></div></div><div class="legend"><div v-for="(item,index) in data.charts.segments" :key="item.key"><i :style="{background: colors[index]}"/><span>{{ item.label }}</span><strong>{{ item.count }}</strong></div></div></div></section>
      <section class="chart-panel"><header><div><p class="eyebrow">用户增长</p><h3>每日新增用户</h3></div><span>共 {{ sum(data.charts.registrations, 'count') }} 人</span></header><div class="bar-scroll"><div class="columns"><div v-for="(item,index) in data.charts.registrations" :key="item.date" class="column"><span>{{ item.count || '' }}</span><div :style="{height: `${height(item.count || 0, registrationMax)}%`} "/><small>{{ axis(index, item.date) }}</small></div></div></div></section>
      <section class="chart-panel"><header><div><p class="eyebrow">学习参与</p><h3>学习时长与打卡人数</h3></div><span>今日 {{ data.summary.study_minutes_today }} 分钟</span></header><div class="bar-scroll"><div class="columns dual"><div v-for="(item,index) in data.charts.learning_activity" :key="item.date" class="column"><span>{{ item.study_minutes || '' }}</span><div class="study" :style="{height: `${height(item.study_minutes || 0, learningMax)}%`} "/><div class="checkin" :style="{height: `${height(item.checkin_users || 0, checkinMax)}%`} "/><small>{{ axis(index, item.date) }}</small></div></div></div><div class="inline-legend"><span><i class="study-dot"/>学习分钟</span><span><i class="checkin-dot"/>打卡人数</span></div></section>
      <section class="chart-panel"><header><div><p class="eyebrow">计划存量</p><h3>计划状态分布</h3></div><span>{{ planTotal }} 个计划</span></header><div class="donut-layout"><div class="donut" :style="donutStyle(data.charts.plan_statuses)"><div><strong>{{ data.summary.active_plans }}</strong><span>进行中</span></div></div><div class="legend"><div v-for="(item,index) in data.charts.plan_statuses" :key="item.key"><i :style="{background: colors[index]}"/><span>{{ item.label }}</span><strong>{{ item.count }}</strong></div></div></div></section>
    </div>
    <p class="loading" v-if="loading">正在计算运营指标...</p>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { AdminApi, type DashboardDay, type DashboardSlice, type OverviewMetrics } from '@/api'
const colors = ['#16765a','#d9a11e','#7d8f87','#c95268','#5f75b8','#a8b2ad']
const data = ref<OverviewMetrics | null>(null), error = ref(''), loading = ref(false)
const registrationMax = computed(() => Math.max(1, ...(data.value?.charts.registrations || []).map(item => item.count || 0)))
const learningMax = computed(() => Math.max(1, ...(data.value?.charts.learning_activity || []).map(item => item.study_minutes || 0)))
const checkinMax = computed(() => Math.max(1, ...(data.value?.charts.learning_activity || []).map(item => item.checkin_users || 0)))
const planTotal = computed(() => data.value?.charts.plan_statuses.reduce((total,item) => total + item.count, 0) || 0)
function ratio(value:number,total:number){return total?Math.round(value*100/total):0}
function sum(rows:DashboardDay[],key:'count'){return rows.reduce((total,row)=>total+(row[key]||0),0)}
function height(value:number,max:number){return value?Math.max(5,Math.round(value/max*100)):1}
function axis(index:number,date:string){return index===0||index===29||index%7===0?date.slice(5):''}
function donutStyle(rows:DashboardSlice[]){const total=Math.max(1,rows.reduce((sum,row)=>sum+row.count,0));let cursor=0;const stops:string[]=[];rows.forEach((row,index)=>{const next=cursor+row.count/total*100;stops.push(`${colors[index]} ${cursor}% ${next}%`);cursor=next});return {background:`conic-gradient(${stops.join(',')})`}}
async function load(){loading.value=true;error.value='';try{data.value=await AdminApi.overview()}catch(err){error.value=err instanceof Error?err.message:'总览加载失败'}finally{loading.value=false}}
onMounted(load)
</script>

<style scoped>
.page-head p{margin:6px 0 0;color:#6b7680}.metric-card small{color:#7a8781}.dashboard-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:18px;margin-top:20px}.chart-panel{min-width:0;padding:22px;border:1px solid #e1e7e4;border-radius:8px;background:#fff}.chart-panel header{display:flex;justify-content:space-between;align-items:flex-start;gap:14px}.chart-panel h3{margin:3px 0 0;font-size:18px}.chart-panel header>span{color:#6e7b75;font-size:13px}.donut-layout{display:flex;align-items:center;gap:28px;min-height:250px}.donut{display:grid;flex:none;width:170px;height:170px;place-items:center;border-radius:50%}.donut>div{display:flex;width:104px;height:104px;align-items:center;justify-content:center;flex-direction:column;border-radius:50%;background:#fff}.donut strong{font-size:28px}.donut span{color:#718079;font-size:12px}.legend{flex:1;display:grid;gap:10px}.legend>div{display:grid;grid-template-columns:10px 1fr auto;align-items:center;gap:8px;font-size:13px}.legend i,.inline-legend i{width:9px;height:9px;border-radius:2px}.bar-scroll{overflow-x:auto;margin-top:24px}.columns{display:flex;align-items:flex-end;gap:5px;min-width:720px;height:230px;border-bottom:1px solid #dfe6e2}.column{position:relative;display:flex;flex:1;height:100%;align-items:center;justify-content:flex-end;flex-direction:column}.column>span{position:absolute;top:-16px;color:#67756e;font-size:9px}.column>div{width:66%;min-height:2px;border-radius:3px 3px 0 0;background:#16765a}.column small{position:absolute;bottom:-22px;color:#7d8983;font-size:9px}.columns.dual .column{flex-direction:row;align-items:flex-end;gap:2px}.columns.dual .column>div{width:36%}.column .study{background:#16765a}.column .checkin{background:#d9a11e}.inline-legend{display:flex;gap:18px;margin-top:32px;color:#68756f;font-size:12px}.inline-legend span{display:flex;align-items:center;gap:6px}.study-dot{background:#16765a}.checkin-dot{background:#d9a11e}.loading{text-align:center;color:#718079}@media(max-width:1000px){.dashboard-grid{grid-template-columns:1fr}.donut-layout{justify-content:center}}@media(max-width:560px){.donut-layout{align-items:stretch;flex-direction:column}.donut{margin:auto}.legend{width:100%}}
</style>
