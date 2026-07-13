export const SUB2API_BRIDGE_PROTOCOL = 'sub2api.image-playground'
export const SUB2API_BRIDGE_VERSION = 1

const WINDOW_NAME_PREFIX = 'sub2api-image-playground:'
const NONCE_PATTERN = /^[A-Za-z0-9_-]{32,128}$/
const STORAGE_SCOPE_PATTERN = /^[1-9][0-9]{0,18}$/
const PROFILE_ID_PATTERN = /^[A-Za-z0-9._-]{1,64}$/
const LOCALE_PATTERN = /^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$/

export interface ManagedProfileConfig {
  id: string
  name: string
  apiMode: 'images' | 'responses'
  model: string
}

export interface ManagedConfig {
  apiKey: string
  apiKeyId: number
  apiKeyName: string
  storageScope: string
  baseUrl: '/v1'
  profiles: ManagedProfileConfig[]
  locale?: string
  theme?: 'light' | 'dark'
}

interface BridgePort {
  onmessage: ((event: { data: unknown }) => void) | null
  postMessage(message: unknown): void
  start(): void
  close(): void
}

interface BridgeMessageEvent {
  origin: string
  source: unknown
  data: unknown
  ports?: BridgePort[]
}

interface BridgeOpener {
  postMessage(message: unknown, targetOrigin: string): void
}

interface BridgeWindow {
  name: string
  location: { origin: string }
  opener: BridgeOpener | null
  addEventListener(type: 'message', listener: (event: BridgeMessageEvent) => void): void
  removeEventListener(type: 'message', listener: (event: BridgeMessageEvent) => void): void
}

interface StartBridgeOptions {
  window: BridgeWindow
  onConfigure(config: ManagedConfig): void | Promise<void>
}

interface ReadyDocument {
  readyState: string
}

