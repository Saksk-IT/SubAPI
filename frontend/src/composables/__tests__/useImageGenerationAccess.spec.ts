import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ApiKey } from '@/types'

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  authStore: {
    isAuthenticated: true,
    user: { id: 1 } as { id: number } | null,
  },
}))

vi.mock('@/api/keys', () => ({
  keysAPI: {
    list: mocks.list,
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mocks.authStore,
}))

import {
  isImageGenerationKey,
  useImageGenerationAccess,
  useImageGenerationKeys,
} from '../useImageGenerationAccess'

function apiKey(
  id: number,
  overrides: Partial<ApiKey> & {
    group?: Partial<NonNullable<ApiKey['group']>> | null
  } = {}
): ApiKey {
  const group = overrides.group === null ? undefined : {
    id,
    name: `Group ${id}`,
    description: '',
    platform: 'openai',
    rate_multiplier: 1,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'standard',
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null,
    allow_image_generation: true,
    allow_batch_image_generation: false,
    image_rate_independent: false,
    image_rate_multiplier: 1,
    batch_image_discount_multiplier: 1,
    batch_image_hold_multiplier: 1,
    video_rate_independent: false,
    sort_order: 0,
    ...(overrides.group || {}),
  } as NonNullable<ApiKey['group']>

  return {
    id,
    user_id: 1,
    key: `sk-key-${id}`,
    name: `Key ${id}`,
    group_id: group?.id ?? null,
    status: 'active',
    ip_whitelist: [],
    ip_blacklist: [],
    last_used_at: null,
    last_used_ip: null,
    quota: 0,
    quota_used: 0,
    expires_at: null,
    created_at: '2026-07-13T00:00:00Z',
    updated_at: '2026-07-13T00:00:00Z',
    current_concurrency: 0,
    rate_limit_5h: 0,
    rate_limit_1d: 0,
    rate_limit_7d: 0,
    usage_5h: 0,
    usage_1d: 0,
    usage_7d: 0,
    window_5h_start: null,
    window_1d_start: null,
    window_7d_start: null,
    reset_5h_at: null,
    reset_1d_at: null,
    reset_7d_at: null,
    ...overrides,
    group: group || undefined,
  }
}

function page(items: ApiKey[], current: number, pages: number) {
  return {
    items,
    total: items.length,
    page: current,
    page_size: 100,
    pages,
  }
}

describe('useImageGenerationAccess', () => {
  beforeEach(() => {
    mocks.list.mockReset()
    mocks.authStore.isAuthenticated = true
    mocks.authStore.user = { id: 1 }
  })

  it('only accepts active OpenAI keys whose group enables image generation', () => {
    expect(isImageGenerationKey(apiKey(1))).toBe(true)
    expect(isImageGenerationKey(apiKey(2, { status: 'inactive' }))).toBe(false)
    expect(isImageGenerationKey(apiKey(3, { group: { platform: 'gemini' } }))).toBe(false)
    expect(isImageGenerationKey(apiKey(4, { group: { allow_image_generation: false } }))).toBe(false)
    expect(isImageGenerationKey(apiKey(5, { group_id: null, group: null }))).toBe(false)
  })

  it('retains only an access boolean and stops after finding an eligible key', async () => {
    mocks.list
      .mockResolvedValueOnce(page([apiKey(1), apiKey(2, { group: { platform: 'gemini' } })], 1, 2))

    const access = useImageGenerationAccess()
    const eligible = await access.refreshImageGenerationAccess(true)

    expect(mocks.list).toHaveBeenNthCalledWith(1, 1, 100, {
      status: 'active',
      sort_by: 'created_at',
      sort_order: 'desc',
    })
    expect(mocks.list).toHaveBeenCalledTimes(1)
    expect(eligible).toBe(true)
    expect(access).not.toHaveProperty('imageGenerationKeys')
    expect(access.canUseImageGeneration.value).toBe(true)
    expect(access.imageGenerationAccessLoaded.value).toBe(true)
    expect(access.imageGenerationAccessError.value).toBeNull()
  })

  it('continues paging until access is found', async () => {
    mocks.list
      .mockResolvedValueOnce(page([apiKey(2, { group: { platform: 'gemini' } })], 1, 2))
      .mockResolvedValueOnce(page([apiKey(3)], 2, 2))

    await expect(useImageGenerationAccess().refreshImageGenerationAccess(true)).resolves.toBe(true)

    expect(mocks.list).toHaveBeenCalledTimes(2)
  })

  it('exposes a retryable access error without retaining credentials', async () => {
    mocks.list.mockRejectedValueOnce(new Error('network unavailable'))

    const access = useImageGenerationAccess()
    const eligible = await access.refreshImageGenerationAccess(true)

    expect(eligible).toBe(false)
    expect(access).not.toHaveProperty('imageGenerationKeys')
    expect(access.canUseImageGeneration.value).toBe(false)
    expect(access.imageGenerationAccessError.value).toBe('network unavailable')

    mocks.list.mockResolvedValueOnce(page([apiKey(6)], 1, 1))
    await access.refreshImageGenerationAccess(true)

    expect(access.canUseImageGeneration.value).toBe(true)
    expect(access.imageGenerationAccessError.value).toBeNull()
  })

  it('does not request or expose access when the user is logged out', async () => {
    mocks.authStore.isAuthenticated = false
    const access = useImageGenerationAccess()

    await expect(access.refreshImageGenerationAccess()).resolves.toBe(false)

    expect(mocks.list).not.toHaveBeenCalled()
    expect(access.canUseImageGeneration.value).toBe(false)
    expect(access.imageGenerationAccessLoaded.value).toBe(true)
  })

  it('shares one in-flight access check across sidebar consumers', async () => {
    mocks.list.mockResolvedValue(page([apiKey(7)], 1, 1))
    const firstSidebar = useImageGenerationAccess()
    const secondSidebar = useImageGenerationAccess()

    const firstLoad = firstSidebar.refreshImageGenerationAccess(true)
    const secondLoad = secondSidebar.refreshImageGenerationAccess()
    await expect(Promise.all([firstLoad, secondLoad])).resolves.toEqual([true, true])

    expect(mocks.list).toHaveBeenCalledTimes(1)
    expect(firstSidebar.canUseImageGeneration.value).toBe(true)
    expect(secondSidebar.canUseImageGeneration.value).toBe(true)
  })

  it('does not retain a previous account access result when users switch during a load', async () => {
    let resolveFirstPage: ((value: ReturnType<typeof page>) => void) | undefined
    mocks.list
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirstPage = resolve }))
      .mockResolvedValueOnce(page([apiKey(9, { user_id: 2 })], 1, 1))

    const access = useImageGenerationAccess()
    const firstLoad = access.refreshImageGenerationAccess(true)
    mocks.authStore.user = { id: 2 }
    const secondLoad = access.refreshImageGenerationAccess()

    expect(mocks.list).toHaveBeenCalledTimes(2)
    resolveFirstPage?.(page([apiKey(8, { user_id: 1 })], 1, 1))
    await Promise.all([firstLoad, secondLoad])

    expect(access.canUseImageGeneration.value).toBe(true)
  })

  it('loads all eligible keys only into a page-local key store', async () => {
    mocks.list
      .mockResolvedValueOnce(page([apiKey(1), apiKey(2, { group: { platform: 'gemini' } })], 1, 2))
      .mockResolvedValueOnce(page([apiKey(3), apiKey(4, { status: 'inactive' })], 2, 2))

    const keysState = useImageGenerationKeys()
    const keys = await keysState.refreshImageGenerationAccess(true)

    expect(keys.map((key) => key.id)).toEqual([1, 3])
    expect(keysState.imageGenerationKeys.value.map((key) => key.id)).toEqual([1, 3])
    expect(mocks.list).toHaveBeenCalledTimes(2)
  })

  it('isolates key arrays per page instance and clears them explicitly', async () => {
    mocks.list.mockResolvedValue(page([apiKey(10)], 1, 1))
    const firstPage = useImageGenerationKeys()
    const secondPage = useImageGenerationKeys()

    await firstPage.refreshImageGenerationAccess(true)

    expect(firstPage.imageGenerationKeys.value.map((key) => key.id)).toEqual([10])
    expect(secondPage.imageGenerationKeys.value).toEqual([])
    expect(firstPage.imageGenerationKeys).not.toBe(secondPage.imageGenerationKeys)

    firstPage.clearImageGenerationKeys()
    expect(firstPage.imageGenerationKeys.value).toEqual([])
  })

  it('discards page-local keys loaded for an account that is no longer active', async () => {
    let resolveFirstPage: ((value: ReturnType<typeof page>) => void) | undefined
    mocks.list
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirstPage = resolve }))
      .mockResolvedValueOnce(page([apiKey(12, { user_id: 2 })], 1, 1))

    const keysState = useImageGenerationKeys()
    const firstLoad = keysState.refreshImageGenerationAccess(true)
    mocks.authStore.user = { id: 2 }
    const secondLoad = keysState.refreshImageGenerationAccess()
    resolveFirstPage?.(page([apiKey(11)], 1, 1))
    await Promise.all([firstLoad, secondLoad])

    expect(keysState.imageGenerationKeys.value.map((key) => key.id)).toEqual([12])
    expect(keysState.imageGenerationKeys.value.every((key) => key.user_id === 2)).toBe(true)
  })
})
