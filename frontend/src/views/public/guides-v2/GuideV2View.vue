<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import mediaManifest from '../../../content/guides-v2/media-manifest.json'
import GuideV2Header from './components/GuideV2Header.vue'
import GuideV2Hero from './components/GuideV2Hero.vue'
import GuideV2MobileToc from './components/GuideV2MobileToc.vue'
import GuideV2Renderer from './components/GuideV2Renderer.vue'
import GuideV2Sidebar from './components/GuideV2Sidebar.vue'
import GuideV2Support from './components/GuideV2Support.vue'
import { createGuideV2Progress, type GuideProgress } from './composables/useGuideV2Progress'
import GuideV2NotFoundView from './GuideV2NotFoundView.vue'
import { parseGuideMarkdown } from './guide-v2.parser'
import { getGuideV2Entry, getGuideV2Navigation } from './guide-v2.registry'
import type { GuideV2Media, GuideV2Slug, ParsedGuideV2 } from './guide-v2.types'
import {
  decodeGuideV2Hash,
  deriveGuideV2Visibility,
  isGuideV2TocAnchor,
} from './guide-v2.visibility'

const route = useRoute()
const guide = ref<ParsedGuideV2>()
const loadError = ref('')
const loading = ref(false)
const activeAnchor = ref<string | null>(null)
const progress = shallowRef<GuideProgress>({
  completedStepIds: Object.freeze([]),
  platform: null,
  lastAnchor: null,
  updatedAt: null,
})
const progressStore = createGuideV2Progress()
let loadSequence = 0

const slug = computed(() => {
  const value = route.params.slug
  return Array.isArray(value) ? (value[0] ?? '') : (value ?? '')
})
const entry = computed(() => getGuideV2Entry(slug.value))
const navigation = computed(() => getGuideV2Navigation(slug.value))
const selectedPlatform = computed(() => {
  const platforms = guide.value?.platforms ?? []
  return progress.value.platform && platforms.includes(progress.value.platform)
    ? progress.value.platform
    : (platforms[0] ?? '')
})
const visibility = computed(() =>
  guide.value
    ? deriveGuideV2Visibility(guide.value, selectedPlatform.value)
    : { blocks: [], toc: [], platformByAnchor: Object.freeze(Object.create(null)) },
)

const currentSlug = (): GuideV2Slug | null => entry.value?.meta.slug ?? null
const syncProgress = (): void => {
  const value = currentSlug()
  if (value) progress.value = progressStore.get(value)
}
const unsubscribe = progressStore.subscribe(syncProgress)
onBeforeUnmount(() => {
  unsubscribe()
})

const markdownBody = (source: string): string => {
  const match = /^---\r?\n[\s\S]*?\r?\n---\r?\n/.exec(source)
  if (!match) throw new Error('教程缺少合法的 front matter')
  return source.slice(match[0].length)
}

const scrollToAnchor = async (anchor: string, updateHash = false): Promise<void> => {
  try {
    const currentGuide = guide.value
    const current = currentSlug()
    if (!currentGuide || !current || !isGuideV2TocAnchor(currentGuide, anchor)) return

    const requiredPlatform = deriveGuideV2Visibility(
      currentGuide,
      selectedPlatform.value,
    ).platformByAnchor[anchor]
    if (requiredPlatform && requiredPlatform !== selectedPlatform.value) {
      try {
        progressStore.setPlatform(current, requiredPlatform)
        await nextTick()
      } catch {
        return
      }
    }

    await nextTick()
    const currentVisibility = deriveGuideV2Visibility(currentGuide, selectedPlatform.value)
    if (!currentVisibility.toc.some((item) => item.anchor === anchor)) return
    const target = document.getElementById(anchor)
    if (!target) return

    activeAnchor.value = anchor
    try {
      progressStore.setLastAnchor(current, anchor)
    } catch {
      // 本地存储或进度状态异常时仍允许用户继续阅读。
    }
    try {
      target.scrollIntoView({
        behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth',
        block: 'start',
      })
    } catch {
      // 不支持 scrollIntoView 的环境保持当前页面可用。
    }
    if (updateHash && route.hash !== `#${anchor}`) {
      try {
        window.history.replaceState(null, '', `#${anchor}`)
      } catch {
        // 浏览器限制 history API 时仅跳过地址栏同步。
      }
    }
  } catch {
    // 所有导航输入都来自 URL 或渐进增强控件，异常不得中断页面。
  }
}

const scrollToRouteHash = (hash: string): void => {
  const anchor = decodeGuideV2Hash(hash)
  if (anchor) void scrollToAnchor(anchor)
}

