<template>
  <view class="page">
    <view class="shell">
      <view class="intro">
        <view class="topbar">
          <view class="brand-mark">学</view>
          <view>
            <view class="brand-name">学习花园</view>
            <view class="brand-sub">把每天的小努力养成会发光的习惯</view>
          </view>
        </view>
        <view class="hero-kicker">今日份学习魔法</view>
        <view class="hero-title">把大目标养成每天会发芽的小任务</view>
        <view class="hero-copy">计划、打卡、AI 拆解、小组陪伴和甜甜休息券，都放在一个轻盈的小花园里。</view>
        <view class="feature-row">
          <text>轻盈计划</text><text>专注打卡</text><text>伙伴陪伴</text>
        </view>
      </view>

      <view class="panel auth-panel">
        <!-- #ifdef H5 -->
        <view class="auth-heading">
          <view class="section-title">{{ h5Title }}</view>
          <view class="section-copy">{{ h5Copy }}</view>
        </view>

        <view class="tabs">
          <view class="tab" :class="{ active: h5Mode === 'login' }" @click="switchH5Mode('login')">登录</view>
          <view class="tab" :class="{ active: h5Mode === 'register' }" @click="switchH5Mode('register')">注册</view>
        </view>

        <view class="form">
          <view class="field">
            <text class="field-label">用户名</text>
            <input v-model="h5Form.username" class="field-input" maxlength="24" placeholder="4-24 位字母、数字或下划线" @input="clearError" />
          </view>
          <view v-if="h5Mode === 'register'" class="field">
            <text class="field-label">邀请码</text>
            <input v-model="h5Form.invite_code" class="field-input" maxlength="64" placeholder="请输入管理员提供的邀请码" @input="clearError" />
          </view>
          <view v-if="h5Mode === 'register'" class="field">
            <text class="field-label">昵称</text>
            <input v-model="h5Form.nickname" class="field-input" maxlength="20" placeholder="2-20 个字符" @input="clearError" />
          </view>
          <view class="field">
            <text class="field-label">密码</text>
            <input v-model="h5Form.password" class="field-input" password maxlength="64" :placeholder="h5Mode === 'login' ? '请输入密码' : '至少 8 位字符'" @input="clearError" />
          </view>
          <view v-if="errMsg" class="error">{{ errMsg }}</view>
          <button class="primary-btn" :loading="submitting" :disabled="submitting" @click="submitH5">{{ h5SubmitText }}</button>
        </view>
        <!-- #endif -->

        <!-- #ifdef MP-WEIXIN -->
        <view v-if="wechatStep === 'login'" class="wechat-entry">
          <view class="auth-heading">
            <view class="section-title">微信快捷登录</view>
            <view class="section-copy">轻轻一点，回到你的学习花园</view>
          </view>
          <view class="wechat-orb">微</view>
          <button class="primary-btn wechat-btn" :loading="submitting" :disabled="submitting" @click="onWechatLogin">微信登录</button>
          <view class="privacy-note">登录即表示你同意相关服务与隐私规则</view>
        </view>

        <view v-else class="registration-form">
          <view class="auth-heading">
            <view class="section-title">第一次见面</view>
            <view class="section-copy">{{ wechatMode === 'register' ? '用邀请码创建账号，下次微信登录会直接回到这里' : '绑定你已经在 H5 创建的账号' }}</view>
          </view>
          <view class="tabs">
            <view class="tab" :class="{ active: wechatMode === 'register' }" @click="switchWechatMode('register')">创建账号</view>
            <view class="tab" :class="{ active: wechatMode === 'link' }" @click="switchWechatMode('link')">已有账号</view>
          </view>
          <view v-if="wechatMode === 'register'" class="field">
            <text class="field-label">邀请码</text>
            <input v-model="wechatForm.invite_code" class="field-input" maxlength="64" placeholder="请输入管理员提供的邀请码" @input="clearError" />
          </view>
          <view class="field">
            <text class="field-label">用户名</text>
            <input v-model="wechatForm.username" class="field-input" maxlength="24" placeholder="4-24 位字母、数字或下划线" @input="clearError" />
          </view>
          <view v-if="wechatMode === 'register'" class="field">
            <text class="field-label">昵称</text>
            <input v-model="wechatForm.nickname" class="field-input" maxlength="20" placeholder="给学习中的自己取个名字" @input="clearError" />
          </view>
          <view class="field">
            <text class="field-label">密码</text>
            <input v-model="wechatForm.password" class="field-input" password maxlength="64" placeholder="至少 8 位字符" @input="clearError" />
          </view>
          <view v-if="errMsg" class="error">{{ errMsg }}</view>
          <button class="primary-btn" :loading="submitting" :disabled="submitting" @click="submitWechatAccount">{{ wechatMode === 'register' ? '注册并开始学习' : '绑定并开始学习' }}</button>
          <button class="text-btn" @click="cancelWechatRegistration">换一个微信账号</button>
        </view>
        <!-- #endif -->

        <button v-if="isDev" class="dev-toggle" @click="showDev = !showDev">{{ showDev ? '收起调试' : '本地调试' }}</button>
      </view>

      <view v-if="showDev" class="panel dev">
        <view class="section-title small">本地调试</view>
        <view class="field">
          <text class="field-label">API 地址</text>
          <input v-model="apiBase" class="field-input" placeholder="留空时 H5 使用当前域名" />
        </view>
        <view class="field">
          <text class="field-label">Mock Code</text>
          <input v-model="mockCode" class="field-input" placeholder="test_user" />
        </view>
        <view class="dev-actions">
          <button class="ghost-btn" @click="saveApiBase">保存地址</button>
          <button class="dark-btn" @click="mockLogin">Mock 登录</button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { AuthApi, type LoginResp, type RegistrationReq, type WechatLoginResp } from '@/api'
