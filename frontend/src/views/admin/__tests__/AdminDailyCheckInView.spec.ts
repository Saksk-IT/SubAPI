import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { adminActivitiesAPI } from '@/api/admin/activities'
import AdminDailyCheckInView from '@/views/admin/activities/AdminDailyCheckInView.vue'

const { mockAdminActivitiesAPI } = vi.hoisted(() => ({
  mockAdminActivitiesAPI: {
    getDailyCheckIn: vi.fn(),
    updateDailyCheckIn: vi.fn(),
  },
}))

vi.mock('@/api/admin/activities', () => ({
  adminActivitiesAPI: mockAdminActivitiesAPI,
  default: mockAdminActivitiesAPI,
}))

describe('AdminDailyCheckInView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(adminActivitiesAPI.getDailyCheckIn).mockResolvedValue({
      data: {
        enabled: true,
        reward_amount: 1.5,
        timezone: 'Asia/Shanghai',
        created_at: '2026-07-19T00:00:00Z',
        updated_at: '2026-07-19T00:00:00Z',
      },
    } as never)
    vi.mocked(adminActivitiesAPI.updateDailyCheckIn).mockResolvedValue({
      data: {
        enabled: true,
        reward_amount: 2.5,
        timezone: 'Asia/Shanghai',
        created_at: '2026-07-19T00:00:00Z',
        updated_at: '2026-07-19T00:00:00Z',
      },
    } as never)
  })

  it('loads and saves the configured daily reward', async () => {
    const wrapper = mount(AdminDailyCheckInView, {
      global: {
        plugins: [
          createPinia(),
          createI18n({ legacy: false, locale: 'zh', messages: { zh: {} }, missingWarn: false, fallbackWarn: false }),
        ],
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
          LoadingSpinner: true,
          Toggle: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<button type="button" @click="$emit(\'update:modelValue\', !modelValue)" />',
          },
        },
      },
    })
    await flushPromises()

    const rewardInput = wrapper.get('[data-testid="daily-check-in-reward"]')
    expect((rewardInput.element as HTMLInputElement).value).toBe('1.5')
    await rewardInput.setValue('2.5')
    const buttons = wrapper.findAll('button')
    await buttons[1].trigger('click')
    await flushPromises()

    expect(adminActivitiesAPI.updateDailyCheckIn).toHaveBeenCalledWith({
      enabled: true,
      reward_amount: 2.5,
    })
  })
})
