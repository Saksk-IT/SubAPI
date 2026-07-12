import { describe, expect, it } from 'vitest'

import type { ParsedGuideV2 } from '../guide-v2.types'
import { deriveGuideV2Visibility } from '../guide-v2.visibility'

const guide: ParsedGuideV2 = Object.freeze({
  meta: Object.freeze({
    title: '平台作用域',
    slug: 'codex',
    summary: '验证平台内容边界',
    duration: '3 分钟',
    platforms: Object.freeze(['Windows', 'macOS']),
    difficulty: '入门',
    updatedAt: '2026-07-13',
    version: 'v2',
  }),
  platforms: Object.freeze(['Windows', 'macOS']),
  toc: Object.freeze([
    { level: 2, title: '定位目录', anchor: 'locate' },
    { level: 3, title: 'Windows', anchor: 'windows' },
    { level: 3, title: 'macOS', anchor: 'macos' },
  ]),
  steps: Object.freeze([{ number: 1, title: '定位目录', anchor: 'locate' }]),
  blocks: Object.freeze([
    { type: 'heading', level: 1, text: '平台作用域' },
    { type: 'heading', level: 2, text: '定位目录', anchor: 'locate', stepNumber: 1 },
    {
      type: 'media',
      id: 'shared-before-branches',
      path: '/img/guides/v2/shared.webp',
      alt: '所有平台共享图片',
    },
    { type: 'heading', level: 3, text: 'Windows', anchor: 'windows', platform: 'Windows' },
    { type: 'paragraph', html: '<p>Windows 专属正文</p>' },
    {
      type: 'media',
      id: 'windows-only',
      path: '/img/guides/v2/windows.webp',
      alt: 'Windows 专属图片',
    },
    { type: 'heading', level: 3, text: 'macOS', anchor: 'macos', platform: 'macOS' },
    { type: 'paragraph', html: '<p>macOS 专属正文</p>' },
  ]),
})

describe('deriveGuideV2Visibility', () => {
  it('让媒体继承结构作用域，只将分支前的共享媒体展示给所有平台', () => {
    const mac = deriveGuideV2Visibility(guide, 'macOS')
    const mediaIds = mac.blocks.flatMap((block) => block.type === 'media' ? [block.id] : [])

    expect(mediaIds).toContain('shared-before-branches')
    expect(mediaIds).not.toContain('windows-only')
    expect(mac.toc.map((item) => item.anchor)).toEqual(['locate', 'macos'])
  })

  it('返回冻结的 null-prototype 平台锚点记录，调用方无法篡改', () => {
    const result = deriveGuideV2Visibility(guide, 'Windows')

    expect(Object.getPrototypeOf(result.platformByAnchor)).toBeNull()
    expect(Object.isFrozen(result.platformByAnchor)).toBe(true)
    expect(result.platformByAnchor.windows).toBe('Windows')
    const mutableRecord = result.platformByAnchor as Record<string, string>
    expect(() => {
      mutableRecord.windows = 'macOS'
    }).toThrow(TypeError)
    expect(result.platformByAnchor.windows).toBe('Windows')
  })
})
