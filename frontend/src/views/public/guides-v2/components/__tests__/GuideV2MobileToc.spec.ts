import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import GuideV2MobileToc from '../GuideV2MobileToc.vue'

const toc = Object.freeze([
  { level: 2 as const, title: '初始化', anchor: 'initialize' },
  { level: 3 as const, title: 'Windows', anchor: 'windows' },
])

describe('GuideV2MobileToc', () => {
  it('支持打开目录、Esc 关闭并把焦点还给触发器', async () => {
    const wrapper = mount(GuideV2MobileToc, {
      attachTo: document.body,
      props: { toc, completedStepIds: [] },
    })
    const trigger = wrapper.get('[data-mobile-toc-trigger]')

    await trigger.trigger('click')
    const dialog = wrapper.get('[role="dialog"]')
    const closeButton = wrapper.get('button[aria-label="关闭目录"]')
    expect(dialog.attributes('aria-modal')).toBe('true')
    expect(document.activeElement).toBe(closeButton.element)

    document.activeElement?.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }),
    )
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)
    wrapper.unmount()
  })

  it('选择目录项时关闭抽屉并发出锚点事件', async () => {
    const target = document.createElement('h2')
    target.id = 'initialize'
    document.body.append(target)
    const wrapper = mount(GuideV2MobileToc, {
      attachTo: document.body,
      props: { toc, completedStepIds: [] },
    })
    await wrapper.get('[data-mobile-toc-trigger]').trigger('click')
    await wrapper.get('a[href="#initialize"]').trigger('click')

    expect(wrapper.emitted('navigate')).toEqual([['initialize']])
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(document.activeElement).toBe(target)
    expect(target.getAttribute('tabindex')).toBe('-1')
    wrapper.unmount()
    target.remove()
  })

  it('将 Tab 和 Shift+Tab 循环限制在目录对话框，取消关闭恢复触发器', async () => {
    const wrapper = mount(GuideV2MobileToc, {
      attachTo: document.body,
      props: { toc, completedStepIds: [] },
    })
    const trigger = wrapper.get('[data-mobile-toc-trigger]')
    await trigger.trigger('click')

    const closeButton = wrapper.get('button[aria-label="关闭目录"]')
    const links = wrapper.findAll('[role="dialog"] a')
    links.at(-1)!.element.focus()
    links.at(-1)!.element.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true }))
    expect(document.activeElement).toBe(closeButton.element)

    closeButton.element.focus()
    closeButton.element.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true }),
    )
    expect(document.activeElement).toBe(links.at(-1)!.element)

    await wrapper.get('.guide-v2-mobile-toc__backdrop').trigger('click')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)
    wrapper.unmount()
  })
})
