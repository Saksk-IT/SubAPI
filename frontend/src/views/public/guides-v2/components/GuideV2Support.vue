<script setup lang="ts">
import { RouterLink } from 'vue-router'

import { Icon } from '@/components/icons'
</script>

<template>
  <aside class="guide-v2-support">
    <div class="guide-v2-support__copy">
      <Icon name="questionCircle" size="lg" aria-hidden="true" />
      <div>
        <h2>遇到问题？先按错误类型排查</h2>
        <p>401、404、429、模型不可用和配置未生效都有统一检查顺序。仍无法解决时，可加入官方社群反馈。</p>
      </div>
    </div>
    <div class="guide-v2-support__actions">
      <RouterLink data-troubleshooting-entry to="/guides/v2/troubleshooting">
        打开公共排错中心
        <Icon name="arrowRight" size="sm" aria-hidden="true" />
      </RouterLink>
      <nav class="guide-v2-support__contacts" aria-label="官方联系方式">
        <a
          data-support-telegram
          href="https://t.me/+aW_Sd-9qDBE2MmMx"
          target="_blank"
          rel="noopener noreferrer"
        >
          Telegram 官方社群
          <Icon name="externalLink" size="xs" aria-hidden="true" />
        </a>
        <a
          data-support-qq
          href="https://qm.qq.com/q/KunflJKpEG"
          target="_blank"
          rel="noopener noreferrer"
        >
          QQ 群
          <Icon name="externalLink" size="xs" aria-hidden="true" />
        </a>
      </nav>
    </div>
  </aside>
</template>
