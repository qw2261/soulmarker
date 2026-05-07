import { post } from './client'
import type { LoginResp } from './types'

export interface RegisterUserReq {
  name: string
  contact: string
  password: string
}

export interface LoginReq {
  contact: string
  password: string
}

export function registerUser(data: RegisterUserReq) {
  return post<LoginResp>('/api/auth/register', data)
}

export function loginUser(data: LoginReq) {
  return post<LoginResp>('/api/auth/login', data)
}
