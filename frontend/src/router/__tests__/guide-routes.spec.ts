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
  it('freezes the seven public V1 guide routes on the shared guide view', async () => {
    const { default: router } = await import('@/router')
    const { default: ClientGuideView } = await import('@/views/public/ClientGuideView.vue')
    const expectedRoutes = [
      {
        name: 'RegistrationKeyGuide',
        path: '/registration-key-guide',
        guideKey: 'registration',
        requiresAuth: false,
      },
      {
        name: 'CodexGuide',
        path: '/codex-guide',
        guideKey: 'codex',
        requiresAuth: false,
      },
      {
        name: 'ClaudeCodeGuide',
        path: '/claude-code-guide',
        guideKey: 'claude',
        requiresAuth: false,
      },
      {
        name: 'OpenCodeGuide',
        path: '/open-code-guide',
        guideKey: 'openCode',
        requiresAuth: false,
      },
      {
        name: 'OpenClawGuide',
        path: '/open-claw-guide',
        guideKey: 'openClaw',
        requiresAuth: false,
      },
      {
        name: 'MobileGuide',
        path: '/mobile-guide',
        guideKey: 'mobile',
        requiresAuth: false,
      },
      {
        name: 'ImageGuide',
        path: '/image-guide',
        guideKey: 'image',
        requiresAuth: false,
      },
    ]

    for (const expectedRoute of expectedRoutes) {
      const route = router.getRoutes().find((record) => record.name === expectedRoute.name)

      expect(route).toBeDefined()
      expect({
        name: route?.name,
        path: route?.path,
        guideKey: route?.meta.guideKey,
        requiresAuth: route?.meta.requiresAuth,
      }).toEqual(expectedRoute)

      const componentLoader = route?.components?.default as () => Promise<{
        default: unknown
      }>
      const componentModule = await componentLoader()
      expect(componentModule.default).toBe(ClientGuideView)
    }
  })
})
