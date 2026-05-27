import { describe, expect, it } from 'vitest'
import {
  calculateSubscriptionTotalQuotaUSD,
  formatSubscriptionValidityUnit,
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

  it('formats validity unit labels from the selected unit', () => {
    const labels = { days: '天', weeks: '周', months: '月' }
    expect(formatSubscriptionValidityUnit(30, 'days', labels)).toBe('30天')
    expect(formatSubscriptionValidityUnit(4, 'weeks', labels)).toBe('4周')
    expect(formatSubscriptionValidityUnit(2, 'months', labels)).toBe('2月')
  })
})
