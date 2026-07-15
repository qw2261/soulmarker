import axios from 'axios'
import type { APIResp } from './types'
import { ElMessage } from 'element-plus'

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('user_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  const adminToken = localStorage.getItem('admin_token')
  if (adminToken) {
    config.headers['X-Admin-Token'] = adminToken
  }
  return config
})

client.interceptors.response.use(
  (response) => response,
  (error) => {
    const msg = error.response?.data?.message || error.message || '网络错误'
    ElMessage.error(msg)
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
