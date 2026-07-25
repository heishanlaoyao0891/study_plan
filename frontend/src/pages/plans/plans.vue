<template>
  <view class="page">
    <view class="summary">
      <view><view class="eyebrow">学习愿望花园</view><view class="summary-title">{{ activeCount }} 个计划正在发芽</view></view>
      <view class="summary-count">{{ plans.length }}</view>
    </view>

    <view class="ai-hero" @click="goAI">
      <view class="ai-badge">AI</view>
      <view class="ai-copy"><view class="ai-title">AI 生成计划</view><view class="ai-desc">说出目标，自动拆成有日期、有节奏的学习任务</view></view>
      <view class="arrow">›</view>
    </view>
    <button class="manual" @click="openCreate">＋ 手动创建</button>

    <view class="tools">
      <view class="tool" @click="goSchedule"><view class="tool-icon">日</view><text>日程</text></view>
      <view class="tool" @click="goGroup"><view class="tool-icon">伴</view><text>小组</text></view>
      <view class="tool" @click="goNotifications"><view class="tool-icon">铃</view><text>提醒</text></view>
      <view class="tool recovery" v-if="overdueCount > 0" @click="goRecovery"><view class="tool-icon">排</view><text>重新安排 {{ overdueCount }}</text></view>
    </view>

    <view class="section-head"><view>我的计划</view><button @click="openLayout">编辑操作</button></view>
    <view class="empty" v-if="!loading && !plans.length"><view class="empty-title">还没有计划</view><view class="empty-desc">从 AI 生成开始，或手动安排清晰的日期与时段。</view></view>
    <view class="cards" v-else>
      <view class="card" v-for="plan in plans" :key="plan.id" :class="{ paused: plan.status === 'paused' }">
        <view class="card-head"><view class="plan-title">{{ plan.title }}</view><view class="pill" :class="plan.status">{{ statusText(plan.status) }}</view></view>
        <view class="plan-desc" v-if="plan.description">{{ plan.description }}</view>
        <view class="schedule">{{ weekdaySummary(plan.study_weekdays) }} · {{ plan.default_planned_start }}-{{ plan.default_planned_end }}</view>
        <view class="progress" v-if="plan.total_tasks > 0">
          <view class="progress-copy"><text>完成 {{ plan.completed_tasks }}/{{ plan.total_tasks }}</text><text>{{ plan.completion_rate ?? calculatedRate(plan) }}%</text></view>
          <view class="track"><view class="fill" :style="{ width: `${plan.completion_rate ?? calculatedRate(plan)}%` }" /></view>
        </view>
        <view class="taskless" v-else>尚未生成任务</view>
        <view class="actions">
          <button v-for="action in directActions" :key="action" :class="['action', { destructive: action === 'delete' }]" @click="runAction(action, plan)">{{ actionLabel(action, plan) }}</button>
          <button class="action more" @click="openMore(plan)">更多</button>
        </view>
      </view>
    </view>
    <view class="settings" @click="goOps">设置与说明</view>

    <view class="sheet" v-if="showForm" @click.self="closeForm"><view class="sheet-body form-sheet">
      <view class="sheet-title">{{ editing ? '编辑计划' : '手动创建计划' }}</view>
      <view class="field"><text>计划名称</text><input v-model="form.title" placeholder="例如：学习 Go 语言" /></view>
      <view class="field"><text>计划说明</text><textarea v-model="form.description" placeholder="阶段目标或范围（可选）" /></view>
      <view class="field" v-if="!editing"><text>每日任务目标</text><textarea v-model="form.objective" maxlength="500" placeholder="具体描述每天要完成什么" /></view>
      <view class="grid"><view class="field"><text>开始日期</text><picker mode="date" :value="form.start_date" @change="setFormValue('start_date', $event)"><view class="picker">{{ form.start_date }}</view></picker></view><view class="field"><text>结束日期</text><picker mode="date" :value="form.end_date" :start="form.start_date" @change="setFormValue('end_date', $event)"><view class="picker">{{ form.end_date }}</view></picker></view></view>
      <view class="grid"><view class="field"><text>默认开始</text><picker mode="time" :value="form.default_planned_start" @change="setFormValue('default_planned_start', $event)"><view class="picker">{{ form.default_planned_start }}</view></picker></view><view class="field"><text>默认结束</text><picker mode="time" :value="form.default_planned_end" @change="setFormValue('default_planned_end', $event)"><view class="picker">{{ form.default_planned_end }}</view></picker></view></view>
      <view class="field"><text>每周目标小时</text><input v-model.number="form.weekly_target_hours" type="number" /></view>
      <view class="field"><text>学习日</text><view class="weekdays"><view v-for="day in weekdays" :key="day.value" :class="['weekday', { selected: form.study_weekdays?.includes(day.value) }]" @click="toggleWeekday(day.value)">{{ day.label }}</view></view></view>
      <view class="form-error" v-if="formError">{{ formError }}</view>
      <label class="confirm" v-if="!editing"><checkbox :checked="!!form.confirm_overload" @click="form.confirm_overload = !form.confirm_overload" />允许继续处理非时间类负荷提醒</label>
      <view class="sheet-actions"><button class="secondary" @click="closeForm">取消</button><button class="primary" @click="save">保存</button></view>
    </view></view>

    <view class="sheet" v-if="showLayout" @click.self="showLayout = false"><view class="sheet-body">
      <view class="sheet-title">编辑操作</view><view class="sheet-note">长按操作项可拖动排序或跨区；也可使用上下移动与区域切换。卡片只展示直接操作的前 2 项，其余仍在更多中。</view>
      <view id="layout-direct-zone" class="layout-zone">
        <view class="layout-title">直接操作（卡片展示前 2 项）</view>
        <view :id="`layout-direct-${action}`" class="layout-row layout-drop-row" :class="{ dragging: dragState?.action === action }" v-for="action in layoutDraft.direct" :key="action" @touchstart="startDrag(action, 'direct', $event)" @touchmove="moveDrag" @touchend="endDrag" @touchcancel="endDrag"><text>{{ actionName(action) }}</text><view><button @click.stop="move(action, 'direct', -1)">上移</button><button @click.stop="move(action, 'direct', 1)">下移</button><button @click.stop="transfer(action, 'direct')">移至更多</button></view></view>
        <view class="layout-empty" v-if="!layoutDraft.direct.length">长按“更多操作”中的项目拖到这里</view>
      </view>
      <view id="layout-overflow-zone" class="layout-zone overflow-zone">
        <view class="layout-title">更多操作</view>
        <view :id="`layout-overflow-${action}`" class="layout-row layout-drop-row" :class="{ dragging: dragState?.action === action }" v-for="action in layoutDraft.overflow" :key="action" @touchstart="startDrag(action, 'overflow', $event)" @touchmove="moveDrag" @touchend="endDrag" @touchcancel="endDrag"><text :class="{ dangerText: action === 'delete' }">{{ actionName(action) }}</text><view><button @click.stop="move(action, 'overflow', -1)">上移</button><button @click.stop="move(action, 'overflow', 1)">下移</button><button @click.stop="transfer(action, 'overflow')">直接操作</button></view></view>
        <view class="layout-empty" v-if="!layoutDraft.overflow.length">长按“直接操作”中的项目拖到这里</view>
      </view>
      <view class="sheet-actions"><button class="secondary" @click="showLayout = false">取消</button><button class="primary" @click="saveLayout">保存布局</button></view>
    </view></view>

    <view class="sheet" v-if="delayPlan" @click.self="delayPlan = null"><view class="sheet-body">
      <view class="sheet-title">延期「{{ delayPlan.title }}」</view><view class="sheet-note">只移动未来未完成任务，历史完成记录保持不变。</view>
      <view class="stepper"><button @click="delayDays = Math.max(1, delayDays - 1)">−</button><picker mode="selector" :range="delayOptions" :value="delayDays - 1" @change="delayDays = Number($event.detail.value) + 1"><view>{{ delayDays }} 天</view></picker><button @click="delayDays = Math.min(30, delayDays + 1)">＋</button></view>
      <view class="date-preview"><view>当前 {{ planRange(delayPlan) }}</view><view>延期后 {{ delayedRange(delayPlan) }}</view></view>
      <view class="sheet-actions"><button class="secondary" @click="delayPlan = null">取消</button><button class="primary" @click="confirmDelay">确认延期</button></view>
    </view></view>

    <view class="sheet" v-if="invitePlan" @click.self="closeInvite"><view class="sheet-body">
      <view class="sheet-title">邀请学习伙伴</view><view class="sheet-note">输入昵称查找学习伙伴，不会展示手机号或学习数据。</view>
      <input class="search" v-model="searchQuery" placeholder="至少输入 2 个字符" @input="scheduleSearch" />
      <view class="search-status" v-if="searchQueryLength < 2">输入昵称查找学习伙伴</view><view class="search-status" v-else-if="searching">正在查找...</view><view class="search-status" v-else-if="!searchResults.length">没有找到匹配的学习伙伴</view>
      <view class="candidate" v-for="candidate in searchResults" :key="candidate.invite_target_id" :class="{ selected: selectedCandidate?.invite_target_id === candidate.invite_target_id }" @click="selectedCandidate = candidate"><image v-if="candidate.avatar_url" :src="candidate.avatar_url" /><view class="avatar" v-else>{{ candidate.nickname.slice(0, 1) }}</view><text>{{ candidate.nickname }}</text></view>
      <view class="sheet-actions"><button class="secondary" @click="closeInvite">取消</button><button class="primary" :disabled="!selectedCandidate" @click="confirmInvite">确认邀请</button></view>
    </view></view>
  </view>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { PlanApi, RecoveryApi, UserApi, type CreatePlanReq, type Plan, type PlanActionId, type PlanActionLayout, type UserSearchResult } from '@/api'
