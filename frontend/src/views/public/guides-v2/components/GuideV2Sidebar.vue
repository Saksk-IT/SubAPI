<script setup lang="ts">
import { Icon } from '@/components/icons'
import type { GuideV2TocItem } from '../guide-v2.types'

defineProps<{
  readonly toc: readonly GuideV2TocItem[]
  readonly completedStepIds: readonly string[]
  readonly activeAnchor?: string | null
}>()

defineEmits<{ navigate: [anchor: string] }>()
</script>

<template>
  <aside class="guide-v2-sidebar" data-guide-v2-sidebar aria-label="本篇目录">
    <p class="guide-v2-sidebar__title">本篇步骤</p>
    <nav>
      <a
        v-for="item in toc"
        :key="item.anchor"
        :class="[
          'guide-v2-sidebar__link',
          `guide-v2-sidebar__link--level-${item.level}`,
          { 'guide-v2-sidebar__link--active': activeAnchor === item.anchor },
        ]"
        :href="`#${item.anchor}`"
        :aria-current="activeAnchor === item.anchor ? 'location' : undefined"
        @click="$emit('navigate', item.anchor)"
      >
        <Icon
          :name="completedStepIds.includes(item.anchor) ? 'checkCircle' : 'chevronRight'"
          size="xs"
          aria-hidden="true"
        />
        <span>{{ item.title }}</span>
      </a>
    </nav>
  </aside>
</template>
