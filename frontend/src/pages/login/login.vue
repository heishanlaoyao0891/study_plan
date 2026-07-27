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

      <view class="panel auth-panel h5-auth-panel">
        <!-- #ifdef H5 -->
        <view class="auth-heading">
          <view class="section-title">{{ h5Title }}</view>
          <view class="section-copy">{{ h5Copy }}</view>
        </view>

        <view class="form h5-form">
          <view v-if="h5Mode === 'register'" class="field">
            <text class="field-label">邀请码</text>
            <input v-model="h5Form.invite_code" class="field-input" maxlength="64" placeholder="请输入管理员提供的邀请码" @input="clearError" />
          </view>
          <view class="field">
            <text class="field-label">用户名</text>
            <input v-model="h5Form.username" class="field-input" maxlength="24" placeholder="4-24 位字母、数字或下划线" @input="clearError" />
          </view>
          <view v-if="h5Mode === 'reset'" class="field">
            <text class="field-label">管理员重置码</text>
            <input v-model="resetCode" class="field-input" maxlength="64" placeholder="输入 30 分钟内有效的重置码" @input="clearError" />
          </view>
          <view v-if="h5Mode === 'register'" class="field">
            <text class="field-label">昵称</text>
            <input v-model="h5Form.nickname" class="field-input" maxlength="20" placeholder="2-20 个字符" @input="clearError" />
          </view>
          <view class="field">
            <view class="field-label-row">
              <text class="field-label">{{ h5Mode === 'reset' ? '新密码' : '密码' }}</text>
              <button v-if="h5Mode === 'login'" class="inline-link forgot-link" type="button" @click="switchH5Mode('reset')">忘记密码？</button>
            </view>
            <input v-model="h5Form.password" class="field-input" password maxlength="72" :placeholder="h5Mode === 'login' ? '请输入密码' : '至少 8 位字符'" @input="clearError" />
          </view>
          <view v-if="errMsg" class="error">{{ errMsg }}</view>
          <button class="primary-btn" :loading="submitting" :disabled="submitting" @click="submitH5">{{ h5SubmitText }}</button>
          <view v-if="h5Mode === 'login'" class="auth-switch">还没有账号？<button class="inline-link register-link" type="button" @click="switchH5Mode('register')">使用邀请码注册</button></view>
          <button v-else class="return-login" type="button" @click="switchH5Mode('login')">{{ h5Mode === 'register' ? '已有账号？返回登录' : '返回登录' }}</button>
        </view>
        <!-- #endif -->

        <!-- #ifdef MP-WEIXIN -->
        <view v-if="miniProgramAuth.status === 'loading' || miniProgramAuth.status === 'idle'" class="wechat-entry">
          <view class="auth-heading">
            <view class="section-title">正在进入学习花园</view>
            <view class="section-copy">正在安全验证微信身份，请稍候</view>
          </view>
          <view class="wechat-orb auth-loading">微</view>
        </view>

        <view v-else-if="miniProgramAuth.status === 'exchange-error'" class="wechat-entry">
          <view class="auth-heading">
            <view class="section-title">暂时无法登录</view>
            <view class="section-copy">微信身份验证没有完成，可以重新尝试</view>
          </view>
          <view class="error">{{ miniProgramAuth.error }}</view>
          <button class="primary-btn wechat-btn" @click="retryWechatLogin">重试</button>
        </view>

        <view v-else-if="miniProgramAuth.status === 'setup-required'" class="registration-form">
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
            <input v-model="wechatForm.password" class="field-input" password maxlength="72" placeholder="至少 8 位字符" @input="clearError" />
          </view>
          <view v-if="errMsg" class="error">{{ errMsg }}</view>
          <button class="primary-btn" :loading="submitting" :disabled="submitting" @click="submitWechatAccount">{{ wechatMode === 'register' ? '注册并开始学习' : '绑定并开始学习' }}</button>
        </view>

        <view v-else-if="miniProgramAuth.status === 'banned'" class="wechat-entry">
          <view class="section-copy">账号访问已暂停，正在前往状态页面</view>
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
import { onLoad } from '@dcloudio/uni-app'
import { computed, reactive, ref, watch } from 'vue'
import { AuthApi, type LoginResp, type RegistrationReq, type WechatLoginResp } from '@/api'
import { getApiBase, setApiBase, setToken } from '@/api/request'
import { normalizeDisplayText, unicodeLength } from '@/utils/text'
import { routeForUser } from '@/utils/auth-routing'
import { clearMiniProgramSetup, miniProgramAuth, resetMiniProgramAuth, startMiniProgramAuth } from '@/utils/mp-auth'

type H5Mode = 'login' | 'register' | 'reset'

