import { computed, ref } from 'vue'

import { keysAPI } from '@/api/keys'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey } from '@/types'

const PAGE_SIZE = 100
const hasImageGenerationAccess = ref(false)
const imageGenerationAccessLoaded = ref(false)
const imageGenerationAccessLoading = ref(false)
const imageGenerationAccessError = ref<string | null>(null)
let accessOwnerId: number | null = null
let accessRequestGeneration = 0
let pendingAccessLoad: { ownerId: number; generation: number; promise: Promise<boolean> } | null = null

function currentUserId(): number | null {
  const authStore = useAuthStore()
  const userId = authStore.user?.id
  return authStore.isAuthenticated && Number.isSafeInteger(userId) && Number(userId) > 0
    ? Number(userId)
    : null
}

function resetAccessState(ownerId: number | null, loaded: boolean): void {
  accessRequestGeneration += 1
  pendingAccessLoad = null
  accessOwnerId = ownerId
  hasImageGenerationAccess.value = false
  imageGenerationAccessLoaded.value = loaded
  imageGenerationAccessLoading.value = false
  imageGenerationAccessError.value = null
}

function errorMessage(error: unknown): string {
  return error instanceof Error && error.message ? error.message : 'Failed to load API keys'
}

export function isImageGenerationKey(key: ApiKey): boolean {
  return (
    key.status === 'active' &&
    key.group?.platform === 'openai' &&
    key.group.allow_image_generation === true
  )
}

export function useImageGenerationAccess() {
  async function refreshImageGenerationAccess(force = false): Promise<boolean> {
    const authStore = useAuthStore()
    const userId = currentUserId()
    if (userId === null) {
      resetAccessState(null, true)
      return false
    }

    if (accessOwnerId !== userId) resetAccessState(userId, false)
    if (pendingAccessLoad?.ownerId === userId) return pendingAccessLoad.promise
    if (imageGenerationAccessLoaded.value && !force) return hasImageGenerationAccess.value

    const generation = ++accessRequestGeneration
    imageGenerationAccessLoading.value = true
    imageGenerationAccessError.value = null
    const isCurrentRequest = () => (
      generation === accessRequestGeneration &&
      accessOwnerId === userId &&
      authStore.isAuthenticated &&
      authStore.user?.id === userId
    )
    const promise = (async () => {
      let eligible = false
      let page = 1
      let pages = 1

      do {
        const response = await keysAPI.list(page, PAGE_SIZE, {
          status: 'active',
          sort_by: 'created_at',
          sort_order: 'desc',
        })
        eligible = (response.items || []).some((key) => (
          key.user_id === userId && isImageGenerationKey(key)
        ))
        pages = Math.max(1, Number(response.pages) || 1)
        page += 1
      } while (!eligible && page <= pages && isCurrentRequest())

      if (!isCurrentRequest()) return false
      hasImageGenerationAccess.value = eligible
      imageGenerationAccessLoaded.value = true
      return eligible
    })()
      .catch((error: unknown) => {
        if (isCurrentRequest()) {
          hasImageGenerationAccess.value = false
          imageGenerationAccessLoaded.value = true
          imageGenerationAccessError.value = errorMessage(error)
        }
        return false
      })
      .finally(() => {
        if (generation === accessRequestGeneration && accessOwnerId === userId) {
          imageGenerationAccessLoading.value = false
        }
        if (pendingAccessLoad?.generation === generation) pendingAccessLoad = null
      })

    pendingAccessLoad = { ownerId: userId, generation, promise }
    return promise
  }

  return {
    canUseImageGeneration: computed(() => hasImageGenerationAccess.value),
    imageGenerationAccessLoaded,
    imageGenerationAccessLoading,
    imageGenerationAccessError,
    refreshImageGenerationAccess,
  }
}

export function useImageGenerationKeys() {
  const imageGenerationKeys = ref<ApiKey[]>([])
  const imageGenerationKeysLoaded = ref(false)
  const imageGenerationKeysLoading = ref(false)
  const imageGenerationKeysError = ref<string | null>(null)
  let ownerId: number | null = null
  let requestGeneration = 0
  let pendingLoad: { ownerId: number; generation: number; promise: Promise<ApiKey[]> } | null = null

  function clearImageGenerationKeys(): void {
    requestGeneration += 1
    pendingLoad = null
    ownerId = null
    imageGenerationKeys.value = []
    imageGenerationKeysLoaded.value = false
    imageGenerationKeysLoading.value = false
    imageGenerationKeysError.value = null
  }

  async function refreshImageGenerationKeys(force = false): Promise<ApiKey[]> {
    const authStore = useAuthStore()
    const userId = currentUserId()
    if (userId === null) {
      clearImageGenerationKeys()
      imageGenerationKeysLoaded.value = true
      return []
    }

    if (ownerId !== userId) {
      clearImageGenerationKeys()
      ownerId = userId
    }
    if (pendingLoad?.ownerId === userId) return pendingLoad.promise
    if (imageGenerationKeysLoaded.value && !force) return imageGenerationKeys.value

    const generation = ++requestGeneration
    imageGenerationKeysLoading.value = true
    imageGenerationKeysError.value = null
    const isCurrentRequest = () => (
      generation === requestGeneration &&
      ownerId === userId &&
      authStore.isAuthenticated &&
      authStore.user?.id === userId
    )
    const promise = (async () => {
      const eligibleKeys: ApiKey[] = []
      let page = 1
      let pages = 1

      do {
        const response = await keysAPI.list(page, PAGE_SIZE, {
          status: 'active',
          sort_by: 'created_at',
          sort_order: 'desc',
        })
        eligibleKeys.push(...(response.items || []).filter((key) => (
          key.user_id === userId && isImageGenerationKey(key)
        )))
        pages = Math.max(1, Number(response.pages) || 1)
        page += 1
      } while (page <= pages && isCurrentRequest())

      if (!isCurrentRequest()) return []
      imageGenerationKeys.value = eligibleKeys
      imageGenerationKeysLoaded.value = true
      return eligibleKeys
    })()
      .catch((error: unknown) => {
        if (isCurrentRequest()) {
          imageGenerationKeys.value = []
          imageGenerationKeysLoaded.value = true
          imageGenerationKeysError.value = errorMessage(error)
        }
        return []
      })
      .finally(() => {
        if (generation === requestGeneration && ownerId === userId) {
          imageGenerationKeysLoading.value = false
        }
        if (pendingLoad?.generation === generation) pendingLoad = null
      })

    pendingLoad = { ownerId: userId, generation, promise }
    return promise
  }

  return {
    imageGenerationKeys,
    canUseImageGeneration: computed(() => imageGenerationKeys.value.length > 0),
    imageGenerationAccessLoaded: imageGenerationKeysLoaded,
    imageGenerationAccessLoading: imageGenerationKeysLoading,
    imageGenerationAccessError: imageGenerationKeysError,
    refreshImageGenerationAccess: refreshImageGenerationKeys,
    clearImageGenerationKeys,
  }
}
