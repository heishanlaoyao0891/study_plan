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
