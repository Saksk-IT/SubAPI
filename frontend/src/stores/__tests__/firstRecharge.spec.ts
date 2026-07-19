import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFirstRechargeStore } from '@/stores/firstRecharge'
import type { FirstRechargeStatus } from '@/types/payment'

vi.mock('@/api/activities', () => ({
  firstRechargeAPI: {
    getStatus: vi.fn(),
    dismissPopup: vi.fn(),
  },
}))

function createStatus(overrides: Partial<FirstRechargeStatus> = {}): FirstRechargeStatus {
  return {
    enabled: true,
    eligible: true,
    completed: false,
    popup_dismissed: false,
    eligibility_scope: 'all_users',
    purchase_mode: 'internal_payment',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    offers: [],
    ...overrides,
  }
}

describe('useFirstRechargeStore availability', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('allows a product-link entry without internal offers', () => {
    const store = useFirstRechargeStore()
    store.status = createStatus({
      purchase_mode: 'product_link',
      product_url: 'https://shop.example.test/first-recharge',
      completed: true,
    })

    expect(store.available).toBe(true)
  })

  it('still requires an offer for internal payment', () => {
    const store = useFirstRechargeStore()
    store.status = createStatus()

    expect(store.available).toBe(false)

    store.status = createStatus({
      offers: [{
        id: 1,
        name: 'Starter',
        description: '',
        price: 10,
        amount: 20,
        enabled: true,
        sort_order: 10,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      }],
    })

    expect(store.available).toBe(true)
  })
})
