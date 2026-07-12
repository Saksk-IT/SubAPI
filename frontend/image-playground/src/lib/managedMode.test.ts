import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { DEFAULT_SETTINGS } from './apiProfiles'
import {
  activateManagedConfig,
  applyManagedPresentation,
  clearManagedConfig,
  enforceManagedSettings,
  getManagedDatabaseName,
  getManagedGenerationSignal,
  getManagedSnapshot,
  getManagedStorageName,
  hasSameManagedCredentialIdentity,
  managedFetch,
  redactSettingsSecrets,
  requireManagedRuntime,
  updateManagedPresentationConfig,
} from './managedMode'
import type { ManagedConfig } from './sub2apiBridge'

function config(scope = '42', apiKey = 'sk-first'): ManagedConfig {
  return {
    apiKey,
    apiKeyId: 7,
    apiKeyName: '生图 Key',
    storageScope: scope,
    baseUrl: '/v1',
    profiles: [
      { id: 'images', name: 'Images', apiMode: 'images', model: 'gpt-image-2' },
      { id: 'responses', name: 'Responses', apiMode: 'responses', model: 'gpt-5.5' },
    ],
  }
}

describe('managed Sub2API runtime', () => {
  beforeEach(() => {
    clearManagedConfig()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('injects only the two same-origin OpenAI profiles in memory', () => {
    const settings = activateManagedConfig(config(), DEFAULT_SETTINGS)

    expect(settings).toMatchObject({
      baseUrl: '/v1',
      apiKey: 'sk-first',
      apiProxy: false,
      customProviders: [],
      providerOrder: ['openai'],
      agentApiConfigMode: 'hybrid',
      agentTextProfileId: 'responses',
      agentImageProfileId: 'images',
    })
    expect(settings.profiles).toEqual([
      expect.objectContaining({ id: 'images', provider: 'openai', baseUrl: '/v1', apiKey: 'sk-first', apiMode: 'images', apiProxy: false, responseFormatB64Json: true }),
      expect.objectContaining({ id: 'responses', provider: 'openai', baseUrl: '/v1', apiKey: 'sk-first', apiMode: 'responses', apiProxy: false }),
    ])
  })

  it('applies every configured theme and locale update to the document root', () => {
    const toggle = vi.fn()
    const root = { classList: { toggle }, style: { colorScheme: '' }, lang: '' }

    applyManagedPresentation(config(), { documentElement: root } as any)
    applyManagedPresentation({ ...config(), theme: 'dark', locale: 'zh-CN' }, { documentElement: root } as any)

    expect(toggle).toHaveBeenLastCalledWith('dark', true)
    expect(root.style.colorScheme).toBe('dark')
    expect(root.lang).toBe('zh-CN')
  })

  it('updates theme and locale metadata without aborting an in-flight generation', () => {
    activateManagedConfig(config(), DEFAULT_SETTINGS)
    const signal = getManagedGenerationSignal()
    const presentationUpdate = { ...config(), theme: 'dark' as const, locale: 'zh-CN' }

    expect(hasSameManagedCredentialIdentity(presentationUpdate)).toBe(true)
    expect(updateManagedPresentationConfig(presentationUpdate)).toBe(true)
    expect(signal.aborted).toBe(false)
    expect(getManagedGenerationSignal()).toBe(signal)
    expect(getManagedSnapshot()?.config).toEqual(presentationUpdate)
  })

  it('treats a key or managed profile change as a new credential identity', () => {
    activateManagedConfig(config(), DEFAULT_SETTINGS)

    expect(hasSameManagedCredentialIdentity(config('42', 'sk-second'))).toBe(false)
    expect(hasSameManagedCredentialIdentity({
      ...config(),
      profiles: config().profiles.map((profile) => profile.apiMode === 'images'
        ? { ...profile, model: 'parent-updated-model' }
        : profile),
    })).toBe(false)
  })

  it('keeps user-edited models across configure updates in the same storage scope', () => {
    const first = activateManagedConfig(config(), DEFAULT_SETTINGS)
    const edited = {
      ...first,
      profiles: first.profiles.map((profile) => ({
        ...profile,
        model: profile.apiMode === 'images' ? 'custom-image-model' : 'custom-responses-model',
      })),
    }
    const updated = activateManagedConfig(config('42', 'sk-second'), edited)

    expect(updated.profiles).toEqual([
      expect.objectContaining({ model: 'custom-image-model', apiKey: 'sk-second', baseUrl: '/v1' }),
      expect.objectContaining({ model: 'custom-responses-model', apiKey: 'sk-second', baseUrl: '/v1' }),
    ])
  })

  it('resets models to parent defaults when the storage scope changes', () => {
    const first = activateManagedConfig(config('42'), DEFAULT_SETTINGS)
    const edited = {
      ...first,
      profiles: first.profiles.map((profile) => ({ ...profile, model: 'other-user-model' })),
    }

    const updated = activateManagedConfig(config('99', 'sk-second'), edited)

    expect(updated.profiles.map((profile) => profile.model)).toEqual(['gpt-image-2', 'gpt-5.5'])
  })

  it('allows model edits while restoring managed credentials and providers', () => {
    activateManagedConfig(config(), DEFAULT_SETTINGS)
    const unsafe = {
      ...DEFAULT_SETTINGS,
      customProviders: [{ id: 'evil' }],
      profiles: [
        { ...DEFAULT_SETTINGS.profiles[0], provider: 'fal', baseUrl: 'https://evil.example', apiKey: 'evil', model: 'custom-image-model', responseFormatB64Json: false },
        { ...DEFAULT_SETTINGS.profiles[0], id: 'responses', apiMode: 'responses', baseUrl: 'https://evil.example', apiKey: 'evil', model: 'custom-responses-model' },
      ],
    } as any

    const safe = enforceManagedSettings(unsafe)

    expect(safe.customProviders).toEqual([])
    expect(safe.profiles).toEqual([
      expect.objectContaining({ provider: 'openai', baseUrl: '/v1', apiKey: 'sk-first', model: 'custom-image-model', responseFormatB64Json: true }),
      expect.objectContaining({ provider: 'openai', baseUrl: '/v1', apiKey: 'sk-first', model: 'custom-responses-model' }),
    ])
  })

  it('aborts the previous generation when the key changes or clears', () => {
    activateManagedConfig(config(), DEFAULT_SETTINGS)
    const firstSignal = getManagedGenerationSignal()

    activateManagedConfig(config('42', 'sk-second'), DEFAULT_SETTINGS)
    const secondSignal = getManagedGenerationSignal()

    expect(firstSignal.aborted).toBe(true)
    expect(secondSignal.aborted).toBe(false)
    expect(getManagedSnapshot()?.config.apiKey).toBe('sk-second')

    clearManagedConfig()
    expect(secondSignal.aborted).toBe(true)
    expect(getManagedSnapshot()).toBeNull()
  })

  it('uses a different IndexedDB database for every validated user scope', () => {
    activateManagedConfig(config('42'), DEFAULT_SETTINGS)
    const first = getManagedDatabaseName()
    activateManagedConfig(config('99'), DEFAULT_SETTINGS)
    const second = getManagedDatabaseName()

    expect(first).toBe('gpt-image-playground-sub2api-42')
    expect(second).toBe('gpt-image-playground-sub2api-99')
    expect(second).not.toBe(first)
  })

  it('uses a different persisted UI state key for every validated user scope', () => {
    activateManagedConfig(config('42'), DEFAULT_SETTINGS)
    const first = getManagedStorageName()
    activateManagedConfig(config('99'), DEFAULT_SETTINGS)
    const second = getManagedStorageName()

    expect(first).toBe('gpt-image-playground-sub2api-42')
    expect(second).toBe('gpt-image-playground-sub2api-99')
    expect(second).not.toBe(first)
  })

  it('removes legacy and profile API keys from serializable settings', () => {
    const settings = activateManagedConfig(config(), DEFAULT_SETTINGS)
    const redacted = redactSettingsSecrets(settings)

    expect(redacted.apiKey).toBe('')
    expect(redacted.profiles.every((profile) => profile.apiKey === '')).toBe(true)
    expect(JSON.stringify(redacted)).not.toContain('sk-first')
  })

  it('fails closed after clear without invoking fetch', async () => {
    const release = requireManagedRuntime()
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('location', { origin: 'https://sub2api.example' })
    activateManagedConfig(config(), DEFAULT_SETTINGS)
    clearManagedConfig()

    await expect(managedFetch('/v1/images/generations', { method: 'POST' }))
      .rejects.toThrow('Sub2API configuration is not available')
    expect(fetchMock).not.toHaveBeenCalled()
    release()
  })

  it.each([
    'https://evil.example/v1/images/generations',
    '//evil.example/v1/images/generations',
    '/v1/chat/completions',
    '/v1/images/tasks/123',
  ])('rejects a managed request outside the fixed same-origin endpoints: %s', async (url) => {
    const release = requireManagedRuntime()
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('location', { origin: 'https://sub2api.example' })
    activateManagedConfig(config(), DEFAULT_SETTINGS)

    await expect(managedFetch(url, { method: 'POST' })).rejects.toThrow('Managed request URL is not allowed')
    expect(fetchMock).not.toHaveBeenCalled()
    release()
  })

  it('forces redirect error on an allowed same-origin request', async () => {
    const release = requireManagedRuntime()
    const fetchMock = vi.fn().mockResolvedValue(new Response('{}'))
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('location', { origin: 'https://sub2api.example' })
    activateManagedConfig(config(), DEFAULT_SETTINGS)

    await managedFetch('/v1/responses', { method: 'POST', redirect: 'follow' })

    expect(fetchMock).toHaveBeenCalledWith('https://sub2api.example/v1/responses', expect.objectContaining({
      redirect: 'error',
      signal: expect.any(AbortSignal),
    }))
    release()
  })
})
