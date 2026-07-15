import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdmissionCredential from './AdmissionCredential.vue'
import type { Admission } from '@/api/types'

const mocks = vi.hoisted(() => ({ toDataURL: vi.fn() }))

vi.mock('qrcode', () => ({
  default: { toDataURL: mocks.toDataURL },
}))

const activeAdmission: Admission = {
  id: 1,
  event_id: 2,
  credential_code: '00112233445566778899aabbccddeeff',
  credential: 'soulmark:admission:00112233445566778899aabbccddeeff',
  status: 'active',
  issued_at: '2030-01-01T00:00:00Z',
}

function mountCredential(admission: Admission) {
  return mount(AdmissionCredential, {
    props: { admission },
    global: {
      stubs: {
        ElTag: { template: '<span class="tag"><slot /></span>' },
        ElButton: { template: '<button @click="$emit(\'click\', $event)"><slot /></button>' },
        ElSkeletonItem: { template: '<span class="skeleton" />' },
      },
    },
  })
}

describe('AdmissionCredential', () => {
  beforeEach(() => {
    mocks.toDataURL.mockResolvedValue('data:image/png;base64,credential')
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
  })

  it('renders a stable QR payload and copies the full credential', async () => {
    const wrapper = mountCredential(activeAdmission)
    await flushPromises()

    expect(mocks.toDataURL).toHaveBeenCalledWith(activeAdmission.credential, expect.objectContaining({ width: 184 }))
    expect(wrapper.get('img').attributes('src')).toBe('data:image/png;base64,credential')
    expect(wrapper.text()).toContain('待入场')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(activeAdmission.credential)
  })

  it('shows checked-in and revoked states without changing layout', async () => {
    const wrapper = mountCredential({ ...activeAdmission, checked_in_at: '2030-01-02T03:04:05Z' })
    await flushPromises()
    expect(wrapper.text()).toContain('已入场')
    expect(wrapper.text()).toContain('核销时间')

    await wrapper.setProps({ admission: { ...activeAdmission, status: 'revoked' } })
    expect(wrapper.text()).toContain('已取消')
    expect(wrapper.classes()).toContain('inactive')
  })
})