import { addLocalDays, localDateKey } from '@/utils/date'
import { formatScheduleConflicts } from '@/utils/schedule'
import { normalizeDisplayText, unicodeLength } from '@/utils/text'

const DEFAULT_LAYOUT: PlanActionLayout = { direct: ['toggle_status', 'edit'], overflow: ['postpone', 'invite', 'delete'] }
const plans = ref<Plan[]>([]), loading = ref(false), overdueCount = ref(0), showForm = ref(false), showLayout = ref(false)
const editing = ref<Plan | null>(null), delayPlan = ref<Plan | null>(null), invitePlan = ref<Plan | null>(null)
const layout = ref<PlanActionLayout>({ ...DEFAULT_LAYOUT, direct: [...DEFAULT_LAYOUT.direct], overflow: [...DEFAULT_LAYOUT.overflow] })
const layoutDraft = reactive<PlanActionLayout>({ direct: [], overflow: [] })
const delayDays = ref(1), delayOptions = Array.from({ length: 30 }, (_, index) => `${index + 1} 天`)
const searchQuery = ref(''), searchResults = ref<UserSearchResult[]>([]), selectedCandidate = ref<UserSearchResult | null>(null), searching = ref(false)
const formError = ref('')
let searchTimer: ReturnType<typeof setTimeout> | null = null
let searchSequence = 0
let dragTimer: ReturnType<typeof setTimeout> | null = null
let measuringDrag = false
const dragState = ref<{ action: PlanActionId; area: keyof PlanActionLayout } | null>(null)
const weekdays = [{ value: 1, label: '一' }, { value: 2, label: '二' }, { value: 3, label: '三' }, { value: 4, label: '四' }, { value: 5, label: '五' }, { value: 6, label: '六' }, { value: 7, label: '日' }]
const form = reactive<CreatePlanReq>({ title: '', description: '', objective: '', weekly_target_hours: 7, start_date: '', end_date: '', default_planned_start: '20:00', default_planned_end: '21:00', study_weekdays: [1,2,3,4,5], confirm_overload: false })
const activeCount = computed(() => plans.value.filter(plan => plan.status === 'active').length)
const directActions = computed(() => layout.value.direct.slice(0, 2))
const overflowActions = computed(() => [...layout.value.direct.slice(2), ...layout.value.overflow])
const searchQueryLength = computed(() => unicodeLength(searchQuery.value))

