import { describe, expect, it, vi } from 'vitest'

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: false,
  isAdmin: false,
  isSimpleMode: false,
  hasPendingAuthSession: false,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'Sub2API',
  backendModeEnabled: false,
  cachedPublicSettings: null as null | Record<string, unknown>,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
}))

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

describe('static guide routes', () => {
  it('registers the registration and API key parent guide as a public route', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.name === 'RegistrationKeyGuide')

    expect(route).toMatchObject({
      path: '/registration-key-guide',
      meta: {
        requiresAuth: false,
        title: '中转注册、兑换与 API 密钥配置教程',
        guideKey: 'registration',
      },
    })
  })

  it('keeps all six client guide routes on the shared guide view', async () => {
    const { default: router } = await import('@/router')
    const expectedRoutes = new Map([
      ['CodexGuide', 'codex'],
      ['ClaudeCodeGuide', 'claude'],
      ['OpenCodeGuide', 'openCode'],
      ['OpenClawGuide', 'openClaw'],
      ['MobileGuide', 'mobile'],
      ['ImageGuide', 'image'],
    ])

    for (const [name, guideKey] of expectedRoutes) {
      const route = router.getRoutes().find((record) => record.name === name)
      expect(route?.meta.guideKey).toBe(guideKey)
    }
  })
})
