import {
  buildConfigureMessage,
  buildConnectMessage,
  createFrameName,
  isConfigurationErrorAckMessage,
  isConnectedMessage,
  isConfiguredAckMessage,
  isTrustedReadyEvent,
  type ImagePlaygroundTheme,
} from './bridge'

const POPUP_URL = '/image-playground/'
const DEFAULT_TIMEOUT_MS = 8_000
const POPUP_CLOSE_POLL_MS = 250

export type PopupLaunchErrorCode =
  | 'popup_blocked'
  | 'connection_timeout'
  | 'popup_closed'
  | 'configuration_failed'
  | 'aborted'

const ERROR_MESSAGES: Readonly<Record<PopupLaunchErrorCode, string>> = Object.freeze({
  popup_blocked: 'Image playground popup was blocked',
  connection_timeout: 'Image playground connection timed out',
  popup_closed: 'Image playground popup was closed',
  configuration_failed: 'Image playground configuration failed',
  aborted: 'Image playground popup session was aborted',
})

type PopupOpener = (url: string, target: string) => Window | null

interface PopupLaunchSnapshot {
  readonly nonce: string
  readonly origin: string
  readonly timeoutMs: number
  readonly popupName: string
  readonly openWindow: PopupOpener
  readonly configureMessage: ReturnType<typeof buildConfigureMessage>
}

export interface OpenImagePlaygroundPopupOptions {
  apiKey: string
  apiKeyId: number
  apiKeyName: string
  storageScope: string
  locale?: string
  theme?: ImagePlaygroundTheme
  timeoutMs?: number
  openWindow?: PopupOpener
}

export interface ImagePlaygroundPopupSession {
  readonly configured: Promise<void>
  readonly closed: Promise<void>
  abort(): void
}

export class PopupLaunchError extends Error {
  readonly code: PopupLaunchErrorCode

  constructor(code: PopupLaunchErrorCode) {
    super(ERROR_MESSAGES[code])
    this.name = 'PopupLaunchError'
    this.code = code
  }
}

type SessionState =
  | 'waiting-ready'
  | 'waiting-connected'
  | 'waiting-configured'
  | 'ended'

function failedSession(code: PopupLaunchErrorCode): ImagePlaygroundPopupSession {
  const configured = Promise.reject<void>(new PopupLaunchError(code))
  return Object.freeze({
    configured,
    closed: Promise.resolve(),
    abort() {},
  })
}

function normalizeTimeoutMs(timeoutMs: number | undefined): number {
  return typeof timeoutMs === 'number' && Number.isFinite(timeoutMs) && timeoutMs > 0
    ? timeoutMs
    : DEFAULT_TIMEOUT_MS
}

function captureLaunchSnapshot(options: OpenImagePlaygroundPopupOptions): PopupLaunchSnapshot {
  const nonce = crypto.randomUUID()
  const origin = new URL(window.location.origin).origin
  if (origin === 'null') {
    throw new Error('window origin must be non-opaque')
  }
  const timeoutMs = normalizeTimeoutMs(options.timeoutMs)
  const openWindow = options.openWindow ?? window.open.bind(window)
  if (typeof openWindow !== 'function') {
    throw new Error('openWindow must be a function')
  }

  return Object.freeze({
    nonce,
    origin,
    timeoutMs,
    popupName: createFrameName(nonce),
    openWindow,
    configureMessage: buildConfigureMessage({
      nonce,
      requestId: 1,
      apiKey: options.apiKey,
      apiKeyId: options.apiKeyId,
      apiKeyName: options.apiKeyName,
      storageScope: options.storageScope,
      locale: options.locale,
      theme: options.theme,
    }),
  })
}

function isPopupClosed(popup: Window): boolean {
  try {
    return popup.closed
  } catch {
    return true
  }
}

function isExpectedPopupLocation(popup: Window, origin: string): boolean {
  try {
    const url = new URL(popup.location.href)
    return url.origin === origin && url.pathname === POPUP_URL && url.search === '' && url.hash === ''
  } catch {
    return false
  }
}