import { getApiBase, setApiBase, setToken } from '@/api/request'
import { normalizeDisplayText, unicodeLength } from '@/utils/text'

type H5Mode = 'login' | 'register'

const errMsg = ref('')
const submitting = ref(false)
const showDev = ref(false)
const isDev = import.meta.env.VITE_ENABLE_DEV_LOGIN === 'true'
const apiBase = ref(getApiBase())
const mockCode = ref('test_user_' + Math.floor(Math.random() * 10000))
const h5Mode = ref<H5Mode>('login')
const h5Form = reactive<RegistrationReq>({ invite_code: '', username: '', nickname: '', password: '' })
const wechatStep = ref<'login' | 'register'>('login')
const registrationToken = ref('')
const wechatMode = ref<'register' | 'link'>('register')
const wechatForm = reactive<RegistrationReq>({ invite_code: '', username: '', nickname: '', password: '' })

const h5Title = computed(() => ({ login: '欢迎回来', register: '种下第一颗种子' })[h5Mode.value])
const h5Copy = computed(() => ({ login: '用用户名和密码继续今天的学习', register: '凭邀请码创建你的专属学习花园' })[h5Mode.value])
const h5SubmitText = computed(() => ({ login: '进入学习花园', register: '注册并开始学习' })[h5Mode.value])

function clearError() {
  errMsg.value = ''
}

function validNickname(value: string) {
  const length = unicodeLength(normalizeDisplayText(value))
  return length >= 2 && length <= 20
}

function switchH5Mode(mode: H5Mode) {
  h5Mode.value = mode
  clearError()
  h5Form.password = ''
}

function validateAccount(form: RegistrationReq, registering: boolean) {
  if (registering && !/^[A-Za-z0-9_]{4,24}$/.test(form.username.trim())) return '用户名需为 4-24 位字母、数字或下划线'
  if (!registering && !form.username.trim()) return '请输入用户名'
  if (registering && !form.invite_code.trim()) return '请输入邀请码'
  if (registering && !validNickname(form.nickname)) return '昵称长度需要为 2-20 个字符'
  if (form.password.length < 8) return '密码至少需要 8 位字符'
  return ''
}

