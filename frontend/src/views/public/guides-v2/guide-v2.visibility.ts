import type {
  GuideV2Block,
  GuideV2TocItem,
  ParsedGuideV2,
} from './guide-v2.types'

const ANCHOR_PATTERN = /^[a-z][a-z0-9-]*$/

export interface GuideV2Visibility {
  readonly blocks: readonly GuideV2Block[]
  readonly toc: readonly GuideV2TocItem[]
  readonly platformByAnchor: Readonly<Record<string, string>>
}

export const decodeGuideV2Hash = (hash: string): string | null => {
  if (!hash.startsWith('#')) return null
  try {
    const anchor = decodeURIComponent(hash.slice(1))
    return ANCHOR_PATTERN.test(anchor) ? anchor : null
  } catch {
    return null
  }
}

export const deriveGuideV2Visibility = (
  guide: ParsedGuideV2,
  selectedPlatform: string,
): GuideV2Visibility => {
  let platformScope: string | null = null
  const platformEntries = guide.blocks.flatMap((block) =>
    block.type === 'heading' && block.platform && block.anchor
      ? [[block.anchor, block.platform] as const]
      : [],
  )
  const platformByAnchor = Object.freeze(
    Object.create(
      null,
      Object.getOwnPropertyDescriptors(Object.fromEntries(platformEntries)),
    ) as Record<string, string>,
  )
  const blocks = guide.blocks.filter((block) => {
    if (block.type === 'heading') {
      if (block.level <= 2) platformScope = null
      if (block.level === 3) platformScope = block.platform ?? null
    }
    return platformScope === null || platformScope === selectedPlatform
  })
  const visibleAnchors = new Set(
    blocks.flatMap((block) =>
      block.type === 'heading' && block.anchor ? [block.anchor] : [],
    ),
  )

  return Object.freeze({
    blocks: Object.freeze([...blocks]),
    toc: Object.freeze(guide.toc.filter((item) => visibleAnchors.has(item.anchor))),
    platformByAnchor,
  })
}

export const isGuideV2TocAnchor = (guide: ParsedGuideV2, anchor: string): boolean =>
  guide.toc.some((item) => item.anchor === anchor)
