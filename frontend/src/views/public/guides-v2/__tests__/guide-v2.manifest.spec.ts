import { mkdtemp, readFile, stat, symlink, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

import { afterEach, describe, expect, it } from 'vitest'

import { buildManifest } from '../../../../../scripts/build-guide-v2-manifest.mjs'

const slugs = [
  'index',
  'get-started',
  'codex',
  'claude-code',
  'opencode',
  'openclaw',
  'chatbox-mobile',
  'cherry-studio-image',
  'troubleshooting',
] as const

const tempDirs: string[] = []

const markdownFor = (slug: (typeof slugs)[number], extra = '') => `---
title: ${slug} 指南
slug: ${slug}
summary: ${slug} 的安全使用说明。
duration: 5 分钟
platforms:
  - Windows
difficulty: 新手
updatedAt: 2026-07-13
version: v2
---
# ${slug} 指南

## 第 1 步：初始化 {#initialize-${slug}}

阅读 [安全站内链接](/guides/v2/codex)。
${extra}
`

const createFixture = async (): Promise<{
  readonly contentDir: string
  readonly outputPath: string
}> => {
  const contentDir = await mkdtemp(join(tmpdir(), 'guide-v2-manifest-'))
  tempDirs.push(contentDir)
  await Promise.all(
    slugs.map((slug) => writeFile(join(contentDir, `${slug}.md`), markdownFor(slug), 'utf8')),
  )
  await writeFile(
    join(contentDir, 'media-manifest.json'),
    JSON.stringify({ version: 'v2', media: [] }, null, 2),
    'utf8',
  )
  return { contentDir, outputPath: join(contentDir, 'manifest.generated.json') }
}

afterEach(async () => {
  await Promise.all(
    tempDirs.splice(0).map(async (path) => {
      const { rm } = await import('node:fs/promises')
      await rm(path, { recursive: true, force: true })
    }),
  )
})

describe('buildManifest', () => {
  it('按固定顺序生成并原子写入 9 条 manifest entries', async () => {
    const fixture = await createFixture()

    const manifest = await buildManifest(fixture)
    const written = JSON.parse(await readFile(fixture.outputPath, 'utf8'))

    expect(manifest).toEqual(written)
    expect(manifest.entries).toHaveLength(9)
    expect(manifest.entries.map((entry) => entry.meta.slug)).toEqual(slugs)
    expect(manifest.entries[0]).toEqual(
      expect.objectContaining({
        meta: expect.objectContaining({ slug: 'index', version: 'v2' }),
        path: '/guides/v2',
        source: 'index.md',
        contentHash: expect.stringMatching(/^[a-f0-9]{64}$/),
      }),
    )
    await expect(stat(`${fixture.outputPath}.tmp`)).rejects.toMatchObject({ code: 'ENOENT' })
  })

  it('--check 缺失或过期时报错且绝不写入', async () => {
    const fixture = await createFixture()

    await expect(buildManifest({ ...fixture, check: true })).rejects.toThrow(/--check.*缺失/i)
    await expect(stat(fixture.outputPath)).rejects.toMatchObject({ code: 'ENOENT' })

    await buildManifest(fixture)
    const previousOutput = await readFile(fixture.outputPath, 'utf8')
    await writeFile(join(fixture.contentDir, 'codex.md'), markdownFor('codex', '\n新内容。'), 'utf8')

    await expect(buildManifest({ ...fixture, check: true })).rejects.toThrow(/--check.*过期/i)
    expect(await readFile(fixture.outputPath, 'utf8')).toBe(previousOutput)
  })

  it('精确要求 9 个指定 md 文件', async () => {
    const fixture = await createFixture()
    const { rm } = await import('node:fs/promises')
    await rm(join(fixture.contentDir, 'openclaw.md'))
    await writeFile(join(fixture.contentDir, 'extra.md'), markdownFor('openclaw'), 'utf8')

    await expect(buildManifest(fixture)).rejects.toThrow(/openclaw\.md.*extra\.md/i)
  })

  it.each([
    ['多个 H1', '\n# 第二个 H1\n', /codex\.md.*H1/i],
    ['重复锚点', '\n## 重复 {#initialize-codex}\n', /codex\.md.*锚点/i],
    ['危险链接', '\n[x](javascript:alert(1))\n', /codex\.md.*危险协议/i],
    ['非法站内链接', '\n[x](/guides/v2/unknown)\n', /codex\.md.*非法内部链接/i],
    ['HTML 非安全外链', '\n<a href="http://example.com">x</a>\n', /codex\.md.*非安全外链/i],
    ['HTML 非法站内链接', '\n<a href="\/guides\/v2\/unknown">x<\/a>\n', /codex\.md.*非法内部链接/i],
  ])('拒绝%s，并指明具体文件', async (_label, extra, expected) => {
    const fixture = await createFixture()
    await writeFile(join(fixture.contentDir, 'codex.md'), markdownFor('codex', extra), 'utf8')

    await expect(buildManifest(fixture)).rejects.toThrow(expected)
  })

  it('校验 media manifest 登记，并拒绝绝对/穿越 source', async () => {
    const fixture = await createFixture()
    await writeFile(
      join(fixture.contentDir, 'codex.md'),
      markdownFor('codex', '\n![未登记](/img/guides/v2/missing.webp)\n'),
      'utf8',
    )
    await expect(buildManifest(fixture)).rejects.toThrow(/codex\.md.*媒体/i)

    await writeFile(join(fixture.contentDir, 'codex.md'), markdownFor('codex'), 'utf8')
    await writeFile(
      join(fixture.contentDir, 'media-manifest.json'),
      JSON.stringify({
        version: 'v2',
        media: [{ id: 'bad', webPath: '/img/guides/v2/bad.webp', exportPath: 'public/img/guides/v2/bad.webp', alt: 'bad', source: '../bad.png' }],
      }),
      'utf8',
    )
    await expect(buildManifest(fixture)).rejects.toThrow(/media-manifest\.json.*source.*穿越/i)

    await writeFile(
      join(fixture.contentDir, 'media-manifest.json'),
      JSON.stringify({
        version: 'v2',
        media: [{ id: 'bad', webPath: '/img/guides/v2/bad.webp', exportPath: 'public/img/guides/v2/bad.webp', alt: 'bad', source: '/tmp/bad.png' }],
      }),
      'utf8',
    )
    await expect(buildManifest(fixture)).rejects.toThrow(/media-manifest\.json.*source.*绝对/i)
  })

  it('接受 Task 3 的 webPath/exportPath 媒体契约', async () => {
    const fixture = await createFixture()
    await writeFile(
      join(fixture.contentDir, 'media-manifest.json'),
      JSON.stringify({
        version: 'v2',
        media: [{
          id: 'install',
          webPath: '/img/guides/v2/install.webp',
          exportPath: 'public/img/guides/v2/install.webp',
          alt: '安装页面',
          source: 'sources/install.png',
        }],
      }),
      'utf8',
    )
    await writeFile(
      join(fixture.contentDir, 'codex.md'),
      markdownFor('codex', '\n![安装页面](/img/guides/v2/install.webp)\n'),
      'utf8',
    )

    await expect(buildManifest(fixture)).resolves.toMatchObject({ entries: { length: 9 } })
  })

  it('允许协议大小写不同的 HTTPS 外链', async () => {
    const fixture = await createFixture()
    await writeFile(
      join(fixture.contentDir, 'codex.md'),
      markdownFor('codex', '\n[Docs](HTTPS://example.com)\n'),
      'utf8',
    )

    await expect(buildManifest(fixture)).resolves.toMatchObject({ entries: { length: 9 } })
  })

  it.each([
    ['Windows 反斜杠绝对路径', String.raw`C:\secrets\image.png`],
    ['Windows 正斜杠绝对路径', 'C:/secrets/image.png'],
    ['Windows drive-relative 路径', 'C:secrets/image.png'],
    ['UNC 路径', String.raw`\\server\share\image.png`],
  ])('跨平台拒绝%s', async (_label, invalidSource) => {
    const fixture = await createFixture()
    await writeFile(
      join(fixture.contentDir, 'media-manifest.json'),
      JSON.stringify({
        version: 'v2',
        media: [{
          id: 'unsafe-source',
          webPath: '/img/guides/v2/image.webp',
          exportPath: 'public/img/guides/v2/image.webp',
          alt: '图片',
          source: invalidSource,
        }],
      }),
      'utf8',
    )

    await expect(buildManifest(fixture)).rejects.toThrow(
      /media-manifest\.json.*source.*(?:绝对|盘符|UNC)/i,
    )
  })

  it('拒绝 guide Markdown 符号链接逃逸 contentDir', async () => {
    const fixture = await createFixture()
    const outsideDir = await mkdtemp(join(tmpdir(), 'guide-v2-outside-'))
    tempDirs.push(outsideDir)
    const outsideGuide = join(outsideDir, 'codex.md')
    await writeFile(outsideGuide, markdownFor('codex'), 'utf8')
    const { rm } = await import('node:fs/promises')
    await rm(join(fixture.contentDir, 'codex.md'))
    await symlink(outsideGuide, join(fixture.contentDir, 'codex.md'))

    await expect(buildManifest(fixture)).rejects.toThrow(/codex\.md.*符号链接/i)
  })

  it('拒绝 media-manifest.json 符号链接逃逸 contentDir', async () => {
    const fixture = await createFixture()
    const outsideDir = await mkdtemp(join(tmpdir(), 'guide-v2-media-outside-'))
    tempDirs.push(outsideDir)
    const outsideManifest = join(outsideDir, 'media-manifest.json')
    await writeFile(
      outsideManifest,
      JSON.stringify({ version: 'v2', media: [] }),
      'utf8',
    )
    const { rm } = await import('node:fs/promises')
    await rm(join(fixture.contentDir, 'media-manifest.json'))
    await symlink(outsideManifest, join(fixture.contentDir, 'media-manifest.json'))

    await expect(buildManifest(fixture)).rejects.toThrow(
      /media-manifest\.json.*符号链接/i,
    )
  })

  it('拒绝通过符号链接输出父目录写出 contentDir', async () => {
    const fixture = await createFixture()
    const outsideDir = await mkdtemp(join(tmpdir(), 'guide-v2-output-outside-'))
    tempDirs.push(outsideDir)
    const linkedOutputDir = join(fixture.contentDir, 'linked-output')
    const escapedOutput = join(outsideDir, 'manifest.generated.json')
    await symlink(outsideDir, linkedOutputDir, 'dir')

    await expect(
      buildManifest({
        contentDir: fixture.contentDir,
        outputPath: join(linkedOutputDir, 'manifest.generated.json'),
      }),
    ).rejects.toThrow(/linked-output.*符号链接/i)
    await expect(stat(escapedOutput)).rejects.toMatchObject({ code: 'ENOENT' })
  })
})
