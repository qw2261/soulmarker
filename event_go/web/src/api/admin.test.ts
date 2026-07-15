import { describe, expect, it } from 'vitest'
import { isPlatformAdminSession } from './admin'

describe('platform admin session contract', () => {
  it('requires both authenticated and the platform_admin principal type', () => {
    expect(isPlatformAdminSession({ authenticated: true, principal_type: 'platform_admin' })).toBe(true)
    expect(isPlatformAdminSession({ authenticated: false, principal_type: 'platform_admin' })).toBe(false)
    expect(isPlatformAdminSession(undefined)).toBe(false)
    expect(isPlatformAdminSession({ authenticated: true } as never)).toBe(false)
    expect(isPlatformAdminSession({ authenticated: true, principal_type: 'organization_member' } as never)).toBe(false)
  })
})
