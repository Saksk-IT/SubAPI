import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { reactive, ref } from 'vue'

import type { ApiKey } from '@/types'
import {
  IMAGE_PLAYGROUND_PROTOCOL,
  IMAGE_PLAYGROUND_PROTOCOL_VERSION,
} from '@/features/imagePlayground/bridge'

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
const authStore = reactive({
  user: { id: 42 },
})

vi.mock('@/composables/useImageGenerationAccess', () => ({
  useImageGenerationKeys: () => accessState,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

class FakeMessagePort {
  messages: unknown[] = []
  closed = false
  started = false
  private readonly messageListeners = new Set<(event: MessageEvent) => void>()

  postMessage(message: unknown) {
    this.messages.push(message)
  }

  start() {
    this.started = true
  }

  addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    if (type !== 'message' || typeof listener !== 'function') return
    this.messageListeners.add(listener as (event: MessageEvent) => void)
  }

  removeEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    if (type !== 'message' || typeof listener !== 'function') return
    this.messageListeners.delete(listener as (event: MessageEvent) => void)
  }

  emitMessage(data: unknown) {
    const event = new MessageEvent('message', { data })
    for (const listener of this.messageListeners) listener(event)
  }

  close() {
    this.closed = true
  }
}

class FakeMessageChannel {
  static instances: FakeMessageChannel[] = []
  port1 = new FakeMessagePort()
  port2 = new FakeMessagePort()

  constructor() {
    FakeMessageChannel.instances.push(this)
  }
}

let ImageGenerationView: object
const wrappers: VueWrapper[] = []
let uuidCounter = 0

function key(id: number): ApiKey {
  return {
    id,
    user_id: 42,
    key: `sk-secret-${id}`,
    name: `Image Key ${id}`,
    group_id: id,
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
    group: {
      id,
      name: `OpenAI ${id}`,
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
    } as NonNullable<ApiKey['group']>,
  }
}

function i18n() {
  const message = (value: string) => () => value
  return createI18n({
    legacy: false,
    locale: 'zh-CN',
    messages: {
      'zh-CN': {
        imageGeneration: {
          title: message('生图'),
          subtitle: message('使用已授权的 API 密钥。'),
          keyLabel: message('API 密钥'),
          loading: message('正在加载生图权限…'),
          loadError: message('加载 API 密钥失败'),
          noKeyTitle: message('暂无可用的生图密钥'),
          noKeyDescription: message('请先创建已启用生图权限的 OpenAI API 密钥。'),
          manageKeys: message('管理 API 密钥'),
          retry: message('重试'),
          waiting: message('正在连接生图工作区…'),
          timeoutTitle: message('生图工作区连接超时'),
          timeoutDescription: message('请重试连接。'),
          connectionErrorTitle: message('无法连接生图工作区'),
          connectionErrorDescription: message('请重试连接。'),
          frameTitle: message('生图工作区'),
        },
      },
    },
  })
}

function mountView() {
  const wrapper = mount(ImageGenerationView, {
    attachTo: document.body,
    global: {
      plugins: [i18n()],
      stubs: {
        AppLayout: {
          name: 'AppLayoutStub',
          props: ['hideFirstRechargeBanner'],
          template: '<div><slot /></div>',
        },
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
      },
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

function readyMessage(nonce: string) {
  return {
    protocol: IMAGE_PLAYGROUND_PROTOCOL,
    version: IMAGE_PLAYGROUND_PROTOCOL_VERSION,
    type: 'ready',
    nonce,
  }
}

function configuredAck(nonce: string, requestId: number, overrides: Record<string, unknown> = {}) {
  return {
    protocol: IMAGE_PLAYGROUND_PROTOCOL,
    version: IMAGE_PLAYGROUND_PROTOCOL_VERSION,
    type: 'ack',
    nonce,
    status: 'configured',
    requestId,
    ...overrides,
  }
}

function configureRequestId(channel: FakeMessageChannel, index = -1) {
  const message = channel.port1.messages.at(index) as { requestId?: unknown } | undefined
  if (!message || typeof message.requestId !== 'number') {
    throw new Error('Expected a configure message with requestId')
  }
  return message.requestId
}

function frameNonce(wrapper: VueWrapper) {
  return wrapper.get('iframe').attributes('name').replace('sub2api-image-playground:', '')
}

async function connect(wrapper: VueWrapper) {
  const frame = wrapper.get('iframe').element as HTMLIFrameElement
  const postMessage = vi.spyOn(frame.contentWindow!, 'postMessage')
  await wrapper.get('iframe').trigger('load')
  window.dispatchEvent(new MessageEvent('message', {
    origin: window.location.origin,
    source: frame.contentWindow,
    data: readyMessage(frameNonce(wrapper)),
  }))
  await flushPromises()
  return { frame, postMessage, channel: FakeMessageChannel.instances.at(-1)! }
}

beforeAll(async () => {
  ImageGenerationView = (await import('../ImageGenerationView.vue')).default
})

beforeEach(() => {
  uuidCounter = 0
  FakeMessageChannel.instances = []
  document.documentElement.classList.remove('dark')
  vi.stubGlobal('MessageChannel', FakeMessageChannel)
  vi.stubGlobal('crypto', {
    ...globalThis.crypto,
    randomUUID: vi.fn(() => `123e4567-e89b-42d3-a456-${String(++uuidCounter).padStart(12, '0')}`),
  })
  accessState = {
    imageGenerationKeys: ref([key(1), key(2)]),
    canUseImageGeneration: ref(true),
    imageGenerationAccessLoaded: ref(true),
    imageGenerationAccessLoading: ref(false),
    imageGenerationAccessError: ref(null),
    refreshImageGenerationAccess: vi.fn(async () => accessState.imageGenerationKeys.value),
    clearImageGenerationKeys: vi.fn(() => {
      accessState.imageGenerationKeys.value = []
    }),
  }
  authStore.user = { id: 42 }
})

afterEach(() => {
  for (const wrapper of wrappers.splice(0)) {
    if (wrapper.exists()) wrapper.unmount()
  }
  document.body.innerHTML = ''
  document.documentElement.classList.remove('dark')
  vi.useRealTimers()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('ImageGenerationView', () => {
  it('renders loading, load-error and no-key states with retry actions', async () => {
    accessState.imageGenerationAccessLoading.value = true
    accessState.imageGenerationAccessLoaded.value = false
    let wrapper = mountView()
    expect(wrapper.text()).toContain('正在加载生图权限')
    expect(wrapper.find('iframe').exists()).toBe(false)
    wrapper.unmount()

    accessState.imageGenerationAccessLoading.value = false
    accessState.imageGenerationAccessLoaded.value = true
    accessState.imageGenerationAccessError.value = 'network unavailable'
    wrapper = mountView()
    expect(wrapper.text()).toContain('加载 API 密钥失败')
    await wrapper.get('[data-testid="access-retry"]').trigger('click')
    expect(accessState.refreshImageGenerationAccess).toHaveBeenCalledWith(true)
    wrapper.unmount()

    accessState.imageGenerationAccessError.value = null
    accessState.imageGenerationKeys.value = []
    accessState.canUseImageGeneration.value = false
    wrapper = mountView()
    expect(wrapper.text()).toContain('暂无可用的生图密钥')
    expect(wrapper.get('a').attributes('href')).toBe('/keys')
    expect(wrapper.find('iframe').exists()).toBe(false)
  })

  it('uses a constant iframe URL, nonce-only name and responsive host controls', async () => {
    const wrapper = mountView()
    await flushPromises()

    const host = wrapper.get('[data-testid="image-generation-host"]')
    expect(wrapper.getComponent({ name: 'AppLayoutStub' }).props('hideFirstRechargeBanner')).toBe(true)
    const frame = wrapper.get('iframe')
    expect(host.classes()).toContain('min-w-0')
    expect(host.classes()).toContain('h-[calc(100dvh-6rem)]')
    expect(frame.attributes('src')).toBe('/image-playground/')
    expect(frame.attributes('src')).not.toContain('?')
    expect(frame.attributes('name')).toMatch(/^sub2api-image-playground:[0-9a-f-]+$/)
    expect(frame.attributes('sandbox')).toContain('allow-scripts')
    expect(frame.attributes('sandbox')).toContain('allow-same-origin')
    expect(frame.attributes('sandbox')).not.toContain('allow-forms')
    expect(frame.attributes('sandbox')).not.toContain('allow-modals')
    expect(frame.attributes('referrerpolicy')).toBe('no-referrer')
    expect(frame.classes()).toContain('min-h-0')
    expect(frame.element.parentElement?.classList.contains('min-h-0')).toBe(true)
    expect(wrapper.get('label').attributes('for')).toBe('image-generation-key')
    expect(wrapper.get('[role="status"]').attributes('aria-live')).toBe('polite')
  })

  it('defers a trusted ready message until iframe load initialization and connects only once', async () => {
    const wrapper = mountView()
    await flushPromises()
    const frame = wrapper.get('iframe').element as HTMLIFrameElement
    vi.spyOn(frame.contentWindow!, 'postMessage')
    const nonce = frameNonce(wrapper)

    window.dispatchEvent(new MessageEvent('message', {
      origin: window.location.origin,
      source: frame.contentWindow,
      data: readyMessage(nonce),
    }))
    await flushPromises()
    expect(FakeMessageChannel.instances).toHaveLength(0)

    await wrapper.get('iframe').trigger('load')
    await flushPromises()
    expect(FakeMessageChannel.instances).toHaveLength(1)

    window.dispatchEvent(new MessageEvent('message', {
      origin: window.location.origin,
      source: frame.contentWindow,
      data: readyMessage(nonce),
    }))
    await flushPromises()
    expect(FakeMessageChannel.instances).toHaveLength(1)
  })

  it('does not cache a ready message with the wrong origin, source or nonce before iframe load', async () => {
    const wrapper = mountView()
    await flushPromises()
    const frame = wrapper.get('iframe').element as HTMLIFrameElement
    const nonce = frameNonce(wrapper)

    for (const message of [
      { origin: 'https://evil.example.com', source: frame.contentWindow, nonce },
      { origin: window.location.origin, source: window, nonce },
      { origin: window.location.origin, source: frame.contentWindow, nonce: `${nonce}-wrong` },
    ]) {
      window.dispatchEvent(new MessageEvent('message', {
        origin: message.origin,
        source: message.source,
        data: readyMessage(message.nonce),
      }))
    }

    await wrapper.get('iframe').trigger('load')
    await flushPromises()
    expect(FakeMessageChannel.instances).toHaveLength(0)
  })

  it('rejects untrusted ready events and sends only the selected key through a transferred port', async () => {
    const wrapper = mountView()
    await flushPromises()
    const frame = wrapper.get('iframe').element as HTMLIFrameElement
    const postMessage = vi.spyOn(frame.contentWindow!, 'postMessage')
    const nonce = frameNonce(wrapper)
    await wrapper.get('iframe').trigger('load')

    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://evil.example.com',
      source: frame.contentWindow,
      data: readyMessage(nonce),
    }))
    window.dispatchEvent(new MessageEvent('message', {
      origin: window.location.origin,
      source: window,
      data: readyMessage(nonce),
    }))
    await flushPromises()
    expect(FakeMessageChannel.instances).toHaveLength(0)

    window.dispatchEvent(new MessageEvent('message', {
      origin: window.location.origin,
      source: frame.contentWindow,
      data: readyMessage(nonce),
    }))
    await flushPromises()

    expect(FakeMessageChannel.instances).toHaveLength(1)
    const channel = FakeMessageChannel.instances[0]
    expect(postMessage).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'connect', nonce }),
      window.location.origin,
      [channel.port2]
    )
    expect(channel.port1.messages).toHaveLength(1)
    expect(channel.port1.messages[0]).toMatchObject({
      type: 'configure',
      requestId: 1,
      payload: {
        apiKey: 'sk-secret-1',
        apiKeyId: 1,
        storageScope: '42',
      },
    })
    expect(JSON.stringify(channel.port1.messages[0])).not.toContain('sk-secret-2')
  })

  it('waits for an exact configured acknowledgement before showing the workspace as connected', async () => {
    const wrapper = mountView()
    await flushPromises()
    const { channel } = await connect(wrapper)
    const nonce = frameNonce(wrapper)
    const requestId = configureRequestId(channel)

    expect(wrapper.text()).toContain('正在连接生图工作区')
    channel.port1.emitMessage(configuredAck('123e4567-e89b-42d3-a456-426614174999', requestId))
    channel.port1.emitMessage(configuredAck(nonce, requestId, { status: 'cleared' }))
    channel.port1.emitMessage(configuredAck(nonce, requestId, { unexpected: true }))
    await flushPromises()
    expect(wrapper.text()).toContain('正在连接生图工作区')

    channel.port1.emitMessage(configuredAck(nonce, requestId))
    await flushPromises()
    expect(wrapper.text()).not.toContain('正在连接生图工作区')
  })

  it('reuses the established channel and waits for a new acknowledgement after a key switch', async () => {
    const wrapper = mountView()
    await flushPromises()
    const first = await connect(wrapper)
    const firstRequestId = configureRequestId(first.channel)

    await wrapper.get('#image-generation-key').setValue('2')
    await flushPromises()
    const secondRequestId = configureRequestId(first.channel)

    expect(first.channel.port1.closed).toBe(false)
    expect(FakeMessageChannel.instances).toHaveLength(1)
    expect(secondRequestId).toBe(firstRequestId + 1)
    expect(first.channel.port1.messages.at(-1)).toMatchObject({
      type: 'configure',
      requestId: secondRequestId,
      payload: { apiKey: 'sk-secret-2', apiKeyId: 2 },
    })
    expect(wrapper.text()).toContain('正在连接生图工作区')

    first.channel.port1.emitMessage(configuredAck(frameNonce(wrapper), firstRequestId))
    await flushPromises()
    expect(wrapper.text()).toContain('正在连接生图工作区')

    first.channel.port1.emitMessage(configuredAck(frameNonce(wrapper), secondRequestId))
    await flushPromises()
    expect(wrapper.text()).not.toContain('正在连接生图工作区')
  })

  it('rebuilds the iframe channel with the new storage scope when the account changes', async () => {
    const wrapper = mountView()
    await flushPromises()
    const first = await connect(wrapper)
    const firstFrameName = wrapper.get('iframe').attributes('name')
    const nextKey = { ...key(3), user_id: 43, key: 'sk-secret-3' }
    accessState.refreshImageGenerationAccess.mockImplementationOnce(async () => {
      accessState.imageGenerationKeys.value = [nextKey]
      return [nextKey]
    })

    authStore.user = { id: 43 }
    await flushPromises()

    expect(first.channel.port1.messages.at(-1)).toMatchObject({ type: 'clear' })
    expect(first.channel.port1.closed).toBe(true)
    expect(wrapper.get('iframe').attributes('name')).not.toBe(firstFrameName)

    const second = await connect(wrapper)
    expect(FakeMessageChannel.instances).toHaveLength(2)
    expect(second.channel.port1.messages[0]).toMatchObject({
      type: 'configure',
      payload: {
        apiKey: 'sk-secret-3',
        apiKeyId: 3,
        storageScope: '43',
      },
    })
  })

  it('re-sends the selected configuration when the document theme changes', async () => {
    const wrapper = mountView()
    await flushPromises()
    const { channel } = await connect(wrapper)
    const firstRequestId = configureRequestId(channel)
    channel.port1.emitMessage(configuredAck(frameNonce(wrapper), firstRequestId))
    await flushPromises()
    const messageCount = channel.port1.messages.length

    document.documentElement.classList.add('dark')
    await flushPromises()

    expect(channel.port1.messages).toHaveLength(messageCount + 1)
    expect(channel.port1.messages.at(-1)).toMatchObject({
      type: 'configure',
      requestId: firstRequestId + 1,
      payload: { apiKey: 'sk-secret-1', theme: 'dark' },
    })
    expect(FakeMessageChannel.instances).toHaveLength(1)
  })

  it('trims the selected key name before transferring configuration', async () => {
    accessState.imageGenerationKeys.value = [{ ...key(1), name: '  Padded Key  ' }]
    const wrapper = mountView()
    await flushPromises()
    const { channel } = await connect(wrapper)

    expect(channel.port1.messages[0]).toMatchObject({
      type: 'configure',
      payload: { apiKeyName: 'Padded Key' },
    })
  })

  it('falls back to the key id when the selected key name is only whitespace', async () => {
    accessState.imageGenerationKeys.value = [{ ...key(1), name: '   ' }]
    const wrapper = mountView()
    await flushPromises()
    const { channel } = await connect(wrapper)

    expect(channel.port1.messages[0]).toMatchObject({
      type: 'configure',
      payload: { apiKeyName: '#1' },
    })
  })

  it('shows a handshake timeout and can reload with a fresh nonce', async () => {
    vi.useFakeTimers()
    const wrapper = mountView()
    await flushPromises()
    const firstName = wrapper.get('iframe').attributes('name')

    await wrapper.get('iframe').trigger('load')
    await vi.advanceTimersByTimeAsync(8_000)

    expect(wrapper.text()).toContain('生图工作区连接超时')
    await wrapper.get('[data-testid="handshake-retry"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('iframe').attributes('name')).not.toBe(firstName)
  })

  it('clears credentials and closes the port on unmount', async () => {
    const wrapper = mountView()
    await flushPromises()
    const { channel } = await connect(wrapper)

    wrapper.unmount()

    expect(channel.port1.messages.at(-1)).toMatchObject({ type: 'clear' })
    expect(channel.port1.closed).toBe(true)
    expect(accessState.clearImageGenerationKeys).toHaveBeenCalledOnce()
  })
})
