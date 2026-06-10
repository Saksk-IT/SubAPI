<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import { Icon } from '@/components/icons'

import ClaudeCodeGuideContent from './client-guides/ClaudeCodeGuideContent.vue'
import ImageGuideContent from './client-guides/ImageGuideContent.vue'
import MobileGuideContent from './client-guides/MobileGuideContent.vue'
import OpenClawGuideContent from './client-guides/OpenClawGuideContent.vue'
import OpenCodeGuideContent from './client-guides/OpenCodeGuideContent.vue'
import { guideLinks, type GuideKey, type GuidePage } from './client-guide-data'

const guidePages: Record<Exclude<GuideKey, 'codex'>, GuidePage> = {
  claude: {
    active: 'claude',
    title: 'Claude Code 配置教程',
    lead: '从注册中转账户、兑换额度、创建 API Key，到通过 settings.json 或系统环境变量手动接入 Claude Code。',
    badges: [
      { icon: 'userPlus', label: '注册中转' },
      { icon: 'gift', label: '兑换额度' },
      { icon: 'cog', label: '手动配置' },
      { icon: 'terminal', label: 'claude 验证' },
    ],
    jumps: [
      { href: '#claudeStart', label: '从零开始' },
      { href: '#claudeManual', label: '手动配置' },
      { href: '#claudeVerify', label: '验证排错' },
    ],
    toc: [
      {
        title: 'Claude Code 教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#claudeStart', label: '准备 API Key' },
          { href: '#claudeManual', label: '手动配置' },
          { href: '#claudePath', label: '配置目录' },
          { href: '#claudeSettings', label: 'settings.json' },
          { href: '#claudeEnv', label: '系统环境变量' },
          { href: '#claudeVerify', label: '验证与排错' },
        ],
      },
    ],
    component: ClaudeCodeGuideContent,
  },
  openCode: {
    active: 'openCode',
    title: 'Open Code 配置教程',
    lead: '从注册中转账户、兑换额度、创建 API Key 开始，完整接入 Open Code CLI；长期使用写入 opencode.json，临时切换可用 /connect。',
    badges: [
      { icon: 'userPlus', label: '注册中转' },
      { icon: 'key', label: '创建 Key' },
      { icon: 'document', label: 'opencode.json' },
      { icon: 'terminal', label: '/connect' },
    ],
    jumps: [
      { href: '#openCodeStart', label: '从零开始' },
      { href: '#openCodeInstall', label: '安装' },
      { href: '#openCodeJson', label: 'JSON' },
      { href: '#openCodeVerify', label: '验证排错' },
    ],
    toc: [
      {
        title: 'Open Code 教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#openCodeStart', label: '准备 API Key' },
          { href: '#openCodeInstall', label: '安装并启动' },
          { href: '#openCodePath', label: '配置目录' },
          { href: '#openCodeJson', label: 'opencode.json' },
          { href: '#openCodeConnect', label: '/connect 临时切换' },
          { href: '#openCodeVerify', label: '验证与排错' },
        ],
      },
    ],
    component: OpenCodeGuideContent,
  },
  openClaw: {
    active: 'openClaw',
    title: 'Open Claw 配置教程',
    lead: '从注册中转账户、兑换额度、创建 API Key 开始，完整接入 Open Claw。支持腾讯云在线配置，也支持本地 ~/.openclaw 配置。',
    badges: [
      { icon: 'userPlus', label: '注册中转' },
      { icon: 'cloud', label: '腾讯云在线配置' },
      { icon: 'document', label: '本地配置' },
      { icon: 'checkCircle', label: '模型测试' },
    ],
    jumps: [
      { href: '#openClawStart', label: '从零开始' },
      { href: '#openClawCloud', label: '云端' },
      { href: '#openClawLocal', label: '本地' },
      { href: '#openClawCheck', label: '检查' },
    ],
    toc: [
      {
        title: 'Open Claw 教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#openClawStart', label: '准备 API Key' },
          { href: '#openClawCloud', label: '腾讯云在线配置' },
          { href: '#openClawLocal', label: '本地配置' },
          { href: '#openClawCheck', label: '验证与检查' },
        ],
      },
    ],
    component: OpenClawGuideContent,
  },
  mobile: {
    active: 'mobile',
    title: '移动端配置教程',
    lead: '使用 Chatbox 在 iOS、Android 等移动设备接入 sak API 服务，从下载应用、添加模型提供方到完成模型选择，按步骤配置即可。',
    badges: [
      { icon: 'download', label: '下载 Chatbox' },
      { icon: 'chat', label: '手机端配置' },
      { icon: 'bolt', label: 'OpenAI response API 兼容' },
      { icon: 'chatBubble', label: '新对话切换模型' },
    ],
    jumps: [
      { href: '#mobileDownload', label: '下载应用' },
      { href: '#mobileConfig', label: '配置步骤' },
      { href: '#mobileCheck', label: '完成检查' },
      { href: '#mobileUse', label: '开始使用' },
    ],
    toc: [
      {
        title: '移动端教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#mobileDownload', label: '下载 Chatbox' },
          { href: '#mobileHome', label: '确认应用界面' },
          { href: '#mobileConfig', label: '配置步骤' },
          { href: '#mobileModels', label: '获取并选择模型' },
          { href: '#mobileCheck', label: '完成检查' },
          { href: '#mobileUse', label: '开始使用' },
        ],
      },
    ],
    component: MobileGuideContent,
  },
  image: {
    active: 'image',
    title: '图像生成教程',
    lead: '使用 Cherry Studio 接入 https://api.sakms.top/，配置 gpt-image-2 图像生成端点，并通过绘画入口完成专业生图。',
    badges: [
      { icon: 'download', label: '下载 Cherry Studio' },
      { icon: 'key', label: '填写 API Key' },
      { icon: 'sparkles', label: 'gpt-image-2' },
      { icon: 'checkCircle', label: '绘画入口验证' },
    ],
    jumps: [
      { href: '#imageDownload', label: '下载' },
      { href: '#imageService', label: '模型服务' },
      { href: '#imageModel', label: '配置模型' },
      { href: '#imageGenerate', label: '开始生图' },
    ],
    toc: [
      {
        title: '图像生成教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#imageGuideIntro', label: '生图路径说明' },
          { href: '#imageDownload', label: '下载 Cherry Studio' },
          { href: '#imageService', label: '配置模型服务' },
          { href: '#imageModel', label: '配置图像模型' },
          { href: '#imageGenerate', label: '开始生图' },
          { href: '#imageCheck', label: '完成检查' },
        ],
      },
    ],
    component: ImageGuideContent,
  },
}

