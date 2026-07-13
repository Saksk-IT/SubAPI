import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive, ref } from 'vue'

const launcher = {
  open: vi.fn(),
}

const authStore = reactive({
  isAuthenticated: true,
  isAdmin: false,
  user: { id: 42 },
})

const router = {
  replace: vi.fn(),
}

vi.mock('@/stores/imageGenerationLauncher', () => ({
  useImageGenerationLauncherStore: () => launcher,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/composables/useImageGenerationAccess', () => ({
  useImageGenerationKeys: () => ({
    imageGenerationKeys: ref([]),
    canUseImageGeneration: ref(false),
    imageGenerationAccessLoaded: ref(true),
    imageGenerationAccessLoading: ref(false),
    imageGenerationAccessError: ref(null),
    refreshImageGenerationAccess: vi.fn(async () => []),
    clearImageGenerationKeys: vi.fn(),
  }),
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

vi.mock('vue-router', () => ({
  useRouter: () => router,
}))

import ImageGenerationView from '../ImageGenerationView.vue'

describe('ImageGenerationView compatibility route', () => {
  beforeEach(() => {
    launcher.open.mockReset()
    router.replace.mockReset()
    authStore.isAdmin = false
  })

  it('opens the global launcher and replaces the user compatibility route', async () => {
    const wrapper = mount(ImageGenerationView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
        },
      },
    })
    await flushPromises()

    expect(launcher.open).toHaveBeenCalledOnce()
    expect(router.replace).toHaveBeenCalledWith('/dashboard')
    expect(wrapper.find('iframe').exists()).toBe(false)
  })

  it('returns admins to the admin dashboard without rendering an iframe', async () => {
    authStore.isAdmin = true

    const wrapper = mount(ImageGenerationView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
        },
      },
    })
    await flushPromises()

    expect(launcher.open).toHaveBeenCalledOnce()
    expect(router.replace).toHaveBeenCalledWith('/admin/dashboard')
    expect(wrapper.find('iframe').exists()).toBe(false)
  })
})
