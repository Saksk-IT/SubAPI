import DOMPurify from 'dompurify'
import { marked, type Token, type Tokens } from 'marked'

import {
  GUIDE_V2_SLUGS,
  type GuideV2Block,
  type GuideV2Media,
  type GuideV2Meta,
  type GuideV2Slug,
  type GuideV2Step,
  type GuideV2TocItem,
  type ParseGuideMarkdownInput,
  type ParsedGuideV2,
} from './guide-v2.types'

const SUPPORTED_PLATFORMS = ['Windows', 'macOS', 'Linux', 'iOS', 'Android'] as const
const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/
const ANCHOR_PATTERN = /^[a-z][a-z0-9-]*$/
const HEADING_WITH_ANCHOR_PATTERN = /^(.*?)\s+\{#([^}]+)\}$/
const STEP_HEADING_PATTERN = /^第\s*(\d+)\s*步：(.+?)\s+\{#([^}]+)\}$/
const EVENT_ATTRIBUTE_PATTERN = /\son[a-z0-9_-]+\s*=/i
const SCRIPT_ELEMENT_PATTERN = /<\s*\/?\s*script\b/i
const ALLOWED_INTERNAL_PATHS = new Set([
  '/guides/v2',
  ...GUIDE_V2_SLUGS.filter((slug) => slug !== 'index').map((slug) => `/guides/v2/${slug}`),
])

const fail = (sourceName: string, message: string): never => {
  throw new Error(`${sourceName}: ${message}`)
}

const isRecord = (value: unknown): value is Readonly<Record<string, unknown>> =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

const isHeading = (token: Token): token is Tokens.Heading =>
  token.type === 'heading' && 'depth' in token && typeof token.depth === 'number'

const isImage = (token: Token): token is Tokens.Image =>
  token.type === 'image' && 'href' in token && typeof token.href === 'string'

const isLink = (token: Token): token is Tokens.Link =>
  token.type === 'link' && 'href' in token && typeof token.href === 'string'

const requireString = (
  metadata: Readonly<Record<string, unknown>>,
  field: string,
  sourceName: string,
): string => {
  const value = metadata[field]
  if (typeof value !== 'string' || value.trim() === '') {
    return fail(sourceName, `缺少或非法字段 ${field}`)
  }
  return value
}

const isRealDate = (value: string): boolean => {
  if (!DATE_PATTERN.test(value)) return false
  const [year, month, day] = value.split('-').map(Number)
  const date = new Date(Date.UTC(year, month - 1, day))
  return (
    date.getUTCFullYear() === year &&
    date.getUTCMonth() === month - 1 &&
    date.getUTCDate() === day
  )
}

const parseMetadata = (metadata: unknown, sourceName: string): GuideV2Meta => {
  if (!isRecord(metadata)) return fail(sourceName, '元数据必须是对象')

  const title = requireString(metadata, 'title', sourceName)
  const slug = requireString(metadata, 'slug', sourceName)
  const summary = requireString(metadata, 'summary', sourceName)
  const duration = requireString(metadata, 'duration', sourceName)
  const difficulty = requireString(metadata, 'difficulty', sourceName)
  const updatedAt = requireString(metadata, 'updatedAt', sourceName)
  const version = requireString(metadata, 'version', sourceName)

  if (!GUIDE_V2_SLUGS.some((candidate) => candidate === slug)) {
    return fail(sourceName, `非法 slug: ${slug}`)
  }
  if (difficulty !== '新手' && difficulty !== '入门') {
    return fail(sourceName, `非法难度: ${difficulty}`)
  }
  if (!isRealDate(updatedAt)) return fail(sourceName, `错误日期: ${updatedAt}`)
  if (version !== 'v2') return fail(sourceName, `版本必须是 v2，收到 ${version}`)
  if (!Array.isArray(metadata.platforms) || metadata.platforms.length === 0) {
    return fail(sourceName, '缺少或非法字段 platforms')
  }
  const platforms = metadata.platforms.map((platform) => {
    if (
      typeof platform !== 'string' ||
      !SUPPORTED_PLATFORMS.some((candidate) => candidate === platform)
    ) {
      return fail(sourceName, `非法平台: ${String(platform)}`)
    }
    return platform
  })

  return {
    title,
    slug: slug as GuideV2Slug,
    summary,
    duration,
    platforms,
    difficulty,
    updatedAt,
    version,
  }
}

const validateLink = (href: string, sourceName: string): void => {
  const normalized = href.trim()
  if (/^(?:javascript|data|vbscript):/i.test(normalized)) {
    return fail(sourceName, `危险协议链接: ${href}`)
  }
  if (/^https:\/\//i.test(normalized) || normalized.startsWith('#')) return
  if (normalized.startsWith('/guides/v2')) {
    const url = new URL(normalized, 'https://guides.local')
    if (!ALLOWED_INTERNAL_PATHS.has(url.pathname) || url.search !== '') {
      return fail(sourceName, `非法内部链接: ${href}`)
    }
    return
  }
  if (/^http:/i.test(normalized)) return fail(sourceName, `非安全外链: ${href}`)
  return fail(sourceName, `非法内部链接: ${href}`)
}

const validateHtmlLinks = (html: string, sourceName: string): void => {
  const hrefPattern = /\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/gi
  Array.from(html.matchAll(hrefPattern)).forEach((match) => {
    validateLink(match[1] ?? match[2] ?? match[3] ?? '', sourceName)
  })
}

const childTokens = (token: Token): readonly Token[] => {
  if ('tokens' in token && Array.isArray(token.tokens)) return token.tokens
  if (token.type === 'table') {
    const table = token as Tokens.Table
    return [
      ...table.header.flatMap((cell) => cell.tokens),
      ...table.rows.flatMap((row) => row.flatMap((cell) => cell.tokens)),
    ]
  }
  if (token.type === 'list') {
    return (token as Tokens.List).items.flatMap((item) => item.tokens)
  }
  return []
}

const walkTokens = (tokens: readonly Token[]): readonly Token[] =>
  tokens.flatMap((token) => [token, ...walkTokens(childTokens(token))])

const registeredMedia = (
  token: Tokens.Image,
  media: readonly GuideV2Media[],
  sourceName: string,
): GuideV2Media => {
  const entry = media.find((candidate) => candidate.webPath === token.href)
  if (!entry) return fail(sourceName, `媒体未在 manifest 登记: ${token.href}`)
  return entry
}

const sanitize = (html: string): string =>
  DOMPurify.sanitize(html, {
    ALLOWED_TAGS: [
      'a',
      'p',
      'strong',
      'em',
      'del',
      'code',
      'br',
      'ul',
      'ol',
      'li',
      'blockquote',
      'table',
      'thead',
      'tbody',
      'tr',
      'th',
      'td',
    ],
    ALLOWED_ATTR: ['href', 'title', 'target', 'rel'],
    ALLOW_DATA_ATTR: false,
  })

const parseHeading = (
  token: Tokens.Heading,
  meta: GuideV2Meta,
  sourceName: string,
  anchors: ReadonlySet<string>,
): {
  readonly block: GuideV2Block
  readonly toc?: GuideV2TocItem
  readonly step?: GuideV2Step
  readonly anchor?: string
} => {
  if (token.depth === 1) return { block: { type: 'heading', level: 1, text: token.text } }

  if (token.depth === 2) {
    if (/^第.*步/.test(token.text)) {
      const step = STEP_HEADING_PATTERN.exec(token.text)
      if (!step) return fail(sourceName, `步骤标题格式错误: ${token.text}`)
      const [, number, title, anchor] = step
      validateAnchor(anchor, sourceName, anchors)
      const parsedStep = { number: Number(number), title: title.trim(), anchor }
      return {
        block: {
          type: 'heading',
          level: 2,
          text: title.trim(),
          anchor,
          stepNumber: Number(number),
        },
        toc: { level: 2, title: title.trim(), anchor },
        step: parsedStep,
        anchor,
      }
    }

    const heading = HEADING_WITH_ANCHOR_PATTERN.exec(token.text)
    if (!heading) return fail(sourceName, `H2 缺少 ASCII 锚点: ${token.text}`)
    const [, title, anchor] = heading
    validateAnchor(anchor, sourceName, anchors)
    return {
      block: { type: 'heading', level: 2, text: title.trim(), anchor },
      toc: { level: 2, title: title.trim(), anchor },
      anchor,
    }
  }

  const anchoredHeading = HEADING_WITH_ANCHOR_PATTERN.exec(token.text)
  const headingText = anchoredHeading?.[1].trim() ?? token.text
  const platform = SUPPORTED_PLATFORMS.find((candidate) => candidate === headingText)
  if (token.depth === 3 && platform) {
    if (!anchoredHeading) return fail(sourceName, `平台 H3 缺少 ASCII 锚点: ${token.text}`)
    const anchor = anchoredHeading[2]
    validateAnchor(anchor, sourceName, anchors)
    if (!meta.platforms.includes(platform)) {
      return fail(sourceName, `平台 ${platform} 未在元数据 platforms 中允许`)
    }
    return {
      block: { type: 'heading', level: 3, text: platform, anchor, platform },
      toc: { level: 3, title: platform, anchor },
      anchor,
    }
  }
  if (token.depth === 3 && anchoredHeading) {
    const anchor = anchoredHeading[2]
    validateAnchor(anchor, sourceName, anchors)
    return {
      block: { type: 'heading', level: 3, text: headingText, anchor },
      toc: { level: 3, title: headingText, anchor },
      anchor,
    }
  }

  return {
    block: {
      type: 'heading',
      level: Math.min(6, Math.max(1, token.depth)) as 1 | 2 | 3 | 4 | 5 | 6,
      text: token.text,
    },
  }
}

const validateAnchor = (
  anchor: string,
  sourceName: string,
  anchors: ReadonlySet<string>,
): void => {
  if (!ANCHOR_PATTERN.test(anchor)) return fail(sourceName, `非法 ASCII 锚点: ${anchor}`)
  if (anchors.has(anchor)) return fail(sourceName, `重复锚点: ${anchor}`)
}

const renderBlock = (
  token: Token,
  media: readonly GuideV2Media[],
  sourceName: string,
): readonly GuideV2Block[] => {
  if (token.type === 'space') return []
  if (token.type === 'code') {
    return [{ type: 'code', language: token.lang ?? '', code: token.text.replace(/\n$/, '') }]
  }
  if (token.type === 'paragraph') {
    const paragraph = token as Tokens.Paragraph
    return splitInlineTokens(paragraph.tokens).flatMap((segment) =>
      segment.type === 'media'
        ? [toMediaBlock(segment.image, media, sourceName)]
        : renderInlineParagraph(segment.tokens),
    )
  }
  if (token.type === 'table') return [{ type: 'table', html: sanitize(marked.parser([token])) }]
  if (token.type === 'list' || token.type === 'blockquote' || token.type === 'html') {
    return [{ type: 'paragraph', html: sanitize(marked.parser([token])) }]
  }
  return []
}

const toMediaBlock = (
  image: Tokens.Image,
  media: readonly GuideV2Media[],
  sourceName: string,
): GuideV2Block => {
  const entry = registeredMedia(image, media, sourceName)
  return {
    type: 'media',
    id: entry.id,
    path: entry.webPath,
    alt: image.text || entry.alt,
    ...(image.title ? { title: image.title } : {}),
  }
}

type InlineSegment =
  | { readonly type: 'tokens'; readonly tokens: readonly Token[] }
  | { readonly type: 'media'; readonly image: Tokens.Image }

const splitInlineTokens = (tokens: readonly Token[]): readonly InlineSegment[] => {
  const splitToken = (token: Token): readonly InlineSegment[] => {
    if (isImage(token)) return [{ type: 'media', image: token }]
    if (!('tokens' in token) || !Array.isArray(token.tokens)) {
      return [{ type: 'tokens', tokens: [token] }]
    }
    if (!walkTokens(token.tokens).some(isImage)) {
      return [{ type: 'tokens', tokens: [token] }]
    }
    return splitInlineTokens(token.tokens).map((segment) => {
      if (segment.type === 'media') return segment
      const raw = segment.tokens.map((child) => child.raw).join('')
      const wrappedToken = {
        ...token,
        raw,
        text: raw,
        tokens: [...segment.tokens],
      } as Token
      return { type: 'tokens', tokens: [wrappedToken] }
    })
  }

  return tokens.flatMap(splitToken).reduce<readonly InlineSegment[]>((segments, segment) => {
    const previous = segments.at(-1)
    if (previous?.type === 'tokens' && segment.type === 'tokens') {
      return [
        ...segments.slice(0, -1),
        { type: 'tokens', tokens: [...previous.tokens, ...segment.tokens] },
      ]
    }
    return [...segments, segment]
  }, [])
}

const renderInlineParagraph = (tokens: readonly Token[]): readonly GuideV2Block[] => {
  if (tokens.length === 0) return []
  const raw = tokens.map((token) => token.raw).join('')
  const paragraph: Tokens.Paragraph = {
    type: 'paragraph',
    raw,
    text: raw,
    tokens: [...tokens],
  }
  return [{ type: 'paragraph', html: sanitize(marked.parser([paragraph])) }]
}

const deepFreeze = <T>(value: T): T => {
  if (typeof value !== 'object' || value === null || Object.isFrozen(value)) return value
  Object.values(value).forEach((child) => deepFreeze(child))
  return Object.freeze(value)
}

export const parseGuideMarkdown = (input: ParseGuideMarkdownInput): ParsedGuideV2 => {
  const { sourceName, body, metadata, media } = input
  if (typeof sourceName !== 'string' || sourceName.trim() === '') {
    throw new Error('sourceName: 必须是非空字符串')
  }
  if (typeof body !== 'string') return fail(sourceName, '正文必须是字符串')

  const meta = parseMetadata(metadata, sourceName)
  const tokens = marked.lexer(body)
  const allTokens = walkTokens(tokens)

  allTokens.forEach((token) => {
    if (isLink(token)) validateLink(token.href, sourceName)
    if (isImage(token)) registeredMedia(token, media, sourceName)
    if (token.type === 'html' && 'text' in token && typeof token.text === 'string') {
      validateHtmlLinks(token.text, sourceName)
    }
  })
  if (SCRIPT_ELEMENT_PATTERN.test(body)) return fail(sourceName, '拒绝 script 元素')
  if (EVENT_ATTRIBUTE_PATTERN.test(body)) return fail(sourceName, '拒绝 on* 事件属性')

  const headingOnes = tokens.filter(isHeading).filter((token) => token.depth === 1)
  if (headingOnes.length !== 1) return fail(sourceName, '正文必须且只能有一个 H1')

  const parsed = tokens.reduce<{
    readonly blocks: readonly GuideV2Block[]
    readonly toc: readonly GuideV2TocItem[]
    readonly steps: readonly GuideV2Step[]
    readonly anchors: ReadonlySet<string>
  }>(
    (state, token) => {
      if (!isHeading(token)) {
        return { ...state, blocks: [...state.blocks, ...renderBlock(token, media, sourceName)] }
      }
      const heading = parseHeading(token, meta, sourceName, state.anchors)
      return {
        blocks: [...state.blocks, heading.block],
        toc: heading.toc ? [...state.toc, heading.toc] : state.toc,
        steps: heading.step ? [...state.steps, heading.step] : state.steps,
        anchors: heading.anchor ? new Set([...state.anchors, heading.anchor]) : state.anchors,
      }
    },
    { blocks: [], toc: [], steps: [], anchors: new Set<string>() },
  )

  return deepFreeze({
    meta: { ...meta, platforms: [...meta.platforms] },
    blocks: [...parsed.blocks],
    toc: [...parsed.toc],
    steps: [...parsed.steps],
    platforms: [...meta.platforms],
  })
}
