<script setup lang="ts">
import { RouterLink } from 'vue-router'

import { Icon } from '@/components/icons'
import type { GuideV2Meta } from '../guide-v2.types'

defineProps<{ readonly meta: GuideV2Meta }>()
defineEmits<{ start: [] }>()
</script>

<template>
  <section class="guide-v2-hero" data-guide-v2-hero>
    <nav class="guide-v2-breadcrumb" aria-label="面包屑">
      <RouterLink to="/guides/v2">教程中心</RouterLink>
      <Icon name="chevronRight" size="xs" aria-hidden="true" />
      <span aria-current="page">{{ meta.title }}</span>
    </nav>
    <h1>{{ meta.title }}</h1>
    <p class="guide-v2-hero__summary">{{ meta.summary }}</p>
    <dl class="guide-v2-meta-list">
      <div>
        <dt><Icon name="clock" size="sm" aria-hidden="true" /> 预计耗时</dt>
        <dd>{{ meta.duration }}</dd>
      </div>
      <div>
        <dt><Icon name="badge" size="sm" aria-hidden="true" /> 难度</dt>
        <dd>{{ meta.difficulty }}</dd>
      </div>
      <div data-guide-platforms role="group" aria-label="支持平台">
        <dt><Icon name="globe" size="sm" aria-hidden="true" /> 支持平台</dt>
        <dd>
          <ul>
            <li v-for="platform in meta.platforms" :key="platform">{{ platform }}</li>
          </ul>
        </dd>
      </div>
      <div>
        <dt><Icon name="calendar" size="sm" aria-hidden="true" /> 最后核验</dt>
        <dd>{{ meta.updatedAt }}</dd>
      </div>
    </dl>
    <button class="guide-v2-button guide-v2-button--primary" type="button" @click="$emit('start')">
      开始本教程
      <Icon name="arrowDown" size="sm" aria-hidden="true" />
    </button>
  </section>
</template>
