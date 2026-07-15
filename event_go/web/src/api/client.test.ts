import { describe, expect, it } from 'vitest'
import { isPublicAuthPath } from './client'

describe('API authentication boundary', () => {
  it('does not attach a stale JWT to public token-confirmation requests', () => {
    expect(isPublicAuthPath('/auth/password-reset/confirm')).toBe(true)
    expect(isPublicAuthPath('/auth/recovery-email/confirm')).toBe(true)
    expect(isPublicAuthPath('/me/recovery-email/request')).toBe(false)
  })
})
