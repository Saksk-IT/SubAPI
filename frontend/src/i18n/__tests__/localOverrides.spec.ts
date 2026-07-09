import { describe, expect, it } from 'vitest'

import { deepMergeLocale } from '../deepMergeLocale'
import en from '../locales/en'
import enAdminAccounts from '../locales/en/admin/accounts'
import enAdminChannels from '../locales/en/admin/channels'
import enAdminOverrides from '../locales/en/admin/local-overrides'
import enAdminOps from '../locales/en/admin/ops'
import enAdminOverview from '../locales/en/admin/overview'
import enAdminResources from '../locales/en/admin/resources'
import enAdminSettings from '../locales/en/admin/settings'
import enCommon from '../locales/en/common'
import enDashboard from '../locales/en/dashboard'
import enLanding from '../locales/en/landing'
import enRootOverrides from '../locales/en/local-overrides'
import enMisc from '../locales/en/misc'
import zh from '../locales/zh'
import zhAdminAccounts from '../locales/zh/admin/accounts'
import zhAdminChannels from '../locales/zh/admin/channels'
import zhAdminOverrides from '../locales/zh/admin/local-overrides'
import zhAdminOps from '../locales/zh/admin/ops'
import zhAdminOverview from '../locales/zh/admin/overview'
import zhAdminResources from '../locales/zh/admin/resources'
import zhAdminSettings from '../locales/zh/admin/settings'
import zhCommon from '../locales/zh/common'
import zhDashboard from '../locales/zh/dashboard'
import zhLanding from '../locales/zh/landing'
import zhRootOverrides from '../locales/zh/local-overrides'
import zhMisc from '../locales/zh/misc'

type LocaleObject = Record<string, unknown>

function flatten(value: unknown, prefix = '', out = new Map<string, unknown>()): Map<string, unknown> {
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    for (const [key, child] of Object.entries(value)) {
      flatten(child, prefix ? `${prefix}.${key}` : key, out)
    }
  } else {
    out.set(prefix, value)
  }
  return out
}

function getPath(value: LocaleObject, path: string): unknown {
  return path.split('.').reduce<unknown>((current, key) => {
    if (!current || typeof current !== 'object') return undefined
    return (current as LocaleObject)[key]
  }, value)
}

const upstreamLocales = {
  en: {
    ...enLanding,
    ...enCommon,
    ...enDashboard,
    admin: {
      ...enAdminOverview,
      ...enAdminChannels,
      ...enAdminAccounts,
      ...enAdminResources,
      ...enAdminOps,
      ...enAdminSettings
    },
    ...enMisc
  },
  zh: {
    ...zhLanding,
    ...zhCommon,
    ...zhDashboard,
    admin: {
      ...zhAdminOverview,
      ...zhAdminChannels,
      ...zhAdminAccounts,
      ...zhAdminResources,
      ...zhAdminOps,
      ...zhAdminSettings
    },
    ...zhMisc
  }
}

describe('local locale overrides', () => {
  it('deep-merges nested overrides without dropping upstream siblings', () => {
    expect(
      deepMergeLocale(
        { admin: { users: { role: 'Role', status: 'Status' } } },
        { admin: { users: { batchDelete: 'Batch delete' } } }
      )
    ).toEqual({
      admin: { users: { role: 'Role', status: 'Status', batchDelete: 'Batch delete' } }
    })
  })

  it('keeps English and Chinese local override key paths in parity', () => {
    const enPaths = [
      ...flatten(enRootOverrides).keys(),
      ...[...flatten(enAdminOverrides).keys()].map((path) => `admin.${path}`)
    ].sort()
    const zhPaths = [
      ...flatten(zhRootOverrides).keys(),
      ...[...flatten(zhAdminOverrides).keys()].map((path) => `admin.${path}`)
    ].sort()

    expect(enPaths).toEqual(zhPaths)
    expect(enPaths).toHaveLength(342)
  })

  it.each([
    ['en', en, enRootOverrides, enAdminOverrides],
    ['zh', zh, zhRootOverrides, zhAdminOverrides]
  ] as const)('applies every %s local override to the assembled locale', (_locale, assembled, root, admin) => {
    for (const [path, expected] of flatten(root)) {
      expect(getPath(assembled as LocaleObject, path), path).toEqual(expected)
    }
    for (const [path, expected] of flatten(admin)) {
      expect(getPath(assembled as LocaleObject, `admin.${path}`), `admin.${path}`).toEqual(expected)
    }
  })

  it.each([
    ['en', en, enRootOverrides, enAdminOverrides],
    ['zh', zh, zhRootOverrides, zhAdminOverrides]
  ] as const)('preserves every upstream %s leaf unless a local override intentionally replaces it', (locale, assembled, root, admin) => {
    const overrideLeaves = new Map([
      ...flatten(root),
      ...[...flatten(admin)].map(([path, value]) => [`admin.${path}`, value] as const)
    ])

    for (const [path, upstreamValue] of flatten(upstreamLocales[locale])) {
      const expected = overrideLeaves.has(path) ? overrideLeaves.get(path) : upstreamValue
      expect(getPath(assembled as LocaleObject, path), path).toEqual(expected)
    }
  })

  it('keeps representative local and upstream features together', () => {
    for (const locale of [en, zh]) {
      expect(locale.admin.users.batchDelete.enable).toBeTruthy()
      expect(locale.admin.groups.rateSchedule.title).toBeTruthy()
      expect(locale.admin.dataDashboard.title).toBeTruthy()
      expect(locale.admin.settings.customMenu.openInNewTab).toBeTruthy()
      expect(locale.payment.purchaseGuide.title).toBeTruthy()
      expect(locale.firstRecharge.banner.title).toBeTruthy()

      expect(locale.nav.batchImage).toBeTruthy()
      expect(locale.common.availableBalance).toBeTruthy()
      expect(locale.admin.users.form.roleLabel).toBeTruthy()
      expect(locale.admin.usage.tokenRanking.subtitle).toBeTruthy()
      expect(locale.version.rollback).toBeTruthy()
    }
  })
})
