<script setup lang="ts">
import { ref } from 'vue'

import { Icon } from '@/components/icons'

const props = defineProps<{ readonly code: string; readonly language?: string }>()
const feedback = ref('')

const copy = async (): Promise<void> => {
  try {
    if (!navigator.clipboard?.writeText) throw new Error('clipboard unavailable')
    await navigator.clipboard.writeText(props.code)
    feedback.value = '已复制到剪贴板'
  } catch {
    feedback.value = '复制失败，请手动选择并复制'
  }
}
</script>

<template>
  <div class="guide-v2-code">
    <div class="guide-v2-code__toolbar">
      <span>{{ language || 'text' }}</span>
      <button type="button" @click="copy">
        <Icon name="copy" size="sm" aria-hidden="true" />
        复制
      </button>
    </div>
    <pre tabindex="0"><code>{{ code }}</code></pre>
    <p v-if="feedback" class="guide-v2-code__feedback" role="status">{{ feedback }}</p>
  </div>
</template>