interface LoadTarget {
  addEventListener(type: 'load', listener: () => void, options?: { once?: boolean }): void
  removeEventListener(type: 'load', listener: () => void): void
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function hasExactKeys(record: Record<string, unknown>, required: string[], optional: string[] = []) {
  const allowed = new Set([...required, ...optional])
  const keys = Object.keys(record)
  return required.every((key) => Object.prototype.hasOwnProperty.call(record, key)) && keys.every((key) => allowed.has(key))
}

function isTrimmedString(value: unknown, maxLength: number): value is string {
  return typeof value === 'string' && value.length > 0 && value.length <= maxLength && value.trim() === value
}

function isBoundedString(value: unknown, maxLength: number): value is string {
  return typeof value === 'string' && value.length > 0 && value.length <= maxLength
}

function parseProfile(value: unknown): ManagedProfileConfig | null {
  if (!isRecord(value) || !hasExactKeys(value, ['id', 'name', 'apiMode', 'model'])) return null
  if (!isTrimmedString(value.id, 64) || !PROFILE_ID_PATTERN.test(value.id)) return null
  if (!isTrimmedString(value.name, 128)) return null
  if (value.apiMode !== 'images' && value.apiMode !== 'responses') return null
  if (!isTrimmedString(value.model, 128)) return null
  return { id: value.id, name: value.name, apiMode: value.apiMode, model: value.model }
}

function parseManagedConfig(value: unknown): ManagedConfig | null {
  if (!isRecord(value) || !hasExactKeys(
    value,
    ['apiKey', 'apiKeyId', 'apiKeyName', 'storageScope', 'baseUrl', 'profiles'],
    ['locale', 'theme'],
  )) return null
  if (!isBoundedString(value.apiKey, 512)) return null
  if (!Number.isSafeInteger(value.apiKeyId) || Number(value.apiKeyId) <= 0) return null
  if (!isTrimmedString(value.apiKeyName, 128)) return null
  if (typeof value.storageScope !== 'string' || !STORAGE_SCOPE_PATTERN.test(value.storageScope)) return null
  if (value.baseUrl !== '/v1') return null
  if (!Array.isArray(value.profiles) || value.profiles.length !== 2) return null
  const profiles = value.profiles.map(parseProfile)
  if (profiles.some((profile) => !profile)) return null
  const validProfiles = profiles as ManagedProfileConfig[]
  if (new Set(validProfiles.map((profile) => profile.id)).size !== 2) return null
  if (validProfiles.filter((profile) => profile.apiMode === 'images').length !== 1) return null
  if (validProfiles.filter((profile) => profile.apiMode === 'responses').length !== 1) return null
  if (value.locale !== undefined && (!isTrimmedString(value.locale, 32) || !LOCALE_PATTERN.test(value.locale))) return null
  if (value.theme !== undefined && value.theme !== 'light' && value.theme !== 'dark') return null
  return {
    apiKey: value.apiKey,
    apiKeyId: Number(value.apiKeyId),
    apiKeyName: value.apiKeyName,
    storageScope: value.storageScope,
    baseUrl: '/v1',
    profiles: validProfiles,
    ...(value.locale === undefined ? {} : { locale: value.locale }),
    ...(value.theme === undefined ? {} : { theme: value.theme }),
  }
}

function isProtocolMessage(value: unknown, type: 'connect' | 'configure', nonce: string) {
  if (!isRecord(value)) return false
  const required = type === 'configure'
    ? ['protocol', 'version', 'type', 'nonce', 'requestId', 'payload']
    : ['protocol', 'version', 'type', 'nonce']
  if (!hasExactKeys(value, required)) return false
  return (type !== 'configure' || (Number.isSafeInteger(value.requestId) && Number(value.requestId) > 0)) &&
    value.protocol === SUB2API_BRIDGE_PROTOCOL &&
    value.version === SUB2API_BRIDGE_VERSION &&
    value.type === type &&
    value.nonce === nonce
}

export function parseBridgeNonce(windowName: string): string | null {
  if (!windowName.startsWith(WINDOW_NAME_PREFIX)) return null
  const nonce = windowName.slice(WINDOW_NAME_PREFIX.length)
  return NONCE_PATTERN.test(nonce) ? nonce : null
}

export function runAfterWindowLoad(document: ReadyDocument, target: LoadTarget, start: () => void) {
  if (document.readyState === 'complete') {
    start()
    return () => undefined
  }
  const handleLoad = () => {
    target.removeEventListener('load', handleLoad)
    start()
  }
  target.addEventListener('load', handleLoad, { once: true })
  return () => target.removeEventListener('load', handleLoad)
}

function clearLaunchCapabilities(target: BridgeWindow): boolean {
  try {
    target.name = ''
    target.opener = null
    return target.name === '' && target.opener === null
  } catch {
    return false
  }
}

export function startSub2ApiBridge(options: StartBridgeOptions) {
  const nonce = parseBridgeNonce(options.window.name)
  const opener = options.window.opener
  let port: BridgePort | null = null
  let resolveConfigured: (config: ManagedConfig) => void = () => undefined
  let acceptedConfiguration = false
  let disposed = false
  const configured = new Promise<ManagedConfig>((resolve) => {
    resolveConfigured = resolve
  })

  if (!nonce || !opener) {
    clearLaunchCapabilities(options.window)
    return { mode: 'direct' as const, configured, dispose: () => undefined }
  }

  const closePort = () => {
    if (!port) return
    const activePort = port
    port = null
    activePort.onmessage = null
    try {
      activePort.close()
    } catch {
      // Cleanup is best-effort after the one-time channel has settled.
    }
  }

  const postToPort = (message: unknown): boolean => {
    if (!port) return false
    try {
      port.postMessage(message)
      return true
    } catch {
      return false
    }
  }

  const sendConfiguredAck = (requestId: number) => {
    return postToPort({
      protocol: SUB2API_BRIDGE_PROTOCOL,
      version: SUB2API_BRIDGE_VERSION,
      type: 'ack',
      nonce,
      status: 'configured',
      requestId,
    })
  }

  const sendErrorAck = (
    requestId: number,
    code: 'invalid_configuration' | 'configuration_failed',
    message: string,
  ) => {
    return postToPort({
      protocol: SUB2API_BRIDGE_PROTOCOL,
      version: SUB2API_BRIDGE_VERSION,
      type: 'ack',
      nonce,
      status: 'error',
      requestId,
      error: { code, message },
    })
  }

  const handlePortMessage = (event: { data: unknown }) => {
    if (
      disposed ||
      acceptedConfiguration ||
      !isProtocolMessage(event.data, 'configure', nonce)
    ) {
      return
    }

    acceptedConfiguration = true
    const message = event.data as Record<string, unknown>
    const requestId = Number(message.requestId)
    const payload = parseManagedConfig(message.payload)
    if (!payload) {
      try {
        sendErrorAck(
          requestId,
          'invalid_configuration',
          'The image playground configuration is invalid',
        )
      } finally {
        closePort()
      }
      return
    }

    void (async () => {
      try {
        await options.onConfigure(payload)
        if (disposed) return
        if (sendConfiguredAck(requestId)) {
          resolveConfigured(payload)
        }
      } catch {
        if (disposed) return
        sendErrorAck(
          requestId,
          'configuration_failed',
          'The image playground configuration could not be applied',
        )
      } finally {
        closePort()
      }
    })()
  }

  const handleWindowMessage = (event: BridgeMessageEvent) => {
    if (disposed || port) return
    if (event.origin !== options.window.location.origin || event.source !== opener) return
    if (!isProtocolMessage(event.data, 'connect', nonce)) return
    if (!Array.isArray(event.ports) || event.ports.length !== 1) return

    const connectedPort = event.ports[0]
    port = connectedPort
    options.window.removeEventListener('message', handleWindowMessage)

    // 保留不含密钥的 nonce 与同源 opener，页面刷新后才能重新建立一次性端口。
    // 父页仍会严格校验 origin、WindowProxy 与 nonce，密钥只经新的 MessagePort 下发。

    try {
      connectedPort.onmessage = handlePortMessage
      connectedPort.start()
      if (!postToPort({
        protocol: SUB2API_BRIDGE_PROTOCOL,
        version: SUB2API_BRIDGE_VERSION,
        type: 'connected',
        nonce,
      })) {
        closePort()
      }
    } catch {
      closePort()
    }
  }

  options.window.addEventListener('message', handleWindowMessage)
  opener.postMessage({
    protocol: SUB2API_BRIDGE_PROTOCOL,
    version: SUB2API_BRIDGE_VERSION,
    type: 'ready',
    nonce,
  }, options.window.location.origin)

  return {
    mode: 'popup' as const,
    configured,
    dispose: () => {
      disposed = true
      options.window.removeEventListener('message', handleWindowMessage)
      closePort()
    },
  }
}
