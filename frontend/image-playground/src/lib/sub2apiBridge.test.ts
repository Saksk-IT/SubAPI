import { describe, expect, it, vi } from 'vitest'

import {
  SUB2API_BRIDGE_PROTOCOL,
  SUB2API_BRIDGE_VERSION,
  parseBridgeNonce,
  runAfterWindowLoad,
  startSub2ApiBridge,
  type ManagedConfig,
} from './sub2apiBridge'

const NONCE = '123e4567-e89b-42d3-a456-426614174000'

function config(overrides: Partial<ManagedConfig> = {}): ManagedConfig {
  return {
    apiKey: 'sk-managed-secret',
    apiKeyId: 7,
    apiKeyName: '生图 Key',
    storageScope: '42',
    baseUrl: '/v1',
    profiles: [
      { id: 'sub2api-images', name: 'Images', apiMode: 'images', model: 'gpt-image-2' },
      { id: 'sub2api-responses', name: 'Responses', apiMode: 'responses', model: 'gpt-5.5' },
    ],
    ...overrides,
  }
}

class FakePort {
  onmessage: ((event: { data: unknown }) => void) | null = null
  readonly sent: unknown[] = []
  started = false
  closed = false

  postMessage(message: unknown) {
    this.sent.push(message)
  }

  start() {
    this.started = true
  }

  close() {
    this.closed = true
  }

  dispatch(data: unknown) {
    this.onmessage?.({ data })
  }
}

function harness(name = `sub2api-image-playground:${NONCE}`) {
  const parent = { postMessage: vi.fn() }
  const listeners = new Set<(event: any) => void>()
  const childWindow = {
    name,
    location: { origin: 'https://sub2api.example' },
    parent,
    addEventListener: vi.fn((_type: string, listener: (event: any) => void) => listeners.add(listener)),
    removeEventListener: vi.fn((_type: string, listener: (event: any) => void) => listeners.delete(listener)),
  }
  return {
    childWindow,
    parent,
    dispatch: (event: unknown) => listeners.forEach((listener) => listener(event)),
  }
}

function connectMessage(overrides: Record<string, unknown> = {}) {
  return {
    protocol: SUB2API_BRIDGE_PROTOCOL,
    version: SUB2API_BRIDGE_VERSION,
    type: 'connect',
    nonce: NONCE,
    ...overrides,
  }
}

function configureMessage(payload: unknown = config(), overrides: Record<string, unknown> = {}) {
  return {
    protocol: SUB2API_BRIDGE_PROTOCOL,
    version: SUB2API_BRIDGE_VERSION,
    type: 'configure',
    nonce: NONCE,
    requestId: 1,
    payload,
    ...overrides,
  }
}

