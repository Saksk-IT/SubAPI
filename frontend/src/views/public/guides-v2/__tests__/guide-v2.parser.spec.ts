import { describe, expect, it } from 'vitest'

import { parseGuideMarkdown } from '../guide-v2.parser'

const metadata = Object.freeze({
  title: '快速开始',
  slug: 'get-started',
  summary: '用最少步骤完成配置。',
  duration: '5 分钟',
  platforms: Object.freeze(['Windows', 'macOS']),
  difficulty: '新手',
  updatedAt: '2026-07-13',
  version: 'v2',
})

const media = Object.freeze([
  Object.freeze({
    id: 'install-screen',
    webPath: '/img/guides/v2/install-screen.webp',
    exportPath: 'public/img/guides/v2/install-screen.webp',
    alt: '安装页面',
  }),
])

const validBody = `# 快速开始

这是 **安全** 段落，可查看 [Codex 指南](/guides/v2/codex)。

## 第 1 步：初始化 {#initialize}

### Windows {#windows}

运行安装命令。

\`\`\`bash
pnpm install
\`\`\`

![安装页面](/img/guides/v2/install-screen.webp)

## 常见问题 {#faq}

### macOS {#macos}

| 项目 | 值 |
| --- | --- |
| 版本 | v2 |
`

describe('parseGuideMarkdown', () => {
  it('生成稳定的步骤、平台、TOC 和结构化代码/媒体块', () => {
    const result = parseGuideMarkdown({
      sourceName: 'get-started.md',
      body: validBody,
      metadata,
      media,
    })

    expect(result.meta.slug).toBe('get-started')
    expect(result.steps).toEqual([
      expect.objectContaining({ number: 1, title: '初始化', anchor: 'initialize' }),
    ])
    expect(result.platforms).toEqual(['Windows', 'macOS'])
    expect(result.toc.map((item) => item.anchor)).toEqual([
      'initialize',
      'windows',
      'faq',
      'macos',
    ])
    expect(result.blocks).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ type: 'code', language: 'bash', code: 'pnpm install' }),
        expect.objectContaining({
          type: 'media',
          path: '/img/guides/v2/install-screen.webp',
        }),
        expect.objectContaining({
          type: 'heading',
          platform: 'Windows',
          anchor: 'windows',
        }),
      ]),
    )
  })

  it('不原地修改输入，并返回递归冻结的结果', () => {
    const input = Object.freeze({
      sourceName: 'get-started.md',
      body: validBody,
      metadata,
      media,
    })

    const result = parseGuideMarkdown(input)

    expect(result.meta).not.toBe(metadata)
    expect(Object.isFrozen(result)).toBe(true)
    expect(Object.isFrozen(result.blocks)).toBe(true)
    expect(Object.isFrozen(result.steps[0])).toBe(true)
    expect(metadata.platforms).toEqual(['Windows', 'macOS'])
  })

  it.each([
    ['缺少字段', { ...metadata, summary: undefined }],
    ['非法平台', { ...metadata, platforms: ['Windows', 'BeOS'] }],
    ['错误日期', { ...metadata, updatedAt: '2026-02-30' }],
    ['错误版本', { ...metadata, version: 'v1' }],
    ['非法 slug', { ...metadata, slug: 'unknown' }],
  ])('拒绝%s并在错误中包含 sourceName', (_label, invalidMetadata) => {
    expect(() =>
      parseGuideMarkdown({
        sourceName: 'broken.md',
        body: validBody,
        metadata: invalidMetadata,
        media,
      }),
    ).toThrow(/broken\.md/)
  })

  it('拒绝重复锚点和格式错误的步骤标题', () => {
    const duplicateAnchor = `${validBody}\n## 其他 {#initialize}\n`
    const malformedStep = validBody.replace('第 1 步：初始化 {#initialize}', '第一步：初始化')

    expect(() =>
      parseGuideMarkdown({ sourceName: 'duplicate.md', body: duplicateAnchor, metadata, media }),
    ).toThrow(/duplicate\.md.*锚点/i)
    expect(() =>
      parseGuideMarkdown({ sourceName: 'step.md', body: malformedStep, metadata, media }),
    ).toThrow(/step\.md.*步骤/i)
  })

  it('将普通 H3 锚点加入 TOC', () => {
    const result = parseGuideMarkdown({
      sourceName: 'ordinary-h3.md',
      body: `${validBody}\n### 补充说明 {#notes}\n\n补充内容。\n`,
      metadata,
      media,
    })

    expect(result.toc).toContainEqual({ level: 3, title: '补充说明', anchor: 'notes' })
  })

  it('拒绝两个普通 H3 使用同一锚点', () => {
    const body = `${validBody}\n### 补充说明 {#notes}\n\n### 附录 {#notes}\n`

    expect(() =>
      parseGuideMarkdown({ sourceName: 'duplicate-h3.md', body, metadata, media }),
    ).toThrow(/duplicate-h3\.md.*重复锚点/i)
  })

  it.each([
    ['[x](javascript:alert(1))', '危险协议'],
    ['[x](data:text/html,<script>alert(1)</script>)', '危险协议'],
    ['[x](http://example.com)', '非安全外链'],
    ['[x](/guides/v2/not-found)', '非法内部链接'],
    ['<a href="http://example.com">x</a>', '非安全外链'],
    ['<a href="/guides/v2/not-found">x</a>', '非法内部链接'],
    ['<script>alert(1)</script>', 'script'],
    ['<span onclick="alert(1)">x</span>', '事件属性'],
  ])('拒绝%s', (fragment, expectedMessage) => {
    expect(() =>
      parseGuideMarkdown({
        sourceName: 'unsafe.md',
        body: `${validBody}\n${fragment}`,
        metadata,
        media,
      }),
    ).toThrow(new RegExp(`unsafe\\.md.*${expectedMessage}`, 'i'))
  })

  it('允许 https 外链，并用白名单净化段落和表格 HTML', () => {
    const body = `${validBody}\n\n<a href="https://example.com" class="safe" style="color:red">Docs</a>`
    const result = parseGuideMarkdown({ sourceName: 'safe.md', body, metadata, media })
    const renderedHtml = result.blocks
      .filter((block) => block.type === 'paragraph' || block.type === 'table')
      .map((block) => block.html)
      .join('')

    expect(renderedHtml).toContain('https://example.com')
    expect(renderedHtml).not.toContain('style=')
  })

  it('允许协议大小写不同的 HTTPS 外链', () => {
    const result = parseGuideMarkdown({
      sourceName: 'uppercase-https.md',
      body: `${validBody}\n\n[Docs](HTTPS://example.com)`,
      metadata,
      media,
    })
    const renderedHtml = result.blocks
      .filter((block) => block.type === 'paragraph')
      .map((block) => block.html)
      .join('')

    expect(renderedHtml).toContain('HTTPS://example.com')
  })

  it('保留同一段落中媒体前后的文字、链接和顺序', () => {
    const mixedParagraph =
      '前文 ![安装页面](/img/guides/v2/install-screen.webp) 后文 [Docs](https://example.com)'
    const result = parseGuideMarkdown({
      sourceName: 'mixed.md',
      body: `${validBody}\n\n${mixedParagraph}`,
      metadata,
      media,
    })
    const mixedBlocks = result.blocks.filter(
      (block) =>
        block.type === 'media' ||
        (block.type === 'paragraph' && (block.html.includes('前文') || block.html.includes('后文'))),
    )

    expect(mixedBlocks.map((block) => block.type)).toEqual(['media', 'paragraph', 'media', 'paragraph'])
    expect(mixedBlocks[1]).toMatchObject({ type: 'paragraph', html: expect.stringContaining('前文') })
    expect(mixedBlocks[3]).toMatchObject({ type: 'paragraph', html: expect.stringContaining('https://example.com') })
  })

  it('将链接和强调文本中的嵌套图片转换为结构化媒体块', () => {
    const linkedImageResult = parseGuideMarkdown({
      sourceName: 'nested-media.md',
      body: `${validBody}\n\n[![链接图片](/img/guides/v2/install-screen.webp)](https://example.com)`,
      metadata,
      media,
    })
    expect(
      linkedImageResult.blocks.filter(
        (block) => block.type === 'media' && block.alt === '链接图片',
      ),
    ).toEqual([
      expect.objectContaining({ type: 'media', path: '/img/guides/v2/install-screen.webp' }),
    ])

    const emphasizedImageResult = parseGuideMarkdown({
      sourceName: 'nested-media.md',
      body: `${validBody}\n\n**前 ![强调图片](/img/guides/v2/install-screen.webp) 后**`,
      metadata,
      media,
    })
    const emphasizedSequence = emphasizedImageResult.blocks.filter(
      (block) =>
        (block.type === 'paragraph' && (block.html.includes('前') || block.html.includes('后'))) ||
        (block.type === 'media' && block.alt === '强调图片'),
    )

    expect(emphasizedSequence.map((block) => block.type)).toEqual([
      'paragraph',
      'media',
      'paragraph',
    ])
    expect(emphasizedSequence[0]).toMatchObject({
      type: 'paragraph',
      html: expect.stringMatching(/<strong>前\s*<\/strong>/),
    })
    expect(emphasizedSequence[2]).toMatchObject({
      type: 'paragraph',
      html: expect.stringMatching(/<strong>\s*后<\/strong>/),
    })
  })

  it('拒绝未在 media manifest 登记的媒体', () => {
    expect(() =>
      parseGuideMarkdown({
        sourceName: 'missing-media.md',
        body: validBody.replace('install-screen.webp', 'missing.webp'),
        metadata,
        media,
      }),
    ).toThrow(/missing-media\.md.*媒体/i)
  })
})
