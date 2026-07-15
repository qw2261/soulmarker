import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNotificationStore } from './notifications'

const mocks = vi.hoisted(() => ({
  getUnreadCount: vi.fn(),
  markRead: vi.fn(),
  markAllRead: vi.fn(),
}))

vi.mock('@/api/notifications', () => ({
  getNotificationUnreadCount: mocks.getUnreadCount,
  markNotificationRead: mocks.markRead,
  markAllNotificationsRead: mocks.markAllRead,
}))

describe('notification store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.setItem('user_token', 'test-token')
    mocks.getUnreadCount.mockReset().mockResolvedValue({ data: { unread: 3 } })
    mocks.markRead.mockReset().mockResolvedValue({ code: 200 })
    mocks.markAllRead.mockReset().mockResolvedValue({ data: { updated: 2 } })
  })

  it('keeps the navigation badge in sync with read operations', async () => {
    const store = useNotificationStore()
    await store.fetchUnreadCount()
    expect(store.unreadCount).toBe(3)

    await store.markRead(7)
    expect(mocks.markRead).toHaveBeenCalledWith(7)
    expect(store.unreadCount).toBe(2)

    await store.markAllRead()
    expect(store.unreadCount).toBe(0)
  })

  it('clears stale counts when there is no user session', async () => {
    const store = useNotificationStore()
    store.unreadCount = 5
    localStorage.removeItem('user_token')
    await store.fetchUnreadCount()
    expect(store.unreadCount).toBe(0)
    expect(mocks.getUnreadCount).not.toHaveBeenCalled()
  })
})
