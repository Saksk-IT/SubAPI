import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { reactive, ref } from 'vue'

const canUseImageGeneration = ref(true)
const route = reactive({ path: '/dashboard' })
const router = { push: vi.fn() }
const launcherStore = { open: vi.fn() }

const appStore = reactive({
  sidebarCollapsed: false,
  mobileOpen: true,
  sidebarScrollTop: 0,
  siteName: 'Sub2API',
  siteLogo: '',
  siteVersion: '1.0.0',
  publicSettingsLoaded: true,
  backendModeEnabled: false,
  cachedPublicSettings: {
    image_generation_enabled: true,
    custom_menu_items: [],
  } as {
    image_generation_enabled?: boolean
    custom_menu_items: []
  },
  toggleSidebar: vi.fn(),
  setMobileOpen: vi.fn((value: boolean) => {
    appStore.mobileOpen = value
  }),
})

const authStore = reactive({
  isAdmin: false,
  isSimpleMode: false,
  user: { id: 42 },
  token: 'jwt-not-forwarded-to-image-launcher',
})

const adminSettingsStore = reactive({
  opsMonitoringEnabled: true,
  paymentEnabled: true,
  customMenuItems: [],
  fetch: vi.fn(),
})

const onboardingStore = {
  isCurrentStep: vi.fn(() => false),
  nextStep: vi.fn(),
}

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => router,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: ref('zh-CN'),
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
  useOnboardingStore: () => onboardingStore,
  useAdminSettingsStore: () => adminSettingsStore,
  useImageGenerationLauncherStore: () => launcherStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/composables/useImageGenerationAccess', () => ({
  useImageGenerationAccess: () => ({
    canUseImageGeneration,
    refreshImageGenerationAccess: vi.fn(async () => canUseImageGeneration.value),
  }),
}))

vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({
    canUseBatchImage: ref(false),
    refreshBatchImageAccess: vi.fn(async () => false),
  }),
}))

let AppSidebar: object
const wrappers: VueWrapper[] = []

beforeAll(async () => {
  vi.stubGlobal('matchMedia', vi.fn(() => ({ matches: false })))
  AppSidebar = (await import('../AppSidebar.vue')).default
})

beforeEach(() => {
  canUseImageGeneration.value = true
  appStore.mobileOpen = true
  appStore.cachedPublicSettings = {
    image_generation_enabled: true,
    custom_menu_items: [],
  }
  launcherStore.open.mockReset()
  router.push.mockReset()
  appStore.setMobileOpen.mockClear()
  localStorage.clear()
})

afterEach(() => {
  for (const wrapper of wrappers.splice(0)) wrapper.unmount()
  vi.useRealTimers()
})

function mountSidebar() {
  const wrapper = mount(AppSidebar, {
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
        VersionBadge: true,
      },
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

function imageGenerationLink(wrapper: VueWrapper) {
  return wrapper.find('a[href="/image-generation"]')
}

describe('AppSidebar image generation action', () => {
  it('shows the action only when the global flag and key access are both enabled', async () => {
    const wrapper = mountSidebar()
    await flushPromises()
    expect(imageGenerationLink(wrapper).exists()).toBe(true)

    appStore.cachedPublicSettings = {
      image_generation_enabled: false,
      custom_menu_items: [],
    }
    await flushPromises()
    expect(imageGenerationLink(wrapper).exists()).toBe(false)

    appStore.cachedPublicSettings = {
      image_generation_enabled: true,
      custom_menu_items: [],
    }
    canUseImageGeneration.value = false
    await flushPromises()
    expect(imageGenerationLink(wrapper).exists()).toBe(false)
  })

  it('prevents navigation, opens the launcher and closes the mobile sidebar', async () => {
    vi.useFakeTimers()
    const wrapper = mountSidebar()
    await flushPromises()
    const event = new MouseEvent('click', { bubbles: true, cancelable: true })

    imageGenerationLink(wrapper).element.dispatchEvent(event)
    await flushPromises()

    expect(event.defaultPrevented).toBe(true)
    expect(launcherStore.open).toHaveBeenCalledOnce()
    expect(router.push).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(150)
    expect(appStore.setMobileOpen).toHaveBeenCalledWith(false)
  })
})