async function load() {
  loading.value = true
  try {
    const [planRows, recovery, savedLayout] = await Promise.all([PlanApi.list(), RecoveryApi.preview().catch(() => null), UserApi.planActionLayout().catch(() => null)])
    plans.value = planRows || []
    overdueCount.value = recovery?.overdue_tasks || 0
    if (savedLayout) layout.value = normalizeLayout(savedLayout)
  } catch (error: any) { uni.showToast({ title: error?.message || '加载失败', icon: 'none' }) }
  finally { loading.value = false }
}
function normalizeLayout(value: PlanActionLayout) {
  const allowed: PlanActionId[] = ['toggle_status', 'edit', 'postpone', 'invite', 'delete']
  const direct = (value.direct || []).filter((id, index, rows) => allowed.includes(id) && rows.indexOf(id) === index)
  const overflow = (value.overflow || []).filter((id, index, rows) => allowed.includes(id) && !direct.includes(id) && rows.indexOf(id) === index)
  for (const action of allowed) if (!direct.includes(action) && !overflow.includes(action)) overflow.push(action)
  return { direct, overflow }
}
function resetForm() { const start = new Date(); Object.assign(form, { title: '', description: '', objective: '', weekly_target_hours: 7, start_date: localDateKey(start), end_date: localDateKey(addLocalDays(start, 6)), default_planned_start: '20:00', default_planned_end: '21:00', study_weekdays: [1,2,3,4,5], confirm_overload: false }); formError.value = '' }
function openCreate() { editing.value = null; resetForm(); showForm.value = true }
function openEdit(plan: Plan) { editing.value = plan; Object.assign(form, { title: plan.title, description: plan.description || '', objective: '', weekly_target_hours: plan.weekly_target_hours, start_date: plan.start_date || '', end_date: plan.end_date || '', default_planned_start: plan.default_planned_start, default_planned_end: plan.default_planned_end, study_weekdays: [...(plan.study_weekdays || [])], confirm_overload: false }); formError.value = ''; showForm.value = true }
function closeForm() { showForm.value = false }
async function save() {
  formError.value = ''
  if (!form.title.trim() || (!editing.value && !form.objective?.trim())) return void (formError.value = '请填写计划名称和具体任务目标')
  if (!form.study_weekdays?.length || !form.start_date || !form.end_date || form.start_date > form.end_date) return void (formError.value = '请选择有效日期范围和至少一个学习日')
  if ((form.default_planned_start || '') >= (form.default_planned_end || '')) return void (formError.value = '结束时间须晚于开始时间')
  try { if (!editing.value) await PlanApi.validateSchedule({ ...form }); editing.value ? await PlanApi.update(editing.value.id, { ...form }) : await PlanApi.create({ ...form }); showForm.value = false; await load(); uni.showToast({ title: '已保存', icon: 'success' }) }
  catch (error: any) { formError.value = scheduleError(error) || error?.message || '保存失败' }
}
function scheduleError(error: any) { const rows = error?.raw?.invalid_tasks; return Array.isArray(rows) ? formatScheduleConflicts(rows.map((row: any) => ({ ...row, id: row.task_id, conflicting_tasks: row.conflicting_tasks || [] }))) : '' }
function runAction(action: PlanActionId, plan: Plan) { if (action === 'toggle_status') togglePause(plan); if (action === 'edit') openEdit(plan); if (action === 'postpone') openDelay(plan); if (action === 'invite') openInvite(plan); if (action === 'delete') removePlan(plan) }
function openMore(plan: Plan) { const actions = overflowActions.value; uni.showActionSheet({ itemList: [...actions.map(action => actionName(action)), '编辑操作'], success: result => result.tapIndex === actions.length ? openLayout() : runAction(actions[result.tapIndex], plan) }) }
async function togglePause(plan: Plan) { try { Object.assign(plan, plan.status === 'paused' ? await PlanApi.resume(plan.id) : await PlanApi.pause(plan.id)) } catch (error: any) { uni.showToast({ title: error?.message || '操作失败', icon: 'none' }) } }
async function removePlan(plan: Plan) { const confirmed = await new Promise<boolean>(resolve => uni.showModal({ title: '删除计划', content: `将删除「${plan.title}」及其任务与关联打卡历史，此操作不可恢复。`, confirmColor: '#c95668', confirmText: '确认删除', success: result => resolve(result.confirm) })); if (!confirmed) return; try { await PlanApi.remove(plan.id); plans.value = plans.value.filter(item => item.id !== plan.id) } catch (error: any) { uni.showToast({ title: error?.message || '删除失败', icon: 'none' }) } }
function openLayout() { Object.assign(layoutDraft, { direct: [...layout.value.direct], overflow: [...layout.value.overflow] }); showLayout.value = true }
function move(action: PlanActionId, area: keyof PlanActionLayout, offset: number) { const rows = layoutDraft[area], index = rows.indexOf(action), next = index + offset; if (next < 0 || next >= rows.length) return; rows.splice(index, 1); rows.splice(next, 0, action) }
function transfer(action: PlanActionId, area: keyof PlanActionLayout) { const target = area === 'direct' ? 'overflow' : 'direct'; layoutDraft[area].splice(layoutDraft[area].indexOf(action), 1); layoutDraft[target].push(action) }
function startDrag(action: PlanActionId, area: keyof PlanActionLayout, event: any) { endDrag(); if (!event.touches?.[0]) return; dragTimer = setTimeout(() => { dragState.value = { action, area }; uni.vibrateShort?.({ type: 'light' }) }, 450) }
function moveDrag(event: any) { if (!dragState.value || measuringDrag) return; const touch = event.touches?.[0]; if (!touch) return; if (typeof event.preventDefault === 'function') event.preventDefault(); measuringDrag = true; const y = touch.clientY ?? touch.pageY; uni.createSelectorQuery().selectAll('.layout-zone,.layout-drop-row').boundingClientRect(result => { measuringDrag = false; reorderAtPoint(y, Array.isArray(result) ? result : [result]) }).exec() }
function reorderAtPoint(y: number, rects: any[]) { const state = dragState.value; if (!state) return; const directZone = rects.find(rect => rect.id === 'layout-direct-zone'), overflowZone = rects.find(rect => rect.id === 'layout-overflow-zone'); const targetArea: keyof PlanActionLayout | null = y >= directZone?.top && y <= directZone?.bottom ? 'direct' : y >= overflowZone?.top && y <= overflowZone?.bottom ? 'overflow' : null; if (!targetArea) return; const rows = layoutDraft[targetArea], source = layoutDraft[state.area], sourceIndex = source.indexOf(state.action); if (sourceIndex >= 0) source.splice(sourceIndex, 1); const targetRects = rects.filter(rect => String(rect.id || '').startsWith(`layout-${targetArea}-`) && !String(rect.id).endsWith(`-${state.action}`)); const targetIndex = targetRects.findIndex(rect => y < rect.top + rect.height / 2); rows.splice(targetIndex < 0 ? rows.length : targetIndex, 0, state.action); state.area = targetArea }
function endDrag() { if (dragTimer) clearTimeout(dragTimer); dragTimer = null; dragState.value = null }
async function saveLayout() { try { layout.value = normalizeLayout(await UserApi.savePlanActionLayout({ direct: [...layoutDraft.direct], overflow: [...layoutDraft.overflow] })); showLayout.value = false } catch (error: any) { uni.showToast({ title: error?.message || '布局保存失败', icon: 'none' }) } }
function openDelay(plan: Plan) { delayPlan.value = plan; delayDays.value = 1 }
async function confirmDelay() { const plan = delayPlan.value; if (!plan) return; try { const result = await PlanApi.delay(plan.id, delayDays.value); delayPlan.value = null; await load(); const moved = result.moved_task_count ?? result.moved_tasks ?? result.moved; const range = result.new_start_date && result.new_end_date ? `，新范围 ${result.new_start_date} 至 ${result.new_end_date}` : ''; uni.showToast({ title: moved === undefined ? `已延期 ${delayDays.value} 天` : `已移动 ${moved} 个任务${range}`, icon: 'none', duration: 3000 }) } catch (error: any) { uni.showToast({ title: scheduleError(error) || error?.message || '延期失败', icon: 'none' }) } }
function openInvite(plan: Plan) { searchSequence++; invitePlan.value = plan; searchQuery.value = ''; searchResults.value = []; selectedCandidate.value = null; searching.value = false }
function closeInvite() { searchSequence++; invitePlan.value = null; searching.value = false; if (searchTimer) clearTimeout(searchTimer) }
function scheduleSearch() { selectedCandidate.value = null; if (searchTimer) clearTimeout(searchTimer); const query = normalizeDisplayText(searchQuery.value); const sequence = ++searchSequence; if (unicodeLength(query) < 2) { searchResults.value = []; searching.value = false; return } searchTimer = setTimeout(() => searchUsers(query, sequence), 350) }
async function searchUsers(query: string, sequence: number) { searching.value = true; try { const results = await UserApi.search(query); if (sequence === searchSequence && query === normalizeDisplayText(searchQuery.value)) searchResults.value = results } catch (error: any) { if (sequence === searchSequence) uni.showToast({ title: error?.message || '搜索失败', icon: 'none' }) } finally { if (sequence === searchSequence) searching.value = false } }
async function confirmInvite() { if (!invitePlan.value || !selectedCandidate.value) return; try { await PlanApi.invite(invitePlan.value.id, selectedCandidate.value.invite_target_id); closeInvite(); uni.showToast({ title: '邀请已发送', icon: 'success' }) } catch (error: any) { uni.showToast({ title: error?.message || '邀请失败', icon: 'none' }) } }
function planRange(plan: Plan) { return plan.start_date && plan.end_date ? `${plan.start_date} 至 ${plan.end_date}` : '未设置日期范围' }
function delayedRange(plan: Plan) { return plan.start_date && plan.end_date ? `${localDateKey(addLocalDays(new Date(`${plan.start_date}T12:00:00`), delayDays.value))} 至 ${localDateKey(addLocalDays(new Date(`${plan.end_date}T12:00:00`), delayDays.value))}` : '未来未完成任务顺延' }
function calculatedRate(plan: Plan) { return plan.total_tasks ? Math.round(plan.completed_tasks / plan.total_tasks * 100) : 0 }
function actionName(action: PlanActionId) { return ({ toggle_status: '暂停/恢复', edit: '编辑', postpone: '延期', invite: '邀请学习伙伴', delete: '删除' })[action] }
function actionLabel(action: PlanActionId, plan: Plan) { return action === 'toggle_status' ? (plan.status === 'paused' ? '恢复' : '暂停') : actionName(action) }
function statusText(status: string) { return status === 'paused' ? '暂停' : status === 'archived' ? '归档' : '进行中' }
function weekdaySummary(selected: number[] = []) { return selected.length === 7 ? '每天' : selected.length ? `周${selected.map(value => weekdays.find(day => day.value === value)?.label).join('、')}` : '未设置学习日' }
function toggleWeekday(value: number) { const selected = form.study_weekdays || []; form.study_weekdays = selected.includes(value) ? selected.filter(item => item !== value) : [...selected, value].sort() }
function setFormValue(field: 'start_date' | 'end_date' | 'default_planned_start' | 'default_planned_end', event: any) { form[field] = event.detail.value; formError.value = '' }
function goAI() { uni.navigateTo({ url: '/pages/ai/ai' }) } function goSchedule() { uni.navigateTo({ url: '/pages/schedule/schedule' }) } function goGroup() { uni.navigateTo({ url: '/pages/group/group' }) } function goNotifications() { uni.navigateTo({ url: '/pages/notifications/notifications' }) } function goRecovery() { uni.navigateTo({ url: '/pages/recovery/recovery' }) } function goOps() { uni.navigateTo({ url: '/pages/ops/ops' }) }
onShow(load)
</script>

