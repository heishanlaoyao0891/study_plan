// API 层封装：统一的请求方法 + token 管理
// 使用 uni.request 发起请求，自动带上 Authorization header

const API_BASE_KEY = 'api_base'
const TOKEN_KEY = 'auth_token'

// 默认本地开发地址：H5 端用 localhost，小程序需要改成你电脑的 LAN IP
// 例如：http://192.168.1.100:8080
export function getApiBase(): string {
	return uni.getStorageSync(API_BASE_KEY) || import.meta.env.VITE_API_BASE || 'http://localhost:8080'
}

export function setApiBase(base: string) {
  uni.setStorageSync(API_BASE_KEY, base)
}

export function getToken(): string {
  return uni.getStorageSync(TOKEN_KEY) || ''
}

export function setToken(token: string) {
  uni.setStorageSync(TOKEN_KEY, token)
}

export function clearToken() {
  uni.removeStorageSync(TOKEN_KEY)
}

// 是否已登录
export function isLoggedIn(): boolean {
  return !!getToken()
}

// 统一响应：{ code, data, message, warnings? }
interface ApiResp<T = any> {
  code: number
  data?: T
  message?: string
  warnings?: string[]
}

export interface ApiError {
  code: number
  message: string
  raw?: any
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  data?: any
  auth?: boolean // 是否需要带 token（默认 true）
}

export async function request<T = any>(path: string, options: RequestOptions = {}): Promise<T> {
  const method = options.method || 'GET'
  const needAuth = options.auth !== false
  const url = getApiBase() + path
  const header: Record<string, string> = { 'Content-Type': 'application/json' }
  if (needAuth) {
    const token = getToken()
    if (!token) {
      // 未登录：跳转登录页
      uni.reLaunch({ url: '/pages/login/login' })
      return Promise.reject({ code: 401, message: '未登录' } as ApiError)
    }
    header['Authorization'] = 'Bearer ' + token
  }

  return new Promise<T>((resolve, reject) => {
    uni.request({
      url,
      method,
      data: options.data,
      header,
      success: (res: any) => {
        const status = res.statusCode || 200
        if (status === 401) {
          clearToken()
          uni.reLaunch({ url: '/pages/login/login' })
          return reject({ code: 401, message: '登录已过期，请重新登录' } as ApiError)
        }
        if (status === 403) {
          const data = res.data || {}
          return reject({ code: 403, message: data.message || '无权访问', raw: data } as ApiError)
        }
        if (status >= 400) {
          const data = res.data || {}
          return reject({ code: status, message: data.message || '请求失败' } as ApiError)
        }
        const body: ApiResp<T> = res.data || {}
        if (body.code !== 0) {
          return reject({ code: body.code, message: body.message || '业务错误' } as ApiError)
        }
        resolve(body.data as T)
      },
      fail: (err) => {
        reject({ code: -1, message: '网络错误：' + (err.errMsg || '') } as ApiError)
      },
    })
  })
}

// 便利方法
export const api = {
  get: <T = any>(path: string, opts?: RequestOptions) => request<T>(path, { ...opts, method: 'GET' }),
  post: <T = any>(path: string, data?: any, opts?: RequestOptions) => request<T>(path, { ...opts, method: 'POST', data }),
  put: <T = any>(path: string, data?: any, opts?: RequestOptions) => request<T>(path, { ...opts, method: 'PUT', data }),
  delete: <T = any>(path: string, opts?: RequestOptions) => request<T>(path, { ...opts, method: 'DELETE' }),
}
