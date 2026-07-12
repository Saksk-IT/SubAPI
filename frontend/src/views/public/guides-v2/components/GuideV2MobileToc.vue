<script setup lang="ts">
import { nextTick, ref } from 'vue'

import { Icon } from '@/components/icons'
import type { GuideV2TocItem } from '../guide-v2.types'

defineProps<{
  readonly toc: readonly GuideV2TocItem[]
  readonly completedStepIds: readonly string[]
}>()

const emit = defineEmits<{ navigate: [anchor: string] }>()
const open = ref(false)
const trigger = ref<HTMLButtonElement>()
const closeButton = ref<HTMLButtonElement>()

const show = async (): Promise<void> => {
  open.value = true
  await nextTick()
  closeButton.value?.focus()
}

const close = async (): Promise<void> => {
  open.value = false
  await nextTick()
  trigger.value?.focus()
}

const navigate = (anchor: string): void => {
  emit('navigate', anchor)
  void close()
}

const onKeydown = (event: KeyboardEvent): void => {
  if (event.key === 'Escape') {
    event.preventDefault()
    void close()
  }
}
</script>

<template>
  <div class="guide-v2-mobile-toc" data-guide-v2-mobile-toc>
    <button
      ref="trigger"
      class="guide-v2-mobile-toc__trigger"
      type="button"
      data-mobile-toc-trigger
      :aria-expanded="open"
      aria-controls="guide-v2-mobile-toc-panel"
      @click="show"
    >
      <Icon name="menu" size="sm" aria-hidden="true" />
      查看本篇步骤
      <Icon name="chevronDown" size="sm" aria-hidden="true" />
    </button>
    <div
      v-if="open"
      id="guide-v2-mobile-toc-panel"
      class="guide-v2-mobile-toc__backdrop"
      role="presentation"
      @click.self="close"
    >
      <section
        class="guide-v2-mobile-toc__dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="guide-v2-mobile-toc-title"
        tabindex="-1"
        @keydown="onKeydown"
      >
        <header>
          <h2 id="guide-v2-mobile-toc-title">本篇步骤</h2>
          <button ref="closeButton" type="button" aria-label="关闭目录" @click="close">
            <Icon name="x" size="sm" aria-hidden="true" />
          </button>
        </header>
        <nav aria-label="移动端本篇目录">
          <a
            v-for="item in toc"
            :key="item.anchor"
            :href="`#${item.anchor}`"
            :class="`guide-v2-mobile-toc__link guide-v2-mobile-toc__link--level-${item.level}`"
            @click="navigate(item.anchor)"
          >
            <Icon
              :name="completedStepIds.includes(item.anchor) ? 'checkCircle' : 'chevronRight'"
              size="sm"
              aria-hidden="true"
            />
            {{ item.title }}
          </a>
        </nav>
      </section>
    </div>
  </div>
</template>
