import { describe, expect, it } from 'vitest'

import {
  IMAGE_PLAYGROUND_PROTOCOL,
  IMAGE_PLAYGROUND_PROTOCOL_VERSION,
  buildConfigureMessage,
  buildConnectMessage,
  buildManagedProfiles,
  createFrameName,
  isConfiguredAckMessage,
  isTrustedReadyEvent,
} from '../bridge'

const NONCE = '123e4567-e89b-42d3-a456-426614174000'

function readyData(overrides: Record<string, unknown> = {}) {
  return {
    protocol: IMAGE_PLAYGROUND_PROTOCOL,
    version: IMAGE_PLAYGROUND_PROTOCOL_VERSION,
    type: 'ready',
    nonce: NONCE,
    ...overrides,
  }
}

function trustedEvent(overrides: Partial<Pick<MessageEvent, 'origin' | 'source' | 'data'>> = {}) {
  const source = {} as Window
  return {
    event: {
      origin: 'https://sub2api.example.com',
      source,
      data: readyData(),
      ...overrides,
    } as MessageEvent,
    source,
  }
}

describe('image playground bridge protocol', () => {
  it('accepts only the expected origin, source, protocol version, nonce and exact ready schema', () => {
    const valid = trustedEvent()
    const options = {
      expectedOrigin: 'https://sub2api.example.com',
      expectedSource: valid.source,
      expectedNonce: NONCE,
    }

    expect(isTrustedReadyEvent(valid.event, options)).toBe(true)
    expect(isTrustedReadyEvent(trustedEvent({ origin: 'https://evil.example.com' }).event, options)).toBe(false)
    expect(isTrustedReadyEvent(trustedEvent({ source: {} as Window }).event, options)).toBe(false)
    expect(isTrustedReadyEvent(trustedEvent({ data: readyData({ version: 2 }) }).event, options)).toBe(false)
    expect(isTrustedReadyEvent(trustedEvent({ data: readyData({ nonce: 'wrong' }) }).event, options)).toBe(false)
    expect(isTrustedReadyEvent(trustedEvent({ data: readyData({ unexpected: true }) }).event, options)).toBe(false)
  })

  it('puts the nonce in iframe.name without adding URL parameters', () => {
    expect(createFrameName(NONCE)).toBe(`sub2api-image-playground:${NONCE}`)
    expect(createFrameName(NONCE)).not.toContain('?')
  })

  it('accepts only an exact configured acknowledgement for the current nonce', () => {
    const ack = {
      protocol: IMAGE_PLAYGROUND_PROTOCOL,
      version: IMAGE_PLAYGROUND_PROTOCOL_VERSION,
      type: 'ack',
      nonce: NONCE,
      status: 'configured',
      requestId: 2,
    }

    expect(isConfiguredAckMessage(ack, NONCE, 2)).toBe(true)
    expect(isConfiguredAckMessage(ack, NONCE, 1)).toBe(false)
    expect(isConfiguredAckMessage({ ...ack, requestId: 0 }, NONCE, 0)).toBe(false)
    expect(isConfiguredAckMessage({ ...ack, nonce: 'wrong' }, NONCE, 2)).toBe(false)
    expect(isConfiguredAckMessage({ ...ack, status: 'cleared' }, NONCE, 2)).toBe(false)
    expect(isConfiguredAckMessage({ ...ack, status: 'error' }, NONCE, 2)).toBe(false)
    expect(isConfiguredAckMessage({ ...ack, unexpected: true }, NONCE, 2)).toBe(false)
  })

  it('builds immutable Images and Responses profiles on the same-origin gateway', () => {
    const profiles = buildManagedProfiles()

    expect(profiles).toEqual([
      {
        id: 'sub2api-images',
        name: 'Sub2API Images',
        apiMode: 'images',
        model: 'gpt-image-2',
      },
      {
        id: 'sub2api-responses',
        name: 'Sub2API Responses',
        apiMode: 'responses',
        model: 'gpt-5.5',
      },
    ])
    expect(Object.isFrozen(profiles)).toBe(true)
    expect(profiles.every(Object.isFrozen)).toBe(true)
    expect(profiles.every((profile) => (
      Object.keys(profile).sort().join(',') === 'apiMode,id,model,name'
    ))).toBe(true)
  })

  it('creates connect and configure messages without JWT or batched credentials', () => {
    const connect = buildConnectMessage(NONCE)
    const configure = buildConfigureMessage({
      nonce: NONCE,
      requestId: 1,
      apiKey: 'sk-current-only',
      apiKeyId: 7,
      apiKeyName: '绘图密钥',
      storageScope: '42',
      locale: 'zh-CN',
      theme: 'dark',
    })

    expect(connect).toEqual({
      protocol: IMAGE_PLAYGROUND_PROTOCOL,
      version: 1,
      type: 'connect',
      nonce: NONCE,
    })
    expect(configure.payload).toMatchObject({
      apiKey: 'sk-current-only',
      apiKeyId: 7,
      apiKeyName: '绘图密钥',
      storageScope: '42',
      baseUrl: '/v1',
      locale: 'zh-CN',
      theme: 'dark',
    })
    expect(configure.requestId).toBe(1)
    expect(Object.keys(configure).sort()).toEqual([
      'nonce',
      'payload',
      'protocol',
      'requestId',
      'type',
      'version',
    ])
    expect(configure.payload).not.toHaveProperty('keys')
    expect(configure.payload).not.toHaveProperty('token')
    expect(Object.keys(configure.payload).sort()).toEqual([
      'apiKey',
      'apiKeyId',
      'apiKeyName',
      'baseUrl',
      'locale',
      'profiles',
      'storageScope',
      'theme',
    ])
    expect(JSON.stringify(configure)).not.toContain('auth_token')
  })

  it('rejects an empty or oversized storage scope', () => {
    const base = {
      nonce: NONCE,
      requestId: 1,
      apiKey: 'sk-current-only',
      apiKeyId: 7,
      apiKeyName: 'Key',
      locale: 'zh-CN',
      theme: 'light' as const,
    }

    expect(() => buildConfigureMessage({ ...base, storageScope: '' })).toThrow('storageScope')
    expect(() => buildConfigureMessage({ ...base, storageScope: '1'.repeat(20) })).toThrow('storageScope')
    expect(() => buildConfigureMessage({ ...base, storageScope: '01' })).toThrow('storageScope')
    expect(() => buildConfigureMessage({ ...base, storageScope: '9'.repeat(19) })).not.toThrow()
  })

  it('rejects credentials and metadata outside the protocol bounds', () => {
    const base = {
      nonce: NONCE,
      requestId: 1,
      apiKey: 'sk-current-only',
      apiKeyId: 7,
      apiKeyName: 'Key',
      storageScope: '42',
      locale: 'zh-CN',
      theme: 'light' as const,
    }

    expect(() => buildConfigureMessage({ ...base, apiKey: '' })).toThrow('apiKey')
    expect(() => buildConfigureMessage({ ...base, requestId: 0 })).toThrow('requestId')
    expect(() => buildConfigureMessage({ ...base, requestId: Number.MAX_SAFE_INTEGER + 1 })).toThrow('requestId')
    expect(() => buildConfigureMessage({ ...base, apiKey: 'x'.repeat(513) })).toThrow('apiKey')
    expect(() => buildConfigureMessage({ ...base, apiKeyId: 0 })).toThrow('apiKeyId')
    expect(() => buildConfigureMessage({ ...base, apiKeyName: '' })).toThrow('apiKeyName')
    expect(() => buildConfigureMessage({ ...base, apiKeyName: 'x'.repeat(129) })).toThrow('apiKeyName')
    expect(() => buildConfigureMessage({ ...base, locale: 'x'.repeat(33) })).toThrow('locale')
    expect(() => buildConfigureMessage({ ...base, locale: 'zh_CN' })).toThrow('locale')
    expect(() => buildConfigureMessage({ ...base, locale: 'zh-Hans-CN' })).not.toThrow()
  })
})
