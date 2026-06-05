import type { UserSubscription } from '@/types'
import type { SubscriptionPlan } from '@/types/payment'

const ONE_DAY_MS = 24 * 60 * 60 * 1000

export interface RemainingDurationParts {
  days: number
  hours: number
  minutes: number
}

export type SubscriptionValidityUnit = 'days' | 'weeks' | 'months'

type SubscriptionQuotaSource = Pick<
  SubscriptionPlan,
  'daily_limit_usd' | 'weekly_limit_usd' | 'monthly_limit_usd'
>

type SubscriptionValiditySource = Pick<SubscriptionPlan, 'validity_days' | 'validity_unit'>

export function isOneTimeDailyQuota(
  subscription: Pick<UserSubscription, 'starts_at' | 'expires_at'>,
): boolean {
  if (!subscription.starts_at || !subscription.expires_at) return false

  const startsAt = new Date(subscription.starts_at).getTime()
  const expiresAt = new Date(subscription.expires_at).getTime()

  if (!Number.isFinite(startsAt) || !Number.isFinite(expiresAt)) return false

  return expiresAt <= startsAt + ONE_DAY_MS
}

export function getRemainingDurationParts(
  targetAt: Date | string,
  now: Date = new Date(),
): RemainingDurationParts | null {
  const targetTime = targetAt instanceof Date ? targetAt.getTime() : new Date(targetAt).getTime()
  const nowTime = now.getTime()

  if (!Number.isFinite(targetTime) || !Number.isFinite(nowTime)) return null

  const diffMs = targetTime - nowTime
  if (diffMs <= 0) return null

  const totalMinutes = Math.floor(diffMs / (1000 * 60))
  const days = Math.floor(totalMinutes / (24 * 60))
  const hours = Math.floor((totalMinutes % (24 * 60)) / 60)
  const minutes = totalMinutes % 60

  return { days, hours, minutes }
}

export function normalizeSubscriptionValidityUnit(unit: string | null | undefined): SubscriptionValidityUnit {
  const normalized = String(unit || 'days').toLowerCase()
  if (normalized === 'week' || normalized === 'weeks') return 'weeks'
  if (normalized === 'month' || normalized === 'months') return 'months'
  return 'days'
}

export function normalizePositiveQuota(value: number | null | undefined): number | null {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric <= 0) return null
  return numeric
}

export function getSubscriptionCycleLimitUSD(
  source: SubscriptionQuotaSource | null | undefined,
  unit: string | null | undefined,
): number | null {
  if (!source) return null
  const normalizedUnit = normalizeSubscriptionValidityUnit(unit)
  if (normalizedUnit === 'weeks') return normalizePositiveQuota(source.weekly_limit_usd)
  if (normalizedUnit === 'months') return normalizePositiveQuota(source.monthly_limit_usd)
  return normalizePositiveQuota(source.daily_limit_usd)
}

export function getLargestActiveSubscriptionQuotaUnit(
  source: SubscriptionQuotaSource | null | undefined,
): SubscriptionValidityUnit | null {
  if (!source) return null
  if (normalizePositiveQuota(source.monthly_limit_usd) != null) return 'months'
  if (normalizePositiveQuota(source.weekly_limit_usd) != null) return 'weeks'
  if (normalizePositiveQuota(source.daily_limit_usd) != null) return 'days'
  return null
}

export function deriveSubscriptionValidityUnitFromQuota(
  source: SubscriptionQuotaSource | null | undefined,
): SubscriptionValidityUnit {
  return getLargestActiveSubscriptionQuotaUnit(source) ?? 'days'
}

export function getSmallestActiveSubscriptionQuotaUnit(
  source: SubscriptionQuotaSource | null | undefined,
): SubscriptionValidityUnit | null {
  if (!source) return null
  if (normalizePositiveQuota(source.daily_limit_usd) != null) return 'days'
  if (normalizePositiveQuota(source.weekly_limit_usd) != null) return 'weeks'
  if (normalizePositiveQuota(source.monthly_limit_usd) != null) return 'months'
  return null
}

export function deriveSubscriptionValidityUnitFromMinimumQuota(
  source: SubscriptionQuotaSource | null | undefined,
): SubscriptionValidityUnit {
  return getSmallestActiveSubscriptionQuotaUnit(source) ?? 'days'
}

export function calculateSubscriptionTotalQuotaUSD(
  plan: SubscriptionValiditySource,
  source: SubscriptionQuotaSource | null | undefined,
): number | null {
  const count = Number(plan.validity_days) || 0
  const cycleLimit = getSubscriptionCycleLimitUSD(source, plan.validity_unit)
  if (count <= 0 || cycleLimit == null) return null
  return Math.round(cycleLimit * count * 100) / 100
}

export function calculateSubscriptionPlanPriceUSD(
  plan: SubscriptionValiditySource,
  source: SubscriptionQuotaSource | null | undefined,
  groupActualRateMultiplier: number | null | undefined,
  planPriceMultiplier: number | null | undefined,
): number | null {
  const count = Number(plan.validity_days) || 0
  const cycleLimit = getSubscriptionCycleLimitUSD(source, plan.validity_unit)
  const actualRate = Number(groupActualRateMultiplier)
  const priceMultiplier = Number(planPriceMultiplier)
  if (
    count <= 0 ||
    cycleLimit == null ||
    !Number.isFinite(actualRate) ||
    actualRate <= 0 ||
    !Number.isFinite(priceMultiplier) ||
    priceMultiplier <= 0
  ) {
    return null
  }
  return Math.round((cycleLimit / actualRate) * count * priceMultiplier * 100) / 100
}

export function formatSubscriptionValidityUnit(
  value: number,
  unit: string | null | undefined,
  labels: Record<SubscriptionValidityUnit, string>,
): string {
  const normalizedUnit = normalizeSubscriptionValidityUnit(unit)
  return `${value}${labels[normalizedUnit]}`
}
