import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  ADMIN_SESSION_EXPIRED_EVENT,
  expireAdminSession,
  isCurrentAdminSessionToken,
} from './admin-session'

describe('admin session expiry', () => {
  beforeEach(() => localStorage.clear())

  it('clears the persisted token and preserves the requested redirect', () => {
    localStorage.setItem('admin_token', 'expired-admin-token')
    const listener = vi.fn()
    window.addEventListener(ADMIN_SESSION_EXPIRED_EVENT, listener)

    expireAdminSession('/admin/events/7/tickets')

    expect(localStorage.getItem('admin_token')).toBeNull()
    expect(listener).toHaveBeenCalledTimes(1)
    expect((listener.mock.calls[0][0] as CustomEvent).detail.redirect).toBe('/admin/events/7/tickets')
    window.removeEventListener(ADMIN_SESSION_EXPIRED_EVENT, listener)
  })

  it('does not expire a newer admin session for a stale response', () => {
    localStorage.setItem('admin_token', 'new-admin-token')
    expect(isCurrentAdminSessionToken('old-admin-token')).toBe(false)
    expect(isCurrentAdminSessionToken('new-admin-token')).toBe(true)
  })
})
