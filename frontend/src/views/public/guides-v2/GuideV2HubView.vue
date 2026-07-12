<script setup lang="ts">
import { RouterLink } from 'vue-router'

import { Icon } from '@/components/icons'
import GuideV2Header from './components/GuideV2Header.vue'
import { GUIDE_V2_REGISTRY } from './guide-v2.registry'

const clientSlugs = Object.freeze([
  'codex',
  'claude-code',
  'opencode',
  'openclaw',
  'chatbox-mobile',
  'cherry-studio-image',
])

const clientEntries = Object.freeze(
  clientSlugs.flatMap((slug) => {
    const entry = GUIDE_V2_REGISTRY.find(({ meta }) => meta.slug === slug)
    return entry ? [entry] : []
  }),
)

const clientIcons = Object.freeze({
  codex: 'terminal',
  'claude-code': 'chatBubble',
  opencode: 'cube',
  openclaw: 'cloud',
  'chatbox-mobile': 'chat',
  'cherry-studio-image': 'sparkles',
} as const)

const setupFlow = Object.freeze([
  { label: '账户', icon: 'userCircle' },
  { label: 'Key', icon: 'key' },
  { label: '客户端', icon: 'cpu' },
  { label: '测试', icon: 'beaker' },
] as const)
</script>

<template>
  <div class="guide-v2-theme" data-guide-v2-hub>
    <GuideV2Header />
    <main class="guide-v2-hub">
      <section class="guide-v2-hub__hero">
        <div class="guide-v2-hub__hero-copy">
          <h1>AI 客户端使用指南</h1>
          <p>把复杂配置拆成一条清晰路径。跟着当前页面给出的值完成设置，新手也能独立连接并验证。</p>
          <div class="guide-v2-hub__entries" aria-label="选择开始方式">
            <RouterLink data-primary-entry to="/guides/v2/get-started">
              从零开始
              <Icon name="arrowRight" size="sm" aria-hidden="true" />
            </RouterLink>
            <a data-primary-entry href="#clients">
              我已有 API Key
              <Icon name="arrowDown" size="sm" aria-hidden="true" />
            </a>
          </div>
        </div>
        <ol class="guide-v2-flow" data-setup-flow aria-label="完整配置流程">
          <li v-for="item in setupFlow" :key="item.label" class="guide-v2-flow__item">
            <span class="guide-v2-flow__icon">
              <Icon :name="item.icon" size="sm" aria-hidden="true" />
            </span>
            <span>{{ item.label }}</span>
          </li>
        </ol>
      </section>

      <section id="clients" class="guide-v2-clients" aria-labelledby="guide-client-title">
        <div class="guide-v2-section-heading">
          <h2 id="guide-client-title">选择你的客户端</h2>
          <p>每篇只保留当前客户端需要的配置、平台分支和验证步骤。</p>
        </div>
        <nav class="guide-v2-client-grid" aria-label="客户端教程">
          <RouterLink
            v-for="entry in clientEntries"
            :key="entry.meta.slug"
            class="guide-v2-client-card"
            data-client-card
            :to="entry.path"
          >
            <span class="guide-v2-client-card__icon">
              <Icon :name="clientIcons[entry.meta.slug as keyof typeof clientIcons]" size="lg" aria-hidden="true" />
            </span>
            <h3>{{ entry.meta.title }}</h3>
            <p>{{ entry.meta.summary }}</p>
            <span class="guide-v2-client-card__meta">
              <span>{{ entry.meta.platforms.join(' · ') }}</span>
              <span>{{ entry.meta.duration }}</span>
              <span>{{ entry.meta.difficulty }}</span>
            </span>
          </RouterLink>
        </nav>
      </section>

      <aside class="guide-v2-legacy" data-legacy-entry>
        <span>仍需查看原有版式？V1 路径和内容保持不变。</span>
        <RouterLink to="/registration-key-guide">
          打开旧版教程
          <Icon name="externalLink" size="xs" aria-hidden="true" />
        </RouterLink>
      </aside>
    </main>
  </div>
</template>

<style src="./styles/tokens.css"></style>
<style src="./styles/layout.css"></style>
<style src="./styles/content.css"></style>
<style src="./styles/responsive.css"></style>
