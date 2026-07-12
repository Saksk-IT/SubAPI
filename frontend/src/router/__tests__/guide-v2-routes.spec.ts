import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

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

vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
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

const expectedV2Routes = [
  ['首页', '/guides/v2'],
  ['入门', '/guides/v2/get-started'],
  ['Codex', '/guides/v2/codex'],
  ['Claude Code', '/guides/v2/claude-code'],
  ['OpenCode', '/guides/v2/opencode'],
  ['OpenClaw', '/guides/v2/openclaw'],
  ['Chatbox 移动端', '/guides/v2/chatbox-mobile'],
  ['Cherry Studio 图像生成', '/guides/v2/cherry-studio-image'],
  ['排错中心', '/guides/v2/troubleshooting'],
] as const

describe('V2 静态指南路由', () => {
  beforeEach(() => {
    vi.stubGlobal('scrollTo', vi.fn())
  })

  it.each(expectedV2Routes)('将%s注册为公开页面：%s', async (_label, path) => {
    const { default: router } = await import('@/router')
    const resolved = router.resolve(path)

    expect(resolved.name).toBe(path === '/guides/v2' ? 'GuideV2Hub' : 'GuideV2Detail')
    expect(resolved.meta.requiresAuth).toBe(false)
    expect(resolved.meta.title).toBe('AI 客户端使用指南')
  })

  it('无效 slug 留在详情路由并渲染 V2 专用 404', async () => {
    const { default: router } = await import('@/router')
    const path = '/guides/v2/does-not-exist'
    const resolved = router.resolve(path)

    expect(resolved.name).toBe('GuideV2Detail')

    const componentLoader = resolved.matched[0]?.components?.default as () => Promise<{
      default: Parameters<typeof mount>[0]
    }>
    const { default: GuideV2View } = await componentLoader()
    await router.push(path)

    const wrapper = mount(GuideV2View, { global: { plugins: [router] } })
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe(path)
    expect(wrapper.find('[data-guide-v2-not-found]').exists()).toBe(true)
    expect(wrapper.text()).toContain('未找到这篇 V2 教程')
  })
})
