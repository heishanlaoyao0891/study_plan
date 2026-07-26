<template>
  <view class="page">
    <view class="panel">
      <view class="eyebrow">消息授权</view><view class="title">提醒设置</view>
      <!-- #ifdef MP-WEIXIN -->
      <view class="desc">每种提醒需要单独授权。微信订阅通常在成功发送一次后被消耗，需要时可再次点击授权。</view>
      <view class="loading" v-if="loading">正在读取提醒设置...</view>
      <view class="reminder-list" v-else>
        <view class="reminder" v-for="template in templates" :key="template.template_id">
          <view class="reminder-head"><view><view class="reminder-name">{{ reminderCopy(template.reminder_type).name }}</view><view class="reminder-purpose">{{ reminderCopy(template.reminder_type).purpose }}</view></view><view :class="['state', { active: subscriptionFor(template.reminder_type) }]">{{ subscriptionFor(template.reminder_type) ? '已留存授权' : '尚未授权' }}</view></view>
          <view class="record" v-if="subscriptionFor(template.reminder_type)">当前记录：{{ recordDate(subscriptionFor(template.reminder_type)?.updated_at) }} 更新。消息发送后仍建议再次授权，确保下一次可用。</view>
          <view class="record" v-else>当前没有可用的授权记录，不会发送此类提醒。</view>
          <button class="authorize" :disabled="authorizing === template.reminder_type" @click="authorize(template)">{{ authorizing === template.reminder_type ? '授权中...' : subscriptionFor(template.reminder_type) ? '再次授权' : '授权此提醒' }}</button>
        </view>
        <view class="empty" v-if="!templates.length">管理员暂未启用提醒模板</view>
      </view>
      <button class="cancel-all" v-if="subscriptions.length" @click="unsubscribe">取消全部提醒授权记录</button>
      <!-- #endif -->
      <!-- #ifndef MP-WEIXIN -->
      <view class="desc unsupported">H5 不支持微信订阅消息授权。请使用已绑定同一账号的微信小程序，在“提醒设置”中逐项完成授权；每次授权通常只支持发送一次。</view>
      <!-- #endif -->
    </view>

    <view class="events"><view class="events-title">今日提醒队列</view><view class="event" v-for="event in events" :key="`${event.type}-${event.task?.id}`"><view><view class="event-type">{{ reminderCopy(event.type).name }}</view><view class="event-task">{{ event.task?.title }}</view></view><view class="event-date">{{ event.task?.date }}</view></view><view class="empty" v-if="!events.length">暂无待发送提醒</view></view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { NotificationApi, type NotificationSubscription, type NotificationTemplate } from '@/api'

const reminderCopies: Record<string, { name: string; purpose: string }> = {
  study_start: { name: '到点学习提醒', purpose: '任务计划开始时，提醒你进入学习状态。' },
  completion: { name: '完成提醒', purpose: '学习达到计划节点时，提醒你完成和记录。' },
  decision_2330: { name: '深夜学习决策', purpose: '23:30 仍在学习时，提醒你继续、结束或推迟。' },
  missed_checkin: { name: '未打卡提醒', purpose: '错过计划开始时间后，提醒你回来打卡。' },
  group_nudge: { name: '小组督学提醒', purpose: '学习伙伴向你发起督学时及时通知。' },
	slack_balance: { name: '躺平币余额提醒', purpose: '可用分钟即将耗尽或进入负数时提醒你及时补回。' },
}
const events = ref<any[]>([]), templates = ref<NotificationTemplate[]>([]), subscriptions = ref<NotificationSubscription[]>([]), loading = ref(false), authorizing = ref('')

async function load() {
  loading.value = true
  const now = new Date(), today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
  try {
    // #ifdef MP-WEIXIN
    const [metadata, records, miniProgramDue] = await Promise.all([NotificationApi.templates(), NotificationApi.list(), NotificationApi.due(today)])
    templates.value = metadata.templates || []; subscriptions.value = records || []; events.value = miniProgramDue.events || []
    // #endif
    // #ifndef MP-WEIXIN
    const h5Due = await NotificationApi.due(today); events.value = h5Due.events || []
    // #endif
  } catch { events.value = [] } finally { loading.value = false }
}

