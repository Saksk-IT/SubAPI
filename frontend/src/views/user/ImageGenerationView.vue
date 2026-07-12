<template>
  <AppLayout :hide-first-recharge-banner="true">
    <section
      data-testid="image-generation-host"
      class="flex h-[calc(100dvh-6rem)] min-w-0 flex-col gap-4 overflow-hidden p-3 sm:p-4 md:h-[calc(100dvh-7rem)] lg:h-[calc(100dvh-8rem)] lg:p-6"
    >
      <header class="min-w-0">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-gray-100">
          {{ t('imageGeneration.title') }}
        </h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('imageGeneration.subtitle') }}
        </p>
      </header>

      <div
        v-if="imageGenerationAccessLoading || (!imageGenerationAccessLoaded && !imageGenerationAccessError)"
        class="flex min-h-64 flex-1 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t('imageGeneration.loading') }}
      </div>

      <div
        v-else-if="imageGenerationAccessError"
        class="flex min-h-64 flex-1 flex-col items-center justify-center gap-4 text-center"
      >
        <div>
          <h2 class="text-lg font-medium text-gray-900 dark:text-gray-100">
            {{ t('imageGeneration.loadError') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('imageGeneration.noKeyDescription') }}
          </p>
        </div>
        <button
          type="button"
          data-testid="access-retry"
          class="btn btn-primary"
          @click="retryAccess"
        >
          {{ t('imageGeneration.retry') }}
        </button>
      </div>

      <div
        v-else-if="!canUseImageGeneration || imageGenerationKeys.length === 0"
        class="flex min-h-64 flex-1 flex-col items-center justify-center gap-4 text-center"
      >
        <div>
          <h2 class="text-lg font-medium text-gray-900 dark:text-gray-100">
            {{ t('imageGeneration.noKeyTitle') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('imageGeneration.noKeyDescription') }}
          </p>
        </div>
        <RouterLink to="/keys" class="btn btn-primary">
          {{ t('imageGeneration.manageKeys') }}
        </RouterLink>
      </div>

      <div v-else class="flex min-h-0 flex-1 flex-col gap-3">
        <div class="flex flex-col gap-2 sm:max-w-md">
          <label
            for="image-generation-key"
            class="text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {{ t('imageGeneration.keyLabel') }}
          </label>
          <select
            id="image-generation-key"
            v-model.number="selectedKeyId"
            class="input w-full"
          >
            <option v-for="apiKey in imageGenerationKeys" :key="apiKey.id" :value="apiKey.id">
              {{ apiKey.name || `#${apiKey.id}` }}
            </option>
          </select>
        </div>

        <div
          class="relative min-h-0 min-w-0 flex-1 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-900"
        >
          <iframe
            :key="frameNonce"
            ref="iframeRef"
            src="/image-playground/"
            :name="frameName"
            :title="t('imageGeneration.frameTitle')"
            class="h-full min-h-0 w-full border-0 bg-white"
            sandbox="allow-scripts allow-same-origin allow-downloads"
            allow="clipboard-read; clipboard-write"
            referrerpolicy="no-referrer"
            @load="handleFrameLoad"
          />

          <div
            v-if="handshakeState === 'waiting'"
            role="status"
            aria-live="polite"
            class="absolute inset-0 flex items-center justify-center bg-white/90 px-4 text-center text-sm text-gray-500 backdrop-blur-sm dark:bg-gray-900/90 dark:text-gray-400"
          >
            {{ t('imageGeneration.waiting') }}
          </div>

          <div
            v-else-if="handshakeState === 'timeout' || handshakeState === 'error'"
            role="alert"
            class="absolute inset-0 flex flex-col items-center justify-center gap-4 bg-white/95 px-4 text-center dark:bg-gray-900/95"
          >
            <div>
              <h2 class="text-lg font-medium text-gray-900 dark:text-gray-100">
                {{ handshakeState === 'timeout'
                  ? t('imageGeneration.timeoutTitle')
                  : t('imageGeneration.connectionErrorTitle') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ handshakeState === 'timeout'
                  ? t('imageGeneration.timeoutDescription')
                  : t('imageGeneration.connectionErrorDescription') }}
              </p>
            </div>
            <button
              type="button"
              data-testid="handshake-retry"
              class="btn btn-primary"
              @click="retryHandshake"
            >
              {{ t('imageGeneration.retry') }}
            </button>
          </div>
        </div>
      </div>
    </section>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import AppLayout from '@/components/layout/AppLayout.vue'
