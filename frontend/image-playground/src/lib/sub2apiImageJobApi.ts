import { getApiErrorMessage } from './imageApiShared'
import { managedFetch } from './managedMode'

const SUB2API_IMAGE_JOB_ID = /^imgjob_[0-9a-f]{32}$/
const SUB2API_IMAGE_JOB_STATUSES = new Set([
  'queued',
  'running',
  'completed',
  'failed',
  'cancelled',
])

export type Sub2APIImageJobStatus = 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface Sub2APIImageJobResponse {
  id: string
  status: Sub2APIImageJobStatus
  cancel_requested?: boolean
  result?: unknown
  error?: { code?: string; message?: string }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function parseImageJobResponse(payload: unknown, expectedJobId: string): Sub2APIImageJobResponse {
  if (!isRecord(payload)) throw new Error('生图任务取消响应无效。')
  const id = typeof payload.id === 'string' ? payload.id : ''
  const status = typeof payload.status === 'string' ? payload.status : ''
  if (id !== expectedJobId || !SUB2API_IMAGE_JOB_STATUSES.has(status)) {
    throw new Error('生图任务取消响应无效。')
  }
  return payload as unknown as Sub2APIImageJobResponse
}

export function isSub2APIImageJobId(jobId: string): boolean {
  return SUB2API_IMAGE_JOB_ID.test(jobId)
}

export async function cancelSub2APIImageJob(
  apiKey: string,
  jobId: string,
  signal?: AbortSignal,
): Promise<Sub2APIImageJobResponse> {
  const normalizedKey = apiKey.trim()
  if (!normalizedKey) throw new Error('缺少可用于取消任务的 API 密钥。')
  if (!isSub2APIImageJobId(jobId)) throw new Error('生图任务 ID 无效。')

  const response = await managedFetch(`/v1/images/jobs/${jobId}/cancel`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${normalizedKey}`,
    },
    cache: 'no-store',
    signal,
  })
  if (!response.ok) throw new Error(await getApiErrorMessage(response))

  let payload: unknown
  try {
    payload = await response.json()
  } catch {
    throw new Error('生图任务取消响应无效。')
  }
  return parseImageJobResponse(payload, jobId)
}
