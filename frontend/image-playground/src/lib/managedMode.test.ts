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
  runManagedConfigurationTransaction,
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

  it('injects an internal async Images provider while keeping Responses on OpenAI', () => {
    const settings = activateManagedConfig(config(), DEFAULT_SETTINGS)

    expect(settings).toMatchObject({
      baseUrl: '/v1',
      apiKey: 'sk-first',
      apiProxy: false,
      providerOrder: ['sub2api-async', 'openai'],
      agentApiConfigMode: 'hybrid',
      agentTextProfileId: 'responses',
      agentImageProfileId: 'images',
    })
    expect(settings.profiles).toEqual([
      expect.objectContaining({ id: 'images', provider: 'sub2api-async', baseUrl: '/v1', apiKey: 'sk-first', apiMode: 'images', apiProxy: false, responseFormatB64Json: true }),
      expect.objectContaining({ id: 'responses', provider: 'openai', baseUrl: '/v1', apiKey: 'sk-first', apiMode: 'responses', apiProxy: false }),
    ])
    expect(settings.customProviders).toEqual([{
      id: 'sub2api-async',
      name: 'Sub2API 异步生图',
      template: 'http-image',
      submit: expect.objectContaining({
        path: 'images/generations/jobs',
        method: 'POST',
        contentType: 'json',
        taskIdPath: 'id',
        useTaskIdAsIdempotencyKey: true,
      }),
      editSubmit: expect.objectContaining({
        path: 'images/edits/jobs',
        method: 'POST',
        contentType: 'multipart',
        taskIdPath: 'id',
        useTaskIdAsIdempotencyKey: true,
      }),
      poll: {
        path: 'images/jobs/{task_id}',
        method: 'GET',
        intervalSeconds: 1,
        statusPath: 'status',
        successValues: ['completed'],
        failureValues: ['failed', 'cancelled'],
        errorPath: 'error.message',
        result: {
          imageUrlPaths: ['result.data.*.url'],
          b64JsonPaths: ['result.data.*.b64_json'],
        },
      },
    }])
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

    expect(safe.customProviders).toHaveLength(1)
    expect(safe.profiles).toEqual([
      expect.objectContaining({ provider: 'sub2api-async', baseUrl: '/v1', apiKey: 'sk-first', model: 'custom-image-model', responseFormatB64Json: true }),
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

  it('clears managed credentials and runs runtime redaction when configuration fails', async () => {
    const failure = new Error('configuration failed')
    const redactRuntime = vi.fn()

    await expect(runManagedConfigurationTransaction(async () => {
      activateManagedConfig(config(), DEFAULT_SETTINGS)
      throw failure
    }, redactRuntime)).rejects.toBe(failure)

    expect(getManagedSnapshot()).toBeNull()
    expect(redactRuntime).toHaveBeenCalledOnce()
  })

  it('preserves the original configuration error if runtime redaction also fails', async () => {
    const failure = new Error('original configuration failure')

    await expect(runManagedConfigurationTransaction(async () => {
      activateManagedConfig(config(), DEFAULT_SETTINGS)
      throw failure
    }, () => {
      throw new Error('redaction failed')
    })).rejects.toBe(failure)

    expect(getManagedSnapshot()).toBeNull()
  })

  it('fails closed after clear without invoking fetch', async () => {
    const release = requireManagedRuntime()
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('location', { origin: 'https://sub2api.example' })
    activateManagedConfig(config(), DEFAULT_SETTINGS)
    clearManagedConfig()

    await expect(managedFetch('/v1/images/generations/jobs', { method: 'POST' }))
      .rejects.toThrow('Sub2API configuration is not available')
    expect(fetchMock).not.toHaveBeenCalled()
    release()
  })

  it.each([
    'https://evil.example/v1/images/generations',
    '//evil.example/v1/images/generations',
    '/v1/chat/completions',
    '/v1/images/tasks/123',
    '/v1/images/jobs/imgjob_0123456789abcdef0123456789abcdef?secret=1',
    '/v1/images/jobs/imgjob_0123456789abcdef0123456789abcdef#fragment',
    '/v1/images/jobs/../responses',
    '/v1/images/jobs/not-a-job-id',
    '/v1/images/jobs/imgjob_0123456789abcdef0123456789abcdeg',
    'http://[::1',
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

  it.each([
    ['/v1/images/generations/jobs', 'GET'],
    ['/v1/images/edits/jobs', 'GET'],
    ['/v1/images/jobs/imgjob_0123456789abcdef0123456789abcdef', 'POST'],
    ['/v1/images/jobs/imgjob_0123456789abcdef0123456789abcdef/cancel', 'GET'],
    ['/v1/responses', 'GET'],
  ])('rejects the wrong method for a managed route: %s %s', async (url, method) => {
    const release = requireManagedRuntime()
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('location', { origin: 'https://sub2api.example' })
    activateManagedConfig(config(), DEFAULT_SETTINGS)

    await expect(managedFetch(url, { method })).rejects.toThrow('Managed request URL is not allowed')
    expect(fetchMock).not.toHaveBeenCalled()
    release()
  })

  it.each([
    ['/v1/images/generations/jobs', 'POST'],
    ['/v1/images/edits/jobs', 'POST'],
    ['/v1/images/jobs/imgjob_0123456789abcdef0123456789abcdef', 'GET'],
    ['/v1/images/jobs/imgjob_0123456789abcdef0123456789abcdef/cancel', 'POST'],
    ['/v1/responses', 'POST'],
  ])('allows only the exact same-origin managed route and method: %s %s', async (url, method) => {
    const release = requireManagedRuntime()
    const fetchMock = vi.fn().mockResolvedValue(new Response('{}'))
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('location', { origin: 'https://sub2api.example' })
    activateManagedConfig(config(), DEFAULT_SETTINGS)

    await managedFetch(url, { method })

    expect(fetchMock).toHaveBeenCalledOnce()
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
