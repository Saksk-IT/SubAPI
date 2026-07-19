import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AffiliateView from '@/views/user/AffiliateView.vue'

const { appState, getAffiliateDetail } = vi.hoisted(() => ({
  appState: {
    cachedPublicSettings: {
      affiliate_redeem_code_enabled: false
    } as { affiliate_redeem_code_enabled: boolean },
    showError: vi.fn(),
    showSuccess: vi.fn()
  },
  getAffiliateDetail: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appState
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ refreshUser: vi.fn() })
}))

vi.mock('@/api/user', () => ({
  default: {
    getAffiliateDetail,
    transferAffiliateQuota: vi.fn()
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: vi.fn() })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AffiliateView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    appState.cachedPublicSettings = { affiliate_redeem_code_enabled: false }
    getAffiliateDetail.mockResolvedValue({
      user_id: 1,
      aff_code: 'INVITE2026',
      inviter_id: null,
      aff_count: 0,
      aff_quota: 0,
      aff_frozen_quota: 0,
      aff_history_quota: 0,
      effective_rebate_rate_percent: 10,
      effective_first_rebate_rate_percent: 10,
      effective_repeat_rebate_rate_percent: 5,
      invitees: []
    })
  })

  function mountView() {
    return mount(AffiliateView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true
        }
      }
    })
  }

  it('shows first-payment and repeat-purchase rebate rates', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('affiliate.stats.firstRebateRate')
    expect(wrapper.text()).toContain('affiliate.stats.repeatRebateRate')
  })

  it('shows redeem-code rebate guidance when the public setting is enabled', async () => {
    appState.cachedPublicSettings = { affiliate_redeem_code_enabled: true }

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('affiliate.tips.redeemCode')
  })

  it('hides redeem-code rebate guidance when the public setting is disabled', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).not.toContain('affiliate.tips.redeemCode')
  })
})
