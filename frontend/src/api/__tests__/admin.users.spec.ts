import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post,
  },
}))

import {
  batchAssign,
  bindUserAuthIdentity,
  type AdminBindAuthIdentityRequest,
  type AdminBoundAuthIdentity,
  type BatchAssignUsersRequest,
  type BatchAssignUsersResult,
} from '@/api/admin/users'

type Assert<T extends true> = T
type IsExact<T, U> = (
  (<G>() => G extends T ? 1 : 2) extends (<G>() => G extends U ? 1 : 2)
    ? ((<G>() => G extends U ? 1 : 2) extends (<G>() => G extends T ? 1 : 2) ? true : false)
    : false
)

type ExpectedAdminBindAuthIdentityRequest = {
  provider_type: string
  provider_key: string
  provider_subject: string
  issuer?: string
  metadata?: Record<string, unknown>
  channel?: {
    channel: string
    channel_app_id: string
    channel_subject: string
    metadata?: Record<string, unknown>
  }
}

type ExpectedAdminBoundAuthIdentity = {
  user_id: number
  provider_type: string
  provider_key: string
  provider_subject: string
  verified_at?: string | null
  issuer?: string | null
  metadata: Record<string, unknown> | null
  created_at: string
  updated_at: string
  channel?: {
    channel: string
    channel_app_id: string
    channel_subject: string
    metadata: Record<string, unknown> | null
    created_at: string
    updated_at: string
  } | null
}

type ExpectedBatchAssignUsersRequest = {
  user_ids?: number[]
  all?: boolean
  balance?: {
    operation: 'add' | 'subtract'
    amount: number
    notes?: string
  }
  subscription?: {
    group_id: number
    validity_days: number
    notes?: string
  }
}

type ExpectedBatchAssignUsersResult = {
  target_count: number
  success_count: number
  failed_count: number
  balance_affected_count: number
  subscription_assigned: number
  subscription_extended: number
  errors?: string[]
}

const requestContractExact: Assert<
  IsExact<AdminBindAuthIdentityRequest, ExpectedAdminBindAuthIdentityRequest>
> = true
const responseContractExact: Assert<
  IsExact<AdminBoundAuthIdentity, ExpectedAdminBoundAuthIdentity>
> = true
const batchAssignRequestContractExact: Assert<
  IsExact<BatchAssignUsersRequest, ExpectedBatchAssignUsersRequest>
> = true
const batchAssignResultContractExact: Assert<
  IsExact<BatchAssignUsersResult, ExpectedBatchAssignUsersResult>
> = true

describe('admin users api auth identity binding', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('posts the backend-compatible auth identity bind payload and returns the backend response shape', async () => {
    const payload: AdminBindAuthIdentityRequest = {
      provider_type: 'wechat',
      provider_key: 'wechat-main',
      provider_subject: 'union-123',
      metadata: { source: 'admin-repair' },
      channel: {
        channel: 'open',
        channel_app_id: 'wx-open',
        channel_subject: 'openid-123',
        metadata: { scene: 'migration' },
      },
    }

    const response: AdminBoundAuthIdentity = {
      user_id: 9,
      provider_type: 'wechat',
      provider_key: 'wechat-main',
      provider_subject: 'union-123',
      verified_at: '2026-04-22T00:00:00Z',
      issuer: null,
      metadata: { source: 'admin-repair' },
      created_at: '2026-04-22T00:00:00Z',
      updated_at: '2026-04-22T00:00:00Z',
      channel: {
        channel: 'open',
        channel_app_id: 'wx-open',
        channel_subject: 'openid-123',
        metadata: { scene: 'migration' },
        created_at: '2026-04-22T00:00:00Z',
        updated_at: '2026-04-22T00:00:00Z',
      },
    }
    post.mockResolvedValue({ data: response })

    const result = await bindUserAuthIdentity(9, payload)

    expect(post).toHaveBeenCalledWith('/admin/users/9/auth-identities', payload)
    expect(result).toEqual(response)
  })

  it('keeps bind auth identity request and response types aligned with the backend contract', () => {
    expect(requestContractExact).toBe(true)
    expect(responseContractExact).toBe(true)
  })

  it('posts batch assignment payloads to the users management endpoint', async () => {
    const response: BatchAssignUsersResult = {
      target_count: 12,
      success_count: 10,
      failed_count: 2,
      balance_affected_count: 10,
      subscription_assigned: 0,
      subscription_extended: 0,
      errors: ['user 3: balance cannot be negative'],
    }
    post.mockResolvedValue({ data: response })

    const payload: BatchAssignUsersRequest = {
      all: true,
      balance: {
        operation: 'subtract',
        amount: 2.5,
        notes: 'manual batch adjustment',
      },
    }

    const result = await batchAssign(payload)

    expect(post).toHaveBeenCalledWith('/admin/users/batch-assign', payload)
    expect(result).toEqual(response)
  })

  it('keeps batch assignment request and response types aligned with the backend contract', () => {
    expect(batchAssignRequestContractExact).toBe(true)
    expect(batchAssignResultContractExact).toBe(true)
  })
})
