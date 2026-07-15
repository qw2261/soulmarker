import ElementPlus from 'element-plus'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountSecurity from './AccountSecurity.vue'

const mocks = vi.hoisted(() => ({
  requestRecoveryEmail: vi.fn(),
  userStore: {
    user: {
      id: 1,
      name: '历史用户',
      contact: '13800138000',
      recovery_email: '',
      recovery_email_verified_at: undefined as string | undefined,
      created_at: '2030-01-01T00:00:00Z',
    },
  },
}))

vi.mock('@/api/auth', () => ({ requestRecoveryEmail: mocks.requestRecoveryEmail }))
vi.mock('@/stores/user', () => ({ useUserStore: () => mocks.userStore }))

describe('AccountSecurity', () => {
  beforeEach(() => {
    mocks.requestRecoveryEmail.mockReset()
    mocks.requestRecoveryEmail.mockResolvedValue({ code: 202 })
    mocks.userStore.user = {
      id: 1,
      name: '历史用户',
      contact: '13800138000',
      recovery_email: '',
      recovery_email_verified_at: undefined,
      created_at: '2030-01-01T00:00:00Z',
    }
  })

  it('keeps the legacy login contact and requests password-confirmed email verification', async () => {
    const wrapper = mount(AccountSecurity, {
      global: { plugins: [ElementPlus], stubs: { NavBar: true } },
    })
    expect(wrapper.text()).toContain('13800138000')
    expect(wrapper.text()).toContain('仍使用原手机号登录')

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('Recovery@Example.COM')
    await inputs[1].setValue('current-password')
    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(mocks.requestRecoveryEmail).toHaveBeenCalledWith({
      email: 'Recovery@Example.COM',
      password: 'current-password',
    })
    expect(wrapper.text()).toContain('验证邮件已发送至 recovery@example.com')
  })

  it('shows verified recovery state without another binding form', () => {
    mocks.userStore.user = {
      ...mocks.userStore.user,
      recovery_email: 'recovery@example.com',
      recovery_email_verified_at: '2030-01-02T00:00:00Z',
    }
    const wrapper = mount(AccountSecurity, {
      global: { plugins: [ElementPlus], stubs: { NavBar: true } },
    })
    expect(wrapper.text()).toContain('recovery@example.com')
    expect(wrapper.text()).toContain('已验证')
    expect(wrapper.find('form').exists()).toBe(false)
  })
})
