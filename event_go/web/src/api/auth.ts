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

export interface PasswordResetRequest {
  contact: string
}

export interface PasswordResetConfirmRequest {
  token: string
  password: string
}

export interface RecoveryEmailRequest {
  email: string
  password: string
}

export interface RecoveryEmailConfirmRequest {
  token: string
}

export function registerUser(data: RegisterUserReq) {
  return post<LoginResp>('/auth/register', data)
}

export function loginUser(data: LoginReq) {
  return post<LoginResp>('/auth/login', data)
}

export function logoutUser() {
  return post('/auth/logout')
}

export function requestPasswordReset(data: PasswordResetRequest) {
  return post('/auth/password-reset/request', data)
}

export function confirmPasswordReset(data: PasswordResetConfirmRequest) {
  return post('/auth/password-reset/confirm', data)
}

export function requestRecoveryEmail(data: RecoveryEmailRequest) {
  return post('/me/recovery-email/request', data)
}

export function confirmRecoveryEmail(data: RecoveryEmailConfirmRequest) {
  return post('/auth/recovery-email/confirm', data)
}
