import { mount, RouterLinkStub } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const appStore = vi.hoisted(() => ({ siteName: '蓝图测试站' }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))

import GuideV2HubView from '../GuideV2HubView.vue'

describe('GuideV2HubView', () => {
  beforeEach(() => {
    appStore.siteName = '蓝图测试站'
  })

  it('使用动态网站品牌且全页只有一个 H1', () => {
    const wrapper = mount(GuideV2HubView, {
      global: { stubs: { RouterLink: RouterLinkStub } },
    })

    expect(wrapper.text()).toContain('蓝图测试站')
    expect(wrapper.text()).not.toContain('SAK AI')
    expect(wrapper.findAll('h1')).toHaveLength(1)
  })

  it('首屏呈现四段流程、六个客户端和仅两个主入口', () => {
    const wrapper = mount(GuideV2HubView, {
      global: { stubs: { RouterLink: RouterLinkStub } },
    })

    expect(wrapper.get('[data-setup-flow]').text()).toMatch(/账户.*Key.*客户端.*测试/s)
    expect(wrapper.findAll('[data-client-card]')).toHaveLength(6)
    expect(wrapper.findAll('[data-primary-entry]')).toHaveLength(2)
    expect(wrapper.get('[data-support-telegram]').attributes('href')).toBe(
      'https://t.me/+aW_Sd-9qDBE2MmMx',
    )
    expect(wrapper.get('[data-support-qq]').attributes('href')).toBe(
      'https://qm.qq.com/q/KunflJKpEG',
    )
    expect(wrapper.get('[data-legacy-entry]').text()).toContain('旧版')
  })
})
