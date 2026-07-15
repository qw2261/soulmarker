import { beforeEach, describe, expect, it, vi } from 'vitest'
import { expireUserSession, isCurrentUserSessionToken, USER_SESSION_EXPIRED_EVENT } from './session'

describe('user session expiry', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('clears persisted identity and emits one redirect event', () => {
    localStorage.setItem('user_token', 'expired-token')
    localStorage.setItem('user_info', '{"id":1}')
    const listener = vi.fn()
    window.addEventListener(USER_SESSION_EXPIRED_EVENT, listener)

    expireUserSession('/me/registrations?page=2')

    expect(localStorage.getItem('user_token')).toBeNull()
    expect(localStorage.getItem('user_info')).toBeNull()
    expect(listener).toHaveBeenCalledTimes(1)
    expect((listener.mock.calls[0][0] as CustomEvent).detail.redirect).toBe('/me/registrations?page=2')
    window.removeEventListener(USER_SESSION_EXPIRED_EVENT, listener)
  })

  it('does not let a stale request invalidate a newly authenticated session', () => {
    localStorage.setItem('user_token', 'new-token')
    expect(isCurrentUserSessionToken('old-token')).toBe(false)
    expect(isCurrentUserSessionToken('new-token')).toBe(true)
  })
})
