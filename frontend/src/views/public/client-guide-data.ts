import type { Component } from 'vue'

import { Icon } from '@/components/icons'

export type GuideKey = 'codex' | 'claude' | 'openCode' | 'openClaw' | 'mobile'

export type GuideLink = {
  key: GuideKey
  path: string
  title: string
  description: string
  icon: InstanceType<typeof Icon>['$props']['name']
}

export type TocSection = {
  title: string
  items: {
    href: string
    label: string
  }[]
}

export type GuidePage = {
  active: GuideKey
  title: string
  lead: string
  badges: {
    icon: InstanceType<typeof Icon>['$props']['name']
    label: string
  }[]
  jumps: {
    href: string
    label: string
  }[]
  toc: TocSection[]
  component: Component
}

export const guideLinks: GuideLink[] = [
  {
    key: 'codex',
    path: '/codex-guide',
    title: 'Codex 配置教程',
    description: 'config.toml / auth.json / API 登录',
    icon: 'terminal',
  },
  {
    key: 'claude',
    path: '/claude-code-guide',
    title: 'Claude Code 配置教程',
    description: 'settings.json / 环境变量 / CLI 验证',
    icon: 'terminal',
  },
  {
    key: 'openCode',
    path: '/open-code-guide',
    title: 'Open Code 配置教程',
    description: 'opencode.json / /connect 临时切换',
    icon: 'cube',
  },
  {
    key: 'openClaw',
    path: '/open-claw-guide',
    title: 'Open Claw 配置教程',
    description: '腾讯云在线配置 / 本地配置',
    icon: 'cloud',
  },
  {
    key: 'mobile',
    path: '/mobile-guide',
    title: '移动端配置教程',
    description: 'Chatbox / 手机配置 / 模型切换',
    icon: 'chat',
  },
]
