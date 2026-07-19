import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import { activitiesAPI, firstRechargeAPI } from '@/api/activities'
import FirstRechargeBanner from '@/components/common/FirstRechargeBanner.vue'
import { useAuthStore } from '@/stores/auth'

vi.mock('@/api/activities', () => {
  const activitiesAPI = {
    list: vi.fn(),
    markViewed: vi.fn(),
    checkIn: vi.fn(),
  }
  const firstRechargeAPI = {
    getStatus: vi.fn(),
    dismissPopup: vi.fn(),
  }
  return {
    default: firstRechargeAPI,
    activitiesAPI,
    firstRechargeAPI,
  }
})

const user = {
  id: 42,
  username: 'banner-user',
  email: 'banner@example.test',
  role: 'user' as const,
  balance: 0,
  concurrency: 1,
  status: 'active' as const,
  allowed_groups: null,
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-07-19T00:00:00Z',
  updated_at: '2026-07-19T00:00:00Z',
}

describe('FirstRechargeBanner activity entries', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(firstRechargeAPI.getStatus).mockResolvedValue({
      data: {
        enabled: true,
        eligible: true,
        completed: false,
        popup_dismissed: false,
        eligibility_scope: 'all_users',
        purchase_mode: 'product_link',
        product_url: 'https://shop.example.test/first-recharge',
        created_at: '2026-07-19T00:00:00Z',
        updated_at: '2026-07-19T00:00:00Z',
        offers: [],
      },
    } as never)
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
  })

  it('shows first recharge and daily check-in together and opens the activity center', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const authStore = useAuthStore()
    authStore.$patch({ token: 'test-token', user })

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/activities', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()

    const wrapper = mount(FirstRechargeBanner, {
      global: {
        plugins: [
          pinia,
          router,
          createI18n({
            legacy: false,
            locale: 'zh',
            messages: {
              zh: {
                activities: {
                  dailyCheckIn: {
                    title: () => '每日签到送额度',
                    bannerSummary: (context: { named: (key: string) => unknown }) =>
                      `今日签到可领取 ${String(context.named('amount'))} 额度。`,
                    bannerCompletedSummary: () => '今日签到已完成，奖励已发放到账户余额。',
                    bannerCta: () => '去签到',
                    checkIn: () => '立即签到',
                    viewDetails: () => '查看活动',
                  },
                },
                firstRecharge: {
                  banner: {
                    title: () => '首充专属活动',
                    cta: () => '去首充',
                    productLinkSummary: () => '前往活动商品页完成首充购买。',
                  },
                },
              },
            },
            missingWarn: false,
            fallbackWarn: false,
          }),
        ],
        stubs: { Icon: true },
      },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="first-recharge-banner"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="daily-check-in-banner"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('每日签到送额度')
    expect(wrapper.text()).toContain('2.50')

    await wrapper.get('[data-testid="daily-check-in-banner-cta"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/activities')
  })
})
