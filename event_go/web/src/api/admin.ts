import { get, put } from './client'
import type { ContentModerationAction, ContentReport, ContentReportStatus, ContentTargetType } from './types'

export interface AdminSession {
  authenticated: boolean
}

export function getAdminSession() {
  return get<AdminSession>('/admin/session')
}

export function listContentReports(params: {
  status?: ContentReportStatus
  target_type?: ContentTargetType
  page?: number
  page_size?: number
}) {
  return get<ContentReport[]>('/admin/content-reports', params)
}

export function resolveContentReport(id: number, resolution: 'remove' | 'dismiss', note: string) {
  return put<ContentReport>(`/admin/content-reports/${id}`, { resolution, note })
}

export function listContentModerationActions(params?: { page?: number; page_size?: number }) {
  return get<ContentModerationAction[]>('/admin/content-actions', params)
}