async function authorize(template: NotificationTemplate) {
  // #ifdef MP-WEIXIN
  authorizing.value = template.reminder_type
  try {
    const results = await new Promise<Record<string, string>>((resolve, reject) => { uni.requestSubscribeMessage({ tmplIds: [template.template_id], success: result => resolve(result as unknown as Record<string, string>), fail: reject }) })
    const saved = await NotificationApi.subscribe(template.reminder_type, template.template_id, results[template.template_id])
    uni.showToast({ title: saved.accepted.includes(template.reminder_type) ? '授权记录已更新' : results[template.template_id] === 'reject' ? '你拒绝了此提醒' : '未获得授权', icon: 'none' })
  } catch (error: any) { uni.showToast({ title: error?.message || '授权失败', icon: 'none' }) } finally { authorizing.value = ''; await load() }
  // #endif
}
async function unsubscribe() {
  const confirmed = await new Promise<boolean>(resolve => uni.showModal({ title: '取消全部提醒', content: '将清除当前账号保存的所有提醒授权记录。', confirmColor: '#c95668', success: result => resolve(result.confirm) }))
  if (!confirmed) return
  try { await NotificationApi.unsubscribe(); await load(); uni.showToast({ title: '已全部取消', icon: 'success' }) } catch (error: any) { uni.showToast({ title: error?.message || '取消失败', icon: 'none' }) }
}
function subscriptionFor(type: string) {
  const template = templates.value.find(item => item.reminder_type === type)
  return template ? subscriptions.value.find(record => record.reminder_type === type && record.template_id === template.template_id && record.subscribed) : undefined
}
function reminderCopy(type: string) { return reminderCopies[type] || { name: type, purpose: '接收与学习计划相关的微信提醒。' } }
function recordDate(value?: string) { if (!value) return '未知时间'; const date = new Date(value); return Number.isNaN(date.getTime()) ? value : `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }

onShow(load)
</script>

<style lang="scss">
.page{min-height:100vh;padding:28rpx;box-sizing:border-box;background:linear-gradient(180deg,#fff0f7,#fffaf0 48%,#f7fbff);color:#4b2b3f}.panel{padding:34rpx;border-radius:30rpx;background:#fff;border:1rpx solid #ffe0ea;box-shadow:0 14rpx 34rpx rgba(255,143,171,.09)}.eyebrow{color:#ff6f91;font-size:21rpx;font-weight:900}.title{margin-top:7rpx;font-size:36rpx;font-weight:900}.desc{margin:14rpx 0 26rpx;color:#7b8498;font-size:24rpx;line-height:1.65}.unsupported{padding:24rpx;border-radius:18rpx;background:#fff7e9;color:#6f5a3f}.loading,.empty{padding:24rpx 0;color:#8a92a6;font-size:24rpx;text-align:center}.reminder-list{display:flex;flex-direction:column;gap:16rpx}.reminder{padding:24rpx;border-radius:24rpx;background:linear-gradient(135deg,#fff8fb,#fffdf8);border:1rpx solid #ffe1ea}.reminder-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16rpx}.reminder-head>view:first-child{flex:1}.reminder-name{font-size:28rpx;font-weight:900}.reminder-purpose{margin-top:8rpx;color:#697286;font-size:22rpx;line-height:1.5}.state{flex:none;padding:7rpx 13rpx;border-radius:99rpx;background:#f0f2f5;color:#8a92a6;font-size:19rpx}.state.active{background:#e8f8ef;color:#278255}.record{margin-top:18rpx;padding:16rpx;border-radius:15rpx;background:#fff;color:#7b8498;font-size:20rpx;line-height:1.55}.authorize{margin:18rpx 0 0;border-radius:99rpx;background:#ff7aa2;color:#fff;font-size:24rpx;font-weight:800}.authorize[disabled]{opacity:.6}.cancel-all{margin:24rpx 0 0;background:transparent;color:#a06d7b;font-size:23rpx}.events{margin-top:22rpx;padding:30rpx;border-radius:28rpx;background:#fff;border:1rpx solid #ffe0ea}.events-title{font-size:30rpx;font-weight:900;margin-bottom:16rpx}.event{display:flex;align-items:center;justify-content:space-between;gap:18rpx;padding:18rpx 0;border-top:1rpx solid #f3e7eb}.event-type{color:#d85e82;font-size:25rpx;font-weight:800}.event-task{margin-top:6rpx;color:#606a80;font-size:23rpx}.event-date{color:#7b8498;font-size:22rpx}
</style>
