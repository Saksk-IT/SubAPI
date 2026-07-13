import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  IMAGE_PLAYGROUND_PROTOCOL,
  IMAGE_PLAYGROUND_PROTOCOL_VERSION,
  buildConfigurationErrorAckMessage,
  buildConnectedMessage,
} from '../bridge'
import {
  PopupLaunchError,
  openImagePlaygroundPopup,
  type ImagePlaygroundPopupSession,
} from '../popupBridge'

const API_KEY = 'sk-popup-super-secret'
const ORIGIN = window.location.origin

class FakeMessagePort {
  readonly messages: unknown[] = []
  closed = false
  started = false
  private readonly listeners = new Set<(event: MessageEvent) => void>()

  postMessage(message: unknown) {
    this.messages.push(message)
  }

  start() {
    this.started = true
  }

  addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    if (type === 'message' && typeof listener === 'function') {
      this.listeners.add(listener as (event: MessageEvent) => void)
    }
  }

  removeEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    if (type === 'message' && typeof listener === 'function') {
      this.listeners.delete(listener as (event: MessageEvent) => void)
    }
  }

  emitMessage(data: unknown) {
    const event = new Event('message') as MessageEvent
    Object.defineProperty(event, 'data', { value: data })
    for (const listener of this.listeners) listener(event)
  }

  listenerCount() {
    return this.listeners.size
  }

  close() {
    this.closed = true
  }
}

class FakeMessageChannel {
  static instances: FakeMessageChannel[] = []
  readonly port1 = new FakeMessagePort()
  readonly port2 = new FakeMessagePort()

  constructor() {
    FakeMessageChannel.instances.push(this)
  }
}

class FakePopup {
  closed = false
  readonly postMessage = vi.fn()
  readonly close = vi.fn(() => {
    this.closed = true
  })
}

let uuidCounter = 0
let addEventListenerSpy: ReturnType<typeof vi.spyOn>
let removeEventListenerSpy: ReturnType<typeof vi.spyOn>
const sessions: ImagePlaygroundPopupSession[] = []