export function openImagePlaygroundPopup(
  options: OpenImagePlaygroundPopupOptions,
): ImagePlaygroundPopupSession {
  let snapshot: PopupLaunchSnapshot
  try {
    snapshot = captureLaunchSnapshot(options)
  } catch {
    return failedSession('configuration_failed')
  }

  let openedPopup: Window | null

  try {
    openedPopup = snapshot.openWindow(POPUP_URL, snapshot.popupName)
  } catch {
    return failedSession('popup_blocked')
  }
  if (!openedPopup) {
    return failedSession('popup_blocked')
  }

  const popup = openedPopup
  const {
    nonce,
    origin,
    timeoutMs,
    configureMessage,
  } = snapshot
  let state: SessionState = 'waiting-ready'
  let requestId = 0
  let hasConfigured = false
  let resolveConfigured!: () => void
  let rejectConfigured!: (reason: PopupLaunchError) => void
  let resolveClosed!: () => void
  let timeoutHandle: number | null = null
  let closePollHandle: number | null = null
  let port1: MessagePort | null = null
  let port2: MessagePort | null = null
  let windowListenerAttached = false
  let portListenerAttached = false

  const configured = new Promise<void>((resolve, reject) => {
    resolveConfigured = resolve
    rejectConfigured = reject
  })
  const closed = new Promise<void>((resolve) => {
    resolveClosed = resolve
  })

  function closePort(port: MessagePort | null): void {
    if (!port) return
    try {
      port.close()
    } catch {
      // A transferred port may already be detached; cleanup remains best-effort.
    }
  }

  function cleanupChannel(): void {
    if (port1 && portListenerAttached) {
      port1.removeEventListener('message', onPortMessage)
      portListenerAttached = false
    }
    closePort(port1)
    closePort(port2)
    port1 = null
    port2 = null
  }

  function cleanup(): void {
    if (windowListenerAttached) {
      window.removeEventListener('message', onWindowMessage)
      windowListenerAttached = false
    }
    if (timeoutHandle !== null) {
      window.clearTimeout(timeoutHandle)
      timeoutHandle = null
    }
    if (closePollHandle !== null) {
      window.clearTimeout(closePollHandle)
      closePollHandle = null
    }
    cleanupChannel()
    resolveClosed()
  }

  function fail(code: Exclude<PopupLaunchErrorCode, 'popup_blocked'>): void {
    if (state === 'ended') return
    state = 'ended'
    cleanup()
    try {
      popup.close()
    } catch {
      // Cleanup and rejection must still complete if the browser refuses close().
    }
    if (!hasConfigured) rejectConfigured(new PopupLaunchError(code))
  }

  function succeed(): void {
    if (state !== 'waiting-configured') return
    cleanupChannel()
    state = 'waiting-ready'
    if (hasConfigured) return

    hasConfigured = true
    if (timeoutHandle !== null) {
      window.clearTimeout(timeoutHandle)
      timeoutHandle = null
    }
    resolveConfigured()
  }

  function onPortMessage(event: MessageEvent): void {
    if (state === 'waiting-connected') {
      if (!isConnectedMessage(event.data, nonce)) return

      state = 'waiting-configured'
      try {
        requestId += 1
        port1?.postMessage({ ...configureMessage, requestId })
      } catch {
        fail('configuration_failed')
      }
      return
    }

    if (state !== 'waiting-configured') return
    if (isConfiguredAckMessage(event.data, nonce, requestId)) {
      succeed()
      return
    }
    if (isConfigurationErrorAckMessage(event.data, nonce, requestId)) {
      fail('configuration_failed')
    }
  }

  function onWindowMessage(event: MessageEvent): void {
    if (state !== 'waiting-ready' || !isTrustedReadyEvent(event, {
      expectedOrigin: origin,
      expectedSource: popup,
      expectedNonce: nonce,
    }) || !isExpectedPopupLocation(popup, origin)) {
      return
    }

    state = 'waiting-connected'
    try {
      const channel = new MessageChannel()
      port1 = channel.port1
      port2 = channel.port2
      port1.addEventListener('message', onPortMessage)
      portListenerAttached = true
      port1.start()
      popup.postMessage(buildConnectMessage(nonce), origin, [port2])
    } catch {
      fail('configuration_failed')
    }
  }

  function scheduleClosePoll(): void {
    closePollHandle = window.setTimeout(() => {
      closePollHandle = null
      if (state === 'ended') return

      if (isPopupClosed(popup)) {
        fail('popup_closed')
        return
      }
      scheduleClosePoll()
    }, Math.min(POPUP_CLOSE_POLL_MS, timeoutMs))
  }

  const abort = () => {
    if (state === 'ended') return
    fail('aborted')
  }

  window.addEventListener('message', onWindowMessage)
  windowListenerAttached = true
  timeoutHandle = window.setTimeout(() => {
    timeoutHandle = null
    if (state === 'ended' || hasConfigured) return
    fail(isPopupClosed(popup) ? 'popup_closed' : 'connection_timeout')
  }, timeoutMs)
  scheduleClosePoll()

  return Object.freeze({ configured, closed, abort })
}
