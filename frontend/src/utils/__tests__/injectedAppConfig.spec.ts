import { afterEach, describe, expect, it, vi } from 'vitest'

import { APP_CONFIG_META_NAME, getInjectedAppConfig } from '@/utils/injectedAppConfig'

const encodeConfig = (value: unknown): string => {
  const json = JSON.stringify(value)
  const bytes = new TextEncoder().encode(json)
  const binary = Array.from(bytes, (byte) => String.fromCharCode(byte)).join('')
  return btoa(binary)
}

describe('injectedAppConfig', () => {
  afterEach(() => {
    document.head.innerHTML = ''
    delete window.__APP_CONFIG__
    vi.restoreAllMocks()
  })

  it('returns existing window config first', () => {
    window.__APP_CONFIG__ = {
      site_name: 'Window Config',
    } as any
    document.head.innerHTML = `<meta name="${APP_CONFIG_META_NAME}" content="${encodeConfig({ site_name: 'Meta Config' })}">`

    expect(getInjectedAppConfig()?.site_name).toBe('Window Config')
  })

  it('decodes public settings from meta config and mirrors them to window', () => {
    document.head.innerHTML = `<meta name="${APP_CONFIG_META_NAME}" content="${encodeConfig({ site_name: '元配置' })}">`

    const config = getInjectedAppConfig()

    expect(config?.site_name).toBe('元配置')
    expect(window.__APP_CONFIG__?.site_name).toBe('元配置')
  })

  it('returns null and logs when meta config is invalid', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    document.head.innerHTML = `<meta name="${APP_CONFIG_META_NAME}" content="not-base64-json">`

    expect(getInjectedAppConfig()).toBeNull()
    expect(consoleError).toHaveBeenCalled()
  })
})
