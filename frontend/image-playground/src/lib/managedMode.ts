import type { AppSettings } from '../types'
import { createDefaultOpenAIProfile, normalizeSettings } from './apiProfiles'
import type { ManagedConfig } from './sub2apiBridge'

interface ManagedSnapshot {
  config: ManagedConfig
  generation: number
  controller: AbortController
}

let generation = 0
let snapshot: ManagedSnapshot | null = null
let managedRuntimeRequirements = 0

const ALLOWED_MANAGED_PATHS = new Set([
  '/v1/images/generations',
  '/v1/images/edits',
  '/v1/responses',
])

export function redactSettingsSecrets(settings: AppSettings): AppSettings {
  const normalized = normalizeSettings(settings)
  return {
    ...normalized,
    apiKey: '',
    profiles: normalized.profiles.map((profile) => ({ ...profile, apiKey: '' })),
  }
}

function getPreservedModel(settings: AppSettings, apiMode: 'images' | 'responses') {
  const model = settings.profiles.find((profile) => profile.apiMode === apiMode)?.model.trim()
  return model && model.length <= 128 ? model : null
}

function buildManagedSettings(config: ManagedConfig, previousSettings: AppSettings, preserveModels: boolean) {
  const profiles = config.profiles.map((profile) => createDefaultOpenAIProfile({
    id: profile.id,
    name: profile.name,
    provider: 'openai',
    baseUrl: '/v1',
    apiKey: config.apiKey,
    model: preserveModels ? getPreservedModel(previousSettings, profile.apiMode) ?? profile.model : profile.model,
    apiMode: profile.apiMode,
    codexCli: false,
    apiProxy: false,
    responseFormatB64Json: true,
  }))
  const imagesProfile = profiles.find((profile) => profile.apiMode === 'images')!
  const responsesProfile = profiles.find((profile) => profile.apiMode === 'responses')!
  const settings = normalizeSettings({
    ...previousSettings,
    baseUrl: imagesProfile.baseUrl,
    apiKey: imagesProfile.apiKey,
    model: imagesProfile.model,
    apiMode: imagesProfile.apiMode,
    codexCli: false,
    apiProxy: false,
    customProviders: [],
    providerOrder: ['openai'],
    profiles,
    activeProfileId: imagesProfile.id,
    agentApiConfigMode: 'hybrid',
    agentTextProfileId: responsesProfile.id,
    agentImageProfileId: imagesProfile.id,
  })
  return { ...settings, providerOrder: ['openai'] }
}

export function activateManagedConfig(config: ManagedConfig, previousSettings: AppSettings): AppSettings {
  const preserveModels = snapshot?.config.storageScope === config.storageScope
  snapshot?.controller.abort()
  generation += 1
  snapshot = { config, generation, controller: new AbortController() }
  return buildManagedSettings(config, previousSettings, preserveModels)
}

function profilesHaveSameIdentity(first: ManagedConfig['profiles'], second: ManagedConfig['profiles']) {
  return first.length === second.length && first.every((profile, index) => {
    const other = second[index]
    return Boolean(
      other &&
      profile.id === other.id &&
      profile.name === other.name &&
      profile.apiMode === other.apiMode &&
      profile.model === other.model
    )
  })
}

export function hasSameManagedCredentialIdentity(config: ManagedConfig) {
  if (!snapshot) return false
  const current = snapshot.config
  return current.apiKey === config.apiKey &&
    current.apiKeyId === config.apiKeyId &&
    current.storageScope === config.storageScope &&
    profilesHaveSameIdentity(current.profiles, config.profiles)
}

export function updateManagedPresentationConfig(config: ManagedConfig) {
  if (!snapshot || !hasSameManagedCredentialIdentity(config)) return false
  snapshot = { ...snapshot, config }
  return true
}

export function enforceManagedSettings(settings: AppSettings) {
  if (!snapshot) return settings
  return buildManagedSettings(snapshot.config, settings, true)
}

export function clearManagedConfig() {
  snapshot?.controller.abort()
  snapshot = null
  generation += 1
}

export async function runManagedConfigurationTransaction<T>(
  operation: () => T | Promise<T>,
  redactRuntimeSettings: () => void = () => undefined,
): Promise<T> {
  try {
    return await operation()
  } catch (error) {
    clearManagedConfig()
    try {
      redactRuntimeSettings()
    } catch {
      // Preserve the original configuration failure after clearing the snapshot.
    }
    throw error
  }
}

export function isManagedMode() {
  return snapshot !== null
}

export function getManagedSnapshot() {
  return snapshot
}

export function getManagedGenerationSignal() {
  if (!snapshot) return AbortSignal.abort('Sub2API configuration is not available')
  return snapshot.controller.signal
}

export function getManagedDatabaseName() {
  if (!snapshot) throw new Error('Sub2API storage scope is not configured')
  return `gpt-image-playground-sub2api-${snapshot.config.storageScope}`
}

export function getManagedStorageName() {
  return snapshot
    ? `gpt-image-playground-sub2api-${snapshot.config.storageScope}`
    : 'gpt-image-playground'
}

export function applyManagedPresentation(
  config: ManagedConfig,
  target: Pick<Document, 'documentElement'> = document,
) {
  const theme = config.theme ?? 'light'
  target.documentElement.classList.toggle('dark', theme === 'dark')
  target.documentElement.style.colorScheme = theme
  if (config.locale) target.documentElement.lang = config.locale
}

export function requireManagedRuntime() {
  managedRuntimeRequirements += 1
  let released = false
  return () => {
    if (released) return
    released = true
    managedRuntimeRequirements = Math.max(0, managedRuntimeRequirements - 1)
  }
}

function combineSignals(first: AbortSignal | null | undefined, second: AbortSignal) {
  if (!first) return second
  if (typeof AbortSignal.any === 'function') return AbortSignal.any([first, second])
  const controller = new AbortController()
  const abort = (event: Event) => controller.abort((event.target as AbortSignal).reason)
  if (first.aborted) controller.abort(first.reason)
  else first.addEventListener('abort', abort, { once: true })
  if (second.aborted) controller.abort(second.reason)
  else second.addEventListener('abort', abort, { once: true })
  return controller.signal
}

export function managedFetch(input: RequestInfo | URL, init: RequestInit = {}) {
  if (!snapshot) {
    if (managedRuntimeRequirements > 0) {
      return Promise.reject(new Error('Sub2API configuration is not available'))
    }
    return fetch(input, init)
  }

  const rawUrl = input instanceof Request ? input.url : String(input)
  const origin = globalThis.location?.origin
  if (!origin) return Promise.reject(new Error('Managed request URL is not allowed'))
  const url = new URL(rawUrl, origin)
  if (
    url.origin !== origin ||
    !ALLOWED_MANAGED_PATHS.has(url.pathname) ||
    url.search !== '' ||
    url.hash !== ''
  ) {
    return Promise.reject(new Error('Managed request URL is not allowed'))
  }

  return fetch(url.toString(), {
    ...init,
    redirect: 'error',
    signal: combineSignals(init.signal, snapshot.controller.signal),
  })
}
