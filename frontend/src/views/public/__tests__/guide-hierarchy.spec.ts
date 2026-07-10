import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ClientGuideView from '@/views/public/ClientGuideView.vue'
import { guideLinks } from '@/views/public/client-guide-data'

const routeState = vi.hoisted(() => ({
  meta: { guideKey: 'codex' },
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
}))

function mountGuide(guideKey: string) {
  routeState.meta.guideKey = guideKey
  return mount(ClientGuideView, {
    global: {
      stubs: {
        Icon: true,
      },
    },
  })
}

describe('static guide hierarchy', () => {
  beforeEach(() => {
    routeState.meta.guideKey = 'codex'
  })

  it('keeps exactly six client guides outside the parent guide', () => {
    expect(guideLinks.map((guide) => guide.key)).toEqual([
      'codex',
      'claude',
      'openCode',
      'openClaw',
      'mobile',
      'image',
    ])
    expect(guideLinks).toHaveLength(6)
  })

  it.each(['codex', 'claude', 'openCode', 'openClaw', 'mobile', 'image'])(
    'shows the parent guide entry on the %s child guide',
    (guideKey) => {
      const wrapper = mountGuide(guideKey)

      expect(wrapper.find('a[href="/registration-key-guide"]').exists()).toBe(true)
      expect(wrapper.find('a[href="/codex-guide#chapterKey"]').exists()).toBe(false)
      expect(wrapper.find('a[href="https://sakai.my/register"]').exists()).toBe(false)
    }
  )

  it('does not render screenshots with full keys or obsolete API addresses', () => {
    const forbiddenImages = new Map([
      ['codex', ['/img/codex-guide/image-5.png', '/img/codex-guide/image-31.png']],
      [
        'mobile',
        [
          '/img/codex-guide/image-38.png',
          '/img/codex-guide/image-39.png',
          '/img/codex-guide/image-41.png',
          '/img/codex-guide/image-43.png',
        ],
      ],
      [
        'image',
        [
          '/img/image-guide/image-6.png',
          '/img/image-guide/image-7.png',
          '/img/image-guide/image-10.png',
          '/img/image-guide/image-11.png',
        ],
      ],
    ])

    for (const [guideKey, imageSources] of forbiddenImages) {
      const wrapper = mountGuide(guideKey)
      for (const imageSource of imageSources) {
        expect(wrapper.find(`img[src="${imageSource}"]`).exists()).toBe(false)
      }
    }
  })

  it('renders Codex as a client-only guide without registration chapters', () => {
    const wrapper = mountGuide('codex')

    expect(wrapper.text()).toContain('Codex API 登录对接教程')
    expect(wrapper.find('#registerAccount').exists()).toBe(false)
    expect(wrapper.find('#redeemBenefits').exists()).toBe(false)
    expect(wrapper.find('#createApiKey').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('注册中转账户')
  })

  it('renders the parent guide with registration, redemption, key creation and six children', () => {
    const wrapper = mountGuide('registration')

    expect(wrapper.text()).toContain('中转注册、兑换与 API 密钥配置教程')
    expect(wrapper.find('#registerAccount').exists()).toBe(true)
    expect(wrapper.find('#redeemBenefits').exists()).toBe(true)
    expect(wrapper.find('#createApiKey').exists()).toBe(true)

    for (const guide of guideLinks) {
      expect(wrapper.find(`a[href="${guide.path}"]`).exists()).toBe(true)
    }
  })

  it('marks the secondary header action so narrow screens can hide it', () => {
    const parent = mountGuide('registration')
    const child = mountGuide('codex')

    expect(
      parent.find('.codex-doc-link--secondary[href="#createApiKey"]').exists()
    ).toBe(true)
    expect(
      child
        .find(
          '.codex-doc-link--secondary[href="/registration-key-guide#createApiKey"]'
        )
        .exists()
    ).toBe(true)
  })
})
