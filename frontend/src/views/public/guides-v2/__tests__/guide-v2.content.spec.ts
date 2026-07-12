import { readFile, readdir, stat } from 'node:fs/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import { parse as parseYaml } from 'yaml'

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

const expectedMedia = [
  'common/setup-flow',
  'common/create-key-group',
  'codex/initialize',
  'codex/config-folder',
  'codex/api-login',
  'claude-code/settings',
  'opencode/config',
  'openclaw/mode-choice',
  'chatbox/add-provider',
  'chatbox/select-model',
  'cherry-studio/add-service',
  'cherry-studio/generate-image',
  'troubleshooting/status-map',
] as const

const testDirectory = dirname(fileURLToPath(import.meta.url))
const frontendDirectory = resolve(testDirectory, '../../../../..')
const contentDirectory = join(frontendDirectory, 'src/content/guides-v2')
const publicDirectory = join(frontendDirectory, 'public')

const splitMarkdown = (source: string) => {
  const match = /^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$/.exec(source)
  if (!match) throw new Error('缺少 frontmatter')
  return { metadata: parseYaml(match[1]), body: match[2] }
}

const readGuides = async () =>
  Promise.all(
    slugs.map(async (slug) => ({
      slug,
      source: await readFile(join(contentDirectory, `${slug}.md`), 'utf8'),
    })),
  )

const readPngDimensions = (buffer: Buffer): readonly [number, number] => {
  expect(buffer.subarray(0, 8)).toEqual(Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]))
  expect(buffer.subarray(12, 16).toString('ascii')).toBe('IHDR')
  return [buffer.readUInt32BE(16), buffer.readUInt32BE(20)]
}

const readWebpDimensions = (buffer: Buffer): readonly [number, number] => {
  expect(buffer.subarray(0, 4).toString('ascii')).toBe('RIFF')
  expect(buffer.subarray(8, 12).toString('ascii')).toBe('WEBP')
  const format = buffer.subarray(12, 16).toString('ascii')
  if (format === 'VP8 ') {
    expect(buffer.subarray(23, 26)).toEqual(Buffer.from([0x9d, 0x01, 0x2a]))
    return [buffer.readUInt16LE(26) & 0x3fff, buffer.readUInt16LE(28) & 0x3fff]
  }
  if (format === 'VP8L') {
    const bits = buffer.readUInt32LE(21)
    return [(bits & 0x3fff) + 1, ((bits >> 14) & 0x3fff) + 1]
  }
  expect(format).toBe('VP8X')
  const width = buffer.readUIntLE(24, 3) + 1
  const height = buffer.readUIntLE(27, 3) + 1
  return [width, height]
}

