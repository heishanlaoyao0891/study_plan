<template>
  <view class="page">
    <view class="hero">
      <view class="eyebrow">SUPPORT</view>
      <view class="hero-title">告诉我们哪里需要改进</view>
      <view class="hero-copy">反馈会关联当前账号，处理结果将显示在下方记录中。</view>
    </view>

    <view class="panel form-panel">
      <view class="section-title">反馈类型</view>
      <view class="categories">
        <view v-for="item in categories" :key="item.value" class="category" :class="{ active: category === item.value }" @click="category = item.value">{{ item.label }}</view>
      </view>
      <view class="field-head"><text>问题或建议</text><text>{{ contentLength }}/1000</text></view>
      <textarea v-model="content" maxlength="1000" placeholder="请描述发生了什么、期望结果或改进建议" />
      <view class="field-head"><text>联系方式（可选）</text><text>{{ contactLength }}/100</text></view>
      <input v-model="contact" maxlength="100" placeholder="手机号、邮箱或其他联系方式" />
      <view v-if="submitError" class="message error">{{ submitError }}</view>
      <view v-if="success" class="message success">反馈已收到，可在下方查看处理进度。</view>
      <button class="primary" :loading="submitting" :disabled="submitting" @click="submit">{{ submitting ? '提交中…' : '提交反馈' }}</button>
    </view>

    <view class="history-head"><view class="section-title">我的反馈</view><view class="refresh" @click="loadHistory">刷新</view></view>
    <view v-if="loading" class="state">正在加载反馈记录…</view>
    <view v-else-if="historyError" class="state error">{{ historyError }}<view class="retry" @click="loadHistory">重新加载</view></view>
    <view v-else-if="!reports.length" class="state">还没有反馈记录</view>
    <view v-else class="history">
      <view v-for="report in reports" :key="report.id" class="report">
        <view class="report-head"><view class="tag">{{ categoryLabel(report.category) }}</view><view class="status" :class="report.status">{{ statusLabel(report.status) }}</view></view>
        <view class="report-content">{{ report.content }}</view>
        <view v-if="report.public_response" class="response"><view class="response-label">管理员回复</view>{{ report.public_response }}</view>
        <view class="time">更新于 {{ formatTime(report.updated_at) }}</view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { OpsApi, type FeedbackCategory, type FeedbackReport, type FeedbackStatus } from '@/api'

const categories: Array<{ value: FeedbackCategory; label: string }> = [
  { value: 'issue', label: '问题故障' }, { value: 'suggestion', label: '功能建议' },
  { value: 'content', label: '内容问题' }, { value: 'account', label: '账号问题' }, { value: 'other', label: '其他' },
]
const category = ref<FeedbackCategory>('issue')
const content = ref('')
const contact = ref('')
const submitting = ref(false)
const success = ref(false)
const submitError = ref('')
const loading = ref(false)
const historyError = ref('')
const reports = ref<FeedbackReport[]>([])
const contentLength = computed(() => Array.from(content.value).length)
const contactLength = computed(() => Array.from(contact.value).length)

