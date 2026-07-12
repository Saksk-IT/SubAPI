export const GUIDE_V2_SLUGS = [
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

export type GuideV2Slug = (typeof GUIDE_V2_SLUGS)[number]
export type GuideV2Difficulty = '新手' | '入门'

export interface GuideV2Meta {
  readonly title: string
  readonly slug: GuideV2Slug
  readonly summary: string
  readonly duration: string
  readonly platforms: readonly string[]
  readonly difficulty: GuideV2Difficulty
  readonly updatedAt: string
  readonly version: 'v2'
}

export interface GuideV2Media {
  readonly id: string
  readonly webPath: string
  readonly exportPath: string
  readonly alt: string
}

export interface GuideV2TocItem {
  readonly level: 2 | 3
  readonly title: string
  readonly anchor: string
}

export interface GuideV2Step {
  readonly number: number
  readonly title: string
  readonly anchor: string
}

export type GuideV2Block =
  | {
      readonly type: 'heading'
      readonly level: 1 | 2 | 3 | 4 | 5 | 6
      readonly text: string
      readonly anchor?: string
      readonly stepNumber?: number
      readonly platform?: string
    }
  | { readonly type: 'paragraph'; readonly html: string }
  | { readonly type: 'table'; readonly html: string }
  | { readonly type: 'code'; readonly language: string; readonly code: string }
  | {
      readonly type: 'media'
      readonly id: string
      readonly path: string
      readonly alt: string
      readonly title?: string
    }

export interface ParsedGuideV2 {
  readonly meta: GuideV2Meta
  readonly blocks: readonly GuideV2Block[]
  readonly toc: readonly GuideV2TocItem[]
  readonly steps: readonly GuideV2Step[]
  readonly platforms: readonly string[]
}

export interface ParseGuideMarkdownInput {
  readonly sourceName: string
  readonly body: string
  readonly metadata: unknown
  readonly media: readonly GuideV2Media[]
}
