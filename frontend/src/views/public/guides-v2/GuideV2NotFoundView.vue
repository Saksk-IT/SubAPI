<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <main class="guide-v2-not-found" data-guide-v2-not-found>
    <p class="guide-v2-not-found__code">404</p>
    <h1>未找到这篇 V2 教程</h1>
    <p>链接可能已失效，或者教程名称输入有误。</p>
    <RouterLink to="/guides/v2">返回教程首页</RouterLink>
  </main>
</template>

<style scoped>
.guide-v2-not-found {
  width: min(100% - 32px, 720px);
  margin: 80px auto;
  text-align: center;
}

.guide-v2-not-found__code {
  margin: 0;
  color: #2563eb;
  font-weight: 700;
  letter-spacing: 0.1em;
}

a {
  display: inline-flex;
  min-height: 44px;
  margin-top: 16px;
  align-items: center;
  color: #1d4ed8;
}
</style>
