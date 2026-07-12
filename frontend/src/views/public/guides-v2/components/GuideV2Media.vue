<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'

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
const lightboxOpen = ref(false)
const zoomTrigger = ref<HTMLButtonElement>()
const lightboxClose = ref<HTMLButtonElement>()
const currentSrc = computed(() => {
  if (!fallbackAttempted.value) return props.src
  return props.fallbackSrc || props.src.replace(/\.webp(?:\?.*)?$/i, '.png')
})

const retry = (): void => {
  fallbackAttempted.value = true
  failed.value = false
  retryKey.value += 1
}

const openLightbox = async (): Promise<void> => {
  if (props.kind !== 'image' || failed.value) return
  lightboxOpen.value = true
  await nextTick()
  lightboxClose.value?.focus()
}

const closeLightbox = async (): Promise<void> => {
  lightboxOpen.value = false
  await nextTick()
  zoomTrigger.value?.focus()
}

const onLightboxKeydown = (event: KeyboardEvent): void => {
  if (event.key === 'Escape') {
    event.preventDefault()
    void closeLightbox()
    return
  }
  if (event.key === 'Tab') {
    event.preventDefault()
    lightboxClose.value?.focus()
  }
}
</script>

<template>
  <figure class="guide-v2-media">
    <button
      v-if="kind === 'image' && !failed"
      ref="zoomTrigger"
      class="guide-v2-media__zoom"
      type="button"
      data-media-zoom-trigger
      :aria-label="`放大查看：${alt}`"
      @click="openLightbox"
      @keydown.enter.prevent="openLightbox"
      @keydown.space.prevent="openLightbox"
    >
      <img
        :key="retryKey"
        :src="currentSrc"
        :alt="alt"
        :width="width"
        :height="height"
        loading="lazy"
        decoding="async"
        @error="failed = true"
      />
    </button>
    <video
      v-else-if="kind === 'video' && !failed"
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
    <div
      v-if="lightboxOpen"
      class="guide-v2-lightbox"
      data-lightbox-backdrop
      role="presentation"
      @click.self="closeLightbox"
    >
      <section
        class="guide-v2-lightbox__dialog"
        role="dialog"
        aria-modal="true"
        :aria-label="`大图预览：${alt}`"
        @keydown="onLightboxKeydown"
      >
        <button ref="lightboxClose" type="button" aria-label="关闭大图" @click="closeLightbox">
          <Icon name="x" size="sm" aria-hidden="true" />
        </button>
        <img :src="currentSrc" :alt="alt" :width="width" :height="height" />
        <p v-if="caption">{{ caption }}</p>
      </section>
    </div>
  </figure>
</template>
