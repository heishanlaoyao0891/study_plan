import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const source = async path => readFile(new URL(`../src/${path}`, import.meta.url), 'utf8')

test('plan utilities live only at the bottom of statistics', async () => {
  const [plans, stats] = await Promise.all([
    source('pages/plans/plans.vue'),
    source('pages/stats/stats.vue'),
  ])

  assert.doesNotMatch(plans, /账号与数据|设置与说明|goAccount|goOps/)
  const accountIndex = stats.indexOf('账号与数据')
  const settingsIndex = stats.indexOf('设置与说明')
  assert.ok(accountIndex >= 0, 'statistics must contain account and data')
  assert.ok(settingsIndex > accountIndex, 'settings must follow account and data')
  assert.match(stats, /\/pages\/account\/account/)
  assert.match(stats, /\/pages\/ops\/ops/)
})

test('H5 auth has one primary path and lower-emphasis alternatives', async () => {
  const login = await source('pages/login/login.vue')

  assert.doesNotMatch(login, /h5Mode === 'login'[^\n]*class="tab"/)
  assert.match(login, /class="inline-link forgot-link"[^>]*>忘记密码？</)
  assert.match(login, /class="inline-link register-link"[^>]*>使用邀请码注册</)
  assert.match(login, /class="return-login"/)
  assert.match(login, /\.h5-auth-panel \{ width: 420px;/)
})

test('mini program authenticates automatically and has no manual login landing action', async () => {
  const [app, login, auth] = await Promise.all([
    source('App.vue'),
    source('pages/login/login.vue'),
    source('utils/mp-auth.ts'),
  ])

  assert.match(app, /startMiniProgramAuth\(invitationFromLaunch\(options\)\)/)
  assert.doesNotMatch(login, />微信登录<\/button>/)
  assert.match(login, /miniProgramAuth\.status === 'setup-required'/)
  assert.match(login, /miniProgramAuth\.status === 'exchange-error'/)
  for (const state of ['loading', 'authenticated', 'setup-required', 'exchange-error', 'banned']) {
    assert.match(auth, new RegExp(`'${state}'`))
  }
  assert.doesNotMatch(auth, /setStorageSync/)
})

test('login page exposes administrator QR entry for invitation requests', async () => {
  const login = await source('pages/login/login.vue')

  assert.match(login, /没有邀请码？/)
  assert.match(login, /添加管理员获取/)
  assert.match(login, /const showInviteQr = ref\(false\)/)
  assert.match(login, /const inviteQrSrc = '\/static\/invite-qrcode\.png'/)
  assert.match(login, /class="qr-mask"/)
  assert.match(login, /uni\.previewImage\(\{ urls: \[inviteQrSrc\], current: inviteQrSrc \}\)/)
  assert.match(login, /\.invite-help \{/)
  assert.match(login, /\.qr-image \{/)
})

test('H5 landing pages expose the ICP filing link', async () => {
  const [footer, login, checkin] = await Promise.all([
    source('components/LegalFooter.vue'),
    source('pages/login/login.vue'),
    source('pages/checkin/checkin.vue'),
  ])

  assert.match(footer, /鄂ICP备2026038065号/)
  assert.match(footer, /https:\/\/beian\.miit\.gov\.cn\//)
  assert.match(footer, /const publicSecurityRecordNo = ''/)
  assert.match(login, /<LegalFooter \/>/)
  assert.match(checkin, /<LegalFooter \/>/)
})

test('account settings let users change login names within the monthly limit', async () => {
  const [account, api] = await Promise.all([
    source('pages/account/account.vue'),
    source('api/index.ts'),
  ])

  assert.match(account, /修改登录名/)
  assert.match(account, /每个自然月最多修改 3 次/)
  assert.match(account, /微信号样式的字母、数字、下划线，也可使用 11 位手机号/)
  assert.match(account, /AuthApi\.updateUsername\(loginUsername\.value\)/)
  assert.match(account, /本月还可修改 \$\{result\.remaining_changes\} 次/)
  assert.match(api, /updateUsername\(username: string\)/)
  assert.match(api, /\/api\/auth\/username/)
})

test('plan screens safely render legacy null schedule arrays', async () => {
  const [plans, detail] = await Promise.all([
    source('pages/plans/plans.vue'),
    source('pages/plan-detail/plan-detail.vue'),
  ])

  assert.match(plans, /function weekdaySummary\(selected:unknown\)\{const days=Array\.isArray\(selected\)\?selected:\[\]/)
  assert.match(detail, /function weekdaySummary\(selected: unknown\) \{ const days = Array\.isArray\(selected\) \? selected : \[\]/)
})

test('rearrangement preview differentiates conflicting and applicable tasks', async () => {
  const recovery = await source('pages/recovery/recovery.vue')

  assert.match(recovery, /'has-conflict': rowHasConflict\(row\), 'is-ready': rowIsReady\(row\)/)
  assert.match(recovery, /存在冲突，请调整/)
  assert.match(recovery, /可应用/)
  assert.match(recovery, /const conflictingTaskIDs = computed\(\(\) =>/)
  assert.match(recovery, /\.task\.is-ready\{background:#f2fff8/)
  assert.match(recovery, /\.task\.has-conflict\{background:#fff3f4/)
})

test('plan editing sheets preserve mini-program input focus and scroll long forms', async () => {
  const [plans, detail] = await Promise.all([
    source('pages/plans/plans.vue'),
    source('pages/plan-detail/plan-detail.vue'),
  ])
  const combined = `${plans}\n${detail}`

  assert.doesNotMatch(combined, /@click\.self/)
  assert.match(plans, /<scroll-view class="sheet-body" scroll-y @click\.stop>/)
  assert.match(detail, /<scroll-view class="sheet-body" scroll-y @click\.stop>/)
  assert.match(detail, /v-if="inviting" @click="inviting = false"/)
  assert.match(plans, /\.sheet-body\{width:100%;height:90vh/)
  assert.match(detail, /\.sheet-body\{width:100%;height:86vh/)
})

test('authenticated clients recover an existing server-side study timer without restarting it', async () => {
  const [api, routing, login, miniProgramAuth, app] = await Promise.all([
    source('api/index.ts'),
    source('utils/auth-routing.ts'),
    source('pages/login/login.vue'),
    source('utils/mp-auth.ts'),
    source('App.vue'),
  ])

  assert.match(api, /active\(\) \{\s*return api\.get<TimerTask \| null>\('\/api\/tasks\/active'\)/)
  assert.match(routing, /export async function routeForAuthenticatedUser/)
  assert.match(routing, /StudyTaskApi\.compensateMidnight\(\)/)
  assert.match(routing, /StudyTaskApi\.active\(\)/)
  assert.match(routing, /return activeTask \? `\/pages\/task\/task\?id=\$\{activeTask\.id\}` : fallback/)
  assert.doesNotMatch(routing, /StudyTaskApi\.(start|resume)\(/)
  assert.match(login, /await routeForAuthenticatedUser\(resp\.user, resp\.nickname_required\)/)
  assert.match(miniProgramAuth, /await routeForAuthenticatedUser\(resp\.user, resp\.nickname_required\)/)
  assert.match(app, /await routeForAuthenticatedUser\(user\)/)
})

test('task detail always exposes a task-list exit without mutating the timer', async () => {
  const task = await source('pages/task/task.vue')

  assert.match(task, /class="task-list" @click="goTaskList">任务列表<\/button>/)
  assert.match(task, /function goTaskList\(\)\{uni\.switchTab\(\{url:'\/pages\/checkin\/checkin'\}\)\}/)
  assert.match(task, /\.anchor \.task-list,\.anchor \.more\{flex:none;width:132rpx/)
  const taskListFunction = task.match(/function goTaskList\(\)\{[^}]+\}/)?.[0] || ''
  assert.doesNotMatch(taskListFunction, /StudyTaskApi\.(start|resume|pause|stop|complete)\(/)
})

test('completed-task overflow only edits reflection and switches sheets without overlap', async () => {
  const task = await source('pages/task/task.vue')

  assert.match(task, /v-if="canCorrectSchedule" @click="openCorrection\('postpone'\)">推迟任务<\/button>/)
  assert.match(task, /v-if="canCorrectSchedule" @click="openCorrection\('makeup'\)">补录学习<\/button>/)
  assert.match(task, /const canCorrectSchedule=computed\(\(\)=>detail\.value\?\.task\.timer_state==='pending'\|\|detail\.value\?\.task\.timer_state==='paused'\)/)
  assert.match(task, /v-if="detail\.task\.timer_state === 'completed'" @click="openReflection">编辑完成心得<\/button>/)
  assert.match(task, /showMore\.value=false;showCorrection\.value=false;await nextTick\(\);showReflection\.value=true/)
  assert.match(task, /:disabled="savingReflection" @click="saveReflection"/)
  assert.match(task, /if\(savingReflection\.value\)return/)
})

test('reminder settings represent WeChat authorization as one pending delivery', async () => {
  const notifications = await source('pages/notifications/notifications.vue')

  assert.match(notifications, /本次授权待使用/)
  assert.match(notifications, /成功发送一条此类提醒后会自动消耗/)
  assert.match(notifications, /补充下一次授权/)
  assert.doesNotMatch(notifications, /已留存授权/)
})

test('statistics chart remounts when the time buckets or dimension change', async () => {
  const [stats, chart] = await Promise.all([
    source('pages/stats/stats.vue'),
    source('components/InsightChart.vue'),
  ])

  assert.match(stats, /<InsightChart :key="chartRenderKey" :points="trend\.series" :dimension="dimension"/)
  assert.match(stats, /const chartRenderKey = computed\(\(\) =>/)
  assert.match(stats, /\$\{dimension\.value\}:\$\{trend\.value\?\.series\.map\(point => point\.key\)\.join\('\|'\) \|\| ''\}/)
  assert.match(chart, /:canvas-id="canvasId"/)
  assert.match(chart, /props\.dimension === 'time' \? 'line' : 'bar'/)
  assert.doesNotMatch(chart, /props\.dimension === 'time' \? 'mix' : 'bar'/)
})

test('completed task detail offers a user-triggered next-reminder authorization', async () => {
  const task = await source('pages/task/task.vue')

  assert.match(task, /为下一次学习提醒授权/)
  assert.match(task, /uni\.requestSubscribeMessage\(\{ tmplIds: \[template\.template_id\]/)
  assert.match(task, /NotificationApi\.subscribe\('study_start',template\.template_id/)
  assert.match(task, /&& canAuthorizeNextReminder"/)
})

test('AI client uses durable jobs without preview or commit state', async () => {
  const [api, ai, plans] = await Promise.all([
    source('api/index.ts'),
    source('pages/ai/ai.vue'),
    source('pages/plans/plans.vue'),
  ])
  const combined = `${api}\n${ai}\n${plans}`

  assert.match(api, /\/api\/ai\/plan-jobs/)
  assert.match(ai, /additional_instructions/)
  assert.match(ai, /overload_confirmation_required/)
  assert.match(ai, /job\.error_message/)
  assert.match(ai, /confirm_overload: confirmOverload/)
  assert.match(ai, /AI 调用成功/)
  assert.match(ai, /本次未使用 AI 结果/)
  assert.match(ai, /ai_decomposed/)
  assert.match(ai, /onHide\(stopPolling\)/)
  assert.match(plans, /currentPlanJob/)
  assert.doesNotMatch(combined, /ai_plan_pending_commit|PlanningPreview|commitPlan\(|regeneratePlan\(/)
})

test('AI page opens an already saved plan instead of presenting it as regeneration', async () => {
  const ai = await source('pages/ai/ai.vue')

  assert.match(ai, /const hasSavedPlan = computed\(/)
  assert.match(ai, /job\.value\?\.status === 'succeeded'/)
  assert.match(ai, /job\.value\.result_plan_id/)
  assert.match(ai, /if \(hasSavedPlan\.value\) return openResult\(\)/)
  assert.match(ai, /查看已保存计划/)
  assert.match(ai, /生成另一个计划/)
  assert.doesNotMatch(ai, /return job\.value \? '重新生成计划'/)
})
