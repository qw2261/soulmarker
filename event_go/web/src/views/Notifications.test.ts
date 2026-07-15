import ElementPlus from 'element-plus'
import { createPinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Notifications from './Notifications.vue'

const mocks = vi.hoisted(() => ({
  listNotifications: vi.fn(),
  getUnreadCount: vi.fn(),
  markRead: vi.fn(),
  markAllRead: vi.fn(),
  routerPush: vi.fn(),
}))

vi.mock('@/api/notifications', () => ({
  listNotifications: mocks.listNotifications,
  getNotificationUnreadCount: mocks.getUnreadCount,
  markNotificationRead: mocks.markRead,
  markAllNotificationsRead: mocks.markAllRead,
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mocks.routerPush }),
}))

const notifications = [
  {
    id: 11,
    event_id: 101,
    type: 'registration_confirmed',
    title: '报名成功',
    body: '你已成功报名活动“通知测试活动”。',
    action_url: '/events/101',
    created_at: '2030-01-01T00:00:00Z',
  },
  {
    id: 12,
    event_id: 101,
    type: 'event_updated',
    title: '活动信息已更新',
    body: '活动地点已更新。',
    action_url: '/events/101',
    read_at: '2030-01-02T00:00:00Z',
    created_at: '2030-01-02T00:00:00Z',
  },
]

describe('Notifications', () => {
  beforeEach(() => {
    localStorage.setItem('user_token', 'test-token')
    mocks.listNotifications.mockReset().mockResolvedValue({ data: notifications.map((item) => ({ ...item })), total: 2 })
    mocks.getUnreadCount.mockReset().mockResolvedValue({ data: { unread: 1 } })
    mocks.markRead.mockReset().mockResolvedValue({ code: 200 })
    mocks.markAllRead.mockReset().mockResolvedValue({ data: { updated: 1 } })
    mocks.routerPush.mockReset()
  })

  it('loads a paginated list and marks a single notification before navigation', async () => {
    const wrapper = mount(Notifications, {
      global: {
        plugins: [createPinia(), ElementPlus],
        stubs: {
          NavBar: true,
          Pagination: true,
          PageLoadError: true,
        },
      },
    })
    await flushPromises()

    expect(mocks.listNotifications).toHaveBeenCalledWith({ unread_only: false, page: 1, page_size: 10 })
    expect(wrapper.text()).toContain('2 条通知')
    expect(wrapper.text()).toContain('报名成功')
    expect(wrapper.findAll('.notification-item.unread')).toHaveLength(1)

    await wrapper.findAll('.notification-content')[0].trigger('click')
    await flushPromises()
    expect(mocks.markRead).toHaveBeenCalledWith(11)
    expect(mocks.routerPush).toHaveBeenCalledWith('/events/101')
    expect(wrapper.findAll('.notification-item.unread')).toHaveLength(0)
  })

  it('marks every visible notification as read', async () => {
    const wrapper = mount(Notifications, {
      global: {
        plugins: [createPinia(), ElementPlus],
        stubs: { NavBar: true, Pagination: true, PageLoadError: true },
      },
    })
    await flushPromises()
    await wrapper.get('.page-header button').trigger('click')
    await flushPromises()

    expect(mocks.markAllRead).toHaveBeenCalledTimes(1)
    expect(wrapper.findAll('.notification-item.unread')).toHaveLength(0)
  })
})