import { useImageGenerationKeys } from '@/composables/useImageGenerationAccess'
import {
  buildClearMessage,
  buildConfigureMessage,
  buildConnectMessage,
  createFrameName,
  isConfiguredAckMessage,
  isTrustedReadyEvent,
} from '@/features/imagePlayground/bridge'
import { useAuthStore } from '@/stores/auth'

const HANDSHAKE_TIMEOUT_MS = 8_000

type HandshakeState = 'waiting' | 'connected' | 'timeout' | 'error'

const { t, locale } = useI18n()
const authStore = useAuthStore()
const {
  imageGenerationKeys,
  canUseImageGeneration,
  imageGenerationAccessLoaded,
  imageGenerationAccessLoading,
  imageGenerationAccessError,
  refreshImageGenerationAccess,
  clearImageGenerationKeys,
} = useImageGenerationKeys()

const iframeRef = ref<HTMLIFrameElement | null>(null)
const selectedKeyId = ref<number | null>(null)
const frameNonce = ref(createNonce())
const frameReady = ref(false)
const handshakeState = ref<HandshakeState>('waiting')

let activePort: MessagePort | null = null
let activePortNonce: string | null = null
let requestSequence = 0
let awaitedRequestId: number | null = null
let handshakeTimer: number | null = null
let themeObserver: MutationObserver | null = null
let frameLoadInitialized = false
let pendingTrustedReady = false

const frameName = computed(() => createFrameName(frameNonce.value))
const selectedKey = computed(() => (
  imageGenerationKeys.value.find((apiKey) => apiKey.id === selectedKeyId.value) ?? null
))
const storageScope = computed(() => {
  const userId = authStore.user?.id
  return Number.isSafeInteger(userId) && Number(userId) > 0 ? String(userId) : ''
})

function createNonce(): string {
  if (typeof globalThis.crypto.randomUUID === 'function') {
    return globalThis.crypto.randomUUID()
  }

  const bytes = globalThis.crypto.getRandomValues(new Uint8Array(16))
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}

