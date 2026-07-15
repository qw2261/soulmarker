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
  return post<LoginResp>('/auth/register', data)
}

export function loginUser(data: LoginReq) {
  return post<LoginResp>('/auth/login', data)
}
