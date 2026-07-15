import { describe, expect, it, vi } from 'vitest'
import { requestErrorMessage } from './request-error'

describe('requestErrorMessage', () => {
  it('distinguishes offline and timeout failures', () => {
    vi.stubGlobal('navigator', { onLine: false })
    expect(requestErrorMessage(new Error('failed'))).toContain('网络不可用')

    vi.stubGlobal('navigator', { onLine: true })
    expect(requestErrorMessage({ isAxiosError: true, code: 'ECONNABORTED' })).toContain('请求超时')
    vi.unstubAllGlobals()
  })

  it('uses the API message when present', () => {
    vi.stubGlobal('navigator', { onLine: true })
    expect(requestErrorMessage({
      isAxiosError: true,
      response: { data: { message: '活动不存在' } },
    })).toBe('活动不存在')
    vi.unstubAllGlobals()
  })
})
