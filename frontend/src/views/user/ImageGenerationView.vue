<template>
  <div data-testid="image-generation-compatibility-route" class="sr-only" aria-live="polite">
    {{ t('imageGeneration.opening') }}
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import { useImageGenerationLauncherStore } from '@/stores/imageGenerationLauncher'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const launcherStore = useImageGenerationLauncherStore()

onMounted(() => {
  launcherStore.open()
  void router.replace(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
})
</script>
