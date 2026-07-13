import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { defineComponent, reactive, ref } from 'vue'

import type { ApiKey } from '@/types'
import { PopupLaunchError } from '@/features/imagePlayground/popupBridge'

interface AccessState {
  imageGenerationKeys: ReturnType<typeof ref<ApiKey[]>>
  canUseImageGeneration: ReturnType<typeof ref<boolean>>
  imageGenerationAccessLoaded: ReturnType<typeof ref<boolean>>
  imageGenerationAccessLoading: ReturnType<typeof ref<boolean>>
  imageGenerationAccessError: ReturnType<typeof ref<string | null>>
  refreshImageGenerationAccess: ReturnType<typeof vi.fn>
  clearImageGenerationKeys: ReturnType<typeof vi.fn>
}

let accessState: AccessState
let ImageGenerationLauncher: object
let useImageGenerationLauncherStore: typeof import('@/stores/imageGenerationLauncher')['useImageGenerationLauncherStore']

const openPopup = vi.hoisted(() => vi.fn())
const authStore = reactive({
  isAuthenticated: true,
  user: { id: 42 } as { id: number } | null,
})
const appStore = reactive({
  cachedPublicSettings: { image_generation_enabled: true } as { image_generation_enabled?: boolean },
  showError: vi.fn(),
})