async function submitH5() {
  clearError()
  const validation = validateAccount(h5Form, h5Mode.value === 'register')
  if (validation) {
    errMsg.value = validation
    return
  }
  submitting.value = true
  try {
    if (h5Mode.value === 'login') {
      afterLogin(await AuthApi.h5Login(h5Form.username.trim(), h5Form.password))
    } else {
      afterLogin(await AuthApi.h5Register({
        invite_code: h5Form.invite_code.trim(),
        username: h5Form.username.trim(),
        nickname: normalizeDisplayText(h5Form.nickname),
        password: h5Form.password,
      }))
    }
  } catch (error: any) {
    errMsg.value = error?.message || '操作失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}

function isRegistrationRequired(resp: WechatLoginResp): resp is Extract<WechatLoginResp, { registration_required: true }> {
  return 'registration_required' in resp && resp.registration_required === true
}

function handleWechatResult(resp: WechatLoginResp) {
  if (!isRegistrationRequired(resp)) {
    afterLogin(resp)
    return
  }
  registrationToken.value = resp.registration_token
  wechatStep.value = 'register'
}

async function onWechatLogin() {
  clearError()
  submitting.value = true
  try {
    // #ifdef MP-WEIXIN
    const loginRes: any = await new Promise((resolve, reject) => uni.login({ provider: 'weixin', success: resolve, fail: reject }))
    if (!loginRes?.code) throw new Error('未能获取微信登录凭证，请重试')
    handleWechatResult(await AuthApi.login(loginRes.code))
    // #endif
  } catch (error: any) {
    errMsg.value = error?.message || '微信登录失败'
  } finally {
    submitting.value = false
  }
}

async function submitWechatRegistration() {
  clearError()
  const validation = validateAccount(wechatForm, true)
  if (validation) {
    errMsg.value = validation
    return
  }
  submitting.value = true
  try {
    afterLogin(await AuthApi.wechatRegister({
      registration_token: registrationToken.value,
      invite_code: wechatForm.invite_code.trim(),
      username: wechatForm.username.trim(),
      nickname: normalizeDisplayText(wechatForm.nickname),
      password: wechatForm.password,
    }))
  } catch (error: any) {
    errMsg.value = error?.message || '注册失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}

function switchWechatMode(mode: 'register' | 'link') {
  wechatMode.value = mode
  clearError()
}

async function submitWechatAccount() {
  if (wechatMode.value === 'register') {
    await submitWechatRegistration()
    return
  }
  clearError()
  const validation = validateAccount(wechatForm, false)
  if (validation) {
    errMsg.value = validation
    return
  }
  submitting.value = true
  try {
    afterLogin(await AuthApi.wechatLink({
      registration_token: registrationToken.value,
      username: wechatForm.username.trim(),
      password: wechatForm.password,
    }))
  } catch (error: any) {
    errMsg.value = error?.message || '绑定失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}

function cancelWechatRegistration() {
  wechatStep.value = 'login'
  registrationToken.value = ''
  clearError()
}

function afterLogin(resp: LoginResp) {
  setToken(resp.token)
  uni.showToast({ title: '登录成功', icon: 'success' })
  const onboardingDone = !!uni.getStorageSync('onboarding_completed')
  const url = resp.nickname_required ? '/pages/nickname/nickname' : (onboardingDone ? '/pages/checkin/checkin' : '/pages/onboarding/onboarding')
  setTimeout(() => uni.reLaunch({ url }), 300)
}

function saveApiBase() {
  setApiBase(apiBase.value.trim())
  uni.showToast({ title: '已保存', icon: 'success' })
}

async function mockLogin() {
  clearError()
  submitting.value = true
  try {
    setApiBase(apiBase.value.trim())
    handleWechatResult(await AuthApi.login(mockCode.value.trim()))
  } catch (error: any) {
    errMsg.value = error?.message || 'Mock 登录失败'
  } finally {
    submitting.value = false
  }
}
</script>

<style lang="scss">
.page { min-height: 100vh; box-sizing: border-box; padding: 44rpx 30rpx 64rpx; background: radial-gradient(circle at 85% 8%, rgba(255,195,106,.28), transparent 26%), linear-gradient(155deg, #fff0f7 0%, #fffaf0 52%, #f3faff 100%); }
.shell { width: 100%; max-width: 1080px; margin: 0 auto; }
.intro { padding: 20rpx 4rpx 34rpx; }
.topbar { display: flex; align-items: center; gap: 20rpx; margin-bottom: 58rpx; }
.brand-mark { display: flex; width: 72rpx; height: 72rpx; align-items: center; justify-content: center; border-radius: 24rpx; background: linear-gradient(135deg, #ff8fab, #ffc36a); color: #fff; font-size: 34rpx; font-weight: 800; box-shadow: 0 12rpx 26rpx rgba(255,122,162,.24); }
.brand-name { color: #2f2330; font-size: 34rpx; font-weight: 800; }
.brand-sub { margin-top: 4rpx; color: #7b7180; font-size: 23rpx; }
.hero-kicker { color: #ff6f91; font-size: 22rpx; font-weight: 800; letter-spacing: 2rpx; }
.hero-title { max-width: 720rpx; margin-top: 18rpx; color: #2b2130; font-size: 48rpx; line-height: 1.22; font-weight: 900; }
.hero-copy { max-width: 760rpx; margin-top: 22rpx; color: #655d6e; font-size: 27rpx; line-height: 1.7; }
.feature-row { display: flex; flex-wrap: wrap; gap: 14rpx; margin-top: 28rpx; }
.feature-row text { padding: 10rpx 18rpx; border: 1rpx solid rgba(255,143,171,.28); border-radius: 999rpx; background: rgba(255,255,255,.58); color: #8c5369; font-size: 22rpx; }
.panel { box-sizing: border-box; background: rgba(255,255,255,.94); border: 1rpx solid #f1dfe5; border-radius: 30rpx; box-shadow: 0 20rpx 52rpx rgba(134,74,96,.13); }
.auth-panel { padding: 38rpx 34rpx 30rpx; }
.auth-heading { margin-bottom: 28rpx; }
.section-title { color: #2f2330; font-size: 36rpx; font-weight: 900; }
.section-title.small { font-size: 30rpx; }
.section-copy { margin-top: 10rpx; color: #867b89; font-size: 24rpx; line-height: 1.5; }
.tabs { display: grid; grid-template-columns: 1fr 1fr; padding: 6rpx; margin-bottom: 28rpx; border-radius: 18rpx; background: #fff3f6; }
.tab { padding: 18rpx; border-radius: 14rpx; color: #9b7180; font-size: 26rpx; text-align: center; font-weight: 700; }
.tab.active { background: #fff; color: #f15f86; box-shadow: 0 6rpx 16rpx rgba(216,105,137,.12); }
.field { margin-bottom: 22rpx; }
.field-label { display: block; margin-bottom: 10rpx; color: #5f4855; font-size: 24rpx; font-weight: 700; }
.field-input { box-sizing: border-box; width: 100%; height: 86rpx; padding: 0 22rpx; border: 2rpx solid #f0dce3; border-radius: 18rpx; background: #fffbfc; color: #2f2330; font-size: 28rpx; }
.field-input:focus { border-color: #ff9ab4; background: #fff; }
button { margin: 0; }
button::after { border: 0; }
.primary-btn { width: 100%; height: 90rpx; margin-top: 12rpx; line-height: 90rpx; border-radius: 999rpx; background: linear-gradient(135deg, #ff739c, #ffb45c); color: #fff; font-size: 29rpx; font-weight: 800; box-shadow: 0 14rpx 28rpx rgba(255,122,162,.23); }
.primary-btn[disabled] { opacity: .68; }
.text-btn, .dev-toggle { margin: 18rpx auto 0; padding: 10rpx 20rpx; background: transparent; color: #a66a7e; font-size: 24rpx; line-height: 1.5; }
.error { margin: 0 0 16rpx; padding: 18rpx 20rpx; border-radius: 14rpx; background: #fff0f1; color: #c94058; font-size: 24rpx; line-height: 1.5; }
.wechat-entry { text-align: center; }
.wechat-entry .auth-heading { text-align: left; }
.wechat-orb { display: flex; width: 132rpx; height: 132rpx; margin: 52rpx auto; align-items: center; justify-content: center; border-radius: 42rpx; background: linear-gradient(145deg, #6ddc91, #39b86a); color: #fff; font-size: 45rpx; font-weight: 900; box-shadow: 0 18rpx 38rpx rgba(57,184,106,.25); transform: rotate(-4deg); }
.wechat-btn { background: linear-gradient(135deg, #42c975, #76dc94); box-shadow: 0 14rpx 28rpx rgba(57,184,106,.2); }
.privacy-note { margin-top: 20rpx; color: #a097a1; font-size: 21rpx; }
.dev { margin-top: 24rpx; padding: 30rpx; }
.dev-actions { display: grid; grid-template-columns: 1fr 1fr; gap: 18rpx; }
.ghost-btn, .dark-btn { height: 82rpx; line-height: 82rpx; border-radius: 16rpx; font-size: 26rpx; }
.ghost-btn { background: #f5eff2; color: #6c5963; }
.dark-btn { background: #392f38; color: #fff; }

@media (min-width: 800px) {
  .page { display: flex; align-items: center; padding: 48px; }
  .shell { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(390px, .9fr); align-items: center; gap: 72px; }
  .intro { padding: 20px 0; }
  .topbar { margin-bottom: 70px; }
  .hero-title { font-size: 58px; }
  .hero-copy { font-size: 18px; }
  .auth-panel { padding: 36px; border-radius: 28px; }
  .dev { grid-column: 2; margin-top: -48px; }
}
</style>