const loadGuide = async (): Promise<void> => {
  const sequence = ++loadSequence
  const currentEntry = entry.value
  guide.value = undefined
  loadError.value = ''
  loading.value = Boolean(currentEntry)
  activeAnchor.value = null

  if (!currentEntry) return
  syncProgress()

  try {
    const source = await currentEntry.load()
    const media = mediaManifest.media.filter(
      (item) => item.guide === currentEntry.meta.slug,
    ) as readonly GuideV2Media[]
    const parsed = parseGuideMarkdown({
      sourceName: currentEntry.source,
      body: markdownBody(source),
      metadata: currentEntry.meta,
      media,
    })
    if (sequence === loadSequence) {
      guide.value = parsed
      const routeAnchor = decodeGuideV2Hash(route.hash)
      const initialAnchor = routeAnchor ?? progress.value.lastAnchor
      if (initialAnchor) void scrollToAnchor(initialAnchor)
    }
  } catch (error) {
    if (sequence === loadSequence) {
      loadError.value = error instanceof Error ? error.message : '教程加载失败'
    }
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

const selectPlatform = (platform: string): void => {
  const current = currentSlug()
  if (current) progressStore.setPlatform(current, platform)
}

const toggleStep = (anchor: string): void => {
  const current = currentSlug()
  if (!current) return
  if (progress.value.completedStepIds.includes(anchor)) {
    progressStore.uncompleteStep(current, anchor)
  } else {
    progressStore.completeStep(current, anchor)
  }
}

const clearGuide = (): void => {
  const current = currentSlug()
  if (current && window.confirm('清除本篇教程的步骤、平台和阅读位置？')) {
    progressStore.clear(current)
  }
}

const clearAll = (): void => {
  if (window.confirm('清除全部 V2 教程的本地进度？此操作不会影响账户数据。')) {
    progressStore.clearAll()
  }
}

const startGuide = (): void => {
  const anchor = guide.value?.steps[0]?.anchor ?? guide.value?.toc[0]?.anchor
  if (anchor) void scrollToAnchor(anchor, true)
}

watch(slug, loadGuide, { immediate: true })
watch(
  () => route.hash,
  (hash) => guide.value && scrollToRouteHash(hash),
)
</script>

<template>
  <GuideV2NotFoundView v-if="!entry" />
  <div v-else class="guide-v2-theme" data-guide-v2-detail>
    <GuideV2Header />
    <main class="guide-v2-main">
      <div v-if="loading" class="guide-v2-loading" role="status">正在加载教程…</div>
      <section v-else-if="loadError" class="guide-v2-load-error" role="alert">
        <h1>教程暂时无法显示</h1>
        <p>{{ loadError }}</p>
      </section>
      <template v-else-if="guide">
        <GuideV2Hero :meta="guide.meta" @start="startGuide" />
        <GuideV2MobileToc
          :toc="visibility.toc"
          :completed-step-ids="progress.completedStepIds"
          @navigate="scrollToAnchor($event, true)"
        />
        <div class="guide-v2-detail-grid">
          <GuideV2Sidebar
            :toc="visibility.toc"
            :completed-step-ids="progress.completedStepIds"
            :active-anchor="activeAnchor"
            @navigate="scrollToAnchor($event)"
          />
          <article class="guide-v2-article">
            <GuideV2Renderer
              :guide="guide"
              :completed-step-ids="progress.completedStepIds"
              :selected-platform="selectedPlatform"
              @select-platform="selectPlatform"
              @toggle-step="toggleStep"
            />
            <GuideV2Support />
            <div class="guide-v2-progress-actions" aria-label="本地进度管理">
              <button type="button" data-clear-guide @click="clearGuide">清除本篇进度</button>
              <button type="button" data-clear-all @click="clearAll">清除全部 V2 进度</button>
            </div>
            <nav class="guide-v2-navigation" aria-label="上一篇和下一篇">
              <RouterLink
                v-if="navigation?.previous"
                data-guide-previous
                :to="navigation.previous.path"
              >
                <span>上一篇</span>
                <strong>{{ navigation.previous.meta.title }}</strong>
              </RouterLink>
              <RouterLink v-if="navigation?.next" data-guide-next :to="navigation.next.path">
                <span>下一篇</span>
                <strong>{{ navigation.next.meta.title }}</strong>
              </RouterLink>
            </nav>
          </article>
        </div>
      </template>
    </main>
  </div>
</template>

<style src="./styles/tokens.css"></style>
<style src="./styles/layout.css"></style>
<style src="./styles/content.css"></style>
<style src="./styles/responsive.css"></style>
