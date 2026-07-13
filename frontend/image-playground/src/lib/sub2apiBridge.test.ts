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

function harness({
  name = `sub2api-image-playground:${NONCE}`,
  withOpener = true,
}: { name?: string; withOpener?: boolean } = {}) {
  const opener = { postMessage: vi.fn() }
  const listeners = new Set<(event: any) => void>()
  const childWindow = {
    name,
    location: { origin: 'https://sub2api.example' },
    opener: withOpener ? opener : null,
    addEventListener: vi.fn((_type: string, listener: (event: any) => void) => listeners.add(listener)),
    removeEventListener: vi.fn((_type: string, listener: (event: any) => void) => listeners.delete(listener)),
  }
  return {
    childWindow,
    opener,
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

function connect(h: ReturnType<typeof harness>, port = new FakePort()) {
  h.dispatch({
    origin: 'https://sub2api.example',
    source: h.opener,
    data: connectMessage(),
    ports: [port],
  })
  return port
}

describe('Sub2API popup bridge', () => {
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

  it('reads only a high-entropy nonce from window.name', () => {
    expect(parseBridgeNonce(`sub2api-image-playground:${NONCE}`)).toBe(NONCE)
    expect(parseBridgeNonce('sub2api-image-playground:short')).toBeNull()
    expect(parseBridgeNonce(`other:${NONCE}`)).toBeNull()
  })

  it.each([
    ['missing name', { name: '', withOpener: true }],
    ['missing opener', { name: `sub2api-image-playground:${NONCE}`, withOpener: false }],
  ])('stays in direct mode with %s', (_name, options) => {
    const h = harness(options)
    const bridge = startSub2ApiBridge({ window: h.childWindow, onConfigure: vi.fn() })

    expect(bridge.mode).toBe('direct')
    expect(h.opener.postMessage).not.toHaveBeenCalled()
    expect(h.childWindow.addEventListener).not.toHaveBeenCalled()
    expect(h.childWindow.name).toBe('')
    expect(h.childWindow.opener).toBeNull()
  })

  it('announces readiness only to the exact opener and same origin without secrets', () => {
    const h = harness()
    startSub2ApiBridge({ window: h.childWindow, onConfigure: vi.fn() })

    expect(h.opener.postMessage).toHaveBeenCalledWith({
      protocol: SUB2API_BRIDGE_PROTOCOL,
      version: SUB2API_BRIDGE_VERSION,
      type: 'ready',
      nonce: NONCE,
    }, 'https://sub2api.example')
    expect(JSON.stringify(h.opener.postMessage.mock.calls)).not.toContain('sk-managed-secret')
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
    startSub2ApiBridge({ window: h.childWindow, onConfigure: vi.fn() })

    h.dispatch({
      origin: 'https://sub2api.example',
      source: h.opener,
      data: connectMessage(),
      ports: [port],
      ...patch,
    })

    expect(port.started).toBe(false)
    expect(h.childWindow.name).toBe(`sub2api-image-playground:${NONCE}`)
    expect(h.childWindow.opener).toBe(h.opener)
  })

  it('locks onto the first valid port, severs opener capabilities and reports connected', () => {
    const h = harness()
    startSub2ApiBridge({ window: h.childWindow, onConfigure: vi.fn() })
    const port = connect(h)
    const secondPort = connect(h)

    expect(port.started).toBe(true)
    expect(port.sent).toEqual([{
      protocol: SUB2API_BRIDGE_PROTOCOL,
      version: SUB2API_BRIDGE_VERSION,
      type: 'connected',
      nonce: NONCE,
    }])
    expect(h.childWindow.name).toBe('')
    expect(h.childWindow.opener).toBeNull()
    expect(h.childWindow.removeEventListener).toHaveBeenCalled()
    expect(secondPort.started).toBe(false)
  })

  it.each([
    ['external base URL', config({ baseUrl: 'https://evil.example/v1' as any })],
    ['empty API key', config({ apiKey: '' })],
    ['invalid storage scope', config({ storageScope: '../other-user' })],
    ['missing Images profile', config({ profiles: [config().profiles[1]] })],
    ['fal profile mode', config({ profiles: [{ id: 'fal', name: 'fal', apiMode: 'fal' as any, model: 'x' }, config().profiles[1]] })],
    ['unknown payload field', { ...config(), proxy: true }],
    ['profile secret field', { ...config(), profiles: [{ ...config().profiles[0], apiKey: 'leak' }, config().profiles[1]] }],
  ])('rejects invalid configure payload with a secret-free error: %s', async (_name, payload) => {
    const h = harness()
    const onConfigure = vi.fn()
    const port = new FakePort()
    startSub2ApiBridge({ window: h.childWindow, onConfigure })
    connect(h, port)

    port.dispatch(configureMessage(payload))
    await vi.waitFor(() => expect(port.closed).toBe(true))

    expect(onConfigure).not.toHaveBeenCalled()
    expect(port.sent.at(-1)).toEqual({
      protocol: SUB2API_BRIDGE_PROTOCOL,
      version: SUB2API_BRIDGE_VERSION,
      type: 'ack',
      nonce: NONCE,
      status: 'error',
      requestId: 1,
      error: {
        code: 'invalid_configuration',
        message: 'The image playground configuration is invalid',
      },
    })
    expect(JSON.stringify(port.sent)).not.toContain('sk-managed-secret')
  })

  it('treats the API key as an opaque bounded string', async () => {
    const h = harness()
    const port = new FakePort()
    const onConfigure = vi.fn()
    const opaqueKeyConfig = config({ apiKey: ' key-with-significant-spaces ' })
    startSub2ApiBridge({ window: h.childWindow, onConfigure })
    connect(h, port)

    port.dispatch(configureMessage(opaqueKeyConfig))

    await vi.waitFor(() => expect(onConfigure).toHaveBeenCalledWith(opaqueKeyConfig))
  })

  it.each([
    ['missing request id', { requestId: undefined }],
    ['zero request id', { requestId: 0 }],
    ['fractional request id', { requestId: 1.5 }],
    ['unsafe request id', { requestId: Number.MAX_SAFE_INTEGER + 1 }],
    ['unknown top-level field', { extra: true }],
  ])('ignores malformed configure with %s', async (_name, overrides) => {
    const h = harness()
    const port = new FakePort()
    const onConfigure = vi.fn()
    startSub2ApiBridge({ window: h.childWindow, onConfigure })
    connect(h, port)

    const message = configureMessage(config(), overrides)
    if ('requestId' in overrides && overrides.requestId === undefined) {
      delete (message as Partial<typeof message>).requestId
    }
    port.dispatch(message)
    await Promise.resolve()

    expect(onConfigure).not.toHaveBeenCalled()
    expect(port.sent).toHaveLength(1)
    expect(port.closed).toBe(false)
  })

  it('applies one strict configuration, acknowledges it, then closes the port', async () => {
    const h = harness()
    const port = new FakePort()
    const onConfigure = vi.fn()
    const bridge = startSub2ApiBridge({ window: h.childWindow, onConfigure })
    connect(h, port)

    port.dispatch(configureMessage())

    await expect(bridge.configured).resolves.toEqual(config())
    expect(onConfigure).toHaveBeenCalledOnce()
    expect(port.sent).toEqual([
      {
        protocol: SUB2API_BRIDGE_PROTOCOL,
        version: SUB2API_BRIDGE_VERSION,
        type: 'connected',
        nonce: NONCE,
      },
      {
        protocol: SUB2API_BRIDGE_PROTOCOL,
        version: SUB2API_BRIDGE_VERSION,
        type: 'ack',
        nonce: NONCE,
        status: 'configured',
        requestId: 1,
      },
    ])
    expect(port.closed).toBe(true)

    port.dispatch(configureMessage(config({ apiKey: 'sk-second' }), { requestId: 2 }))
    expect(onConfigure).toHaveBeenCalledOnce()
  })

  it('ignores a second configuration while the first one is still applying', async () => {
    const h = harness()
    const port = new FakePort()
    let releaseConfigure: () => void = () => undefined
    const pendingConfigure = new Promise<void>((resolve) => {
      releaseConfigure = resolve
    })
    const onConfigure = vi.fn(() => pendingConfigure)
    startSub2ApiBridge({ window: h.childWindow, onConfigure })
    connect(h, port)

    port.dispatch(configureMessage(config({ apiKey: 'sk-first' }), { requestId: 1 }))
    port.dispatch(configureMessage(config({ apiKey: 'sk-second' }), { requestId: 2 }))

    expect(onConfigure).toHaveBeenCalledOnce()
    releaseConfigure()
    await vi.waitFor(() => expect(port.closed).toBe(true))
    expect(port.sent.filter((message: any) => message.type === 'ack')).toEqual([
      expect.objectContaining({ status: 'configured', requestId: 1 }),
    ])
  })

  it('returns a generic secret-free error when applying configuration fails', async () => {
    const h = harness()
    const port = new FakePort()
    const onConfigure = vi.fn(() => {
      throw new Error('failed while using sk-managed-secret')
    })
    startSub2ApiBridge({ window: h.childWindow, onConfigure })
    connect(h, port)

    port.dispatch(configureMessage())
    await vi.waitFor(() => expect(port.closed).toBe(true))

    expect(port.sent.at(-1)).toEqual({
      protocol: SUB2API_BRIDGE_PROTOCOL,
      version: SUB2API_BRIDGE_VERSION,
      type: 'ack',
      nonce: NONCE,
      status: 'error',
      requestId: 1,
      error: {
        code: 'configuration_failed',
        message: 'The image playground configuration could not be applied',
      },
    })
    expect(JSON.stringify(port.sent)).not.toContain('sk-managed-secret')
  })

  it('disposes a connected port without accepting later configuration', () => {
    const h = harness()
    const port = new FakePort()
    const onConfigure = vi.fn()
    const bridge = startSub2ApiBridge({ window: h.childWindow, onConfigure })
    connect(h, port)

    bridge.dispose()
    port.dispatch(configureMessage())

    expect(port.closed).toBe(true)
    expect(onConfigure).not.toHaveBeenCalled()
  })
})