vi.mock('@/composables/useImageGenerationAccess', () => ({
  useImageGenerationKeys: () => accessState,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/features/imagePlayground/popupBridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/features/imagePlayground/popupBridge')>()
  return {
    ...actual,
    openImagePlaygroundPopup: openPopup,
  }
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialogStub',
  props: {
    show: Boolean,
    title: String,
  },
  emits: ['close'],
  template: `
    <div v-if="show" role="dialog">
      <h2>{{ title }}</h2>
      <button type="button" data-testid="dialog-close" @click="$emit('close')">关闭</button>
      <slot />
      <slot name="footer" />
    </div>
  `,
})

function key(id: number, name = `Image Key ${id}`): ApiKey {
  return {
    id,
    user_id: 42,
    key: `sk-secret-${id}`,
    name,
    status: 'active',
    group: {
      platform: 'openai',
      allow_image_generation: true,
    },
  } as ApiKey
}

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

function i18n() {
  const message = (value: string) => () => value
  return createI18n({
    legacy: false,
    locale: 'zh-CN',
    messages: {
      'zh-CN': {
        common: { cancel: message('取消') },
        imageGeneration: {
          launcherTitle: message('选择生图密钥'),
          launcherDescription: message('选择用于新标签页生图工作区的 API 密钥。'),
          keyLabel: message('API 密钥'),
          loading: message('正在加载生图权限…'),
          loadError: message('加载 API 密钥失败'),
          noKeyTitle: message('暂无可用的生图密钥'),
          noKeyDescription: message('请先创建已启用生图权限的 OpenAI API 密钥。'),
          manageKeys: message('管理 API 密钥'),
          retry: message('重试'),
          openNewTab: message('在新标签页打开'),
          opening: message('正在打开…'),
          disabled: message('管理员已关闭生图入口。'),
          userChanged: message('账号已切换，请重新打开生图入口。'),
          popupBlocked: message('浏览器拦截了新标签页，请允许弹窗后重试。'),
          popupClosed: message('生图标签页已关闭，请重试。'),
          connectionTimeout: message('生图工作区连接超时，请重试。'),
          configurationFailed: message('生图工作区配置失败，请重试。'),
        },
      },
    },
  })
}

const wrappers: VueWrapper[] = []

function mountLauncher() {
  const wrapper = mount(ImageGenerationLauncher, {
    attachTo: document.body,
    global: {
      plugins: [i18n()],
      stubs: {
        BaseDialog: BaseDialogStub,
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
      },
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

beforeAll(async () => {
  ImageGenerationLauncher = (await import('../ImageGenerationLauncher.vue')).default
  useImageGenerationLauncherStore = (await import('@/stores/imageGenerationLauncher')).useImageGenerationLauncherStore
})

beforeEach(() => {
  setActivePinia(createPinia())
  document.documentElement.classList.remove('dark')
  authStore.isAuthenticated = true
  authStore.user = { id: 42 }
  appStore.cachedPublicSettings = { image_generation_enabled: true }
  appStore.showError.mockReset()
  accessState = {
    imageGenerationKeys: ref([key(1), key(2)]),
    canUseImageGeneration: ref(true),
    imageGenerationAccessLoaded: ref(true),
    imageGenerationAccessLoading: ref(false),
    imageGenerationAccessError: ref(null),
    refreshImageGenerationAccess: vi.fn(async () => accessState.imageGenerationKeys.value),
    clearImageGenerationKeys: vi.fn(() => {
      accessState.imageGenerationKeys.value = []
      accessState.canUseImageGeneration.value = false
      accessState.imageGenerationAccessLoaded.value = false
    }),
  }
  openPopup.mockReset()
})

afterEach(() => {
  for (const wrapper of wrappers.splice(0)) wrapper.unmount()
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('ImageGenerationLauncher', () => {
  it('keeps only modal visibility in Pinia and loads eligible keys when opened', async () => {
    const wrapper = mountLauncher()
    const launcher = useImageGenerationLauncherStore()

    expect(launcher.$state).toEqual({ isOpen: false })
    expect(launcher.$state).not.toHaveProperty('apiKey')

    launcher.open()
    await flushPromises()

    expect(accessState.refreshImageGenerationAccess).toHaveBeenCalledOnce()
    expect(wrapper.get('[role="dialog"]').text()).toContain('选择生图密钥')
    expect(wrapper.findAll('#image-generation-launcher-key option')).toHaveLength(2)
  })

  it('renders load-error and no-key states with retry and key management actions', async () => {
    accessState.imageGenerationAccessError.value = 'network unavailable'
    let wrapper = mountLauncher()
    let launcher = useImageGenerationLauncherStore()
    launcher.open()
    await flushPromises()

    expect(wrapper.text()).toContain('加载 API 密钥失败')
    await wrapper.get('[data-testid="image-generation-access-retry"]').trigger('click')
    expect(accessState.refreshImageGenerationAccess).toHaveBeenLastCalledWith(true)
    wrapper.unmount()

    setActivePinia(createPinia())
    accessState.imageGenerationAccessError.value = null
    accessState.imageGenerationKeys.value = []
    accessState.canUseImageGeneration.value = false
    accessState.imageGenerationAccessLoaded.value = true
    accessState.imageGenerationAccessLoading.value = false
    wrapper = mountLauncher()
    launcher = useImageGenerationLauncherStore()
    launcher.open()
    await flushPromises()

    expect(wrapper.text()).toContain('暂无可用的生图密钥')
    expect(wrapper.get('a').attributes('href')).toBe('/keys')
  })

  it('opens synchronously with the selected key and closes after configuration succeeds', async () => {
    const deferred = createDeferred<void>()
    const closed = createDeferred<void>()
    const abort = vi.fn()
    openPopup.mockReturnValue({ configured: deferred.promise, closed: closed.promise, abort })
    const wrapper = mountLauncher()
    const launcher = useImageGenerationLauncherStore()
    launcher.open()
    await flushPromises()

    await wrapper.get('#image-generation-launcher-key').setValue('2')
    const trigger = wrapper.get('[data-testid="image-generation-open"]').trigger('click')

    expect(openPopup).toHaveBeenCalledOnce()
    expect(openPopup).toHaveBeenCalledWith(expect.objectContaining({
      apiKey: 'sk-secret-2',
      apiKeyId: 2,
      apiKeyName: 'Image Key 2',
      storageScope: '42',
      locale: 'zh-CN',
      theme: 'light',
    }))
    expect(JSON.stringify(openPopup.mock.calls[0])).not.toContain('sk-secret-1')

    deferred.resolve()
    await trigger
    await flushPromises()
    expect(launcher.isOpen).toBe(false)
    expect(accessState.clearImageGenerationKeys).toHaveBeenCalled()
    expect(abort).not.toHaveBeenCalled()

    authStore.user = { id: 43 }
    await flushPromises()
    expect(abort).toHaveBeenCalledOnce()
  })

  it('keeps the modal open and shows a friendly popup error without exposing credentials', async () => {
    const deferred = createDeferred<void>()
    openPopup.mockReturnValue({
      configured: deferred.promise,
      closed: Promise.resolve(),
      abort: vi.fn(),
    })
    const wrapper = mountLauncher()
    const launcher = useImageGenerationLauncherStore()
    launcher.open()
    await flushPromises()

    await wrapper.get('[data-testid="image-generation-open"]').trigger('click')
    deferred.reject(new PopupLaunchError('popup_blocked'))
    await flushPromises()

    expect(launcher.isOpen).toBe(true)
    expect(wrapper.get('[role="alert"]').text()).toContain('浏览器拦截了新标签页')
    expect(wrapper.get('[role="alert"]').text()).not.toContain('sk-secret')
  })

  it('closes and aborts the active launch when the feature is disabled or the user changes', async () => {
    const deferred = createDeferred<void>()
    const abort = vi.fn()
    openPopup.mockReturnValue({ configured: deferred.promise, closed: Promise.resolve(), abort })
    const wrapper = mountLauncher()
    const launcher = useImageGenerationLauncherStore()
    launcher.open()
    await flushPromises()
    await wrapper.get('[data-testid="image-generation-open"]').trigger('click')

    authStore.user = { id: 43 }
    await flushPromises()
    expect(abort).toHaveBeenCalledOnce()
    expect(launcher.isOpen).toBe(false)

    appStore.showError.mockReset()
    launcher.open()
    await flushPromises()
    appStore.cachedPublicSettings = { image_generation_enabled: false }
    await flushPromises()
    expect(launcher.isOpen).toBe(false)
    expect(appStore.showError).toHaveBeenCalledWith('管理员已关闭生图入口。')
  })
})
