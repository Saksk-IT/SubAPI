<script setup lang="ts">
import { RouterLink } from 'vue-router'

import { Icon } from '@/components/icons'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
</script>

<template>
  <header class="guide-v2-header">
    <div class="guide-v2-header__inner">
      <RouterLink class="guide-v2-header__brand" to="/guides/v2" aria-label="返回教程中心">
        <span class="guide-v2-header__mark" aria-hidden="true">
          <Icon name="cloud" size="sm" />
        </span>
        <span>{{ appStore.siteName }}</span>
      </RouterLink>
      <nav class="guide-v2-header__nav" aria-label="教程快捷导航">
        <RouterLink to="/guides/v2">教程中心</RouterLink>
        <RouterLink to="/guides/v2/troubleshooting">排错中心</RouterLink>
      </nav>
    </div>
  </header>
</template>
