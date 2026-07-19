import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import { activitiesAPI } from '@/api/activities'
import ActivitiesView from '@/views/user/ActivitiesView.vue'

vi.mock('@/api/activities', () => ({
  default: {},
  activitiesAPI: {
    list: vi.fn(),
    markViewed: vi.fn(),
    checkIn: vi.fn(),
  },
}))

describe('ActivitiesView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(activitiesAPI.list).mockResolvedValue({
      data: [{
        id: 'first_recharge',
        type: 'first_recharge',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
        first_recharge: {
          enabled: true,
          eligible: true,
          completed: false,
          popup_dismissed: false,
          eligibility_scope: 'all_users',
          purchase_mode: 'product_link',
          product_url: 'https://shop.example.test/first-recharge',
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
          offers: [],
        },
      }],
    } as never)
    vi.mocked(activitiesAPI.markViewed).mockResolvedValue({ data: { message: 'ok' } } as never)
  })

  it('renders eligible activities and records them as viewed on entry', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/activities', component: ActivitiesView }],
    })
    await router.push('/activities')
    await router.isReady()

    const wrapper = mount(ActivitiesView, {
      global: {
        plugins: [
          createPinia(),
          router,
          createI18n({
            legacy: false,
            locale: 'zh',
            messages: { zh: {} },
            missingWarn: false,
            fallbackWarn: false,
          }),
        ],
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
          LoadingSpinner: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="activity-card-first_recharge"]').exists()).toBe(true)
    expect(activitiesAPI.markViewed).toHaveBeenCalledWith('first_recharge')
  })

  it('checks in once and updates the daily activity state', async () => {
    vi.mocked(activitiesAPI.list).mockResolvedValue({
      data: [{
        id: 'daily_check_in',
        type: 'daily_check_in',
        created_at: '2026-07-19T00:00:00Z',
        updated_at: '2026-07-19T00:00:00Z',
        daily_check_in: {
          enabled: true,
          checked_in_today: false,
          reward_amount: 2.5,
          timezone: 'Asia/Shanghai',
          total_check_ins: 4,
          created_at: '2026-07-19T00:00:00Z',
          updated_at: '2026-07-19T00:00:00Z',
        },
      }],
    } as never)
    vi.mocked(activitiesAPI.checkIn).mockResolvedValue({
      data: {
        reward_amount: 2.5,
        balance_after: 12.5,
        check_in_date: '2026-07-19',
        checked_in_at: '2026-07-19T01:00:00Z',
      },
    } as never)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/activities', component: ActivitiesView }],
    })
    await router.push('/activities')
    await router.isReady()

    const wrapper = mount(ActivitiesView, {
      global: {
        plugins: [
          createPinia(),
          router,
          createI18n({ legacy: false, locale: 'zh', messages: { zh: {} }, missingWarn: false, fallbackWarn: false }),
        ],
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
          LoadingSpinner: true,
        },
      },
    })
    await flushPromises()

    const button = wrapper.get('[data-testid="daily-check-in-button"]')
    await button.trigger('click')
    await flushPromises()

    expect(activitiesAPI.checkIn).toHaveBeenCalledOnce()
    expect(button.attributes('disabled')).toBeDefined()
  })
})
