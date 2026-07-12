<script setup lang="ts">
import { computed } from 'vue'

import { Icon } from '@/components/icons'
import GuideV2SafeHtml from './GuideV2SafeHtml.vue'

const props = withDefaults(
  defineProps<{
    readonly tone?: 'note' | 'tip' | 'warning' | 'success'
    readonly html: string
  }>(),
  { tone: 'note' },
)

const labels = Object.freeze({ note: '说明', tip: '提示', warning: '请注意', success: '完成' })
const icons = Object.freeze({
  note: 'infoCircle',
  tip: 'lightbulb',
  warning: 'exclamationTriangle',
  success: 'checkCircle',
} as const)
const label = computed(() => labels[props.tone])
const icon = computed(() => icons[props.tone])
</script>

<template>
  <aside :class="`guide-v2-notice guide-v2-notice--${tone}`" data-guide-v2-notice>
    <div class="guide-v2-notice__label">
      <Icon :name="icon" size="sm" aria-hidden="true" />
      {{ label }}
    </div>
    <GuideV2SafeHtml :html="html" />
  </aside>
</template>