describe('Sub2API managed bridge', () => {
  it('waits for child load before starting the handshake', () => {
    let load: () => void = () => undefined
    const start = vi.fn()
    const target = {
      addEventListener: vi.fn((_type: string, listener: () => void) => {
        load = listener
      }),
      removeEventListener: vi.fn(),
    }

    runAfterWindowLoad({ readyState: 'loading' }, target, start)
    expect(start).not.toHaveBeenCalled()

    load()
    expect(start).toHaveBeenCalledOnce()
  })

  it('starts immediately when child load already completed', () => {
    const start = vi.fn()
    const target = { addEventListener: vi.fn(), removeEventListener: vi.fn() }

    runAfterWindowLoad({ readyState: 'complete' }, target, start)

    expect(start).toHaveBeenCalledOnce()
    expect(target.addEventListener).not.toHaveBeenCalled()
  })

  it('reads only a high-entropy nonce from iframe.name', () => {
    expect(parseBridgeNonce(`sub2api-image-playground:${NONCE}`)).toBe(NONCE)
    expect(parseBridgeNonce('sub2api-image-playground:short')).toBeNull()
    expect(parseBridgeNonce(`other:${NONCE}`)).toBeNull()
  })

  it('does not start when opened directly', () => {
    const h = harness('')
    const bridge = startSub2ApiBridge({
      window: h.childWindow,
      onConfigure: vi.fn(),
      onClear: vi.fn(),
    })

    expect(bridge.mode).toBe('direct')
    expect(h.parent.postMessage).not.toHaveBeenCalled()
  })

  it('announces readiness to the exact same origin without secrets', () => {
    const h = harness()
    startSub2ApiBridge({ window: h.childWindow, onConfigure: vi.fn(), onClear: vi.fn() })

    expect(h.parent.postMessage).toHaveBeenCalledWith({
      protocol: SUB2API_BRIDGE_PROTOCOL,
      version: SUB2API_BRIDGE_VERSION,
      type: 'ready',
      nonce: NONCE,
    }, 'https://sub2api.example')
    expect(JSON.stringify(h.parent.postMessage.mock.calls)).not.toContain('sk-managed-secret')
  })

  it.each([
    ['wrong origin', { origin: 'https://evil.example' }],
    ['wrong source', { source: {} }],
    ['wrong version', { data: connectMessage({ version: 2 }) }],
    ['wrong nonce', { data: connectMessage({ nonce: `${NONCE}x` }) }],
    ['unknown field', { data: connectMessage({ extra: true }) }],
    ['missing port', { ports: [] }],
    ['multiple ports', { ports: [new FakePort(), new FakePort()] }],
  ])('rejects connect with %s', (_name, patch) => {
    const h = harness()
    const port = new FakePort()
    startSub2ApiBridge({ window: h.childWindow, onConfigure: vi.fn(), onClear: vi.fn() })

    h.dispatch({
      origin: 'https://sub2api.example',
      source: h.parent,
      data: connectMessage(),
      ports: [port],
      ...patch,
    })

    expect(port.started).toBe(false)
  })

  it.each([
    ['external base URL', config({ baseUrl: 'https://evil.example/v1' as any })],
    ['empty API key', config({ apiKey: '' })],
    ['invalid storage scope', config({ storageScope: '../other-user' })],
    ['missing Images profile', config({ profiles: [config().profiles[1]] })],
    ['fal profile mode', config({ profiles: [{ id: 'fal', name: 'fal', apiMode: 'fal' as any, model: 'x' }, config().profiles[1]] })],
    ['unknown payload field', { ...config(), proxy: true }],
    ['profile secret field', { ...config(), profiles: [{ ...config().profiles[0], apiKey: 'leak' }, config().profiles[1]] }],
  ])('rejects invalid configure payload: %s', async (_name, payload) => {
    const h = harness()
    const port = new FakePort()
    const onConfigure = vi.fn()
    startSub2ApiBridge({ window: h.childWindow, onConfigure, onClear: vi.fn() })
    h.dispatch({ origin: 'https://sub2api.example', source: h.parent, data: connectMessage(), ports: [port] })

    port.dispatch(configureMessage(payload))
    await Promise.resolve()

    expect(onConfigure).not.toHaveBeenCalled()
    expect(port.sent).toEqual([])
  })

  it('treats the API key as an opaque bounded string', async () => {
    const h = harness()
    const port = new FakePort()
    const onConfigure = vi.fn()
    const opaqueKeyConfig = config({ apiKey: ' key-with-significant-spaces ' })
    startSub2ApiBridge({ window: h.childWindow, onConfigure, onClear: vi.fn() })
    h.dispatch({ origin: 'https://sub2api.example', source: h.parent, data: connectMessage(), ports: [port] })

    port.dispatch(configureMessage(opaqueKeyConfig))
    await vi.waitFor(() => expect(onConfigure).toHaveBeenCalledWith(opaqueKeyConfig))
  })

  it.each([
    ['missing request id', { requestId: undefined }],
    ['zero request id', { requestId: 0 }],
    ['fractional request id', { requestId: 1.5 }],
    ['unsafe request id', { requestId: Number.MAX_SAFE_INTEGER + 1 }],
    ['unknown top-level field', { extra: true }],
  ])('rejects configure with %s', async (_name, overrides) => {
    const h = harness()
    const port = new FakePort()
    const onConfigure = vi.fn()
    startSub2ApiBridge({ window: h.childWindow, onConfigure, onClear: vi.fn() })
    h.dispatch({ origin: 'https://sub2api.example', source: h.parent, data: connectMessage(), ports: [port] })

    const message = configureMessage(config(), overrides)
    if ('requestId' in overrides && overrides.requestId === undefined) {
      delete (message as Partial<typeof message>).requestId
    }
    port.dispatch(message)
    await Promise.resolve()

    expect(onConfigure).not.toHaveBeenCalled()
    expect(port.sent).toEqual([])
  })

  it('serializes configure requests and echoes each positive request id in its ack', async () => {
    const h = harness()
    const port = new FakePort()
    let releaseFirst: () => void = () => undefined
    const firstPending = new Promise<void>((resolve) => {
      releaseFirst = resolve
    })
    const applied: string[] = []
    const onConfigure = vi.fn(async (payload: ManagedConfig) => {
      if (payload.apiKey === 'sk-first') await firstPending
      applied.push(payload.apiKey)
    })
    startSub2ApiBridge({ window: h.childWindow, onConfigure, onClear: vi.fn() })
    h.dispatch({ origin: 'https://sub2api.example', source: h.parent, data: connectMessage(), ports: [port] })

    port.dispatch(configureMessage(config({ apiKey: 'sk-first' }), { requestId: 1 }))
    port.dispatch(configureMessage(config({ apiKey: 'sk-second' }), { requestId: 2 }))
    await Promise.resolve()

    expect(onConfigure).toHaveBeenCalledTimes(1)
    expect(port.sent).toEqual([])

    releaseFirst()
    await vi.waitFor(() => expect(onConfigure).toHaveBeenCalledTimes(2))
    await vi.waitFor(() => expect(port.sent).toHaveLength(2))

    expect(applied).toEqual(['sk-first', 'sk-second'])
    expect(applied.at(-1)).toBe('sk-second')
    expect(port.sent).toEqual([
      {
        protocol: SUB2API_BRIDGE_PROTOCOL,
        version: SUB2API_BRIDGE_VERSION,
        type: 'ack',
        nonce: NONCE,
        status: 'configured',
        requestId: 1,
      },
      {
        protocol: SUB2API_BRIDGE_PROTOCOL,
        version: SUB2API_BRIDGE_VERSION,
        type: 'ack',
        nonce: NONCE,
        status: 'configured',
        requestId: 2,
      },
    ])
  })

  it('queues clear behind an in-flight configure operation', async () => {
    const h = harness()
    const port = new FakePort()
    let releaseConfigure: () => void = () => undefined
    const pendingConfigure = new Promise<void>((resolve) => {
      releaseConfigure = resolve
    })
    const onConfigure = vi.fn(() => pendingConfigure)
    const onClear = vi.fn()
    startSub2ApiBridge({ window: h.childWindow, onConfigure, onClear })
    h.dispatch({ origin: 'https://sub2api.example', source: h.parent, data: connectMessage(), ports: [port] })

    port.dispatch(configureMessage())
    port.dispatch({
      protocol: SUB2API_BRIDGE_PROTOCOL,
      version: SUB2API_BRIDGE_VERSION,
      type: 'clear',
      nonce: NONCE,
    })
    await Promise.resolve()

    expect(onConfigure).toHaveBeenCalledOnce()
    expect(onClear).not.toHaveBeenCalled()

    releaseConfigure()
    await vi.waitFor(() => expect(onClear).toHaveBeenCalledOnce())
    expect(port.sent.map((message: any) => message.status)).toEqual(['configured', 'cleared'])
  })

  it('accepts strict configure updates and clear messages on the transferred port', async () => {
    const h = harness()
    const port = new FakePort()
    const onConfigure = vi.fn()
    const onClear = vi.fn()
    const bridge = startSub2ApiBridge({ window: h.childWindow, onConfigure, onClear })
    h.dispatch({ origin: 'https://sub2api.example', source: h.parent, data: connectMessage(), ports: [port] })

    port.dispatch(configureMessage())
    await Promise.resolve()

    expect(port.started).toBe(true)
    expect(onConfigure).toHaveBeenCalledWith(config())
    await expect(bridge.configured).resolves.toEqual(config())
    expect(port.sent).toContainEqual({
      protocol: SUB2API_BRIDGE_PROTOCOL,
      version: SUB2API_BRIDGE_VERSION,
      type: 'ack',
      nonce: NONCE,
      status: 'configured',
      requestId: 1,
    })

    port.dispatch({
      protocol: SUB2API_BRIDGE_PROTOCOL,
      version: SUB2API_BRIDGE_VERSION,
      type: 'clear',
      nonce: NONCE,
    })
    await vi.waitFor(() => expect(onClear).toHaveBeenCalledOnce())
    await vi.waitFor(() => expect(port.sent).toContainEqual(expect.objectContaining({ type: 'ack', status: 'cleared' })))

    bridge.dispose()
    expect(port.closed).toBe(true)
  })
})
