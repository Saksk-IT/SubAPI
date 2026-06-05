import { describe, expect, it } from 'vitest'
import {
  calculateSubscriptionPlanPriceUSD,
  calculateSubscriptionTotalQuotaUSD,
  deriveSubscriptionValidityUnitFromMinimumQuota,
  deriveSubscriptionValidityUnitFromQuota,
  formatSubscriptionValidityUnit,
  getLargestActiveSubscriptionQuotaUnit,
  getSmallestActiveSubscriptionQuotaUnit,
  getSubscriptionCycleLimitUSD,
} from '../subscriptionQuota'

const quotaSource = {
  daily_limit_usd: 23,
  weekly_limit_usd: 140,
  monthly_limit_usd: 500,
}

describe('subscriptionQuota', () => {
  it('selects the limit matching the validity unit', () => {
    expect(getSubscriptionCycleLimitUSD(quotaSource, 'days')).toBe(23)
    expect(getSubscriptionCycleLimitUSD(quotaSource, 'weeks')).toBe(140)
    expect(getSubscriptionCycleLimitUSD(quotaSource, 'months')).toBe(500)
  })

  it('calculates total quota from validity count and matching cycle limit', () => {
    expect(calculateSubscriptionTotalQuotaUSD({ validity_days: 30, validity_unit: 'days' }, quotaSource)).toBe(690)
    expect(calculateSubscriptionTotalQuotaUSD({ validity_days: 4, validity_unit: 'weeks' }, quotaSource)).toBe(560)
    expect(calculateSubscriptionTotalQuotaUSD({ validity_days: 2, validity_unit: 'months' }, quotaSource)).toBe(1000)
  })

  it('treats missing or zero cycle limits as unlimited', () => {
    expect(calculateSubscriptionTotalQuotaUSD({ validity_days: 2, validity_unit: 'months' }, { ...quotaSource, monthly_limit_usd: null })).toBeNull()
    expect(calculateSubscriptionTotalQuotaUSD({ validity_days: 2, validity_unit: 'months' }, { ...quotaSource, monthly_limit_usd: 0 })).toBeNull()
  })

  it('derives the validity unit from the largest active quota period', () => {
    expect(getLargestActiveSubscriptionQuotaUnit({
      daily_limit_usd: 10,
      weekly_limit_usd: 50,
      monthly_limit_usd: 0,
    })).toBe('weeks')
    expect(getLargestActiveSubscriptionQuotaUnit({
      daily_limit_usd: 10,
      weekly_limit_usd: 50,
      monthly_limit_usd: 200,
    })).toBe('months')
    expect(getLargestActiveSubscriptionQuotaUnit({
      daily_limit_usd: 10,
      weekly_limit_usd: 0,
      monthly_limit_usd: null,
    })).toBe('days')
    expect(getLargestActiveSubscriptionQuotaUnit({
      daily_limit_usd: 0,
      weekly_limit_usd: null,
      monthly_limit_usd: 0,
    })).toBeNull()
    expect(deriveSubscriptionValidityUnitFromQuota({
      daily_limit_usd: 0,
      weekly_limit_usd: null,
      monthly_limit_usd: 0,
    })).toBe('days')
  })

  it('derives the validity unit from the smallest active quota period', () => {
    expect(getSmallestActiveSubscriptionQuotaUnit({
      daily_limit_usd: 10,
      weekly_limit_usd: 50,
      monthly_limit_usd: 200,
    })).toBe('days')
    expect(getSmallestActiveSubscriptionQuotaUnit({
      daily_limit_usd: 0,
      weekly_limit_usd: 50,
      monthly_limit_usd: 200,
    })).toBe('weeks')
    expect(getSmallestActiveSubscriptionQuotaUnit({
      daily_limit_usd: null,
      weekly_limit_usd: 0,
      monthly_limit_usd: 200,
    })).toBe('months')
    expect(getSmallestActiveSubscriptionQuotaUnit({
      daily_limit_usd: 0,
      weekly_limit_usd: null,
      monthly_limit_usd: 0,
    })).toBeNull()
    expect(deriveSubscriptionValidityUnitFromMinimumQuota({
      daily_limit_usd: 0,
      weekly_limit_usd: null,
      monthly_limit_usd: 0,
    })).toBe('days')
  })

  it('calculates subscription plan price from minimum cycle cost, actual group rate, cycle count, and plan multiplier', () => {
    expect(calculateSubscriptionPlanPriceUSD(
      { validity_days: 7, validity_unit: 'days' },
      { daily_limit_usd: 120, weekly_limit_usd: null, monthly_limit_usd: null },
      2.5,
      0.14,
    )).toBe(47.04)
  })

  it('does not calculate subscription plan price when formula inputs are invalid', () => {
    expect(calculateSubscriptionPlanPriceUSD({ validity_days: 0, validity_unit: 'days' }, quotaSource, 2.5, 0.14)).toBeNull()
    expect(calculateSubscriptionPlanPriceUSD({ validity_days: 7, validity_unit: 'days' }, quotaSource, 0, 0.14)).toBeNull()
    expect(calculateSubscriptionPlanPriceUSD({ validity_days: 7, validity_unit: 'days' }, quotaSource, 2.5, 0)).toBeNull()
    expect(calculateSubscriptionPlanPriceUSD({ validity_days: 7, validity_unit: 'days' }, { ...quotaSource, daily_limit_usd: null }, 2.5, 0.14)).toBeNull()
  })

  it('formats validity unit labels from the selected unit', () => {
    const labels = { days: '天', weeks: '周', months: '月' }
    expect(formatSubscriptionValidityUnit(30, 'days', labels)).toBe('30天')
    expect(formatSubscriptionValidityUnit(4, 'weeks', labels)).toBe('4周')
    expect(formatSubscriptionValidityUnit(2, 'months', labels)).toBe('2月')
  })
})
