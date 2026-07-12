import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import GuideV2Renderer from '../GuideV2Renderer.vue'
import type { ParsedGuideV2 } from '../../guide-v2.types'

const guide: ParsedGuideV2 = Object.freeze({
  meta: Object.freeze({
    title: 'Codex 配置',
    slug: 'codex',
    summary: '完成配置',
    duration: '8 分钟',
    platforms: Object.freeze(['Windows', 'macOS']),
    difficulty: '入门',
    updatedAt: '2026-07-13',
    version: 'v2',
  }),
  platforms: Object.freeze(['Windows', 'macOS']),
  toc: Object.freeze([{ level: 2, title: '初始化', anchor: 'initialize' }]),
  steps: Object.freeze([{ number: 1, title: '初始化', anchor: 'initialize' }]),
  blocks: Object.freeze([
    Object.freeze({ type: 'heading' as const, level: 1 as const, text: 'Codex 配置' }),
    Object.freeze({
      type: 'heading' as const,
      level: 2 as const,
      text: '初始化',
      anchor: 'initialize',
      stepNumber: 1,
    }),
    Object.freeze({ type: 'paragraph' as const, html: '<p>先打开一次。</p>' }),
    Object.freeze({
      type: 'paragraph' as const,
      html: '<blockquote><p>[!WARNING] 完全关闭客户端。</p></blockquote>',
    }),
    Object.freeze({ type: 'code' as const, language: 'toml', code: 'model = "example"' }),
    Object.freeze({
      type: 'heading' as const,
      level: 3 as const,
      text: 'Windows',
      anchor: 'windows',
      platform: 'Windows',
    }),
    Object.freeze({ type: 'paragraph' as const, html: '<p>Windows 路径</p>' }),
    Object.freeze({
      type: 'heading' as const,
      level: 3 as const,
      text: 'macOS',
      anchor: 'macos',
      platform: 'macOS',
    }),
    Object.freeze({ type: 'paragraph' as const, html: '<p>macOS 路径</p>' }),
    Object.freeze({
      type: 'media' as const,
      id: 'codex/config',
      path: '/img/guides/v2/codex/config.webp',
      alt: '配置示意',
      title: '复制当前值',
    }),
  ]),
})

describe('GuideV2Renderer', () => {
  it('把规范化块映射为步骤、提示、代码和媒体，不重复 H1', () => {
    const wrapper = mount(GuideV2Renderer, {
      props: { guide, completedStepIds: [], selectedPlatform: 'Windows' },
    })

    expect(wrapper.find('h1').exists()).toBe(false)
    expect(wrapper.get('[data-guide-v2-step]').text()).toContain('初始化')
    expect(wrapper.get('[data-guide-v2-notice]').text()).toContain('完全关闭客户端')
    expect(wrapper.get('pre').text()).toContain('model')
    expect(wrapper.get('img').attributes('alt')).toBe('配置示意')
  })

  it('平台标签支持方向键与 Enter/Space，并只显示当前平台分支', async () => {
    const wrapper = mount(GuideV2Renderer, {
      props: { guide, completedStepIds: [], selectedPlatform: 'Windows' },
    })
    const tabs = wrapper.findAll('[role="tab"]')

    expect(wrapper.text()).toContain('Windows 路径')
    expect(wrapper.text()).not.toContain('macOS 路径')
    await tabs[0].trigger('keydown', { key: 'ArrowRight' })
    await tabs[1].trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('select-platform')).toContainEqual(['macOS'])
    await tabs[1].trigger('keydown', { key: ' ' })
    expect(wrapper.emitted('select-platform')).toContainEqual(['macOS'])
  })

  it('步骤按钮可完成与取消且提供可见文字状态', async () => {
    const wrapper = mount(GuideV2Renderer, {
      props: { guide, completedStepIds: [], selectedPlatform: 'Windows' },
    })
    await wrapper.get('[data-step-toggle]').trigger('click')
    expect(wrapper.emitted('toggle-step')).toEqual([['initialize']])
    expect(wrapper.text()).toContain('标记完成')
  })
})