const route = useRoute()

const page = computed(() => {
  const key = route.meta.guideKey as Exclude<GuideKey, 'codex'>
  return guidePages[key] ?? guidePages.claude
})
</script>

<template>
  <div class="codex-guide-page codex-client-guide-page">
    <header class="codex-doc-header">
      <nav class="codex-doc-nav" aria-label="页面导航">
        <a class="codex-doc-brand" href="/" aria-label="返回 GPT Team 服务台首页">
          <img src="/logo.png" alt="Logo">
          <span>
            <strong>GPT Team</strong>
            <small>客户端配置文档</small>
          </span>
        </a>
        <div class="codex-doc-actions">
          <a href="/codex-guide" class="codex-doc-link">
            <Icon name="book" class="codex-icon" /> Codex 总教程
          </a>
          <a href="/codex-guide#chapterKey" class="codex-doc-link">
            <Icon name="key" class="codex-icon" /> 先创建 Key
          </a>
          <a href="https://api.sakms.top/profile" target="_blank" rel="noopener noreferrer" class="codex-doc-cta">
            <Icon name="externalLink" class="codex-icon" /> 打开额度查询
          </a>
        </div>
      </nav>
    </header>

    <div class="codex-guide-frame">
      <aside class="codex-doc-toc" aria-label="教程目录">
        <div class="codex-doc-toc__heading">
          <Icon name="book" class="codex-icon" />
          <div>
            <strong>教程目录</strong>
            <span>点击章节即可跳转</span>
          </div>
        </div>
        <nav class="codex-doc-toc__nav">
          <section v-for="section in page.toc" :key="section.title" class="codex-doc-toc-group">
            <p>{{ section.title }}</p>
            <a
              v-for="item in section.items"
              :key="item.href"
              class="codex-doc-toc-link"
              :href="item.href"
            >
              <Icon name="chevronRight" class="codex-icon" />
              <span>{{ item.label }}</span>
            </a>
          </section>
        </nav>
      </aside>

      <main class="codex-doc-shell">
        <section class="codex-doc-hero" aria-labelledby="guideTitle">
          <p class="codex-doc-base">API base_url: https://api.sakms.top/</p>
          <h1 id="guideTitle">{{ page.title }}</h1>
          <p class="codex-doc-lead">{{ page.lead }}</p>
          <div class="codex-doc-badges" aria-label="教程要点">
            <span v-for="badge in page.badges" :key="badge.label">
              <Icon :name="badge.icon" class="codex-icon" /> {{ badge.label }}
            </span>
          </div>
          <nav class="codex-doc-jump" aria-label="章节快捷入口">
            <a v-for="jump in page.jumps" :key="jump.href" :href="jump.href">{{ jump.label }}</a>
          </nav>

          <nav class="codex-guide-switcher" aria-label="客户端教程互跳入口">
            <p class="codex-guide-switcher__label">客户端配置教程</p>
            <div class="codex-client-guide-grid codex-client-guide-grid--all">
              <a
                v-for="guide in guideLinks"
                :key="guide.key"
                class="codex-client-guide-card codex-client-guide-card--compact"
                :class="{ 'codex-client-guide-card--active': guide.key === page.active }"
                :href="guide.path"
                :aria-current="guide.key === page.active ? 'page' : undefined"
              >
                <Icon :name="guide.icon" class="codex-icon" />
                <span>
                  <strong>{{ guide.title }}</strong>
                  <small>{{ guide.description }}</small>
                </span>
              </a>
            </div>
          </nav>
        </section>

        <div class="codex-doc-main-grid">
          <article class="codex-doc-article">
            <component :is="page.component" />
          </article>
        </div>
      </main>
    </div>
  </div>
</template>

<style src="@/styles/codex-guide.css"></style>