function readyMessage(nonce: string, overrides: Record<string, unknown> = {}) {
  return {
    protocol: IMAGE_PLAYGROUND_PROTOCOL,
    version: IMAGE_PLAYGROUND_PROTOCOL_VERSION,
    type: 'ready',
    nonce,
    ...overrides,
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

function dispatchWindowMessage(options: {
  popup: FakePopup
  nonce: string
  origin?: string
  source?: unknown
  data?: unknown
}) {
  const event = new Event('message') as MessageEvent
  Object.defineProperties(event, {
    origin: { value: options.origin ?? ORIGIN },
    source: { value: options.source ?? options.popup },
    data: { value: options.data ?? readyMessage(options.nonce) },
  })
  window.dispatchEvent(event)
}

function nonceFromOpenCall(openWindow: ReturnType<typeof vi.fn>, index = 0) {
  const name = openWindow.mock.calls[index]?.[1]
  if (typeof name !== 'string') throw new Error('Expected a popup frame name')
  return name.replace('sub2api-image-playground:', '')
}

function requestIdFromConfigure(channel: FakeMessageChannel) {
  const message = channel.port1.messages[0] as { requestId?: unknown } | undefined
  if (!message || typeof message.requestId !== 'number') {
    throw new Error('Expected a configure request')
  }
  return message.requestId
}

function openSession(popup: FakePopup | null, overrides: Record<string, unknown> = {}) {
  const openWindow = vi.fn(() => popup as unknown as Window | null)
  const session = openImagePlaygroundPopup({
    apiKey: API_KEY,
    apiKeyId: 7,
    apiKeyName: 'Image Key',
    storageScope: '42',
    locale: 'zh-CN',
    theme: 'dark',
    timeoutMs: 8_000,
    openWindow,
    ...overrides,
  })
  sessions.push(session)
  void session.configured.catch(() => undefined)
  return { openWindow, session }
}

function connectPopup(popup: FakePopup, nonce: string) {
  dispatchWindowMessage({ popup, nonce })
  const channel = FakeMessageChannel.instances.at(-1)
  if (!channel) throw new Error('Expected a message channel')
  channel.port1.emitMessage(buildConnectedMessage(nonce))
  return channel
}

function messageListenerCalls(spy: ReturnType<typeof vi.spyOn>) {
  return spy.mock.calls.filter(([type]) => type === 'message')
}

beforeEach(() => {
  uuidCounter = 0
  FakeMessageChannel.instances = []
  vi.useFakeTimers()
  vi.stubGlobal('MessageChannel', FakeMessageChannel)
  vi.stubGlobal('crypto', {
    ...globalThis.crypto,
    randomUUID: vi.fn(() => `123e4567-e89b-42d3-a456-${String(++uuidCounter).padStart(12, '0')}`),
  })
  localStorage.clear()
  sessionStorage.clear()
  addEventListenerSpy = vi.spyOn(window, 'addEventListener')
  removeEventListenerSpy = vi.spyOn(window, 'removeEventListener')
})

afterEach(() => {
  for (const session of sessions.splice(0)) session.abort()
  vi.clearAllTimers()
  vi.useRealTimers()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('image playground popup bridge', () => {
  it('rejects a blocked popup without registering listeners or timers', async () => {
    const { openWindow, session } = openSession(null)

    expect(openWindow).toHaveBeenCalledOnce()
    expect(openWindow).toHaveBeenCalledWith(
      '/image-playground/',
      'sub2api-image-playground:123e4567-e89b-42d3-a456-000000000001',
    )
    await expect(session.configured).rejects.toMatchObject({
      name: 'PopupLaunchError',
      code: 'popup_blocked',
    })
    expect(messageListenerCalls(addEventListenerSpy)).toHaveLength(0)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('opens synchronously with a constant URL and nonce-only name without serializing the key', () => {
    const popup = new FakePopup()
    const { openWindow, session } = openSession(popup)

    expect(openWindow).toHaveBeenCalledOnce()
    const serializedCall = JSON.stringify(openWindow.mock.calls)
    expect(serializedCall).not.toContain(API_KEY)
    expect(openWindow.mock.calls[0]).toHaveLength(2)
    expect(openWindow.mock.calls[0]?.[0]).toBe('/image-playground/')
    expect(openWindow.mock.calls[0]?.[1]).toMatch(/^sub2api-image-playground:[0-9a-f-]+$/)
    expect(JSON.stringify(session)).not.toContain(API_KEY)
    expect(localStorage.length).toBe(0)
    expect(sessionStorage.length).toBe(0)
  })

  it('snapshots the session configuration before the popup opener can mutate its input', () => {
    const popup = new FakePopup()
    const launchOptions = {
      apiKey: API_KEY,
      apiKeyId: 7,
      apiKeyName: 'Original Key',
      storageScope: '42',
      locale: 'zh-CN',
      theme: 'dark' as const,
      timeoutMs: 8_000,
      openWindow: vi.fn(() => {
        launchOptions.apiKey = 'sk-mutated-during-open'
        launchOptions.apiKeyName = 'Mutated Key'
        return popup as unknown as Window
      }),
    }

    const session = openImagePlaygroundPopup(launchOptions)
    sessions.push(session)
    void session.configured.catch(() => undefined)
    const nonce = nonceFromOpenCall(launchOptions.openWindow)
    const channel = connectPopup(popup, nonce)

    expect(channel.port1.messages[0]).toMatchObject({
      payload: {
        apiKey: API_KEY,
        apiKeyName: 'Original Key',
      },
    })
  })

  it('snapshots the timeout before the popup opener can mutate its input', async () => {
    const popup = new FakePopup()
    const launchOptions = {
      apiKey: API_KEY,
      apiKeyId: 7,
      apiKeyName: 'Image Key',
      storageScope: '42',
      timeoutMs: 1_000,
      openWindow: vi.fn(() => {
        launchOptions.timeoutMs = 9_000
        return popup as unknown as Window
      }),
    }
    const session = openImagePlaygroundPopup(launchOptions)
    sessions.push(session)
    let errorCode: string | undefined
    void session.configured.catch((error: PopupLaunchError) => {
      errorCode = error.code
    })

    await vi.advanceTimersByTimeAsync(1_000)

    expect(errorCode).toBe('connection_timeout')
  })

  it('rejects a throwing option getter safely before opening a popup', async () => {
    const popup = new FakePopup()
    const openWindow = vi.fn(() => popup as unknown as Window)
    const launchOptions = {
      get apiKey(): string {
        throw new Error(`Getter exposed ${API_KEY}`)
      },
      apiKeyId: 7,
      apiKeyName: 'Image Key',
      storageScope: '42',
      timeoutMs: 1_000,
      openWindow,
    }
    let session: ImagePlaygroundPopupSession | undefined

    expect(() => {
      session = openImagePlaygroundPopup(launchOptions)
    }).not.toThrow()
    expect(openWindow).not.toHaveBeenCalled()
    const error = await session!.configured.catch((reason: unknown) => reason)
    expect(error).toBeInstanceOf(PopupLaunchError)
    expect(error).toMatchObject({ code: 'configuration_failed' })
    expect(String(error)).not.toContain(API_KEY)
    expect(messageListenerCalls(addEventListenerSpy)).toHaveLength(0)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('ignores invalid ready events and transfers exactly one port after the exact ready event', () => {
    const popup = new FakePopup()
    const otherPopup = new FakePopup()
    const { openWindow } = openSession(popup)
    const nonce = nonceFromOpenCall(openWindow)

    for (const invalid of [
      { origin: 'https://evil.example.com' },
      { source: otherPopup },
      { data: readyMessage('123e4567-e89b-42d3-a456-999999999999') },
      { data: readyMessage(nonce, { version: 2 }) },
      { data: readyMessage(nonce, { unexpected: true }) },
      { data: configuredAck(nonce, 1) },
    ]) {
      dispatchWindowMessage({ popup, nonce, ...invalid })
    }

    expect(FakeMessageChannel.instances).toHaveLength(0)
    dispatchWindowMessage({ popup, nonce })

    expect(FakeMessageChannel.instances).toHaveLength(1)
    const channel = FakeMessageChannel.instances[0]
    expect(popup.postMessage).toHaveBeenCalledOnce()
    expect(popup.postMessage).toHaveBeenCalledWith(
      {
        protocol: IMAGE_PLAYGROUND_PROTOCOL,
        version: IMAGE_PLAYGROUND_PROTOCOL_VERSION,
        type: 'connect',
        nonce,
      },
      ORIGIN,
      [channel.port2],
    )
    expect(popup.postMessage.mock.calls[0]?.[2]).toHaveLength(1)
    expect(channel.port1.started).toBe(true)
  })

  it('waits for exact connected and sends configure only once despite duplicate events', () => {
    const popup = new FakePopup()
    const { openWindow } = openSession(popup)
    const nonce = nonceFromOpenCall(openWindow)

    dispatchWindowMessage({ popup, nonce })
    dispatchWindowMessage({ popup, nonce })
    const channel = FakeMessageChannel.instances[0]
    expect(FakeMessageChannel.instances).toHaveLength(1)
    expect(channel.port1.messages).toHaveLength(0)

    channel.port1.emitMessage(configuredAck(nonce, 1))
    channel.port1.emitMessage({ ...buildConnectedMessage(nonce), nonce: 'wrong' })
    channel.port1.emitMessage({ ...buildConnectedMessage(nonce), unexpected: true })
    expect(channel.port1.messages).toHaveLength(0)

    channel.port1.emitMessage(buildConnectedMessage(nonce))
    channel.port1.emitMessage(buildConnectedMessage(nonce))
    dispatchWindowMessage({ popup, nonce })

    expect(channel.port1.messages).toHaveLength(1)
    expect(channel.port1.messages[0]).toMatchObject({
      protocol: IMAGE_PLAYGROUND_PROTOCOL,
      version: IMAGE_PLAYGROUND_PROTOCOL_VERSION,
      type: 'configure',
      nonce,
      requestId: 1,
      payload: {
        apiKey: API_KEY,
        apiKeyId: 7,
        apiKeyName: 'Image Key',
        storageScope: '42',
        locale: 'zh-CN',
        theme: 'dark',
      },
    })
  })

  it('resolves only for the exact configured acknowledgement and cleans up everything', async () => {
    const popup = new FakePopup()
    const { openWindow, session } = openSession(popup)
    const nonce = nonceFromOpenCall(openWindow)
    const channel = connectPopup(popup, nonce)
    const requestId = requestIdFromConfigure(channel)
    let settled = false
    void session.configured.finally(() => {
      settled = true
    })

    for (const invalid of [
      configuredAck(nonce, requestId + 1),
      configuredAck('123e4567-e89b-42d3-a456-999999999999', requestId),
      configuredAck(nonce, requestId, { version: 2 }),
      configuredAck(nonce, requestId, { unexpected: true }),
      { ...configuredAck(nonce, requestId), status: 'error' },
    ]) {
      channel.port1.emitMessage(invalid)
    }
    await Promise.resolve()
    expect(settled).toBe(false)

    channel.port1.emitMessage(configuredAck(nonce, requestId))
    await expect(session.configured).resolves.toBeUndefined()

    expect(messageListenerCalls(removeEventListenerSpy)).toHaveLength(1)
    expect(channel.port1.listenerCount()).toBe(0)
    expect(channel.port1.closed).toBe(true)
    expect(channel.port2.closed).toBe(true)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('rejects an exact child error with a sanitized configuration failure and cleans up', async () => {
    const popup = new FakePopup()
    const { openWindow, session } = openSession(popup)
    const nonce = nonceFromOpenCall(openWindow)
    const channel = connectPopup(popup, nonce)
    const requestId = requestIdFromConfigure(channel)

    channel.port1.emitMessage(buildConfigurationErrorAckMessage({
      nonce,
      requestId,
      errorCode: 'child_rejected',
      errorMessage: `Do not expose ${API_KEY}`,
    }))

    const error = await session.configured.catch((reason: unknown) => reason)
    expect(error).toBeInstanceOf(PopupLaunchError)
    expect(error).toMatchObject({ code: 'configuration_failed' })
    expect(String(error)).not.toContain(API_KEY)
    expect(String(error)).not.toContain('child_rejected')
    expect(messageListenerCalls(removeEventListenerSpy)).toHaveLength(1)
    expect(channel.port1.listenerCount()).toBe(0)
    expect(channel.port1.closed).toBe(true)
    expect(channel.port2.closed).toBe(true)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('rejects on timeout and removes its listener and polling timer', async () => {
    const popup = new FakePopup()
    const { session } = openSession(popup, { timeoutMs: 1_000 })

    await vi.advanceTimersByTimeAsync(1_000)

    await expect(session.configured).rejects.toMatchObject({ code: 'connection_timeout' })
    expect(messageListenerCalls(removeEventListenerSpy)).toHaveLength(1)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('reports popup_closed when the popup closes just before the timeout callback', async () => {
    const popup = new FakePopup()
    const { session } = openSession(popup, { timeoutMs: 1_000 })

    await vi.advanceTimersByTimeAsync(900)
    popup.closed = true
    await vi.advanceTimersByTimeAsync(100)

    await expect(session.configured).rejects.toMatchObject({ code: 'popup_closed' })
    expect(messageListenerCalls(removeEventListenerSpy)).toHaveLength(1)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('sanitizes a closed getter failure at the timeout boundary and still cleans up', async () => {
    const popup = new FakePopup()
    let throwOnClosedRead = false
    Object.defineProperty(popup, 'closed', {
      configurable: true,
      get() {
        if (throwOnClosedRead) throw new Error(`Getter exposed ${API_KEY}`)
        return false
      },
    })
    const { session } = openSession(popup, { timeoutMs: 1_000 })

    await vi.advanceTimersByTimeAsync(900)
    throwOnClosedRead = true
    await vi.advanceTimersByTimeAsync(100)

    const error = await session.configured.catch((reason: unknown) => reason)
    expect(error).toBeInstanceOf(PopupLaunchError)
    expect(error).toMatchObject({ code: 'popup_closed' })
    expect(String(error)).not.toContain(API_KEY)
    expect(messageListenerCalls(removeEventListenerSpy)).toHaveLength(1)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('rejects when the popup closes before configuration', async () => {
    const popup = new FakePopup()
    const { session } = openSession(popup)
    popup.closed = true

    await vi.advanceTimersByTimeAsync(250)

    await expect(session.configured).rejects.toMatchObject({ code: 'popup_closed' })
    expect(messageListenerCalls(removeEventListenerSpy)).toHaveLength(1)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('abort closes the popup, rejects once and cleans listeners, timers and ports', async () => {
    const popup = new FakePopup()
    const { openWindow, session } = openSession(popup)
    const nonce = nonceFromOpenCall(openWindow)
    const channel = connectPopup(popup, nonce)

    session.abort()
    session.abort()

    await expect(session.configured).rejects.toMatchObject({ code: 'aborted' })
    expect(popup.close).toHaveBeenCalledOnce()
    expect(messageListenerCalls(removeEventListenerSpy)).toHaveLength(1)
    expect(channel.port1.listenerCount()).toBe(0)
    expect(channel.port1.closed).toBe(true)
    expect(channel.port2.closed).toBe(true)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('isolates two concurrent popup sessions by popup source and nonce', async () => {
    const popupA = new FakePopup()
    const popupB = new FakePopup()
    const sessionA = openSession(popupA, { apiKey: 'sk-session-a', apiKeyId: 1 })
    const sessionB = openSession(popupB, { apiKey: 'sk-session-b', apiKeyId: 2 })
    const nonceA = nonceFromOpenCall(sessionA.openWindow)
    const nonceB = nonceFromOpenCall(sessionB.openWindow)

    dispatchWindowMessage({ popup: popupA, nonce: nonceB })
    dispatchWindowMessage({ popup: popupB, nonce: nonceA })
    expect(FakeMessageChannel.instances).toHaveLength(0)

    const channelA = connectPopup(popupA, nonceA)
    expect(FakeMessageChannel.instances).toHaveLength(1)
    expect(JSON.stringify(channelA.port1.messages)).toContain('sk-session-a')
    expect(JSON.stringify(channelA.port1.messages)).not.toContain('sk-session-b')
    channelA.port1.emitMessage(configuredAck(nonceA, requestIdFromConfigure(channelA)))
    await expect(sessionA.session.configured).resolves.toBeUndefined()

    const channelB = connectPopup(popupB, nonceB)
    expect(FakeMessageChannel.instances).toHaveLength(2)
    expect(JSON.stringify(channelB.port1.messages)).toContain('sk-session-b')
    expect(JSON.stringify(channelB.port1.messages)).not.toContain('sk-session-a')
    channelB.port1.emitMessage(configuredAck(nonceB, requestIdFromConfigure(channelB)))
    await expect(sessionB.session.configured).resolves.toBeUndefined()
  })
})
