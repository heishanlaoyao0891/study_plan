export interface BanState {
  account_banned: true
  reason: string
  banned_until: string
  permanent: boolean
  server_now: string
  received_at: number
  token_retained: boolean
}

const BAN_STATE_KEY = 'account_ban_state'
const BANNED_ROUTE = '/pages/banned/banned'
const MAX_DATE_LENGTH = 40
const MIN_BAN_YEAR = 2020
const MAX_BAN_YEAR = 2100
let routing = false

function asRecord(value: unknown): Record<string, any> | null {
  return value && typeof value === 'object' ? value as Record<string, any> : null
}

export function parseBanPayload(body: unknown, tokenRetained: boolean): BanState | null {
  const envelope = asRecord(body)
  const nested = asRecord(envelope?.data)
  const candidate = nested?.account_banned === true ? nested : envelope
  if (!candidate || candidate.account_banned !== true) return null
  const bannedUntil = String(candidate.banned_until || '')
  const serverNow = String(candidate.server_now || '')
  if (bannedUntil.length > MAX_DATE_LENGTH || serverNow.length > MAX_DATE_LENGTH) return null
  const bannedUntilMs = Date.parse(bannedUntil)
  const serverNowMs = Date.parse(serverNow)
  if (!validServerDate(bannedUntilMs) || !validServerDate(serverNowMs)) return null
  return {
    account_banned: true,
    reason: typeof candidate.reason === 'string' ? candidate.reason.trim().slice(0, 256) : '',
    banned_until: bannedUntil,
    permanent: candidate.permanent === true,
    server_now: serverNow,
    received_at: Date.now(),
    token_retained: tokenRetained,
  }
}

function validServerDate(timestamp: number): boolean {
  if (!Number.isFinite(timestamp)) return false
  const year = new Date(timestamp).getUTCFullYear()
  return year >= MIN_BAN_YEAR && year <= MAX_BAN_YEAR
}

export function getBanState(): BanState | null {
  const value = asRecord(uni.getStorageSync(BAN_STATE_KEY))
  const parsed = value ? parseBanPayload(value, value.token_retained === true) : null
  if (!parsed) {
    uni.removeStorageSync(BAN_STATE_KEY)
    return null
  }
  parsed.received_at = Number.isFinite(value!.received_at) ? Number(value!.received_at) : Date.now()
  return parsed
}

export function clearBanState() {
  uni.removeStorageSync(BAN_STATE_KEY)
}

export function serverTime(state: BanState): number {
  return Date.parse(state.server_now) + Math.max(0, Date.now() - state.received_at)
}

function currentRoute(): string {
  const pages = getCurrentPages()
  const route = pages.length ? `/${pages[pages.length - 1].route || ''}` : ''
  return route
}

export function retainBanAndRoute(body: unknown, tokenRetained: boolean): BanState | null {
  const state = parseBanPayload(body, tokenRetained)
  if (!state) return null
  uni.setStorageSync(BAN_STATE_KEY, state)
  if (currentRoute() === BANNED_ROUTE || routing) return state
  routing = true
  uni.reLaunch({
    url: BANNED_ROUTE,
    complete: () => { routing = false },
  })
  return state
}
