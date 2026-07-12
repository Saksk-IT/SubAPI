import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import GuideV2Media from '../GuideV2Media.vue'

describe('GuideV2Media', () => {
  it('为正文图片提供固定尺寸、懒加载和 PNG 降级重试', async () => {
    const wrapper = mount(GuideV2Media, {
      props: {
        kind: 'image',
        src: '/img/guides/v2/codex/config-folder.webp',
        fallbackSrc: '/img/guides/v2/codex/config-folder.png',
        alt: 'Codex 配置目录',
        caption: '不同系统使用相同目录概念',
        width: 1280,
        height: 720,
      },
    })
    const image = wrapper.get('img')

    expect(image.attributes()).toMatchObject({
      alt: 'Codex 配置目录',
      width: '1280',
      height: '720',
      loading: 'lazy',
    })

    await image.trigger('error')
    expect(wrapper.text()).toContain('图片加载失败')
    expect(wrapper.get('button').text()).toContain('重试')
    await wrapper.get('button').trigger('click')
    expect(wrapper.get('img').attributes('src')).toContain('config-folder.png')
  })

  it('视频不自动播放并提供封面、文字链接和元数据预载', () => {
    const wrapper = mount(GuideV2Media, {
      props: {
        kind: 'video',
        src: '/video/setup.mp4',
        poster: '/img/setup-poster.webp',
        alt: '完整配置演示',
        caption: '跟随视频完成配置',
        transcriptUrl: 'https://example.com/setup-text',
        width: 1280,
        height: 720,
      },
    })
    const video = wrapper.get('video')

    expect(video.attributes('autoplay')).toBeUndefined()
    expect(video.attributes('preload')).toBe('metadata')
    expect(video.attributes('poster')).toBe('/img/setup-poster.webp')
    expect(wrapper.get('a').text()).toContain('查看文字步骤')
  })
})
