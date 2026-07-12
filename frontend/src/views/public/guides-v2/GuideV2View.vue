<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

import mediaManifest from '../../../content/guides-v2/media-manifest.json'
import GuideV2NotFoundView from './GuideV2NotFoundView.vue'
import { parseGuideMarkdown } from './guide-v2.parser'
import { getGuideV2Entry } from './guide-v2.registry'
import type { GuideV2Media, ParsedGuideV2 } from './guide-v2.types'

const route = useRoute()
const guide = ref<ParsedGuideV2>()
const loadError = ref('')
const loading = ref(false)
let loadSequence = 0

const slug = computed(() => {
  const value = route.params.slug
  return Array.isArray(value) ? (value[0] ?? '') : (value ?? '')
})
const entry = computed(() => getGuideV2Entry(slug.value))

const markdownBody = (source: string): string => {
  const match = /^---\r?\n[\s\S]*?\r?\n---\r?\n/.exec(source)
  if (!match) throw new Error('教程缺少合法的 front matter')
  return source.slice(match[0].length)
}

const loadGuide = async (): Promise<void> => {
  const sequence = ++loadSequence
  const currentEntry = entry.value
  guide.value = undefined
  loadError.value = ''
  loading.value = Boolean(currentEntry)

  if (!currentEntry) return

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
    if (sequence === loadSequence) guide.value = parsed
  } catch (error) {
    if (sequence === loadSequence) {
      loadError.value = error instanceof Error ? error.message : '教程加载失败'
    }
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

watch(slug, loadGuide, { immediate: true })
</script>

<template>
  <GuideV2NotFoundView v-if="!entry" />
  <main v-else class="guide-v2" data-guide-v2-detail>
    <p v-if="loading" role="status">正在加载教程…</p>
    <section v-else-if="loadError" role="alert">
      <h1>教程暂时无法显示</h1>
      <p>{{ loadError }}</p>
    </section>
    <article v-else-if="guide">
      <template v-for="(block, index) in guide.blocks" :key="`${block.type}-${index}`">
        <component
          :is="`h${block.level}`"
          v-if="block.type === 'heading'"
          :id="block.anchor"
        >
          {{ block.text }}
        </component>
        <div
          v-else-if="block.type === 'paragraph' || block.type === 'table'"
          class="guide-v2__rich-text"
          v-html="block.html"
        />
        <pre v-else-if="block.type === 'code'"><code>{{ block.code }}</code></pre>
        <figure v-else-if="block.type === 'media'">
          <img :src="block.path" :alt="block.alt" loading="lazy" />
          <figcaption v-if="block.title">{{ block.title }}</figcaption>
        </figure>
      </template>
    </article>
  </main>
</template>

<style scoped>
.guide-v2 {
  width: min(100% - 32px, 880px);
  margin: 48px auto;
}

img {
  display: block;
  width: 100%;
  height: auto;
  border-radius: 12px;
}

pre {
  overflow-x: auto;
  padding: 16px;
  border-radius: 12px;
  background: #0f172a;
  color: #e2e8f0;
}

figure {
  margin: 24px 0;
}

figcaption {
  margin-top: 8px;
  color: #64748b;
  text-align: center;
}
</style>
