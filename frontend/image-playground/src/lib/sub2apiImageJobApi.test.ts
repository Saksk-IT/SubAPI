import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { DEFAULT_SETTINGS } from './apiProfiles'
import { activateManagedConfig, clearManagedConfig } from './managedMode'
import { cancelSub2APIImageJob } from './sub2apiImageJobApi'

const JOB_ID = 'imgjob_0123456789abcdef0123456789abcdef'

describe('Sub2API image job cancellation', () => {
  beforeEach(() => {
    vi.stubGlobal('location', { origin: 'https://sub2api.example' })
    clearManagedConfig()
    activateManagedConfig({
      apiKey: 'sk-managed',
      apiKeyId: 7,
      apiKeyName: 'Image key',
      storageScope: '42',
      baseUrl: '/v1',
      profiles: [
        { id: 'images', name: 'Images', apiMode: 'images', model: 'gpt-image-2' },
        { id: 'responses', name: 'Responses', apiMode: 'responses', model: 'gpt-5.5' },
      ],
    }, DEFAULT_SETTINGS)
  })

  afterEach(() => {
    clearManagedConfig()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('posts an authenticated same-origin cancellation request', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      id: JOB_ID,
      status: 'running',
      cancel_requested: true,
    }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }))

    await expect(cancelSub2APIImageJob('sk-managed', JOB_ID)).resolves.toMatchObject({
      id: JOB_ID,
      status: 'running',
      cancel_requested: true,
    })

    expect(fetchMock).toHaveBeenCalledOnce()
    const [url, init] = fetchMock.mock.calls[0]
    expect(new URL(String(url)).pathname).toBe(`/v1/images/jobs/${JOB_ID}/cancel`)
    expect(init).toMatchObject({ method: 'POST', cache: 'no-store', redirect: 'error' })
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer sk-managed')
  })

  it('accepts an idempotent completed response so completion can win the race', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      id: JOB_ID,
      status: 'completed',
      result: { data: [{ b64_json: 'aW1hZ2U=' }] },
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    await expect(cancelSub2APIImageJob('sk-managed', JOB_ID)).resolves.toMatchObject({
      status: 'completed',
      result: { data: [{ b64_json: 'aW1hZ2U=' }] },
    })
  })

  it.each([
    '',
    'task-1',
    'imgjob_0123456789abcdef0123456789abcdeg',
    'imgjob_0123456789abcdef0123456789abcdef/../responses',
  ])('rejects malformed job id %j before fetch', async (jobId) => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')

    await expect(cancelSub2APIImageJob('sk-managed', jobId)).rejects.toThrow('任务 ID')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('rejects a mismatched or malformed server payload', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      id: 'imgjob_ffffffffffffffffffffffffffffffff',
      status: 'cancelled',
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    await expect(cancelSub2APIImageJob('sk-managed', JOB_ID)).rejects.toThrow('响应无效')
  })

  it('surfaces the API error without exposing the bearer key', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      error: { message: 'image generation job not found' },
    }), { status: 404, headers: { 'Content-Type': 'application/json' } }))

    await expect(cancelSub2APIImageJob('sk-managed', JOB_ID)).rejects.toThrow('image generation job not found')
  })
})
