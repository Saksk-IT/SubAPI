import type { PublicSettings } from '@/types'

export const APP_CONFIG_META_NAME = 'sub2api-app-config'

const decodeBase64Utf8 = (value: string): string => {
  const binary = atob(value)
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
  if (typeof TextDecoder !== 'undefined') {
    return new TextDecoder().decode(bytes)
  }
  return decodeURIComponent(
    Array.from(bytes, (byte) => `%${byte.toString(16).padStart(2, '0')}`).join('')
  )
}

const readMetaConfig = (): PublicSettings | null => {
  if (typeof document === 'undefined') return null

  const meta = document.querySelector<HTMLMetaElement>(`meta[name="${APP_CONFIG_META_NAME}"]`)
  const encodedConfig = meta?.content?.trim()
  if (!encodedConfig) return null

  try {
    const parsed = JSON.parse(decodeBase64Utf8(encodedConfig))
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null
    return parsed as PublicSettings
  } catch (error) {
    console.error('Failed to parse injected app config:', error)
    return null
  }
}

export const getInjectedAppConfig = (): PublicSettings | null => {
  if (typeof window !== 'undefined' && window.__APP_CONFIG__) {
    return { ...window.__APP_CONFIG__ }
  }

  const metaConfig = readMetaConfig()
  if (!metaConfig) return null

  if (typeof window !== 'undefined') {
    window.__APP_CONFIG__ = { ...metaConfig }
  }
  return { ...metaConfig }
}
