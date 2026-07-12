import { createHash } from 'node:crypto'
import { mkdir, mkdtemp, readFile, stat, symlink, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

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
const frontendDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '../../../../..')

const sha256 = (value: string | Buffer): string =>
  createHash('sha256').update(value).digest('hex')

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
  readonly assetRoot: string
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
  return {
    contentDir,
    outputPath: join(contentDir, 'manifest.generated.json'),
    assetRoot: contentDir,
  }
}

const createVerifiedMedia = async (
  fixture: Awaited<ReturnType<typeof createFixture>>,
): Promise<{
  readonly entry: Readonly<Record<string, string>>
  readonly pngPath: string
  readonly webpPath: string
}> => {
  const [source, png, webp] = await Promise.all([
    readFile(
      join(frontendDirectory, 'src/content/guides-v2/asset-sources/common-setup-flow.svg'),
    ),
    readFile(join(frontendDirectory, 'public/img/guides/v2/common/setup-flow.png')),
    readFile(join(frontendDirectory, 'public/img/guides/v2/common/setup-flow.webp')),
  ])
  const sourcePath = join(fixture.contentDir, 'sources/install.svg')
  const pngPath = join(fixture.assetRoot, 'frontend/public/img/guides/v2/install.png')
  const webpPath = join(fixture.assetRoot, 'frontend/public/img/guides/v2/install.webp')
  await Promise.all([
    mkdir(join(fixture.contentDir, 'sources'), { recursive: true }),
    mkdir(join(fixture.assetRoot, 'frontend/public/img/guides/v2'), { recursive: true }),
  ])
  await Promise.all([
    writeFile(sourcePath, source),
    writeFile(pngPath, png),
    writeFile(webpPath, webp),
  ])
  return {
    entry: {
      id: 'install',
      webPath: '/img/guides/v2/install.webp',
      exportPath: 'frontend/public/img/guides/v2/install.png',
      alt: '安装页面',
      source: 'sources/install.svg',
      sourceSha256: sha256(source),
      pngSha256: sha256(png),
      webpSha256: sha256(webp),
    },
    pngPath,
    webpPath,
  }
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
    const media = await createVerifiedMedia(fixture)
    await writeFile(
      join(fixture.contentDir, 'media-manifest.json'),
      JSON.stringify({
        version: 'v2',
        media: [media.entry],
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

  it('拒绝与 SVG、PNG 或 WebP 登记哈希不一致的素材', async () => {
    const fixture = await createFixture()
    const media = await createVerifiedMedia(fixture)
    await writeFile(
      join(fixture.contentDir, 'media-manifest.json'),
      JSON.stringify({ version: 'v2', media: [media.entry] }),
      'utf8',
    )

    await expect(buildManifest(fixture)).resolves.toMatchObject({ entries: { length: 9 } })
    await writeFile(media.webpPath, 'tampered-webp-output', 'utf8')

    await expect(buildManifest(fixture)).rejects.toThrow(/media-manifest\.json.*WebP.*SHA-256/i)
  })

  it('替换为另一张有效 PNG 并同步哈希后仍拒绝视觉错配', async () => {
    const fixture = await createFixture()
    const media = await createVerifiedMedia(fixture)
    const replacement = await readFile(
      join(frontendDirectory, 'public/img/guides/v2/common/create-key-group.png'),
    )
    const tamperedEntry = { ...media.entry, pngSha256: sha256(replacement) }
    await writeFile(media.pngPath, replacement)
    await writeFile(
      join(fixture.contentDir, 'media-manifest.json'),
      JSON.stringify({ version: 'v2', media: [tamperedEntry] }),
      'utf8',
    )

    await expect(buildManifest(fixture)).rejects.toThrow(/media-manifest\.json.*PNG.*视觉/i)
  })

  it('拒绝 reference-style link 指向不存在的跨页 fragment', async () => {
    const fixture = await createFixture()
    await writeFile(
      join(fixture.contentDir, 'codex.md'),
      markdownFor(
        'codex',
        '\n[失效锚点][bad]\n\n[bad]: /guides/v2/opencode#missing-anchor\n',
      ),
      'utf8',
    )

    await expect(buildManifest(fixture)).rejects.toThrow(/codex\.md.*fragment.*missing-anchor/i)
  })

  it('拒绝 HTML a 标签中的非法 HTTPS URL', async () => {
    const fixture = await createFixture()
    await writeFile(
      join(fixture.contentDir, 'codex.md'),
      markdownFor('codex', '\n<a href="https://">损坏链接</a>\n'),
      'utf8',
    )

    await expect(buildManifest(fixture)).rejects.toThrow(/codex\.md.*HTTPS.*URL/i)
  })

  it('代码块中的伪 anchor 不能满足跨页 fragment', async () => {
    const fixture = await createFixture()
    await writeFile(
      join(fixture.contentDir, 'opencode.md'),
      markdownFor('opencode', '\n```md\n## 伪标题 {#fake-anchor}\n```\n'),
      'utf8',
    )
    await writeFile(
      join(fixture.contentDir, 'codex.md'),
      markdownFor('codex', '\n[伪锚点](/guides/v2/opencode#fake-anchor)\n'),
      'utf8',
    )

    await expect(buildManifest(fixture)).rejects.toThrow(/codex\.md.*fragment.*fake-anchor/i)
  })

  it('接受 marked heading token 提供的真实跨页 fragment', async () => {
    const fixture = await createFixture()
    await writeFile(
      join(fixture.contentDir, 'codex.md'),
      markdownFor('codex', '\n[有效锚点](/guides/v2/opencode#initialize-opencode)\n'),
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
