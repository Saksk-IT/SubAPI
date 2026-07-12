<script setup lang="ts">
import { computed, ref } from 'vue'

import { Icon } from '@/components/icons'

const props = withDefaults(
  defineProps<{
    readonly kind?: 'image' | 'video'
    readonly src: string
    readonly fallbackSrc?: string
    readonly alt: string
    readonly caption?: string
    readonly width: number
    readonly height: number
    readonly poster?: string
    readonly transcriptUrl?: string
  }>(),
  { kind: 'image', fallbackSrc: '', caption: '', poster: '', transcriptUrl: '' },
)

const failed = ref(false)
const fallbackAttempted = ref(false)
const retryKey = ref(0)
const currentSrc = computed(() => {
  if (!fallbackAttempted.value) return props.src
  return props.fallbackSrc || props.src.replace(/\.webp(?:\?.*)?$/i, '.png')
})

const retry = (): void => {
  fallbackAttempted.value = true
  failed.value = false
  retryKey.value += 1
}
</script>

<template>
  <figure class="guide-v2-media">
    <img
      v-if="kind === 'image'"
      v-show="!failed"
      :key="retryKey"
      :src="currentSrc"
      :alt="alt"
      :width="width"
      :height="height"
      loading="lazy"
      decoding="async"
      @error="failed = true"
    />
    <video
      v-else
      v-show="!failed"
      :key="retryKey"
      :src="src"
      :poster="poster || undefined"
      :width="width"
      :height="height"
      preload="metadata"
      controls
      :aria-label="alt"
      @error="failed = true"
    />
    <div v-if="failed" class="guide-v2-media__error" role="alert">
      <Icon name="exclamationTriangle" size="lg" aria-hidden="true" />
      <strong>{{ kind === 'image' ? '图片加载失败' : '视频加载失败' }}</strong>
      <span>文字步骤仍可继续阅读。</span>
      <button type="button" @click="retry">
        <Icon name="refresh" size="sm" aria-hidden="true" />
        {{ kind === 'image' ? '重试并使用 PNG 备用图' : '重试加载' }}
      </button>
    </div>
    <figcaption v-if="caption">{{ caption }}</figcaption>
    <a
      v-if="kind === 'video' && transcriptUrl"
      class="guide-v2-media__text-link"
      :href="transcriptUrl"
    >
      查看文字步骤
      <Icon name="externalLink" size="xs" aria-hidden="true" />
    </a>
  </figure>
</template>