describe('V2 指南单一源内容', () => {
  it('精确包含固定顺序的 9 篇 Markdown 与合规 frontmatter', async () => {
    const markdownFiles = (await readdir(contentDirectory))
      .filter((name) => name.endsWith('.md'))
      .sort()
    expect(markdownFiles).toEqual(slugs.map((slug) => `${slug}.md`).sort())

    const guides = await readGuides()
    const parsed = guides.map(({ source }) => splitMarkdown(source))
    expect(parsed.map(({ metadata }) => metadata.slug)).toEqual(slugs)
    parsed.forEach(({ metadata }) => {
      expect(metadata).toEqual(
        expect.objectContaining({
          title: expect.any(String),
          slug: expect.any(String),
          summary: expect.any(String),
          duration: expect.any(String),
          difficulty: expect.stringMatching(/^(新手|入门)$/),
          updatedAt: '2026-07-13',
          version: 'v2',
        }),
      )
      expect(metadata.platforms.length).toBeGreaterThan(0)
      expect(
        metadata.platforms.every((platform: string) =>
          ['Windows', 'macOS', 'Linux', 'iOS', 'Android'].includes(platform),
        ),
      ).toBe(true)
    })
  })

  it('每篇只有一个 H1，H2/H3 均有 ASCII 锚点且步骤连续', async () => {
    const guides = await readGuides()
    guides.forEach(({ slug, source }) => {
      const { body } = splitMarkdown(source)
      expect(body.match(/^#\s+.+$/gm), slug).toHaveLength(1)
      const headings = Array.from(body.matchAll(/^(#{2,3})\s+(.+)$/gm))
      expect(headings.length, slug).toBeGreaterThan(0)
      headings.forEach(([, , heading]) => {
        expect(heading, `${slug}: ${heading}`).toMatch(/\s\{#[a-z][a-z0-9-]*\}$/)
      })
      const steps = headings
        .filter(([, hashes, heading]) => hashes === '##' && /^第 \d+ 步：/.test(heading))
        .map(([, , heading]) => Number(/^第 (\d+) 步：/.exec(heading)?.[1]))
      expect(steps, slug).toEqual(steps.map((_number, index) => index + 1))
      expect(steps.length, slug).toBeGreaterThan(0)
    })
  })

  it('图片独立成段、alt 非空，并且全部引用已登记 WebP', async () => {
    const manifest = JSON.parse(
      await readFile(join(contentDirectory, 'media-manifest.json'), 'utf8'),
    ) as {
      readonly media: readonly {
        readonly webPath: string
        readonly caption: string
      }[]
    }
    const registeredMedia = new Map(manifest.media.map((media) => [media.webPath, media]))

    const guides = await readGuides()
    let imageCount = 0
    guides.forEach(({ slug, source }) => {
      const { body } = splitMarkdown(source)
      const images = Array.from(body.matchAll(/!\[([^\]]*)\]\(([^)]+)\)/g))
      imageCount += images.length
      images.forEach((match) => {
        expect(match[1].trim(), slug).not.toBe('')
        const target = /^(\S+)\s+"([^"]+)"$/.exec(match[2])
        expect(target, `${slug}: 图片必须包含非空 title`).not.toBeNull()
        const media = registeredMedia.get(target?.[1] ?? '')
        expect(media, `${slug}: ${target?.[1]}`).toBeDefined()
        expect(target?.[2], `${slug}: caption`).toBe(media?.caption)
        const line = body.slice(0, match.index).split('\n').length
        expect(body.split('\n')[line - 1].trim(), `${slug}:${line}`).toBe(match[0])
      })
    })
    expect(imageCount).toBe(13)
  })

  it('13 组素材均有可编辑 SVG、PNG 和 WebP，签名与尺寸一致', async () => {
    const manifest = JSON.parse(
      await readFile(join(contentDirectory, 'media-manifest.json'), 'utf8'),
    ) as {
      readonly version: string
      readonly media: readonly {
        readonly id: string
        readonly webPath: string
        readonly exportPath: string
        readonly alt: string
        readonly source: string
        readonly width: number
        readonly height: number
        readonly simulated: boolean
        readonly sourceTool: string
      }[]
    }

    expect(manifest.version).toBe('v2')
    expect(manifest.media.map(({ id }) => id)).toEqual(expectedMedia)
    expect(manifest.media).toHaveLength(13)

    await Promise.all(
      manifest.media.map(async (media) => {
        expect(media).toEqual(
          expect.objectContaining({
            alt: expect.stringMatching(/\S/),
            width: 1280,
            height: 720,
            simulated: true,
            sourceTool: 'deterministic-svg',
          }),
        )
        expect(media.webPath).toBe(`/img/guides/v2/${media.id}.webp`)
        expect(media.exportPath).toBe(`frontend/public/img/guides/v2/${media.id}.png`)
        expect(media.source).toBe(`asset-sources/${media.id.replace('/', '-')}.svg`)

        const pngPath = join(frontendDirectory, '..', media.exportPath)
        const webpPath = join(publicDirectory, media.webPath.replace(/^\//, ''))
        const svgPath = join(contentDirectory, media.source)
        await expect(stat(svgPath)).resolves.toMatchObject({ size: expect.any(Number) })
        const [png, webp, svg] = await Promise.all([
          readFile(pngPath),
          readFile(webpPath),
          readFile(svgPath, 'utf8'),
        ])
        expect(svg).toMatch(/<svg[^>]+viewBox="0 0 1280 720"/)
        expect(readPngDimensions(png)).toEqual([media.width, media.height])
        expect(readWebpDimensions(webp)).toEqual([media.width, media.height])
      }),
    )
  })

  it('不包含真实密钥、旧域名、真实账号、余额或用量数据', async () => {
    const guides = await readGuides()
    const sourceNames = expectedMedia.map((id) => `asset-sources/${id.replace('/', '-')}.svg`)
    const sources = await Promise.all([
      ...guides.map(({ source }) => Promise.resolve(source)),
      ...sourceNames.map((name) => readFile(join(contentDirectory, name), 'utf8')),
    ])
    const combined = sources.join('\n').replaceAll('sk-example-not-a-real-key', '')

    expect(combined).not.toMatch(/api\.sakms\.top/i)
    expect(combined).not.toMatch(/sk-[a-z0-9_-]{16,}/i)
    expect(combined).not.toMatch(/[\w.+-]+@[\w.-]+\.[a-z]{2,}/i)
    expect(combined).not.toMatch(/(?:余额|用量|消费)\s*[:：]?\s*[¥￥$]?\d/i)
    expect(combined).not.toMatch(/(?:账号|用户)\s*[:：]\s*[a-z0-9_-]{5,}/i)
  })

  it('易变域名和具体 GPT 模型只出现在允许的集中说明中', async () => {
    const guides = await readGuides()
    guides
      .filter(({ slug }) => slug !== 'get-started')
      .forEach(({ slug, source }) => expect(source, slug).not.toMatch(/sakai\.my/i))
    guides
      .filter(({ slug }) => slug !== 'cherry-studio-image')
      .forEach(({ slug, source }) => expect(source, slug).not.toMatch(/\bgpt-[a-z0-9.-]+/i))

    const sourceNames = expectedMedia.map((id) => `asset-sources/${id.replace('/', '-')}.svg`)
    const sourceEntries = await Promise.all(
      sourceNames.map(async (name) => ({
        name,
        source: await readFile(join(contentDirectory, name), 'utf8'),
      })),
    )
    sourceEntries.forEach(({ name, source }) => expect(source, name).not.toMatch(/sakai\.my/i))
    sourceEntries
      .filter(({ name }) => !name.startsWith('asset-sources/cherry-studio-'))
      .forEach(({ name, source }) => expect(source, name).not.toMatch(/\bgpt-[a-z0-9.-]+/i))
  })

  it('Hub 提供两种起点、六个客户端入口和排错入口', async () => {
    const index = splitMarkdown(await readFile(join(contentDirectory, 'index.md'), 'utf8')).body
    expect(index).toMatch(/从零开始/)
    expect(index).toMatch(/已有 API Key/)
    ;[
      '/guides/v2/codex',
      '/guides/v2/claude-code',
      '/guides/v2/opencode',
      '/guides/v2/openclaw',
      '/guides/v2/chatbox-mobile',
      '/guides/v2/cherry-studio-image',
      '/guides/v2/troubleshooting',
    ].forEach((path) => expect(index).toContain(`](${path})`))
  })

  it('排错页覆盖五类错误，且每个客户端页都链接排错页', async () => {
    const guides = await readGuides()
    const troubleshooting = splitMarkdown(
      guides.find(({ slug }) => slug === 'troubleshooting')?.source ?? '',
    ).body
    ;[/401/, /404/, /429/, /model not found/i, /配置未生效/, /扩展名/, /媒体.*失败/].forEach(
      (pattern) => expect(troubleshooting).toMatch(pattern),
    )

    guides
      .filter(({ slug }) => !['index', 'get-started', 'troubleshooting'].includes(slug))
      .forEach(({ slug, source }) => {
        expect(source, slug).toContain('](/guides/v2/troubleshooting)')
      })
  })
})