const errMsg = ref('')
const submitting = ref(false)
const showDev = ref(false)
const isDev = import.meta.env.VITE_ENABLE_DEV_LOGIN === 'true'
const apiBase = ref(getApiBase())
const mockCode = ref('test_user_' + Math.floor(Math.random() * 10000))
const h5Mode = ref<H5Mode>('login')
const resetCode = ref('')
const h5Form = reactive<RegistrationReq>({ invite_code: '', username: '', nickname: '', password: '' })
const wechatMode = ref<'register' | 'link'>('register')
const wechatForm = reactive<RegistrationReq>({ invite_code: '', username: '', nickname: '', password: '' })

const h5Title = computed(() => ({ login: '欢迎回来', register: '种下第一颗种子', reset: '重置密码' })[h5Mode.value])
const h5Copy = computed(() => ({ login: '用用户名和密码继续今天的学习', register: '凭邀请码创建你的专属学习花园', reset: '使用管理员提供的一次性重置码' })[h5Mode.value])
const h5SubmitText = computed(() => ({ login: '登录', register: '注册并开始学习', reset: '重置密码' })[h5Mode.value])

onLoad(() => {
  // #ifdef MP-WEIXIN
  if (miniProgramAuth.status === 'authenticated') resetMiniProgramAuth()
  if (miniProgramAuth.status === 'idle') startMiniProgramAuth()
  // #endif
})

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
    if (h5Mode.value === 'reset') {
      if (!resetCode.value.trim()) throw new Error('请输入重置码')
      await AuthApi.resetPassword(h5Form.username.trim(), resetCode.value.trim(), h5Form.password)
      uni.showToast({ title: '密码已重置', icon: 'success' })
      switchH5Mode('login')
    } else if (h5Mode.value === 'login') {
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
  miniProgramAuth.registrationToken = resp.registration_token
  miniProgramAuth.status = 'setup-required'
}

function retryWechatLogin() {
  startMiniProgramAuth()
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
      registration_token: miniProgramAuth.registrationToken,
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
       registration_token: miniProgramAuth.registrationToken,
      username: wechatForm.username.trim(),
      password: wechatForm.password,
    }))
  } catch (error: any) {
    errMsg.value = error?.message || '绑定失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}

function afterLogin(resp: LoginResp) {
  setToken(resp.token)
  clearMiniProgramSetup()
  uni.showToast({ title: '登录成功', icon: 'success' })
  const url = routeForUser(resp.user, resp.nickname_required)
  setTimeout(() => uni.reLaunch({ url }), 300)
}

watch(() => miniProgramAuth.status, (status) => {
  if (status === 'setup-required' && miniProgramAuth.launchInvitation && !wechatForm.invite_code) {
    wechatForm.invite_code = miniProgramAuth.launchInvitation
  }
}, { immediate: true })

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

/* #ifdef H5 */
.h5-auth-panel { border-radius: 18rpx; box-shadow: 0 12rpx 32rpx rgba(76,52,64,.1); }
.h5-form .field-label-row { display: flex; min-height: 34rpx; align-items: center; justify-content: space-between; gap: 20rpx; margin-bottom: 10rpx; }
.h5-form .field-label-row .field-label { margin-bottom: 0; }
.inline-link, .return-login { min-width: 0; height: auto; padding: 0; border: 0; border-radius: 0; background: transparent; font-size: 23rpx; line-height: 1.5; }
.inline-link { display: inline; margin: 0; color: #8b7d85; }
.forgot-link { flex: none; }
.auth-switch { display: flex; flex-wrap: wrap; align-items: baseline; justify-content: center; gap: 6rpx; margin-top: 24rpx; color: #827780; font-size: 23rpx; line-height: 1.6; text-align: center; }
.register-link { color: #e95f85; font-weight: 700; }
.return-login { margin: 22rpx auto 0; color: #8b7d85; }
.h5-form .primary-btn { height: 86rpx; line-height: 86rpx; border-radius: 14rpx; box-shadow: 0 8rpx 18rpx rgba(255,122,162,.18); }
/* #endif */

@media (min-width: 800px) {
  .page { display: flex; align-items: center; padding: 48px; }
  .shell { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(390px, .9fr); align-items: center; gap: 72px; }
  .intro { padding: 20px 0; }
  .topbar { margin-bottom: 70px; }
  .hero-title { font-size: 58px; }
  .hero-copy { font-size: 18px; }
  .auth-panel { padding: 36px; border-radius: 28px; }
  /* #ifdef H5 */
  .shell { grid-template-columns: minmax(0, 1fr) 420px; }
  .h5-auth-panel { width: 420px; padding: 34px; border-radius: 10px; }
  .h5-auth-panel .field-input, .h5-auth-panel .primary-btn { height: 44px; line-height: 44px; }
  /* #endif */
  .dev { grid-column: 2; margin-top: -48px; }
}
</style>
