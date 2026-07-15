import { get } from './client'

export interface AdminSession {
  authenticated: boolean
}

export function getAdminSession() {
  return get<AdminSession>('/admin/session')
}
