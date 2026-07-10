<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import { Icon } from '@/components/icons'

import ClaudeCodeGuideContent from './client-guides/ClaudeCodeGuideContent.vue'
import CodexGuideContent from './client-guides/CodexGuideContent.vue'
import ImageGuideContent from './client-guides/ImageGuideContent.vue'
import MobileGuideContent from './client-guides/MobileGuideContent.vue'
import OpenClawGuideContent from './client-guides/OpenClawGuideContent.vue'
import OpenCodeGuideContent from './client-guides/OpenCodeGuideContent.vue'
import RegistrationKeyGuideContent from './client-guides/RegistrationKeyGuideContent.vue'
import {
  guideLinks,
  parentGuideLink,
  type GuideKey,
  type GuidePage,
} from './client-guide-data'

const guidePages: Record<GuideKey, GuidePage> = {
  registration: {
    active: 'registration',
    title: '中转注册、兑换与 API 密钥配置教程',
    baseLabel: 'API base_url: https://sakai.my/',
    lead: '所有客户端配置的父教程：先完成中转账户注册、权益兑换、API 密钥创建与分组选择，再进入对应的客户端子教程。',
    badges: [
      { icon: 'userPlus', label: '注册中转账户' },
      { icon: 'gift', label: '兑换权益' },
      { icon: 'key', label: '创建 API 密钥' },
      { icon: 'checkCircle', label: '选择正确分组' },
    ],
    jumps: [
      { href: '#guideHierarchy', label: '教程目录' },
      { href: '#registerAccount', label: '注册' },
      { href: '#redeemBenefits', label: '兑换' },
      { href: '#createApiKey', label: '创建 Key' },
    ],
    toc: [
      {
        title: '父教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#guideHierarchy', label: '父子目录关系' },
          { href: '#usageNotes', label: '使用前说明' },
          { href: '#registerAccount', label: '注册中转账户' },
          { href: '#redeemBenefits', label: '获取并兑换权益' },
          { href: '#createApiKey', label: '创建 API 密钥' },
          { href: '#clientConfig', label: '查看客户端配置' },
          { href: '#parentFaq', label: '常见问题' },
        ],
      },
    ],
    component: RegistrationKeyGuideContent,
  },
  codex: {
    active: 'codex',
    title: 'Codex API 登录对接教程',
    baseLabel: 'base_url 与 API Key：以父教程“使用密钥”弹窗为准',
    lead: '完成父教程后，配置 Codex 的 config.toml 与 auth.json，使用自己的 API Key 登录并验证连接。',
    badges: [
      { icon: 'download', label: '下载并初始化 Codex' },
      { icon: 'document', label: 'config.toml / auth.json' },
      { icon: 'key', label: 'API 登录' },
      { icon: 'checkCircle', label: '验证与排错' },
    ],
    jumps: [
      { href: '#codexStart', label: '开始前准备' },
      { href: '#codexManual', label: '手动配置' },
      { href: '#codexLogin', label: 'API 登录' },
      { href: '#codexVerify', label: '验证排错' },
    ],
    toc: [
      {
        title: 'Codex 教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#codexStart', label: '开始前准备' },
          { href: '#codexManual', label: '手动配置 Codex' },
          { href: '#codexWindows', label: 'Windows 配置' },
          { href: '#codexMac', label: 'macOS 配置' },
          { href: '#codexLogin', label: '重新登录' },
          { href: '#codexVerify', label: '验证与排错' },
        ],
      },
    ],
    component: CodexGuideContent,
  },
  claude: {
    active: 'claude',
    title: 'Claude Code 配置教程',
    baseLabel: 'base_url 与 API Key：以父教程“使用密钥”弹窗为准',
    lead: '完成父教程后，通过 settings.json 或系统环境变量接入 Claude Code，并使用新终端验证配置。',
    badges: [
      { icon: 'book', label: '先完成父教程' },
      { icon: 'document', label: 'settings.json' },
      { icon: 'cog', label: '系统环境变量' },
      { icon: 'terminal', label: 'claude 验证' },
    ],
    jumps: [
      { href: '#claudeStart', label: '开始前准备' },
      { href: '#claudeManual', label: '手动配置' },
      { href: '#claudeVerify', label: '验证排错' },
    ],
    toc: [
      {
        title: 'Claude Code 教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#claudeStart', label: '开始前准备' },
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
    baseLabel: 'base_url 与 API Key：以父教程“使用密钥”弹窗为准',
    lead: '完成父教程后接入 Open Code CLI；长期使用写入 opencode.json，临时切换可使用 /connect。',
    badges: [
      { icon: 'book', label: '先完成父教程' },
      { icon: 'download', label: '安装 Open Code' },
      { icon: 'document', label: 'opencode.json' },
      { icon: 'terminal', label: '/connect' },
    ],
    jumps: [
      { href: '#openCodeStart', label: '开始前准备' },
      { href: '#openCodeInstall', label: '安装' },
      { href: '#openCodeJson', label: 'JSON' },
      { href: '#openCodeVerify', label: '验证排错' },
    ],
    toc: [
      {
        title: 'Open Code 教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#openCodeStart', label: '开始前准备' },
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
    baseLabel: 'base_url 与 API Key：以父教程“使用密钥”弹窗为准',
    lead: '完成父教程后接入 Open Claw，支持腾讯云在线配置和 Windows、macOS、Linux 本地配置。',
    badges: [
      { icon: 'book', label: '先完成父教程' },
      { icon: 'cloud', label: '腾讯云在线配置' },
      { icon: 'document', label: '本地配置' },
      { icon: 'checkCircle', label: '模型测试' },
    ],
    jumps: [
      { href: '#openClawStart', label: '开始前准备' },
      { href: '#openClawCloud', label: '云端' },
      { href: '#openClawLocal', label: '本地' },
      { href: '#openClawCheck', label: '检查' },
    ],
    toc: [
      {
        title: 'Open Claw 教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#openClawStart', label: '开始前准备' },
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
    title: '移动端 Chatbox 配置教程',
    baseLabel: 'API 主机与 API Key：以父教程“使用密钥”弹窗为准',
    lead: '完成父教程后，在 iOS、Android 等移动设备使用 Chatbox 添加模型提供方并选择可用模型。',
    badges: [
      { icon: 'book', label: '先完成父教程' },
      { icon: 'download', label: '下载 Chatbox' },
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
          { href: '#mobileStart', label: '开始前准备' },
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
    title: 'Cherry Studio 图像生成教程',
    baseLabel: 'API 地址与 API Key：以父教程“使用密钥”弹窗为准',
    lead: '完成父教程后，在 Cherry Studio 配置 gpt-image-2 图像生成端点，并通过绘画入口完成生图。',
    badges: [
      { icon: 'book', label: '先完成父教程' },
      { icon: 'download', label: '下载 Cherry Studio' },
      { icon: 'sparkles', label: 'gpt-image-2' },
      { icon: 'checkCircle', label: '绘画入口验证' },
    ],
    jumps: [
      { href: '#imageStart', label: '开始前准备' },
      { href: '#imageDownload', label: '下载' },
      { href: '#imageModel', label: '配置模型' },
      { href: '#imageGenerate', label: '开始生图' },
    ],
    toc: [
      {
        title: '图像生成教程',
        items: [
          { href: '#guideTitle', label: '教程总览' },
          { href: '#imageStart', label: '开始前准备' },
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
  const key = route.meta.guideKey as GuideKey
  return guidePages[key] ?? guidePages.registration
})

const isParentGuide = computed(() => page.value.active === 'registration')
</script>

<template>
  <div class="codex-guide-page codex-client-guide-page">
    <header class="codex-doc-header">
      <nav class="codex-doc-nav" aria-label="页面导航">
        <a class="codex-doc-brand" href="/" aria-label="返回 GPT Team 服务台首页">
          <img src="/logo.png" alt="Logo">
          <span>
            <strong>GPT Team</strong>
            <small>{{ isParentGuide ? '中转接入父教程' : '客户端配置子教程' }}</small>
          </span>
        </a>
        <div class="codex-doc-actions">
          <template v-if="isParentGuide">
            <a href="#createApiKey" class="codex-doc-link codex-doc-link--secondary">
              <Icon name="key" class="codex-icon" /> 创建 API 密钥
            </a>
            <a href="https://sakai.my/register" target="_blank" rel="noopener noreferrer" class="codex-doc-cta">
              <Icon name="externalLink" class="codex-icon" /> 打开中转注册
            </a>
          </template>
          <template v-else>
            <a :href="parentGuideLink.path" class="codex-doc-link">
              <Icon name="book" class="codex-icon" /> 父教程
            </a>
            <a :href="parentGuideLink.keyPath" class="codex-doc-link codex-doc-link--secondary">
              <Icon name="key" class="codex-icon" /> 先创建 Key
            </a>
            <a href="https://sakai.my/profile" target="_blank" rel="noopener noreferrer" class="codex-doc-cta">
              <Icon name="externalLink" class="codex-icon" /> 打开额度查询
            </a>
          </template>
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
          <p class="codex-doc-base">{{ page.baseLabel }}</p>
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
            <p class="codex-guide-switcher__label">
              {{ isParentGuide ? '选择客户端子教程' : '切换客户端子教程' }}
            </p>
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
