import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  getNotificationUnreadCount,
  markAllNotificationsRead,
  markNotificationRead,
} from '@/api/notifications'

export const useNotificationStore = defineStore('notifications', () => {
  const unreadCount = ref(0)
  const loading = ref(false)

  async function fetchUnreadCount() {
    if (!localStorage.getItem('user_token')) {
      clear()
      return
    }
    loading.value = true
    try {
      const response = await getNotificationUnreadCount()
      unreadCount.value = response.data?.unread || 0
    } catch {
      unreadCount.value = 0
    } finally {
      loading.value = false
    }
  }

  async function markRead(notificationId: number) {
    await markNotificationRead(notificationId)
    unreadCount.value = Math.max(0, unreadCount.value - 1)
  }

  async function markAllRead() {
    const response = await markAllNotificationsRead()
    unreadCount.value = 0
    return response.data?.updated || 0
  }

  function clear() {
    unreadCount.value = 0
    loading.value = false
  }

  return { unreadCount, loading, fetchUnreadCount, markRead, markAllRead, clear }
})
