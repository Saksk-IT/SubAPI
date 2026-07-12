import { readFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

import {
  GUIDE_V2_REGISTRY,
  getGuideV2Entry,
  getGuideV2Navigation,
} from '../guide-v2.registry'
import { GUIDE_V2_SLUGS } from '../guide-v2.types'

const testDirectory = dirname(fileURLToPath(import.meta.url))

describe('V2 指南 registry', () => {
  it('按 generated manifest 注册 9 个唯一 slug 和规范路径', () => {
    expect(GUIDE_V2_REGISTRY.map(({ meta }) => meta.slug)).toEqual(GUIDE_V2_SLUGS)
    expect(new Set(GUIDE_V2_REGISTRY.map(({ meta }) => meta.slug))).toHaveProperty('size', 9)
    expect(GUIDE_V2_REGISTRY.map(({ path }) => path)).toEqual([
      '/guides/v2',
      ...GUIDE_V2_SLUGS.slice(1).map((slug) => `/guides/v2/${slug}`),
    ])
    expect(Object.isFrozen(GUIDE_V2_REGISTRY)).toBe(true)
    GUIDE_V2_REGISTRY.forEach((entry) => {
      expect(Object.isFrozen(entry)).toBe(true)
      expect(Object.isFrozen(entry.meta)).toBe(true)
    })
  })

  it('为 9 篇正文使用显式 ?raw 动态 import，不用 eager glob', async () => {
    const source = await readFile(resolve(testDirectory, '../guide-v2.registry.ts'), 'utf8')
    const dynamicImports = Array.from(
      source.matchAll(/import\(['"]([^'"]+\.md\?raw)['"]\)/g),
      (match) => match[1],
    )

    expect(dynamicImports).toHaveLength(9)
    expect(dynamicImports.map((path) => path.split('/').at(-1)?.replace('.md?raw', ''))).toEqual(
      GUIDE_V2_SLUGS,
    )
    expect(source).not.toContain('import.meta.glob')
    expect(source).not.toMatch(/^import\s+.+\.md\?raw/m)
  })

  it('按 slug 查询并仅在调用 load 时返回对应正文', async () => {
    const entry = getGuideV2Entry('codex')
    expect(entry?.meta.slug).toBe('codex')
    expect(entry?.source).toBe('codex.md')
    expect(await entry?.load()).toMatch(/^---[\s\S]+# Codex/m)
    expect(getGuideV2Entry('not-a-guide')).toBeUndefined()
  })

  it('返回不可变的前后篇导航且不修改 registry', () => {
    const before = GUIDE_V2_REGISTRY.map(({ meta }) => meta.slug)
    const middle = getGuideV2Navigation('opencode')
    const first = getGuideV2Navigation('index')
    const last = getGuideV2Navigation('troubleshooting')

    expect(middle.previous?.meta.slug).toBe('claude-code')
    expect(middle.next?.meta.slug).toBe('openclaw')
    expect(first.previous).toBeUndefined()
    expect(first.next?.meta.slug).toBe('get-started')
    expect(last.previous?.meta.slug).toBe('cherry-studio-image')
    expect(last.next).toBeUndefined()
    expect(Object.isFrozen(middle)).toBe(true)
    expect(GUIDE_V2_REGISTRY.map(({ meta }) => meta.slug)).toEqual(before)
    expect(getGuideV2Navigation('not-a-guide')).toBeUndefined()
  })
})
