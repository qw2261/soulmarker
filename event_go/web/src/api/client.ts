import axios from 'axios'
import type { APIResp } from './types'
import { ElMessage } from 'element-plus'
import { expireUserSession, isCurrentUserSessionToken } from '@/auth/session'
import { expireAdminSession, isCurrentAdminSessionToken } from '@/auth/admin-session'

const publicAuthPaths = new Set([
  '/auth/register',
  '/auth/login',
  '/auth/password-reset/request',
  '/auth/password-reset/confirm',
])

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('user_token')
  if (token && !publicAuthPaths.has(config.url || '')) {
    config.headers.Authorization = `Bearer ${token}`
    ;(config as typeof config & { soulmarkUserToken?: string }).soulmarkUserToken = token
  }
  const adminToken = localStorage.getItem('admin_token')
  if (adminToken) {
    config.headers['X-Admin-Token'] = adminToken
    ;(config as typeof config & { soulmarkAdminToken?: string }).soulmarkAdminToken = adminToken
  }
  return config
})

client.interceptors.response.use(
  (response) => response,
  (error) => {
    const msg = error.response?.data?.message || error.message || '网络错误'
    const errorCode = error.response?.data?.error_code
    const sentUserToken = (error.config as typeof error.config & { soulmarkUserToken?: string })?.soulmarkUserToken
    const sentAdminToken = (error.config as typeof error.config & { soulmarkAdminToken?: string })?.soulmarkAdminToken
    if (error.response?.status === 401 && errorCode === 'ADMIN_AUTH_INVALID' && isCurrentAdminSessionToken(sentAdminToken)) {
      if (error.config?.url === '/admin/session') {
        ElMessage.error(msg)
      } else {
        expireAdminSession(window.location.pathname + window.location.search)
        ElMessage.warning('管理登录已失效，请重新验证')
      }
    } else if (error.response?.status === 401 && isCurrentUserSessionToken(sentUserToken) && ['USER_TOKEN_INVALID', 'USER_AUTH_REQUIRED'].includes(errorCode)) {
      expireUserSession(window.location.pathname + window.location.search)
      ElMessage.warning('登录已过期，请重新登录')
    } else {
      ElMessage.error(msg)
    }
    return Promise.reject(error)
  }
)

export async function get<T>(url: string, params?: any): Promise<APIResp<T>> {
  const res = await client.get<APIResp<T>>(url, { params })
  return res.data
}

export async function post<T>(url: string, data?: any): Promise<APIResp<T>> {
  const res = await client.post<APIResp<T>>(url, data)
  return res.data
}

export async function put<T>(url: string, data?: any): Promise<APIResp<T>> {
  const res = await client.put<APIResp<T>>(url, data)
  return res.data
}

export async function del<T>(url: string, data?: any): Promise<APIResp<T>> {
  const res = await client.delete<APIResp<T>>(url, { data })
  return res.data
}

export function setAuthToken(token: string) {
  client.defaults.headers.common['Authorization'] = `Bearer ${token}`
}

export function clearAuthToken() {
  delete client.defaults.headers.common['Authorization']
}

export default client
