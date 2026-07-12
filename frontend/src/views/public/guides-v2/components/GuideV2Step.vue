<script setup lang="ts">
import { Icon } from '@/components/icons'

defineProps<{
  readonly number: number
  readonly title: string
  readonly anchor: string
  readonly completed: boolean
}>()

defineEmits<{ toggle: [] }>()
</script>

<template>
  <section :id="anchor" class="guide-v2-step" data-guide-v2-step>
    <div class="guide-v2-step__heading">
      <span class="guide-v2-step__number" aria-hidden="true">{{ number }}</span>
      <h2>{{ title }}</h2>
      <button
        type="button"
        data-step-toggle
        :class="['guide-v2-step__toggle', { 'guide-v2-step__toggle--complete': completed }]"
        :aria-pressed="completed"
        @click="$emit('toggle')"
      >
        <Icon :name="completed ? 'checkCircle' : 'check'" size="sm" aria-hidden="true" />
        {{ completed ? '已完成，点击取消' : '标记完成' }}
      </button>
    </div>
  </section>
</template>
