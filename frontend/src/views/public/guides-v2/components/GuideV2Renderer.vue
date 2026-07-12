<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'

import mediaManifest from '../../../../content/guides-v2/media-manifest.json'
import type { GuideV2Block, ParsedGuideV2 } from '../guide-v2.types'
import { deriveGuideV2Visibility } from '../guide-v2.visibility'
import GuideV2CodeBlock from './GuideV2CodeBlock.vue'
import GuideV2Media from './GuideV2Media.vue'
import GuideV2Notice from './GuideV2Notice.vue'
import GuideV2SafeHtml from './GuideV2SafeHtml.vue'
import GuideV2Step from './GuideV2Step.vue'

const props = defineProps<{
  readonly guide: ParsedGuideV2
  readonly completedStepIds: readonly string[]
  readonly selectedPlatform: string
}>()

const emit = defineEmits<{
  'toggle-step': [anchor: string]
  'select-platform': [platform: string]
}>()

const tabRefs = ref<HTMLElement[]>([])

const visibility = computed(() => deriveGuideV2Visibility(props.guide, props.selectedPlatform))
const panelId = computed(() => `guide-v2-platform-panel-${props.guide.meta.slug}`)
const platformId = (platform: string): string =>
  `guide-v2-platform-tab-${props.guide.meta.slug}-${platform.toLowerCase()}`
const selectedTabId = computed(() => platformId(props.selectedPlatform))

const mediaMeta = (id: string) =>
  mediaManifest.media.find((candidate) => candidate.id === id)

const pngFallback = (path: string): string => path.replace(/\.webp(?:\?.*)?$/i, '.png')

const notice = (
  block: GuideV2Block,
): { readonly tone: 'note' | 'tip' | 'warning' | 'success'; readonly html: string } | null => {
  if (block.type !== 'paragraph') return null
  const match = /^<blockquote>\s*<p>\s*\[!(NOTE|TIP|WARNING|SUCCESS)\]\s*/i.exec(block.html)
  if (!match) return null
  const tone = match[1].toLowerCase() as 'note' | 'tip' | 'warning' | 'success'
  const html = block.html
    .replace(match[0], '<p>')
    .replace(/<\/blockquote>\s*$/i, '')
  return Object.freeze({ tone, html })
}

const setTabRef = (element: unknown, index: number): void => {
  if (element instanceof HTMLElement) tabRefs.value[index] = element
}

const onTabKeydown = async (event: KeyboardEvent, platform: string, index: number): Promise<void> => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    emit('select-platform', platform)
    return
  }
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return

  event.preventDefault()
  const last = props.guide.platforms.length - 1
  const target = event.key === 'Home'
    ? 0
    : event.key === 'End'
      ? last
      : event.key === 'ArrowRight'
        ? (index + 1) % props.guide.platforms.length
        : (index - 1 + props.guide.platforms.length) % props.guide.platforms.length
  await nextTick()
  tabRefs.value[target]?.focus()
}
</script>

<template>
  <div class="guide-v2-renderer">
    <section v-if="guide.platforms.length > 1" class="guide-v2-platforms" aria-labelledby="guide-platform-title">
      <div>
        <h2 id="guide-platform-title">选择你的系统</h2>
        <p>系统选择只保存在当前浏览器，不会上传。</p>
      </div>
      <div class="guide-v2-platforms__tabs" role="tablist" aria-label="平台分支">
        <button
          v-for="(platform, index) in guide.platforms"
          :key="platform"
          :ref="(element) => setTabRef(element, index)"
          type="button"
          role="tab"
          :id="platformId(platform)"
          :aria-controls="panelId"
          :aria-selected="selectedPlatform === platform"
          :tabindex="selectedPlatform === platform ? 0 : -1"
          @click="emit('select-platform', platform)"
          @keydown="onTabKeydown($event, platform, index)"
        >
          {{ platform }}
        </button>
      </div>
    </section>

    <div
      :id="panelId"
      :role="guide.platforms.length > 1 ? 'tabpanel' : undefined"
      :aria-labelledby="guide.platforms.length > 1 ? selectedTabId : undefined"
      class="guide-v2-platform-panel"
    >
      <template v-for="(block, index) in visibility.blocks" :key="`${block.type}-${index}`">
        <GuideV2Step
          v-if="block.type === 'heading' && block.stepNumber && block.anchor"
          :number="block.stepNumber"
          :title="block.text"
          :anchor="block.anchor"
          :completed="completedStepIds.includes(block.anchor)"
          @toggle="emit('toggle-step', block.anchor)"
        />
        <component
          :is="`h${block.level}`"
          v-else-if="block.type === 'heading' && block.level > 1"
          :id="block.anchor"
          :class="`guide-v2-heading guide-v2-heading--${block.level}`"
        >
          {{ block.text }}
        </component>
        <GuideV2Notice
          v-else-if="notice(block)"
          :tone="notice(block)!.tone"
          :html="notice(block)!.html"
        />
        <GuideV2SafeHtml
          v-else-if="block.type === 'paragraph' || block.type === 'table'"
          :html="block.html"
        />
        <GuideV2CodeBlock
          v-else-if="block.type === 'code'"
          :language="block.language"
          :code="block.code"
        />
        <GuideV2Media
          v-else-if="block.type === 'media'"
          kind="image"
          :src="block.path"
          :fallback-src="pngFallback(block.path)"
          :alt="block.alt"
          :caption="block.title"
          :width="mediaMeta(block.id)?.width ?? 1280"
          :height="mediaMeta(block.id)?.height ?? 720"
        />
      </template>
    </div>
  </div>
</template>
