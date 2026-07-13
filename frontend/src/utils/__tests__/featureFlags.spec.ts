import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAppStore } from '@/stores/app'
import type { PublicSettings } from '@/types'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

describe('image generation feature flag', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('registers image generation as an opt-out public setting', () => {
    expect(FeatureFlags.imageGeneration).toEqual({
      key: 'image_generation_enabled',
      mode: 'opt-out',
      label: 'Image Generation',
    })
  })

  it.each([
    ['a missing value', {}, true],
    ['an explicit true value', { image_generation_enabled: true }, true],
    ['an explicit false value', { image_generation_enabled: false }, false],
  ])('resolves %s to %s', (_case, settings, expected) => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = settings as PublicSettings

    expect(isFeatureFlagEnabled(FeatureFlags.imageGeneration)).toBe(expected)
  })
})
