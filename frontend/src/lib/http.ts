import axios, { type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'
import type { ApiResponse } from '@/types/api'

// token 存储
const TOKEN_KEY = 'gamero_access_token'
const REFRESH_KEY = 'gamero_refresh_token'

export const tokenStorage = {
  getAccess: () => localStorage.getItem(TOKEN_KEY) ?? '',
  setAccess: (t: string) => localStorage.setItem(TOKEN_KEY, t),
  getRefresh: () => localStorage.getItem(REFRESH_KEY) ?? '',
  setRefresh: (t: string) => localStorage.setItem(REFRESH_KEY, t),
  clear: () => {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 15_000,
  headers: { 'Content-Type': 'application/json' },
  validateStatus: () => true,
})

// 请求拦截：自动注入 Authorization
http.interceptors.request.use((cfg) => {
  const token = tokenStorage.getAccess()
  if (token) cfg.headers.Authorization = `Bearer ${token}`
  return cfg
})

// Token 刷新状态
let refreshing = false
let refreshQueue: Array<{ resolve: (token: string) => void; reject: (err: unknown) => void }> = []

// 响应拦截：处理 token 过期（因为 validateStatus=true，401 进入 success 分支）
let hasShownExpiredToast = false
http.interceptors.response.use(
  (res) => {
    // 检测 token 过期 (code 3001 = Token无效, 3003 = Token过期)
    // Token 错误码: 3001=无效 3002=过期 3003=吊销 3004=缺失
    const tokenExpired = [3001, 3002, 3003, 3004].includes(res.data?.code)
    // 不拦截 refresh 端点自身，否则会死循环
    if (tokenExpired && !res.config.url?.includes('/auth/refresh')) {
      return handleTokenExpired(res.config) as Promise<AxiosResponse>
    }
    return res
  },
  async (err) => {
    // 网络错误等 axios 级错误
    if (err.response?.status === 401) {
      return handleTokenExpired(err.config) as Promise<AxiosResponse>
    }
    return Promise.reject(err)
  },
)

async function handleTokenExpired(originalConfig: InternalAxiosRequestConfig & { _retry?: boolean }) {
  if (originalConfig._retry) {
    // 已经重试过，直接跳转登录
    clearAndRedirect()
    return Promise.reject(new Error('token expired'))
  }

  if (refreshing) {
    return new Promise((resolve, reject) => {
      refreshQueue.push({
        resolve: (token) => {
          originalConfig.headers.Authorization = `Bearer ${token}`
          resolve(http(originalConfig))
        },
        reject,
      })
    })
  }

  originalConfig._retry = true
  refreshing = true

  try {
    const refreshToken = tokenStorage.getRefresh()
    if (!refreshToken) {
      clearAndRedirect()
      return Promise.reject(new Error('no refresh token'))
    }
    const res = await axios.post<ApiResponse<{ access_token: string; refresh_token: string }>>(
      '/api/v1/auth/refresh',
      { refresh_token: refreshToken },
    )
    if (res.data.code !== 0 || !res.data.data) {
      clearAndRedirect()
      return Promise.reject(new Error('refresh failed'))
    }
    const { access_token, refresh_token } = res.data.data
    tokenStorage.setAccess(access_token)
    tokenStorage.setRefresh(refresh_token)
    hasShownExpiredToast = false
    refreshQueue.forEach((cb) => cb.resolve(access_token))
    refreshQueue = []
    return http(originalConfig)
  } catch {
    clearAndRedirect()
    return Promise.reject(new Error('refresh error'))
  } finally {
    refreshing = false
  }
}

function clearAndRedirect() {
  tokenStorage.clear()
  refreshQueue.forEach((cb) => cb.reject(new Error('token expired')))
  refreshQueue = []
  if (!hasShownExpiredToast) {
    hasShownExpiredToast = true
    // 跳转登录页
    if (window.location.pathname !== '/login') {
      window.location.href = '/login?expired=1'
    }
  }
}

export function extractData<T>(
  res: AxiosResponse<ApiResponse<T>>,
): T {
  if (res.data.code !== 0) {
    throw res.data
  }
  return res.data.data as T
}

export default http
