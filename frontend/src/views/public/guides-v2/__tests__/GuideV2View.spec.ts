import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const appStore = vi.hoisted(() => ({ siteName: '蓝图测试站' }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))

import GuideV2View from '../GuideV2View.vue'
import { GUIDE_V2_PROGRESS_STORAGE_KEY } from '../composables/useGuideV2Progress'

const mountAt = async (path = '/guides/v2/codex') => {
  const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/guides/v2/:slug', component: GuideV2View }],
  })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(GuideV2View, {
    global: { plugins: [router], stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  await vi.waitFor(() => expect(wrapper.find('h1').exists()).toBe(true))
  return { wrapper, router }
}

describe('GuideV2View', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('详情页只有一个 H1，移动目录紧跟 Hero，桌面目录粘性布局', async () => {
    const { wrapper } = await mountAt()

    expect(wrapper.findAll('h1')).toHaveLength(1)
    expect(wrapper.get('[data-guide-v2-hero]').element.nextElementSibling).toBe(
      wrapper.get('[data-guide-v2-mobile-toc]').element,
    )
    expect(wrapper.get('[data-guide-v2-sidebar]').classes()).toContain('guide-v2-sidebar')
  })

  it('保存平台和步骤进度，支持清除本篇与全部进度', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const { wrapper } = await mountAt()
    const macTab = wrapper.findAll('[role="tab"]').find((tab) => tab.text() === 'macOS')
    expect(macTab).toBeDefined()
    await macTab!.trigger('click')
    await wrapper.get('[data-step-toggle]').trigger('click')

    expect(localStorage.getItem(GUIDE_V2_PROGRESS_STORAGE_KEY)).toContain('macOS')
    expect(localStorage.getItem(GUIDE_V2_PROGRESS_STORAGE_KEY)).toContain('install-and-initialize')

    wrapper.unmount()
    const { wrapper: restored } = await mountAt()
    expect(restored.get('[role="tab"][aria-selected="true"]').text()).toBe('macOS')

    await restored.get('[data-clear-guide]').trigger('click')
    expect(restored.get('[role="tab"][aria-selected="true"]').text()).toBe('Windows')
    await restored.get('[data-clear-all]').trigger('click')
    expect(localStorage.getItem(GUIDE_V2_PROGRESS_STORAGE_KEY)).toBeNull()
  })

  it('提供上一篇/下一篇和公共排错入口，并在 hash 变化后定位锚点', async () => {
    const scrollIntoView = vi.fn()
    vi.spyOn(document, 'getElementById').mockReturnValue({ scrollIntoView } as unknown as HTMLElement)
    const { wrapper, router } = await mountAt()

    expect(wrapper.find('[data-guide-previous]').exists()).toBe(true)
    expect(wrapper.find('[data-guide-next]').exists()).toBe(true)
    expect(wrapper.getComponent('[data-troubleshooting-entry]').props('to')).toBe(
      '/guides/v2/troubleshooting',
    )

    await router.push('/guides/v2/codex#write-config-files')
    await flushPromises()
    expect(scrollIntoView).toHaveBeenCalled()
  })
})
