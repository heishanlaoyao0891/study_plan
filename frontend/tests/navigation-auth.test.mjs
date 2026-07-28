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
