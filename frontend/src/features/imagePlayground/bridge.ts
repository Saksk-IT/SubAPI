export const IMAGE_PLAYGROUND_PROTOCOL = 'sub2api.image-playground' as const
export const IMAGE_PLAYGROUND_PROTOCOL_VERSION = 1 as const

const FRAME_NAME_PREFIX = 'sub2api-image-playground:'
const NONCE_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const STORAGE_SCOPE_PATTERN = /^[1-9][0-9]{0,18}$/
const LOCALE_PATTERN = /^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$/

export type ImagePlaygroundTheme = 'light' | 'dark'
export type ImagePlaygroundApiMode = 'images' | 'responses'

export interface ImagePlaygroundProfile {
  readonly id: string
  readonly name: string
  readonly apiMode: ImagePlaygroundApiMode
  readonly model: string
}

export interface ConfigureMessageInput {
  nonce: string
  requestId: number
  apiKey: string
  apiKeyId: number
  apiKeyName: string
  storageScope: string
  locale?: string
  theme?: ImagePlaygroundTheme
}

interface ReadyEventOptions {
  expectedOrigin: string
  expectedSource: MessageEventSource | null
  expectedNonce: string
}

const MANAGED_PROFILES: readonly Readonly<ImagePlaygroundProfile>[] = Object.freeze([
  Object.freeze({
    id: 'sub2api-images',
    name: 'Sub2API Images',
    apiMode: 'images' as const,
    model: 'gpt-image-2',
  }),
  Object.freeze({
    id: 'sub2api-responses',
    name: 'Sub2API Responses',
    apiMode: 'responses' as const,
    model: 'gpt-5.5',
  }),
])

function assertNonce(nonce: string): void {
  if (!NONCE_PATTERN.test(nonce)) {
    throw new Error('nonce must be a UUID')
  }
}

function assertBoundedString(value: string, field: string, maxLength: number): void {
  if (typeof value !== 'string' || value.length < 1 || value.length > maxLength) {
    throw new Error(`${field} must contain between 1 and ${maxLength} characters`)
  }
}

function hasExactKeys(value: object, expectedKeys: readonly string[]): boolean {
  const actualKeys = Object.keys(value).sort()
  const sortedExpectedKeys = [...expectedKeys].sort()
  return actualKeys.length === sortedExpectedKeys.length &&
    actualKeys.every((key, index) => key === sortedExpectedKeys[index])
}

export function buildManagedProfiles(): readonly Readonly<ImagePlaygroundProfile>[] {
  return MANAGED_PROFILES
}

export function createFrameName(nonce: string): string {
  assertNonce(nonce)
  return `${FRAME_NAME_PREFIX}${nonce}`
}

export function isTrustedReadyEvent(
  event: MessageEvent,
  options: ReadyEventOptions,
): boolean {
  if (event.origin !== options.expectedOrigin || event.source !== options.expectedSource) {
    return false
  }

  if (!event.data || typeof event.data !== 'object' || Array.isArray(event.data)) {
    return false
  }

  const data = event.data as Record<string, unknown>
  if (!hasExactKeys(data, ['protocol', 'version', 'type', 'nonce'])) {
    return false
  }

  return data.protocol === IMAGE_PLAYGROUND_PROTOCOL &&
    data.version === IMAGE_PLAYGROUND_PROTOCOL_VERSION &&
    data.type === 'ready' &&
    data.nonce === options.expectedNonce
}

export function isConfiguredAckMessage(
  data: unknown,
  expectedNonce: string,
  expectedRequestId: number,
): boolean {
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    return false
  }

  const message = data as Record<string, unknown>
  if (!hasExactKeys(message, ['protocol', 'version', 'type', 'nonce', 'status', 'requestId'])) {
    return false
  }

  return message.protocol === IMAGE_PLAYGROUND_PROTOCOL &&
    message.version === IMAGE_PLAYGROUND_PROTOCOL_VERSION &&
    message.type === 'ack' &&
    message.nonce === expectedNonce &&
    message.status === 'configured' &&
    Number.isSafeInteger(message.requestId) &&
    Number(message.requestId) > 0 &&
    message.requestId === expectedRequestId
}

export function buildConnectMessage(nonce: string) {
  assertNonce(nonce)
  return Object.freeze({
    protocol: IMAGE_PLAYGROUND_PROTOCOL,
    version: IMAGE_PLAYGROUND_PROTOCOL_VERSION,
    type: 'connect' as const,
    nonce,
  })
}

export function buildClearMessage(nonce: string) {
  assertNonce(nonce)
  return Object.freeze({
    protocol: IMAGE_PLAYGROUND_PROTOCOL,
    version: IMAGE_PLAYGROUND_PROTOCOL_VERSION,
    type: 'clear' as const,
    nonce,
  })
}

export function buildConfigureMessage(input: ConfigureMessageInput) {
  assertNonce(input.nonce)
  if (!Number.isSafeInteger(input.requestId) || input.requestId <= 0) {
    throw new Error('requestId must be a positive integer')
  }
  assertBoundedString(input.apiKey, 'apiKey', 512)
  assertBoundedString(input.apiKeyName, 'apiKeyName', 128)

  if (!Number.isSafeInteger(input.apiKeyId) || input.apiKeyId <= 0) {
    throw new Error('apiKeyId must be a positive integer')
  }
  if (!STORAGE_SCOPE_PATTERN.test(input.storageScope)) {
    throw new Error('storageScope must be a positive decimal user id')
  }
  if (input.locale !== undefined) {
    assertBoundedString(input.locale, 'locale', 32)
    if (!LOCALE_PATTERN.test(input.locale)) {
      throw new Error('locale must be a valid BCP 47 language tag')
    }
  }
  if (input.theme !== undefined && input.theme !== 'light' && input.theme !== 'dark') {
    throw new Error('theme must be light or dark')
  }

  const payload = Object.freeze({
    apiKey: input.apiKey,
    apiKeyId: input.apiKeyId,
    apiKeyName: input.apiKeyName,
    storageScope: input.storageScope,
    baseUrl: '/v1' as const,
    profiles: MANAGED_PROFILES,
    ...(input.locale === undefined ? {} : { locale: input.locale }),
    ...(input.theme === undefined ? {} : { theme: input.theme }),
  })

  return Object.freeze({
    protocol: IMAGE_PLAYGROUND_PROTOCOL,
    version: IMAGE_PLAYGROUND_PROTOCOL_VERSION,
    type: 'configure' as const,
    nonce: input.nonce,
    requestId: input.requestId,
    payload,
  })
}
