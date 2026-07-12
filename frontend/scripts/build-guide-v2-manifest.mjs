import { createHash, randomUUID } from 'node:crypto'
import { lstat, mkdir, readFile, readdir, realpath, rename, rm, writeFile } from 'node:fs/promises'
import { dirname, isAbsolute, join, relative, resolve, sep, win32 } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { marked } from 'marked'
import sharp from 'sharp'
import { parse as parseYaml } from 'yaml'

const GUIDE_SLUGS = Object.freeze([
  'index',
  'get-started',
  'codex',
  'claude-code',
  'opencode',
  'openclaw',
  'chatbox-mobile',
  'cherry-studio-image',
  'troubleshooting',
])
const GUIDE_FILES = Object.freeze(GUIDE_SLUGS.map((slug) => `${slug}.md`))
const SUPPORTED_PLATFORMS = Object.freeze(['Windows', 'macOS', 'Linux', 'iOS', 'Android'])
const ALLOWED_INTERNAL_PATHS = new Set([
  '/guides/v2',
  ...GUIDE_SLUGS.filter((slug) => slug !== 'index').map((slug) => `/guides/v2/${slug}`),
])
const FRONTMATTER_PATTERN = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?/
const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/
const SHA256_PATTERN = /^[a-f0-9]{64}$/
const HEADING_ANCHOR_PATTERN = /\s+\{#([a-z][a-z0-9-]*)\}$/
const DANGEROUS_PROTOCOL_PATTERN = /^(?:javascript|data|vbscript):/i
const EVENT_ATTRIBUTE_PATTERN = /\son[a-z0-9_-]+\s*=/i
const SCRIPT_ELEMENT_PATTERN = /<\s*\/?\s*script\b/i
const VISUAL_WIDTH = 128
const VISUAL_HEIGHT = 72
const MAX_VISUAL_MAE = 4
const MIN_INK_JACCARD = 0.6
const INK_LUMINANCE_THRESHOLD = 246

const scriptDirectory = dirname(fileURLToPath(import.meta.url))
const defaultContentDir = join(scriptDirectory, '../src/content/guides-v2')
const defaultAssetRoot = resolve(scriptDirectory, '../..')

const fail = (source, message) => {
  throw new Error(`${source}: ${message}`)
}

const isRecord = (value) => typeof value === 'object' && value !== null && !Array.isArray(value)

const isRealDate = (value) => {
  if (typeof value !== 'string' || !DATE_PATTERN.test(value)) return false
  const [year, month, day] = value.split('-').map(Number)
  const date = new Date(Date.UTC(year, month - 1, day))
  return (
    date.getUTCFullYear() === year &&
    date.getUTCMonth() === month - 1 &&
    date.getUTCDate() === day
  )
}

const requiredString = (metadata, field, source) => {
  const value = metadata[field]
  if (typeof value !== 'string' || value.trim() === '') {
    return fail(source, `缺少或非法字段 ${field}`)
  }
  return value
}

const requiredSha256 = (entry, field, source) => {
  const value = requiredString(entry, field, source)
  if (!SHA256_PATTERN.test(value)) return fail(source, `${field} 必须是小写 SHA-256`)
  return value
}

const validateMetadata = (rawMetadata, expectedSlug, source) => {
  if (!isRecord(rawMetadata)) return fail(source, 'Frontmatter 必须是对象')
  const title = requiredString(rawMetadata, 'title', source)
  const slug = requiredString(rawMetadata, 'slug', source)
  const summary = requiredString(rawMetadata, 'summary', source)
  const duration = requiredString(rawMetadata, 'duration', source)
  const difficulty = requiredString(rawMetadata, 'difficulty', source)
  const updatedAt = requiredString(rawMetadata, 'updatedAt', source)
  const version = requiredString(rawMetadata, 'version', source)

  if (slug !== expectedSlug) return fail(source, `slug 必须是 ${expectedSlug}，收到 ${slug}`)
  if (difficulty !== '新手' && difficulty !== '入门') {
    return fail(source, `非法难度: ${difficulty}`)
  }
  if (!isRealDate(updatedAt)) return fail(source, `错误日期: ${updatedAt}`)
  if (version !== 'v2') return fail(source, `版本必须是 v2，收到 ${version}`)
  if (!Array.isArray(rawMetadata.platforms) || rawMetadata.platforms.length === 0) {
    return fail(source, '缺少或非法字段 platforms')
  }
  const platforms = rawMetadata.platforms.map((platform) => {
    if (typeof platform !== 'string' || !SUPPORTED_PLATFORMS.includes(platform)) {
      return fail(source, `非法平台: ${String(platform)}`)
    }
    return platform
  })

  if (Object.hasOwn(rawMetadata, 'source')) validateRelativeSource(rawMetadata.source, source)

  return { title, slug, summary, duration, platforms, difficulty, updatedAt, version }
}

const splitFrontmatter = (sourceText, source) => {
  const match = FRONTMATTER_PATTERN.exec(sourceText)
  if (!match) return fail(source, '缺少有效 Frontmatter')
  try {
    return { metadata: parseYaml(match[1]), body: sourceText.slice(match[0].length) }
  } catch (error) {
    return fail(source, `Frontmatter YAML 解析失败: ${error.message}`)
  }
}

const validateRelativePath = (pathValue, field, sourceName) => {
  if (typeof pathValue !== 'string' || pathValue.trim() === '') {
    return fail(sourceName, `${field} 必须是非空相对路径`)
  }
  if (/^(?:\\\\|\/\/)/.test(pathValue)) {
    return fail(sourceName, `${field} 拒绝 UNC 路径: ${pathValue}`)
  }
  if (/^[a-z]:/i.test(pathValue)) {
    return fail(sourceName, `${field} 拒绝 Windows 盘符路径: ${pathValue}`)
  }
  if (isAbsolute(pathValue) || win32.isAbsolute(pathValue)) {
    return fail(sourceName, `${field} 拒绝绝对路径: ${pathValue}`)
  }
  if (pathValue.split(/[\\/]+/).includes('..')) {
    return fail(sourceName, `${field} 拒绝目录穿越: ${pathValue}`)
  }
  return pathValue
}

const validateRelativeSource = (sourceValue, sourceName) =>
  validateRelativePath(sourceValue, 'source', sourceName)

const assertInsideContentDir = (contentDirReal, targetReal, source, description) => {
  const relativePath = relative(contentDirReal, targetReal)
  if (relativePath === '..' || relativePath.startsWith(`..${sep}`) || isAbsolute(relativePath)) {
    return fail(source, `${description}的真实路径逃逸 contentDir: ${targetReal}`)
  }
}

const assertSafeInputFile = async (contentDirReal, absolutePath, source) => {
  let fileStat
  try {
    fileStat = await lstat(absolutePath)
  } catch (error) {
    return fail(source, `无法检查输入文件: ${error.message}`)
  }
  if (fileStat.isSymbolicLink()) return fail(source, `拒绝符号链接: ${absolutePath}`)
  if (!fileStat.isFile()) return fail(source, `输入必须是普通文件: ${absolutePath}`)
  const fileReal = await realpath(absolutePath)
  assertInsideContentDir(contentDirReal, fileReal, source, '输入文件')
  return fileReal
}

const assertSafeMediaFile = async (assetRootReal, absolutePath, source, label) => {
  let fileStat
  try {
    fileStat = await lstat(absolutePath)
  } catch (error) {
    return fail(source, `无法检查 ${label}: ${error.message}`)
  }
  if (fileStat.isSymbolicLink()) return fail(source, `${label} 拒绝符号链接: ${absolutePath}`)
  if (!fileStat.isFile()) return fail(source, `${label} 必须是普通文件: ${absolutePath}`)
  const fileReal = await realpath(absolutePath)
  assertInsideContentDir(assetRootReal, fileReal, source, label)
  return fileReal
}

const fileSha256 = async (path) =>
  createHash('sha256').update(await readFile(path)).digest('hex')

const normalizedVisualPixels = async (path) =>
  sharp(path, { density: 96 })
    .flatten({ background: '#ffffff' })
    .resize(VISUAL_WIDTH, VISUAL_HEIGHT, { fit: 'fill', kernel: 'lanczos3' })
    .blur(0.6)
    .toColourspace('srgb')
    .removeAlpha()
    .raw()
    .toBuffer()

const visualDifference = (sourcePixels, outputPixels) => {
  if (sourcePixels.length !== outputPixels.length || sourcePixels.length % 3 !== 0) {
    return { meanAbsoluteError: Number.POSITIVE_INFINITY, inkJaccard: 0 }
  }
  let absoluteDifference = 0
  let inkIntersection = 0
  let inkUnion = 0
  for (let index = 0; index < sourcePixels.length; index += 3) {
    for (let channel = 0; channel < 3; channel += 1) {
      absoluteDifference += Math.abs(sourcePixels[index + channel] - outputPixels[index + channel])
    }
    const sourceLuminance =
      (sourcePixels[index] + sourcePixels[index + 1] + sourcePixels[index + 2]) / 3
    const outputLuminance =
      (outputPixels[index] + outputPixels[index + 1] + outputPixels[index + 2]) / 3
    const sourceInk = sourceLuminance < INK_LUMINANCE_THRESHOLD
    const outputInk = outputLuminance < INK_LUMINANCE_THRESHOLD
    if (sourceInk && outputInk) inkIntersection += 1
    if (sourceInk || outputInk) inkUnion += 1
  }
  return {
    meanAbsoluteError: absoluteDifference / sourcePixels.length,
    inkJaccard: inkUnion === 0 ? 1 : inkIntersection / inkUnion,
  }
}

const assertVisualMatch = async (sourceFile, outputFile, source, label) => {
  let sourcePixels
  let outputPixels
  try {
    const pixels = await Promise.all([
      normalizedVisualPixels(sourceFile),
      normalizedVisualPixels(outputFile),
    ])
    sourcePixels = pixels[0]
    outputPixels = pixels[1]
  } catch (error) {
    return fail(source, `${label} 无法进行跨平台视觉解码: ${error.message}`)
  }
  const { meanAbsoluteError, inkJaccard } = visualDifference(sourcePixels, outputPixels)
  if (meanAbsoluteError > MAX_VISUAL_MAE || inkJaccard < MIN_INK_JACCARD) {
    return fail(
      source,
      `${label} 视觉内容与 SVG source 不一致（MAE=${meanAbsoluteError.toFixed(3)}, ink=${inkJaccard.toFixed(3)}）`,
    )
  }
}

const inspectOutputSegments = async (currentPath, segments, source) => {
  if (segments.length === 0) return
  const [segment, ...remaining] = segments
  const nextPath = join(currentPath, segment)
  let pathStat
  try {
    pathStat = await lstat(nextPath)
  } catch (error) {
    if (error.code === 'ENOENT') return
    return fail(source, `无法检查输出父目录: ${error.message}`)
  }
  if (pathStat.isSymbolicLink()) return fail(source, `输出父目录包含符号链接: ${nextPath}`)
  if (!pathStat.isDirectory()) return fail(source, `输出父路径不是目录: ${nextPath}`)
  return inspectOutputSegments(nextPath, remaining, source)
}

const assertSafeOutputPath = async ({ contentDir, contentDirReal, outputPath, createParent }) => {
  const source = outputPath
  const contentDirAbsolute = resolve(contentDir)
  const outputAbsolute = resolve(outputPath)
  const outputParent = dirname(outputAbsolute)
  const relativeParent = relative(contentDirAbsolute, outputParent)
  if (
    relativeParent === '..' ||
    relativeParent.startsWith(`..${sep}`) ||
    isAbsolute(relativeParent)
  ) {
    return fail(source, '输出父目录必须位于 contentDir 内')
  }
  const segments = relativeParent === '' ? [] : relativeParent.split(sep)
  await inspectOutputSegments(contentDirAbsolute, segments, source)
  if (createParent) await mkdir(outputParent, { recursive: true })

  let outputParentReal
  try {
    outputParentReal = await realpath(outputParent)
  } catch (error) {
    if (!createParent && error.code === 'ENOENT') return
    return fail(source, `无法解析输出父目录: ${error.message}`)
  }
  assertInsideContentDir(contentDirReal, outputParentReal, source, '输出父目录')

  try {
    const outputStat = await lstat(outputAbsolute)
    if (outputStat.isSymbolicLink()) return fail(source, `拒绝输出符号链接: ${outputAbsolute}`)
    if (!outputStat.isFile()) return fail(source, `输出必须是普通文件: ${outputAbsolute}`)
  } catch (error) {
    if (error.code !== 'ENOENT') throw error
  }
}

const loadMediaManifest = async (contentDir, contentDirReal, assetRoot, assetRootReal) => {
  const source = 'media-manifest.json'
  let parsed
  try {
    const manifestPath = join(contentDir, source)
    await assertSafeInputFile(contentDirReal, manifestPath, source)
    parsed = JSON.parse(await readFile(manifestPath, 'utf8'))
  } catch (error) {
    return fail(source, `读取或解析失败: ${error.message}`)
  }
  if (!isRecord(parsed) || parsed.version !== 'v2' || !Array.isArray(parsed.media)) {
    return fail(source, '必须包含 version=v2 和 media 数组')
  }

  const media = await Promise.all(parsed.media.map(async (entry, index) => {
    if (!isRecord(entry)) return fail(source, `media[${index}] 必须是对象`)
    const id = requiredString(entry, 'id', source)
    const webPath = requiredString(entry, 'webPath', source)
    const exportPath = validateRelativePath(entry.exportPath, 'exportPath', source)
    const alt = requiredString(entry, 'alt', source)
    const parsedWebPath = new URL(webPath, 'https://guides.local')
    if (
      !webPath.startsWith('/img/guides/v2/') ||
      parsedWebPath.pathname !== webPath ||
      parsedWebPath.search !== '' ||
      parsedWebPath.hash !== ''
    ) {
      return fail(source, `media[${index}] webPath 必须位于 /img/guides/v2/: ${webPath}`)
    }
    const sourcePath = Object.hasOwn(entry, 'source')
      ? validateRelativeSource(entry.source, source)
      : undefined
    if (!sourcePath) return fail(source, `media[${index}] 缺少 source`)
    if (
      !exportPath.startsWith('frontend/public/img/guides/v2/') ||
      !exportPath.endsWith('.png')
    ) {
      return fail(source, `media[${index}] exportPath 必须是 V2 PNG: ${exportPath}`)
    }
    if (!webPath.endsWith('.webp')) {
      return fail(source, `media[${index}] webPath 必须是 WebP: ${webPath}`)
    }
    const sourceSha256 = requiredSha256(entry, 'sourceSha256', source)
    const pngSha256 = requiredSha256(entry, 'pngSha256', source)
    const webpSha256 = requiredSha256(entry, 'webpSha256', source)
    const sourceFile = await assertSafeInputFile(
      contentDirReal,
      join(contentDir, sourcePath),
      source,
    )
    const pngFile = await assertSafeMediaFile(
      assetRootReal,
      resolve(assetRoot, exportPath),
      source,
      `media[${index}] PNG`,
    )
    const webpFile = await assertSafeMediaFile(
      assetRootReal,
      resolve(assetRoot, 'frontend/public', webPath.slice(1)),
      source,
      `media[${index}] WebP`,
    )
    const [actualSourceSha256, actualPngSha256, actualWebpSha256] = await Promise.all([
      fileSha256(sourceFile),
      fileSha256(pngFile),
      fileSha256(webpFile),
    ])
    if (actualSourceSha256 !== sourceSha256) {
      return fail(source, `media[${index}] SVG source SHA-256 不匹配`)
    }
    if (actualPngSha256 !== pngSha256) {
      return fail(source, `media[${index}] PNG SHA-256 不匹配`)
    }
    if (actualWebpSha256 !== webpSha256) {
      return fail(source, `media[${index}] WebP SHA-256 不匹配`)
    }
    await Promise.all([
      assertVisualMatch(sourceFile, pngFile, source, `media[${index}] PNG`),
      assertVisualMatch(sourceFile, webpFile, source, `media[${index}] WebP`),
    ])
    return { id, webPath, exportPath, alt, source: sourcePath }
  }))
  const uniqueIds = new Set(media.map((entry) => entry.id))
  const uniquePaths = new Set(media.map((entry) => entry.webPath))
  if (uniqueIds.size !== media.length) return fail(source, '存在重复 media id')
  if (uniquePaths.size !== media.length) return fail(source, '存在重复 media path')
  return media
}

const nestedTokens = (token) => {
  if (Array.isArray(token.tokens)) return token.tokens
  if (token.type === 'table') {
    return [
      ...token.header.flatMap((cell) => cell.tokens),
      ...token.rows.flatMap((row) => row.flatMap((cell) => cell.tokens)),
    ]
  }
  if (token.type === 'list') return token.items.flatMap((item) => item.tokens)
  return []
}

const walkTokens = (tokens) =>
  tokens.flatMap((token) => [token, ...walkTokens(nestedTokens(token))])

const validateInternalFragment = (url, source, currentPath, anchorsByPath) => {
  if (url.hash === '') return
  let fragment
  try {
    fragment = decodeURIComponent(url.hash.slice(1))
  } catch {
    return fail(source, `fragment 编码非法: ${url.hash}`)
  }
  const targetPath = url.pathname === '/' ? currentPath : url.pathname
  const targetAnchors = anchorsByPath.get(targetPath)
  if (!targetAnchors?.has(fragment)) {
    return fail(source, `fragment 不存在: ${targetPath}#${fragment}`)
  }
}

const validateLink = (href, source, currentPath, anchorsByPath) => {
  const normalized = href.trim()
  if (DANGEROUS_PROTOCOL_PATTERN.test(normalized)) {
    return fail(source, `危险协议链接: ${href}`)
  }
  if (normalized.startsWith('#')) {
    const url = new URL(normalized, `https://guides.local${currentPath}`)
    return validateInternalFragment(url, source, currentPath, anchorsByPath)
  }
  if (normalized.startsWith('/guides/v2')) {
    let url
    try {
      url = new URL(normalized, 'https://guides.local')
    } catch {
      return fail(source, `非法内部 URL: ${href}`)
    }
    if (!ALLOWED_INTERNAL_PATHS.has(url.pathname) || url.search !== '') {
      return fail(source, `非法内部链接: ${href}`)
    }
    return validateInternalFragment(url, source, currentPath, anchorsByPath)
  }
  let external
  try {
    external = new URL(normalized)
  } catch {
    return fail(source, `非法 HTTPS URL: ${href}`)
  }
  if (external.protocol !== 'https:' || external.hostname === '') {
    return fail(source, `非安全外链，必须是含主机名的 HTTPS URL: ${href}`)
  }
}

const validateHtmlLinks = (html, source, currentPath, anchorsByPath) => {
  const hrefPattern = /\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/gi
  Array.from(html.matchAll(hrefPattern)).forEach((match) => {
    validateLink(match[1] ?? match[2] ?? match[3] ?? '', source, currentPath, anchorsByPath)
  })
}

const inspectBodyTokens = (body, source, media) => {
  const tokens = marked.lexer(body)
  const headings = tokens.filter((token) => token.type === 'heading')
  if (headings.filter((heading) => heading.depth === 1).length !== 1) {
    return fail(source, '正文必须且只能有一个 H1')
  }
  const anchoredHeadings = headings.filter((heading) => heading.depth === 2 || heading.depth === 3)
  const anchors = anchoredHeadings.map((heading) => {
    const match = HEADING_ANCHOR_PATTERN.exec(heading.text)
    if (!match) return fail(source, `H2 缺少合法 ASCII 锚点: ${heading.text}`)
    return match[1]
  })
  if (new Set(anchors).size !== anchors.length) return fail(source, '存在重复锚点')

  const allTokens = walkTokens(tokens)
  allTokens.forEach((token) => {
    if (token.type === 'image' && !media.some((entry) => entry.webPath === token.href)) {
      fail(source, `媒体未在 media-manifest.json 登记: ${token.href}`)
    }
  })
  if (SCRIPT_ELEMENT_PATTERN.test(body)) return fail(source, '拒绝 script 元素')
  if (EVENT_ATTRIBUTE_PATTERN.test(body)) return fail(source, '拒绝 on* 事件属性')
  return { tokens, anchors: new Set(anchors) }
}

const validateTokenLinks = (tokens, source, currentPath, anchorsByPath) => {
  walkTokens(tokens).forEach((token) => {
    if (token.type === 'link') {
      validateLink(token.href, source, currentPath, anchorsByPath)
    }
    if (token.type === 'html' && typeof token.text === 'string') {
      validateHtmlLinks(token.text, source, currentPath, anchorsByPath)
    }
  })
}

const assertExactGuideFiles = async (contentDir) => {
  const actualFiles = (await readdir(contentDir)).filter((name) => name.endsWith('.md')).sort()
  const missing = GUIDE_FILES.filter((name) => !actualFiles.includes(name))
  const extra = actualFiles.filter((name) => !GUIDE_FILES.includes(name))
  if (missing.length > 0 || extra.length > 0) {
    const details = [
      ...(missing.length > 0 ? [`缺少: ${missing.join(', ')}`] : []),
      ...(extra.length > 0 ? [`多余: ${extra.join(', ')}`] : []),
    ].join('；')
    fail(contentDir, `必须精确包含 9 个指南 Markdown；${details}`)
  }
}

const deepFreeze = (value) => {
  if (typeof value !== 'object' || value === null || Object.isFrozen(value)) return value
  Object.values(value).forEach((child) => deepFreeze(child))
  return Object.freeze(value)
}

const stableJson = (manifest) => `${JSON.stringify(manifest, null, 2)}\n`

const atomicWrite = async (outputPath, contents) => {
  await mkdir(dirname(outputPath), { recursive: true })
  const temporaryPath = join(dirname(outputPath), `.${outputPath.split(sep).at(-1)}.${randomUUID()}.tmp`)
  try {
    await writeFile(temporaryPath, contents, { encoding: 'utf8', flag: 'wx' })
    await rename(temporaryPath, outputPath)
  } catch (error) {
    await rm(temporaryPath, { force: true })
    throw error
  }
}

export const buildManifest = async ({
  contentDir = defaultContentDir,
  outputPath = join(contentDir, 'manifest.generated.json'),
  assetRoot = defaultAssetRoot,
  check = false,
} = {}) => {
  let contentDirReal
  let assetRootReal
  try {
    contentDirReal = await realpath(contentDir)
    assetRootReal = await realpath(assetRoot)
  } catch (error) {
    return fail(contentDir, `无法解析 contentDir 或 assetRoot: ${error.message}`)
  }
  await assertExactGuideFiles(contentDir)
  await assertSafeOutputPath({ contentDir, contentDirReal, outputPath, createParent: !check })
  const media = await loadMediaManifest(contentDir, contentDirReal, assetRoot, assetRootReal)
  const drafts = await Promise.all(
    GUIDE_SLUGS.map(async (slug) => {
      const source = validateRelativeSource(`${slug}.md`, `${slug}.md`)
      const absoluteSource = join(contentDir, source)
      const relativeSource = relative(contentDir, absoluteSource)
      if (relativeSource.startsWith('..') || isAbsolute(relativeSource)) {
        return fail(source, '拒绝 source 逃逸 contentDir')
      }
      await assertSafeInputFile(contentDirReal, absoluteSource, source)
      const sourceText = await readFile(absoluteSource, 'utf8')
      const { metadata: rawMetadata, body } = splitFrontmatter(sourceText, source)
      const meta = validateMetadata(rawMetadata, slug, source)
      const path = slug === 'index' ? '/guides/v2' : `/guides/v2/${slug}`
      const inspected = inspectBodyTokens(body, source, media)
      return {
        meta,
        path,
        source,
        body,
        ...inspected,
      }
    }),
  )
  const anchorsByPath = new Map(drafts.map((draft) => [draft.path, draft.anchors]))
  drafts.forEach((draft) => {
    validateTokenLinks(draft.tokens, draft.source, draft.path, anchorsByPath)
  })
  const entries = drafts.map(({ meta, path, source, body }) => ({
    meta,
    path,
    source,
    contentHash: createHash('sha256').update(body, 'utf8').digest('hex'),
  }))
  const manifest = deepFreeze({ version: 'v2', entries })
  const expectedOutput = stableJson(manifest)

  if (check) {
    let currentOutput
    try {
      currentOutput = await readFile(outputPath, 'utf8')
    } catch (error) {
      if (error.code === 'ENOENT') return fail('--check', `manifest 缺失: ${outputPath}`)
      throw error
    }
    if (currentOutput !== expectedOutput) return fail('--check', `manifest 过期: ${outputPath}`)
    return manifest
  }

  await atomicWrite(outputPath, expectedOutput)
  return manifest
}

const isCli = process.argv[1] && pathToFileURL(process.argv[1]).href === import.meta.url
if (isCli) {
  const unknownArguments = process.argv.slice(2).filter((argument) => argument !== '--check')
  if (unknownArguments.length > 0) {
    console.error(`build-guide-v2-manifest.mjs: 未知参数 ${unknownArguments.join(' ')}`)
    process.exitCode = 1
  } else {
    buildManifest({ check: process.argv.includes('--check') }).catch((error) => {
      console.error(error instanceof Error ? error.message : String(error))
      process.exitCode = 1
    })
  }
}
