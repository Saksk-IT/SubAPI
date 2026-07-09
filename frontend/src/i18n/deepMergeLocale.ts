export type LocaleRecord = Record<string, unknown>

function isLocaleRecord(value: unknown): value is LocaleRecord {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

export function deepMergeLocale<T extends LocaleRecord, U extends LocaleRecord>(
  base: T,
  overrides: U
): T & U {
  const merged: LocaleRecord = { ...base }

  for (const [key, overrideValue] of Object.entries(overrides)) {
    const baseValue = merged[key]
    merged[key] =
      isLocaleRecord(baseValue) && isLocaleRecord(overrideValue)
        ? deepMergeLocale(baseValue, overrideValue)
        : overrideValue
  }

  return merged as T & U
}
