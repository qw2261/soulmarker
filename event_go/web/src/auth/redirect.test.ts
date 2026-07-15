import { describe, expect, it } from 'vitest'
import { safeRedirectPath } from './redirect'

describe('authentication redirect', () => {
  it('keeps only same-origin absolute paths', () => {
    expect(safeRedirectPath('/organization-invitations/accept?token=value')).toBe('/organization-invitations/accept?token=value')
    expect(safeRedirectPath('//evil.example/path')).toBe('/')
    expect(safeRedirectPath('https://evil.example/path')).toBe('/')
    expect(safeRedirectPath(undefined, '/workspace')).toBe('/workspace')
  })
})