async function submit() {
  if (submitting.value) return
  const trimmedContent = content.value.trim()
  const trimmedContact = contact.value.trim()
  submitError.value = ''
  success.value = false
  if (!trimmedContent) { submitError.value = '请填写问题或建议'; return }
  if (Array.from(trimmedContent).length > 1000) { submitError.value = '内容不能超过 1000 个字符'; return }
  if (Array.from(trimmedContact).length > 100) { submitError.value = '联系方式不能超过 100 个字符'; return }
  submitting.value = true
  try {
    await OpsApi.feedback({ category: category.value, content: trimmedContent, contact: trimmedContact })
    content.value = ''
    contact.value = ''
    success.value = true
    await loadHistory()
  } catch (error: any) {
    submitError.value = error?.message || '提交失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}

async function loadHistory() {
  loading.value = true
  historyError.value = ''
  try { reports.value = await OpsApi.feedbackHistory() }
  catch (error: any) { historyError.value = error?.message || '反馈记录加载失败' }
  finally { loading.value = false }
}

function categoryLabel(value: FeedbackCategory) { return categories.find(item => item.value === value)?.label || value }
function statusLabel(value: FeedbackStatus) { return { open: '待处理', processing: '处理中', resolved: '已解决', closed: '已关闭' }[value] }
function formatTime(value: string) { return value ? new Date(value).toLocaleString() : '-' }

onShow(loadHistory)
</script>

<style lang="scss">
.page { min-height: 100vh; box-sizing: border-box; padding: 28rpx 28rpx 64rpx; background: linear-gradient(180deg,#fff0f7,#fffaf0 42%,#f7fbff); color: #4b2b3f; }
.hero { padding: 22rpx 8rpx 30rpx; }
.eyebrow { color: #ff6f91; font-size: 21rpx; font-weight: 900; letter-spacing: 4rpx; }
.hero-title { margin-top: 10rpx; font-size: 40rpx; font-weight: 900; }
.hero-copy { margin-top: 10rpx; color: #7b6571; font-size: 24rpx; line-height: 1.6; }
.panel,.report,.state { padding: 30rpx; border: 1rpx solid #ffe0ea; border-radius: 28rpx; background: #fff; box-shadow: 0 14rpx 32rpx rgba(255,143,171,.09); }
.section-title { font-size: 31rpx; font-weight: 900; }
.categories { display: flex; flex-wrap: wrap; gap: 14rpx; margin-top: 20rpx; }
.category { padding: 14rpx 22rpx; border-radius: 999rpx; background: #f7f4f6; color: #7b6571; font-size: 23rpx; }
.category.active { background: #ff769f; color: #fff; font-weight: 800; }
.field-head { display: flex; justify-content: space-between; margin: 28rpx 4rpx 12rpx; color: #67505d; font-size: 23rpx; }
textarea,input { box-sizing: border-box; width: 100%; border: 1rpx solid #eadde3; border-radius: 18rpx; background: #fffafd; font-size: 26rpx; }
textarea { height: 240rpx; padding: 20rpx; }
input { height: 84rpx; padding: 0 20rpx; }
.primary { margin-top: 24rpx; border-radius: 999rpx; background: #111827; color: #fff; }
.primary[disabled] { opacity: .65; }
.message { margin-top: 18rpx; padding: 16rpx 20rpx; border-radius: 14rpx; font-size: 23rpx; }
.error { color: #b64059; background: #fff0f3; }
.success { color: #2d7657; background: #edf9f2; }
.history-head { display: flex; justify-content: space-between; align-items: center; margin: 36rpx 6rpx 16rpx; }
.refresh,.retry { color: #d85e82; font-size: 24rpx; font-weight: 800; }
.history { display: flex; flex-direction: column; gap: 18rpx; }
.report-head { display: flex; justify-content: space-between; align-items: center; }
.tag,.status { padding: 9rpx 16rpx; border-radius: 999rpx; font-size: 21rpx; }
.tag { background: #fff0f6; color: #d85e82; }
.status { background: #f1f3f6; color: #697386; }
.status.processing { background: #fff4d8; color: #8a6414; }
.status.resolved { background: #e9f8ef; color: #2d7657; }
.status.closed { background: #edf0f4; color: #596273; }
.report-content { margin-top: 20rpx; color: #30242b; font-size: 27rpx; line-height: 1.65; white-space: pre-wrap; }
.response { margin-top: 20rpx; padding: 20rpx; border-radius: 18rpx; background: #f7f4ff; color: #51466d; font-size: 25rpx; line-height: 1.6; white-space: pre-wrap; }
.response-label { margin-bottom: 6rpx; color: #8c72b4; font-size: 21rpx; font-weight: 900; }
.time { margin-top: 18rpx; color: #9a8b93; font-size: 21rpx; }
.state { color: #7b6571; text-align: center; font-size: 25rpx; }
.retry { margin-top: 12rpx; }
</style>
