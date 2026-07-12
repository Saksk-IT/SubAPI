<script setup lang="ts">
import { RouterLink } from 'vue-router'

import { GUIDE_V2_REGISTRY } from './guide-v2.registry'

const guideEntries = Object.freeze(
  GUIDE_V2_REGISTRY.filter(({ meta }) => meta.slug !== 'index'),
)
</script>

<template>
  <main class="guide-v2-hub" data-guide-v2-hub>
    <p class="guide-v2-hub__eyebrow">V2 · 新手友好</p>
    <h1>AI 客户端使用指南</h1>
    <p>从创建 API Key 到客户端配置与故障排查，按你的使用场景选择教程。</p>
    <nav aria-label="V2 教程列表">
      <RouterLink v-for="entry in guideEntries" :key="entry.meta.slug" :to="entry.path">
        <strong>{{ entry.meta.title }}</strong>
        <span>{{ entry.meta.summary }}</span>
      </RouterLink>
    </nav>
  </main>
</template>

<style scoped>
.guide-v2-hub {
  width: min(100% - 32px, 960px);
  margin: 48px auto;
}

.guide-v2-hub__eyebrow {
  color: #2563eb;
  font-weight: 700;
}

nav {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 16px;
  margin-top: 32px;
}

a {
  display: flex;
  min-height: 96px;
  padding: 20px;
  flex-direction: column;
  gap: 8px;
  border: 1px solid #dbeafe;
  border-radius: 16px;
  color: inherit;
  text-decoration: none;
}

a:hover {
  border-color: #60a5fa;
}

span {
  color: #475569;
}
</style>
