import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import MyRegistrations from './MyRegistrations.vue'
import type { MyActivity } from '@/api/types'

const mocks = vi.hoisted(() => ({ listMyActivities: vi.fn() }))

vi.mock('@/api/events', () => ({
  listMyActivities: mocks.listMyActivities,
}))

const admission = {
  id: 11,
  event_id: 101,
  credential_code: '00112233445566778899aabbccddeeff',
  credential: 'soulmark:admission:00112233445566778899aabbccddeeff',
  status: 'active' as const,
  issued_at: '2030-01-01T00:00:00Z',
}

const activities: MyActivity[] = [
  {
    id: 11,
    kind: 'admission',
    event_id: 101,
    event_title: '免费活动',
    event_time: '2099-12-31T10:00:00Z',
    location: 'A 场地',
    event_status: 'published',
    joined_at: '2030-01-01T00:00:00Z',
    admission,
  },
  {
    id: 22,
    kind: 'registration',
    registration_id: 22,
    event_id: 202,
    event_title: '付费活动',
    event_time: '2099-12-30T10:00:00Z',
    location: 'B 场地',
    event_status: 'published',
    ticket_name: '标准票',
    joined_at: '2030-01-02T00:00:00Z',
  },
]

const PaginationStub = defineComponent({
  props: { total: Number, currentPage: Number, pageSize: Number },
  emits: ['page-change', 'size-change'],
  template: '<button class="next-page" @click="$emit(\'page-change\', 2)">next</button>',
})

describe('MyRegistrations', () => {
  beforeEach(() => {
    mocks.listMyActivities.mockReset()
    mocks.listMyActivities.mockResolvedValue({ data: activities, total: 23 })
  })

  it('uses one exact activity timeline for rendering and pagination', async () => {
    const wrapper = mount(MyRegistrations, {
      global: {
        mocks: { $router: { push: vi.fn() } },
        stubs: {
          NavBar: true,
          Pagination: PaginationStub,
          AdmissionCredential: { props: ['admission'], template: '<div class="credential-stub" />' },
          ElMain: { template: '<main><slot /></main>' },
          ElSkeleton: true,
          ElEmpty: true,
          ElTag: { template: '<span><slot /></span>' },
        },
      },
    })
    await flushPromises()

    expect(mocks.listMyActivities).toHaveBeenCalledTimes(1)
    expect(mocks.listMyActivities).toHaveBeenLastCalledWith({ page: 1, page_size: 10 })
    expect(wrapper.text()).toContain('23 项')
    expect(wrapper.text()).toContain('免费活动')
    expect(wrapper.text()).toContain('付费活动')
    expect(wrapper.text()).toContain('标准票')
    expect(wrapper.findAll('.credential-stub')).toHaveLength(1)

    await wrapper.get('.next-page').trigger('click')
    await flushPromises()
    expect(mocks.listMyActivities).toHaveBeenCalledTimes(2)
    expect(mocks.listMyActivities).toHaveBeenLastCalledWith({ page: 2, page_size: 10 })
  })
})
