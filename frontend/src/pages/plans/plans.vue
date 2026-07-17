<template>
  <view class="plans-page">
    <view class="actions">
      <button class="btn-add" @click="openCreate">+ 新建计划</button>
    </view>

    <view class="empty" v-if="!loading && plans.length === 0">
      <view class="empty-icon">📋</view>
      <view>还没有学习计划</view>
      <button class="link" @click="openCreate">创建第一个计划</button>
    </view>

    <view class="plan-list" v-else>
      <view class="plan-card" v-for="p in plans" :key="p.id" :class="{ paused: p.status === 'paused' }">
        <view class="plan-title" @click="goToday(p)">{{ p.title }}</view>
        <view class="plan-desc" v-if="p.description">{{ p.description }}</view>
        <view class="plan-meta">
          <text>目标 {{ p.weekly_target_hours || 0 }} h/周</text>
          <text class="tag" :class="p.status">{{ statusText(p.status) }}</text>
        </view>
        <view class="plan-actions">
          <view class="pact" @click="togglePause(p)">{{ p.status === 'paused' ? '恢复' : '暂停' }}</view>
          <view class="pact" @click="openEdit(p)">编辑</view>
          <view class="pact danger" @click="del(p)">删除</view>
        </view>
      </view>
    </view>

    <!-- 新建/编辑 弹层 -->
    <view class="modal" v-if="showModal" @click.self="closeModal">
      <view class="modal-body">
        <view class="modal-title">{{ editing ? '编辑计划' : '新建学习计划' }}</view>
        <view class="form-row">
          <text class="label">标题</text>
          <input v-model="form.title" placeholder="如：学习Go语言" />
        </view>
        <view class="form-row">
          <text class="label">描述</text>
          <textarea v-model="form.description" placeholder="可选" />
        </view>
        <view class="form-row">
          <text class="label">每周目标(小时)</text>
          <input v-model.number="form.weekly_target_hours" type="number" placeholder="如 28" />
        </view>
        <view class="warnings" v-if="warnings.length > 0">
          <view class="warn-item" v-for="w in warnings" :key="w">⚠️ {{ w }}</view>
          <view class="warn-confirm">
            <label><input type="checkbox" v-model="form.confirm_overload" />我已知晓，仍要创建</label>
          </view>
        </view>
        <view class="modal-actions">
          <button @click="closeModal">取消</button>
          <button class="primary" @click="save">{{ editing ? '保存' : '创建' }}</button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { PlanApi, type Plan, type CreatePlanReq } from '@/api'

interface FormState extends CreatePlanReq {
  confirm_overload?: boolean
}

const plans = ref<Plan[]>([])
const loading = ref(false)
const showModal = ref(false)
const editing = ref<Plan | null>(null)
const warnings = ref<string[]>([])
const form = reactive<FormState>({
  title: '',
  description: '',
  weekly_target_hours: 0,
  start_date: '',
  end_date: '',
  confirm_overload: false,
})

async function load() {
  loading.value = true
  try {
    plans.value = await PlanApi.list()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '加载失败', icon: 'none' })
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.title = ''
  form.description = ''
  form.weekly_target_hours = 0
  form.start_date = ''
  form.end_date = ''
  form.confirm_overload = false
  warnings.value = []
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
  form.start_date = p.start_date || ''
  form.end_date = p.end_date || ''
  form.confirm_overload = false
  warnings.value = []
  showModal.value = true
}

function closeModal() {
  showModal.value = false
}

async function save() {
  if (!form.title?.trim()) {
    uni.showToast({ title: '请输入计划标题', icon: 'none' })
    return
  }
  try {
    if (editing.value) {
      await PlanApi.update(editing.value.id, {
        title: form.title,
        description: form.description,
        weekly_target_hours: form.weekly_target_hours,
        start_date: form.start_date,
        end_date: form.end_date,
      })
      uni.showToast({ title: '已保存', icon: 'success' })
    } else {
      // 注意：后端在有警告但未 confirm 时会返回错误
      try {
        await PlanApi.create({ ...form })
        uni.showToast({ title: '已创建', icon: 'success' })
      } catch (e: any) {
        // 后端在超负荷且未 confirm 时返回 code != 0；
        // 我们的封装会把 message 通过 e.message 透出
        if (e?.code === 400 && /overload/i.test(e?.message || '')) {
          // 这里走二次提示，要求用户勾选 confirm
          uni.showModal({
            title: '超负荷提示',
            content: e.message + '\n\n如要继续创建，请在表单中勾选"我已知晓"',
            showCancel: false,
          })
          return
        }
        throw e
      }
    }
    showModal.value = false
    await load()
  } catch (e: any) {
    uni.showToast({ title: e?.message || '保存失败', icon: 'none' })
  }
}

