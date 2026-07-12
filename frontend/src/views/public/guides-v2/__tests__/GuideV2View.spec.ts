import { flushPromises, mount, RouterLinkStub, type VueWrapper } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const appStore = vi.hoisted(() => ({ siteName: '蓝图测试站' }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))

import GuideV2View from '../GuideV2View.vue'
import {
  createGuideV2Progress,
  GUIDE_V2_PROGRESS_STORAGE_KEY,
} from '../composables/useGuideV2Progress'

const mountedWrappers: VueWrapper[] = []

const mountAt = async (path = '/guides/v2/codex') => {
  const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/guides/v2/:slug', component: GuideV2View }],
  })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(GuideV2View, {
    attachTo: document.body,
    global: { plugins: [router], stubs: { RouterLink: RouterLinkStub } },
  })
  mountedWrappers.push(wrapper)
  await flushPromises()
  await vi.waitFor(() => expect(wrapper.find('h1').exists()).toBe(true))
  return { wrapper, router }
}

describe('GuideV2View', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    createGuideV2Progress(localStorage).clearAll()
    document.title = '原始标题'
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: vi.fn(),
    })
  })

  afterEach(() => {
    mountedWrappers.splice(0).forEach((wrapper) => {
      if (wrapper.element.isConnected) wrapper.unmount()
    })
  })

  it('详情页只有一个 H1，移动目录紧跟 Hero，桌面目录粘性布局', async () => {
    const { wrapper, router } = await mountAt()

    expect(wrapper.findAll('h1')).toHaveLength(1)
    const platforms = wrapper.get('[data-guide-platforms]')
    expect(platforms.text()).toMatch(/Windows.*macOS.*Linux/s)
    expect(platforms.attributes('aria-label')).toBe('支持平台')
    expect(platforms.attributes('role')).toBe('group')
    expect(wrapper.get('[data-guide-v2-hero]').element.nextElementSibling).toBe(
      wrapper.get('[data-guide-v2-mobile-toc]').element,
    )
    expect(wrapper.get('[data-guide-v2-sidebar]').classes()).toContain('guide-v2-sidebar')
    expect(document.title).toBe('原始标题')
    document.title = '全局路由标题'
    await router.push('/guides/v2/claude-code')
    await vi.waitFor(() => expect(wrapper.get('h1').text()).toBe('Claude Code 配置'))
    expect(document.title).toBe('全局路由标题')
    wrapper.unmount()
    expect(document.title).toBe('全局路由标题')
  })

  it('平台切换同步过滤正文、桌面目录和移动目录，同时保留共享步骤', async () => {
    const { wrapper } = await mountAt()
    const sidebar = wrapper.get('[data-guide-v2-sidebar]')

    expect(sidebar.text()).toContain('定位平台配置目录')
    expect(sidebar.text()).toContain('Windows')
    expect(sidebar.text()).not.toContain('macOS')
    expect(sidebar.text()).not.toContain('Linux')

    await wrapper.get('[data-mobile-toc-trigger]').trigger('click')
    expect(wrapper.get('[role="dialog"]').text()).toContain('Windows')
    expect(wrapper.get('[role="dialog"]').text()).not.toContain('macOS')
    await wrapper.get('button[aria-label="关闭目录"]').trigger('click')

    const macTab = wrapper.findAll('[role="tab"]').find((tab) => tab.text() === 'macOS')!
    await macTab.trigger('click')
    expect(sidebar.text()).toContain('定位平台配置目录')
    expect(sidebar.text()).toContain('macOS')
    expect(sidebar.text()).not.toContain('Windows')
    expect(sidebar.text()).not.toContain('Linux')
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

  it('平台专属深链自动选择相应平台并在目标存在后记录和滚动', async () => {
    const { wrapper } = await mountAt('/guides/v2/codex#macos')

    await vi.waitFor(() => {
      expect(wrapper.get('[role="tab"][aria-selected="true"]').text()).toBe('macOS')
    })
    const target = wrapper.get('#macos').element as HTMLElement
    expect(target.scrollIntoView).toBe(HTMLElement.prototype.scrollIntoView)
    await vi.waitFor(() => {
      expect(localStorage.getItem(GUIDE_V2_PROGRESS_STORAGE_KEY)).toContain('"lastAnchor":"macos"')
    })
    expect(HTMLElement.prototype.scrollIntoView).toHaveBeenCalled()
  })

  it.each(['#foo_bar', '#%E0%A4%A', '#missing-anchor'])(
    '安全忽略非法、恶意编码或未知 hash：%s',
    async (hash) => {
      const { wrapper } = await mountAt(`/guides/v2/codex${hash}`)

      expect(wrapper.get('[role="tab"][aria-selected="true"]').text()).toBe('Windows')
      expect(HTMLElement.prototype.scrollIntoView).not.toHaveBeenCalled()
      expect(localStorage.getItem(GUIDE_V2_PROGRESS_STORAGE_KEY) ?? '').not.toContain(hash.slice(1))
    },
  )

  it('目标 DOM 缺失时不记录阅读位置且不会产生未处理异常', async () => {
    const nativeGetElementById = document.getElementById.bind(document)
    vi.spyOn(document, 'getElementById').mockImplementation((id) =>
      id === 'write-config-files' ? null : nativeGetElementById(id),
    )
    await mountAt('/guides/v2/codex#write-config-files')

    expect(HTMLElement.prototype.scrollIntoView).not.toHaveBeenCalled()
    expect(localStorage.getItem(GUIDE_V2_PROGRESS_STORAGE_KEY) ?? '').not.toContain(
      'write-config-files',
    )
  })

  it('滚动 API 抛错时捕获异常并保持页面可继续使用', async () => {
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: vi.fn(() => {
        throw new Error('scroll blocked')
      }),
    })
    const { wrapper } = await mountAt('/guides/v2/codex#write-config-files')

    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.get('h1').text()).toBe('Codex API 配置')
  })
})
