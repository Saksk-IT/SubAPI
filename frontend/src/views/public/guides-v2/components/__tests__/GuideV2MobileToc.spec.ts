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
    expect(wrapper.get('[role="dialog"]').attributes('aria-modal')).toBe('true')

    await wrapper.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)
    wrapper.unmount()
  })

  it('选择目录项时关闭抽屉并发出锚点事件', async () => {
    const wrapper = mount(GuideV2MobileToc, { props: { toc, completedStepIds: [] } })
    await wrapper.get('[data-mobile-toc-trigger]').trigger('click')
    await wrapper.get('a[href="#initialize"]').trigger('click')

    expect(wrapper.emitted('navigate')).toEqual([['initialize']])
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })
})
