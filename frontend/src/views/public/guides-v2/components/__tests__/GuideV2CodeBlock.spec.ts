import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import GuideV2CodeBlock from '../GuideV2CodeBlock.vue'

describe('GuideV2CodeBlock', () => {
  afterEach(() => vi.restoreAllMocks())

  it('复制成功后显示非动画的文字反馈', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
    const wrapper = mount(GuideV2CodeBlock, {
      props: { code: 'pnpm install', language: 'bash' },
    })

    await wrapper.get('button').trigger('click')

    expect(writeText).toHaveBeenCalledWith('pnpm install')
    expect(wrapper.get('[role="status"]').text()).toContain('已复制')
  })

  it('复制失败时明确提示手动选择并复制', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('denied'))
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
    const wrapper = mount(GuideV2CodeBlock, {
      props: { code: 'example', language: 'text' },
    })

    await wrapper.get('button').trigger('click')

    expect(wrapper.text()).toContain('请手动选择并复制')
  })
})
