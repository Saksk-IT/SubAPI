import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post, put } = vi.hoisted(() => ({
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post,
    put,
  },
}))

import { create, generate, update } from '@/api/admin/redeem'

describe('admin redeem api', () => {
  beforeEach(() => {
    post.mockReset()
    put.mockReset()
    post.mockResolvedValue({ data: [] })
    put.mockResolvedValue({ data: {} })
  })

  it('creates a single redeem code via management API', async () => {
    post.mockResolvedValue({ data: { id: 1, code: 'R-1' } })

    const result = await create({ code: 'R-1', type: 'balance', value: 10, notes: 'manual' })

    expect(post).toHaveBeenCalledWith('/admin/redeem-codes', {
      code: 'R-1',
      type: 'balance',
      value: 10,
      notes: 'manual',
    })
    expect(result).toEqual({ id: 1, code: 'R-1' })
  })

  it('updates a redeem code via management API', async () => {
    put.mockResolvedValue({ data: { id: 2, code: 'R-2', status: 'expired' } })

    const result = await update(2, { status: 'expired' })

    expect(put).toHaveBeenCalledWith('/admin/redeem-codes/2', { status: 'expired' })
    expect(result).toEqual({ id: 2, code: 'R-2', status: 'expired' })
  })

  it('passes group and validity days for generated subscription codes', async () => {
    await generate(3, 'subscription', 0, 9, 45)

    expect(post).toHaveBeenCalledWith('/admin/redeem-codes/generate', {
      count: 3,
      type: 'subscription',
      value: 0,
      group_id: 9,
      validity_days: 45,
    })
  })
})
