<script setup lang="ts">
import { RouterLink } from 'vue-router'

import { Icon } from '@/components/icons'
import GuideV2Header from './components/GuideV2Header.vue'
</script>

<template>
  <div class="guide-v2-theme" data-guide-v2-not-found>
    <GuideV2Header />
    <main class="guide-v2-not-found">
      <div class="guide-v2-not-found__inner">
        <p class="guide-v2-not-found__code">404 · GUIDE NOT FOUND</p>
        <h1>未找到这篇 V2 教程</h1>
        <p>这篇教程还不在蓝图里。链接可能已失效，返回教程中心重新选择客户端即可。</p>
        <RouterLink to="/guides/v2">
          返回教程中心
          <Icon name="arrowRight" size="sm" aria-hidden="true" />
        </RouterLink>
      </div>
    </main>
  </div>
</template>

<style src="./styles/tokens.css"></style>
<style src="./styles/layout.css"></style>
<style src="./styles/content.css"></style>
<style src="./styles/responsive.css"></style>