function currentTheme(): 'light' | 'dark' {
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

function clearHandshakeTimer(): void {
  if (handshakeTimer === null) return
  window.clearTimeout(handshakeTimer)
  handshakeTimer = null
}

function startHandshakeTimer(): void {
  clearHandshakeTimer()
  handshakeTimer = window.setTimeout(() => {
    if (handshakeState.value === 'waiting') {
      handshakeState.value = 'timeout'
    }
    handshakeTimer = null
  }, HANDSHAKE_TIMEOUT_MS)
}

function disconnectActivePort(): void {
  const port = activePort
  const nonce = activePortNonce
  activePort = null
  activePortNonce = null
  awaitedRequestId = null
  if (!port) return

  try {
    port.removeEventListener('message', handlePortMessage)
    if (nonce) port.postMessage(buildClearMessage(nonce))
  } catch (error) {
    console.warn('[ImageGeneration] Failed to clear the previous bridge port', error)
  } finally {
    port.close()
  }
}

function reportBridgeError(error: unknown): void {
  clearHandshakeTimer()
  handshakeState.value = 'error'
  console.error('[ImageGeneration] Failed to establish the image playground bridge', error)
}

function nextRequestId(): number {
  if (requestSequence >= Number.MAX_SAFE_INTEGER) {
    throw new Error('Image playground request sequence exhausted')
  }
  requestSequence += 1
  return requestSequence
}

function configurationMessage(requestId: number) {
  const apiKey = selectedKey.value
  if (!apiKey) throw new Error('No selected image-generation API key')
  const trimmedName = (apiKey.name || `#${apiKey.id}`).trim()
  const apiKeyName = trimmedName || `#${apiKey.id}`

  return buildConfigureMessage({
    nonce: frameNonce.value,
    requestId,
    apiKey: apiKey.key,
    apiKeyId: apiKey.id,
    apiKeyName,
    storageScope: storageScope.value,
    locale: locale.value,
    theme: currentTheme(),
  })
}

function connectChannel(): void {
  const frameWindow = iframeRef.value?.contentWindow
  if (!frameWindow || !selectedKey.value) return

  disconnectActivePort()

  try {
    const requestId = nextRequestId()
    const configureMessage = configurationMessage(requestId)
    const channel = new MessageChannel()
    activePort = channel.port1
    activePortNonce = frameNonce.value
    awaitedRequestId = requestId
    channel.port1.addEventListener('message', handlePortMessage)
    channel.port1.start()
    handshakeState.value = 'waiting'
    startHandshakeTimer()
    frameWindow.postMessage(
      buildConnectMessage(frameNonce.value),
      window.location.origin,
      [channel.port2],
    )
    channel.port1.postMessage(configureMessage)
  } catch (error) {
    disconnectActivePort()
    reportBridgeError(error)
  }
}

function sendConfiguration(): void {
  if (!activePort) return
  try {
    const requestId = nextRequestId()
    const message = configurationMessage(requestId)
    awaitedRequestId = requestId
    handshakeState.value = 'waiting'
    startHandshakeTimer()
    activePort.postMessage(message)
  } catch (error) {
    disconnectActivePort()
    reportBridgeError(error)
  }
}

function handlePortMessage(event: MessageEvent): void {
  if (
    !activePortNonce ||
    awaitedRequestId === null ||
    !isConfiguredAckMessage(event.data, activePortNonce, awaitedRequestId)
  ) return
  awaitedRequestId = null
  clearHandshakeTimer()
  handshakeState.value = 'connected'
}

function handleWindowMessage(event: MessageEvent): void {
  const frameWindow = iframeRef.value?.contentWindow ?? null
  if (!isTrustedReadyEvent(event, {
    expectedOrigin: window.location.origin,
    expectedSource: frameWindow,
    expectedNonce: frameNonce.value,
  })) {
    return
  }

  if (!frameLoadInitialized) {
    pendingTrustedReady = true
    return
  }

  frameReady.value = true
  if (!activePort) connectChannel()
}

function handleFrameLoad(): void {
  disconnectActivePort()
  frameLoadInitialized = true
  frameReady.value = false
  handshakeState.value = 'waiting'
  startHandshakeTimer()
  if (pendingTrustedReady) {
    pendingTrustedReady = false
    frameReady.value = true
    connectChannel()
  }
}

function retryHandshake(): void {
  clearHandshakeTimer()
  disconnectActivePort()
  frameLoadInitialized = false
  pendingTrustedReady = false
  frameReady.value = false
  handshakeState.value = 'waiting'
  frameNonce.value = createNonce()
  startHandshakeTimer()
}

async function retryAccess(): Promise<void> {
  await refreshImageGenerationAccess(true)
}

watch(imageGenerationKeys, (keys) => {
  if (keys.some((apiKey) => apiKey.id === selectedKeyId.value)) return
  selectedKeyId.value = keys[0]?.id ?? null
}, { immediate: true })

watch(selectedKeyId, (nextKeyId, previousKeyId) => {
  if (nextKeyId === previousKeyId) return
  if (nextKeyId === null) {
    disconnectActivePort()
    return
  }
  if (!activePort && !frameReady.value) startHandshakeTimer()
  if (activePort) {
    sendConfiguration()
    return
  }
  if (frameReady.value) connectChannel()
}, { immediate: true })

watch(() => authStore.user?.id, (nextUserId, previousUserId) => {
  if (nextUserId === previousUserId) return
  clearImageGenerationKeys()
  retryHandshake()
  void refreshImageGenerationAccess(true)
})

watch(locale, () => sendConfiguration())

onMounted(() => {
  window.addEventListener('message', handleWindowMessage)
  themeObserver = new MutationObserver(() => sendConfiguration())
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  void refreshImageGenerationAccess()
})

onBeforeUnmount(() => {
  window.removeEventListener('message', handleWindowMessage)
  themeObserver?.disconnect()
  themeObserver = null
  clearHandshakeTimer()
  disconnectActivePort()
  clearImageGenerationKeys()
})
</script>
