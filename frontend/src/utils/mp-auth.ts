import { reactive } from 'vue'
import { AuthApi, type WechatLoginResp } from '@/api'
import { setToken } from '@/api/request'
import { routeForAuthenticatedUser } from '@/utils/auth-routing'
import { getBanState } from '@/utils/ban-state'

export type MiniProgramAuthStatus = 'idle' | 'loading' | 'authenticated' | 'setup-required' | 'exchange-error' | 'banned'

export const miniProgramAuth = reactive({
  status: 'idle' as MiniProgramAuthStatus,
  error: '',
  registrationToken: '',
  launchInvitation: '',
})

function cleanInvitation(value: unknown): string {
  if (typeof value !== 'string') return ''
  try {
    return decodeURIComponent(value.replace(/\+/g, ' ')).trim().slice(0, 64)
  } catch {
    return value.trim().slice(0, 64)
  }
}

export function invitationFromLaunch(options: any): string {
  const query = options?.query || {}
  const direct = cleanInvitation(query.invite_code || query.invitation || query.invite)
  if (direct) return direct

  // WeChat's top-level scene is a numeric launch source; query.scene carries QR parameters.
  const scene = cleanInvitation(query.scene)
  if (!scene) return ''
  const params = scene.split('&')
  for (const part of params) {
    const separator = part.indexOf('=')
    if (separator < 0) continue
    const key = part.slice(0, separator)
    if (key === 'invite_code' || key === 'invitation' || key === 'invite') {
      return cleanInvitation(part.slice(separator + 1))
    }
  }
  return scene.includes('=') ? '' : scene
}

function registrationRequired(resp: WechatLoginResp): resp is Extract<WechatLoginResp, { registration_required: true }> {
  return 'registration_required' in resp && resp.registration_required === true
}

function wechatCode(): Promise<string> {
  return new Promise((resolve, reject) => {
    uni.login({
      provider: 'weixin',
      success: (result: any) => result?.code ? resolve(result.code) : reject(new Error('未能获取微信登录凭证，请重试')),
      fail: reject,
    })
  })
}

export async function startMiniProgramAuth(invitation?: string) {
  if (miniProgramAuth.status === 'loading') return
  if (invitation !== undefined) miniProgramAuth.launchInvitation = cleanInvitation(invitation)
  miniProgramAuth.status = 'loading'
  miniProgramAuth.error = ''
  miniProgramAuth.registrationToken = ''

  try {
    const resp = await AuthApi.login(await wechatCode())
    if (registrationRequired(resp)) {
      miniProgramAuth.registrationToken = resp.registration_token
      miniProgramAuth.status = 'setup-required'
      return
    }

    setToken(resp.token)
    miniProgramAuth.launchInvitation = ''
    miniProgramAuth.status = 'authenticated'
    const route = await routeForAuthenticatedUser(resp.user, resp.nickname_required)
    uni.reLaunch({ url: route })
  } catch (error: any) {
    if (getBanState()) {
      miniProgramAuth.status = 'banned'
      return
    }
    miniProgramAuth.status = 'exchange-error'
    miniProgramAuth.error = error?.message || '微信登录失败，请稍后重试'
  }
}

export function clearMiniProgramSetup() {
  miniProgramAuth.registrationToken = ''
  miniProgramAuth.launchInvitation = ''
}

export function resetMiniProgramAuth() {
  clearMiniProgramSetup()
  miniProgramAuth.status = 'idle'
  miniProgramAuth.error = ''
}