async function togglePause(p: Plan) {
  try {
    if (p.status === 'paused') {
      await PlanApi.resume(p.id)
      p.status = 'active'
      uni.showToast({ title: '已恢复', icon: 'success' })
    } else {
      await PlanApi.pause(p.id)
      p.status = 'paused'
      uni.showToast({ title: '已暂停', icon: 'success' })
    }
  } catch (e: any) {
    uni.showToast({ title: e?.message || '操作失败', icon: 'none' })
  }
}

async function del(p: Plan) {
  const res = await new Promise<boolean>(resolve =>
    uni.showModal({
      title: '删除计划',
      content: `确定删除「${p.title}」？相关打卡记录也会被删除`,
      success: r => resolve(r.confirm),
    })
  )
  if (!res) return
  try {
    await PlanApi.remove(p.id)
    plans.value = plans.value.filter(x => x.id !== p.id)
    uni.showToast({ title: '已删除', icon: 'success' })
  } catch (e: any) {
    uni.showToast({ title: e?.message || '删除失败', icon: 'none' })
  }
}

function goToday(p: Plan) {
  uni.switchTab({ url: '/pages/checkin/checkin' })
}

function statusText(s: string) {
  return s === 'paused' ? '已暂停' : s === 'archived' ? '已归档' : '进行中'
}

onShow(load)
</script>

<style lang="scss">
.plans-page { min-height: 100vh; background: #f5f6fa; padding-bottom: 40rpx; }
.actions { padding: 20rpx 30rpx 0; }
.btn-add {
  background: #4C8BF5; color: #fff; border-radius: 40rpx; font-size: 28rpx;
  width: 240rpx; margin-left: auto;
}

.empty { text-align: center; padding: 120rpx 40rpx; color: #888; }
.empty-icon { font-size: 120rpx; margin-bottom: 30rpx; }
.link { margin-top: 30rpx; color: #4C8BF5; background: transparent; font-size: 28rpx; }

.plan-list { padding: 30rpx; }
.plan-card {
  background: #fff; border-radius: 16rpx; padding: 30rpx; margin-bottom: 20rpx;
  box-shadow: 0 2rpx 12rpx rgba(0,0,0,.04);
}
.plan-card.paused { opacity: .6; }
.plan-title { font-size: 34rpx; font-weight: 600; color: #333; }
.plan-desc { font-size: 26rpx; color: #888; margin-top: 10rpx; }
.plan-meta { display: flex; justify-content: space-between; margin-top: 20rpx; font-size: 24rpx; color: #666; }
.tag { padding: 4rpx 16rpx; border-radius: 20rpx; background: #e8f5e9; color: #2e7d32; }
.tag.paused { background: #fff3e0; color: #fb8c00; }
.tag.archived { background: #f5f5f5; color: #888; }
.plan-actions { display: flex; margin-top: 24rpx; border-top: 1rpx solid #eee; padding-top: 16rpx; }
.pact { flex: 1; text-align: center; color: #4C8BF5; font-size: 26rpx; }
.pact.danger { color: #e53935; }

.modal {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,.4); display: flex; align-items: center; justify-content: center;
  z-index: 99;
}
.modal-body {
  width: 640rpx; background: #fff; border-radius: 20rpx; padding: 30rpx;
}
.modal-title { font-size: 34rpx; font-weight: 600; margin-bottom: 24rpx; text-align: center; }
.form-row { margin-bottom: 24rpx; }
.label { display: block; font-size: 26rpx; color: #666; margin-bottom: 10rpx; }
.form-row input, .form-row textarea {
  width: 100%; border: 1rpx solid #ddd; border-radius: 12rpx; padding: 16rpx;
  font-size: 28rpx;
}
.warnings { margin: 16rpx 0; padding: 16rpx; background: #fffbe6; border-radius: 12rpx; }
.warn-item { font-size: 24rpx; color: #d46b08; margin-bottom: 8rpx; }
.warn-confirm { font-size: 24rpx; color: #555; margin-top: 10rpx; }
.warn-confirm input { margin-right: 8rpx; }
.modal-actions { display: flex; margin-top: 30rpx; }
.modal-actions button { flex: 1; margin: 0 8rpx; }
.modal-actions .primary { background: #4C8BF5; color: #fff; }
</style>