<style lang="scss">
.page { min-height: 100vh; box-sizing: border-box; padding: 28rpx 28rpx 64rpx; background: linear-gradient(180deg,#fff0f7,#fffaf0 42%,#f7fbff); color: #4b2b3f; }
.summary { display:flex; justify-content:space-between; align-items:center; padding:30rpx 32rpx; border-radius:30rpx; background:#fff; border:1rpx solid #ffe0ea; box-shadow:0 16rpx 36rpx rgba(255,143,171,.12); }.eyebrow{color:#ff6f91;font-size:22rpx;font-weight:900}.summary-title{margin-top:8rpx;font-size:36rpx;font-weight:900}.summary-count{color:#ff8fab;font-size:56rpx;font-weight:900}
.ai-hero{display:flex;align-items:center;gap:22rpx;margin-top:20rpx;padding:32rpx;border-radius:32rpx;background:linear-gradient(135deg,#ff769f,#ffb35c);color:#fff;box-shadow:0 20rpx 42rpx rgba(255,111,145,.24)}.ai-badge{display:flex;align-items:center;justify-content:center;width:82rpx;height:82rpx;border-radius:26rpx;background:rgba(255,255,255,.22);font-size:28rpx;font-weight:900}.ai-copy{flex:1}.ai-title{font-size:35rpx;font-weight:900}.ai-desc{margin-top:8rpx;color:rgba(255,255,255,.9);font-size:23rpx;line-height:1.45}.arrow{font-size:52rpx}.manual{margin:14rpx 0 0;border-radius:999rpx;background:#fff;color:#ff6f91;border:1rpx solid #ffd3e0;font-size:26rpx;font-weight:800}
.tools{display:flex;gap:12rpx;margin-top:20rpx}.tool{flex:1;min-width:0;padding:18rpx 6rpx;border-radius:22rpx;background:#fff;text-align:center;border:1rpx solid #ffe6ed}.tool-icon{display:flex;align-items:center;justify-content:center;width:50rpx;height:50rpx;margin:auto;border-radius:16rpx;background:#fff0f6;color:#ff6f91;font-size:21rpx;font-weight:900}.tool text{display:block;margin-top:8rpx;color:#606a80;font-size:21rpx;white-space:nowrap}.tool.recovery{flex:1.35;background:#fff8e9;border-color:#ffe0a8}.tool.recovery .tool-icon{background:#ffedc7;color:#b46c10}
.section-head{display:flex;align-items:center;justify-content:space-between;margin:32rpx 4rpx 16rpx;font-size:32rpx;font-weight:900}.section-head button{margin:0;padding:0 20rpx;height:56rpx;line-height:56rpx;border-radius:99rpx;background:#fff0f6;color:#ff6f91;font-size:22rpx}.cards{display:flex;flex-direction:column;gap:18rpx}.card{padding:28rpx;border-radius:28rpx;background:#fff;border:1rpx solid #ffe0ea;box-shadow:0 12rpx 30rpx rgba(255,143,171,.1)}.card.paused{opacity:.65}.card-head,.progress-copy{display:flex;align-items:center;justify-content:space-between;gap:16rpx}.plan-title{font-size:31rpx;font-weight:900}.pill{padding:8rpx 16rpx;border-radius:99rpx;background:#fff0f6;color:#ff6f91;font-size:21rpx;font-weight:800}.pill.paused{background:#fff5df;color:#a26817}.plan-desc{margin-top:12rpx;color:#606a80;font-size:24rpx;line-height:1.5}.schedule{margin-top:12rpx;color:#9a6176;font-size:22rpx}.progress{margin-top:22rpx}.progress-copy{font-size:23rpx;font-weight:800}.track{height:14rpx;margin-top:12rpx;border-radius:99rpx;background:#edf1f6;overflow:hidden}.fill{height:100%;border-radius:99rpx;background:linear-gradient(90deg,#ff8fab,#ffc36a)}.taskless{margin-top:20rpx;padding:18rpx;border-radius:16rpx;background:#f8fafc;color:#8a92a6;font-size:23rpx}.actions{display:grid;grid-template-columns:repeat(3,1fr);gap:10rpx;margin-top:22rpx}.action{margin:0;padding:0 8rpx;height:64rpx;line-height:64rpx;border-radius:99rpx;background:#fff2f7;color:#d85e82;font-size:22rpx}.action.more{background:#f3f6fb;color:#606a80}.action.destructive{background:#f7f4f4;color:#9a7379}.settings{margin-top:30rpx;text-align:center;color:#9aa0ae;font-size:23rpx}.empty{padding:48rpx 32rpx;border-radius:28rpx;background:#fff}.empty-title{font-size:31rpx;font-weight:900}.empty-desc{margin-top:10rpx;color:#7b8498;font-size:24rpx}
.sheet{position:fixed;inset:0;z-index:99;display:flex;align-items:flex-end;background:rgba(48,34,43,.46)}.sheet-body{width:100%;max-height:86vh;box-sizing:border-box;padding:34rpx 30rpx 44rpx;border-radius:30rpx 30rpx 0 0;background:#fff;overflow-y:auto}.sheet-title{font-size:34rpx;font-weight:900}.sheet-note{margin:10rpx 0 24rpx;color:#7b8498;font-size:23rpx;line-height:1.5}.field{margin-bottom:20rpx}.field>text{display:block;margin-bottom:9rpx;color:#606a80;font-size:23rpx}.field input,.field textarea,.picker,.search{box-sizing:border-box;width:100%;padding:0 20rpx;border:1rpx solid #dfe4ed;border-radius:14rpx;background:#f9fbff;font-size:26rpx}.field input,.picker,.search{height:78rpx;line-height:78rpx}.field textarea{height:126rpx;padding-top:16rpx}.grid{display:grid;grid-template-columns:1fr 1fr;gap:14rpx}.weekdays{display:grid;grid-template-columns:repeat(7,1fr);gap:8rpx}.weekday{height:58rpx;line-height:58rpx;border-radius:50%;background:#f1f4f8;text-align:center;font-size:22rpx}.weekday.selected{background:#ff7aa2;color:#fff;font-weight:900}.confirm{display:flex;align-items:center;margin:4rpx 0 22rpx;color:#606a80;font-size:22rpx}.form-error{margin:-4rpx 0 18rpx;padding:16rpx;border-radius:12rpx;background:#fff1f3;color:#c44b61;font-size:22rpx;white-space:pre-line}.sheet-actions{display:grid;grid-template-columns:1fr 1fr;gap:14rpx;margin-top:24rpx}.primary,.secondary{margin:0;border-radius:99rpx;font-size:26rpx}.primary{background:linear-gradient(135deg,#ff7aa2,#ffb45c);color:#fff}.secondary{background:#f3f6fb;color:#606a80}.layout-title{margin-top:18rpx;color:#ff6f91;font-size:24rpx;font-weight:900}.overflow-title{margin-top:28rpx}.layout-row{display:flex;align-items:center;justify-content:space-between;gap:12rpx;padding:16rpx 0;border-bottom:1rpx solid #eef1f5;font-size:25rpx;font-weight:800}.layout-row>view{display:flex;gap:6rpx}.layout-row button{margin:0;padding:0 12rpx;height:50rpx;line-height:50rpx;background:#f4f6fa;color:#606a80;font-size:19rpx}.dangerText{color:#a96b75}.stepper{display:grid;grid-template-columns:80rpx 1fr 80rpx;align-items:center;gap:16rpx}.stepper button{margin:0;background:#fff0f6;color:#ff6f91}.stepper picker{padding:20rpx;border-radius:14rpx;background:#f8fafc;text-align:center;font-size:30rpx;font-weight:900}.date-preview{margin-top:22rpx;padding:20rpx;border-radius:16rpx;background:#fff8eb;color:#775631;font-size:24rpx;line-height:1.7}.search-status{padding:26rpx;text-align:center;color:#8a92a6;font-size:23rpx}.candidate{display:flex;align-items:center;gap:16rpx;padding:16rpx;border-radius:16rpx;border:2rpx solid transparent}.candidate.selected{border-color:#ff8fab;background:#fff7fa}.candidate image,.avatar{width:62rpx;height:62rpx;border-radius:50%}.avatar{display:flex;align-items:center;justify-content:center;background:#fff0f6;color:#ff6f91;font-weight:900}.candidate text{font-size:26rpx;font-weight:800}
.layout-zone{min-height:100rpx;margin-top:18rpx;padding:0 16rpx 10rpx;border:1rpx dashed #ffd3e0;border-radius:18rpx}.overflow-zone{margin-top:22rpx;border-color:#dfe4ed}.layout-row.dragging{opacity:.35}.layout-empty{padding:22rpx 0;color:#9aa0ae;font-size:21rpx;text-align:center}
</style>
