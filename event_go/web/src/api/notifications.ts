import { get, put } from './client'
import type { Notification, NotificationsMarkedRead, NotificationUnreadCount } from './types'

export interface ListNotificationsParams {
  unread_only?: boolean
  page?: number
  page_size?: number
}

export function listNotifications(params?: ListNotificationsParams) {
  return get<Notification[]>('/me/notifications', params)
}

export function getNotificationUnreadCount() {
  return get<NotificationUnreadCount>('/me/notifications/unread-count')
}

export function markNotificationRead(notificationId: number) {
  return put(`/me/notifications/${notificationId}/read`)
}

export function markAllNotificationsRead() {
  return put<NotificationsMarkedRead>('/me/notifications/read-all')
}
