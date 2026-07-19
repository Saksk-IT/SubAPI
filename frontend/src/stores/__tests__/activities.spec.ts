import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { activitiesAPI } from '@/api/activities'
import { useActivityStore } from '@/stores/activities'
import type { UserActivity } from '@/types/payment'

vi.mock('@/api/activities', () => ({
  activitiesAPI: {
    list: vi.fn(),
    markViewed: vi.fn(),
    checkIn: vi.fn(),
  },
}))

function createActivity(overrides: Partial<UserActivity> = {}): UserActivity {
  return {
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
    ...overrides,
  }
}

describe('useActivityStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('collects unseen activities for one popup and supports a session-only dismissal', async () => {
    vi.mocked(activitiesAPI.list).mockResolvedValue({ data: [createActivity()] } as never)
    const store = useActivityStore()

    await store.fetchActivities()
    expect(store.popupActivities).toHaveLength(1)

    store.dismissPopupForSession()
    expect(store.popupActivities).toHaveLength(0)
    expect(store.activities[0].viewed_at).toBeUndefined()
    expect(activitiesAPI.markViewed).not.toHaveBeenCalled()
  })

  it('marks every activity viewed after the user enters the activity center', async () => {
    vi.mocked(activitiesAPI.list).mockResolvedValue({ data: [createActivity()] } as never)
    vi.mocked(activitiesAPI.markViewed).mockResolvedValue({ data: { message: 'ok' } } as never)
    const store = useActivityStore()

    await store.fetchActivities()
    await store.markAllViewed()

    expect(activitiesAPI.markViewed).toHaveBeenCalledWith('first_recharge')
    expect(store.unviewedActivities).toHaveLength(0)
    expect(store.activities[0].viewed_at).toBeTruthy()
  })
})
