<template>
  <BaseDialog
    :show="launcherStore.isOpen"
    :title="t('imageGeneration.launcherTitle')"
    width="normal"
    @close="closeLauncher"
  >
    <div class="max-h-[70dvh] space-y-5 overflow-y-auto pr-1">
      <p class="text-sm text-gray-600 dark:text-gray-400">
        {{ t('imageGeneration.launcherDescription') }}
      </p>

      <div
        v-if="imageGenerationAccessLoading || (!imageGenerationAccessLoaded && !imageGenerationAccessError)"
        role="status"
        aria-live="polite"
        class="flex min-h-32 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t('imageGeneration.loading') }}
      </div>

      <div
        v-else-if="imageGenerationAccessError"
        class="space-y-4 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-900/70 dark:bg-red-950/30"
      >
        <p role="alert" class="text-sm text-red-700 dark:text-red-300">
          {{ t('imageGeneration.loadError') }}
        </p>
        <button
          type="button"
          data-testid="image-generation-access-retry"
          class="btn btn-secondary min-h-11 w-full sm:w-auto"
          @click="retryAccess"
        >
          {{ t('imageGeneration.retry') }}
        </button>
      </div>

      <div
        v-else-if="!canUseImageGeneration || imageGenerationKeys.length === 0"
        class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
      >
        <div>
          <h3 class="font-medium text-gray-900 dark:text-gray-100">
            {{ t('imageGeneration.noKeyTitle') }}
          </h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('imageGeneration.noKeyDescription') }}
          </p>
        </div>
        <RouterLink
          to="/keys"
          class="btn btn-primary min-h-11 w-full sm:w-auto"
          @click="closeLauncher"
        >
          {{ t('imageGeneration.manageKeys') }}
        </RouterLink>
      </div>

      <div v-else class="space-y-2">
        <label
          for="image-generation-launcher-key"
          class="block text-sm font-medium text-gray-700 dark:text-gray-300"
        >
          {{ t('imageGeneration.keyLabel') }}
        </label>
        <select
          id="image-generation-launcher-key"
          v-model.number="selectedKeyId"
          class="input min-h-11 w-full"
          :disabled="launching"
        >
          <option v-for="apiKey in imageGenerationKeys" :key="apiKey.id" :value="apiKey.id">
            {{ displayKeyName(apiKey) }}
          </option>
        </select>
      </div>

      <p
        v-if="launchError"
        role="alert"
        class="rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/70 dark:bg-red-950/30 dark:text-red-300"
      >
        {{ launchError }}
      </p>
    </div>

    <template #footer>
      <div class="flex w-full flex-col-reverse gap-3 sm:flex-row sm:justify-end">
        <button
          type="button"
          class="btn btn-secondary min-h-11 w-full sm:w-auto"
          :disabled="launching"
          @click="closeLauncher"
        >
          {{ t('common.cancel') }}
        </button>
        <button
          v-if="canUseImageGeneration && imageGenerationKeys.length > 0"
          type="button"
          data-testid="image-generation-open"
          class="btn btn-primary min-h-11 w-full sm:w-auto"
          :disabled="launching || selectedKey === null"
          @click="openPlayground"
        >
          {{ launching ? t('imageGeneration.opening') : t('imageGeneration.openNewTab') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import BaseDialog from '@/components/common/BaseDialog.vue'
import { useImageGenerationKeys } from '@/composables/useImageGenerationAccess'
import {
  openImagePlaygroundPopup,
  PopupLaunchError,
  type ImagePlaygroundPopupSession,
  type PopupLaunchErrorCode,
} from '@/features/imagePlayground/popupBridge'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useImageGenerationLauncherStore } from '@/stores/imageGenerationLauncher'
import type { ApiKey } from '@/types'

const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const launcherStore = useImageGenerationLauncherStore()
const {
  imageGenerationKeys,
  canUseImageGeneration,
  imageGenerationAccessLoaded,
  imageGenerationAccessLoading,
  imageGenerationAccessError,
  refreshImageGenerationAccess,
  clearImageGenerationKeys,
} = useImageGenerationKeys()

const selectedKeyId = ref<number | null>(null)
const launching = ref(false)
const launchError = ref<string | null>(null)
let activeSession: ImagePlaygroundPopupSession | null = null

const selectedKey = computed(() => (
  imageGenerationKeys.value.find((apiKey) => apiKey.id === selectedKeyId.value) ?? null
))

function displayKeyName(apiKey: ApiKey): string {
  return apiKey.name?.trim() || `#${apiKey.id}`
}

function selectAvailableKey(): void {
  if (selectedKey.value) return
  selectedKeyId.value = imageGenerationKeys.value[0]?.id ?? null
}

function abortActiveSession(): void {
  const session = activeSession
  activeSession = null
  if (!session) return
  session.abort()
}

function resetLocalState(): void {
  abortActiveSession()
  launching.value = false
  launchError.value = null
  selectedKeyId.value = null
  clearImageGenerationKeys()
}

function closeLauncher(): void {
  resetLocalState()
  launcherStore.close()
}

async function loadKeys(force = false): Promise<void> {
  launchError.value = null
  await refreshImageGenerationAccess(force)
  selectAvailableKey()
}

async function retryAccess(): Promise<void> {
  await loadKeys(true)
}

function launchErrorMessage(code: PopupLaunchErrorCode): string {
  const keys: Record<PopupLaunchErrorCode, string> = {
    popup_blocked: 'imageGeneration.popupBlocked',
    connection_timeout: 'imageGeneration.connectionTimeout',
    popup_closed: 'imageGeneration.popupClosed',
    configuration_failed: 'imageGeneration.configurationFailed',
    aborted: 'imageGeneration.configurationFailed',
  }
  return t(keys[code])
}

async function openPlayground(): Promise<void> {
  if (launching.value) return
  const apiKey = selectedKey.value
  const ownerId = authStore.user?.id
  if (!apiKey || !Number.isSafeInteger(ownerId) || Number(ownerId) <= 0) {
    launchError.value = t('imageGeneration.configurationFailed')
    return
  }

  launching.value = true
  launchError.value = null
  const session = openImagePlaygroundPopup({
    apiKey: apiKey.key,
    apiKeyId: apiKey.id,
    apiKeyName: displayKeyName(apiKey),
    storageScope: String(ownerId),
    locale: locale.value,
    theme: document.documentElement.classList.contains('dark') ? 'dark' : 'light',
  })
  activeSession = session

  try {
    await session.configured
    if (activeSession !== session || authStore.user?.id !== ownerId) return
    activeSession = null
    closeLauncher()
  } catch (error: unknown) {
    if (activeSession !== session || !launcherStore.isOpen) return
    activeSession = null
    launchError.value = error instanceof PopupLaunchError
      ? launchErrorMessage(error.code)
      : t('imageGeneration.configurationFailed')
  } finally {
    if (activeSession === session) activeSession = null
    launching.value = false
  }
}

watch(imageGenerationKeys, selectAvailableKey)

watch(
  () => launcherStore.isOpen,
  async (isOpen) => {
    if (!isOpen) {
      resetLocalState()
      return
    }
    if (appStore.cachedPublicSettings?.image_generation_enabled === false) {
      closeLauncher()
      appStore.showError(t('imageGeneration.disabled'))
      return
    }
    await loadKeys()
  },
)

watch(
  () => appStore.cachedPublicSettings?.image_generation_enabled,
  (enabled) => {
    if (enabled !== false || !launcherStore.isOpen) return
    closeLauncher()
    appStore.showError(t('imageGeneration.disabled'))
  },
)

watch(
  () => authStore.user?.id,
  (userId, previousUserId) => {
    if (previousUserId === undefined || userId === previousUserId || !launcherStore.isOpen) return
    closeLauncher()
    appStore.showError(t('imageGeneration.userChanged'))
  },
)

onBeforeUnmount(resetLocalState)
</script>
