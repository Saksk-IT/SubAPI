<script setup lang="ts">
import { RouterLink } from 'vue-router'

import { Icon } from '@/components/icons'
</script>

<template>
  <aside class="guide-v2-support">
    <div>
      <Icon name="questionCircle" size="lg" aria-hidden="true" />
      <div>
        <h2>遇到问题？先按错误类型排查</h2>
        <p>401、404、429、模型不可用和配置未生效都有统一检查顺序。</p>
      </div>
    </div>
    <RouterLink data-troubleshooting-entry to="/guides/v2/troubleshooting">
      打开公共排错中心
      <Icon name="arrowRight" size="sm" aria-hidden="true" />
    </RouterLink>
  </aside>
</template>
