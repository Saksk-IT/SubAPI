import manifest from '../../../content/guides-v2/manifest.generated.json'

import type { GuideV2Meta, GuideV2Slug } from './guide-v2.types'

type GuideContentLoader = () => Promise<string>

export interface GuideV2RegistryEntry {
  readonly meta: GuideV2Meta
  readonly path: string
  readonly source: string
  readonly contentHash: string
  readonly load: GuideContentLoader
}

export interface GuideV2Navigation {
  readonly previous?: GuideV2RegistryEntry
  readonly next?: GuideV2RegistryEntry
}

const contentLoaders: Readonly<Record<GuideV2Slug, GuideContentLoader>> = Object.freeze({
  index: () => import('../../../content/guides-v2/index.md?raw').then((module) => module.default),
  'get-started': () =>
    import('../../../content/guides-v2/get-started.md?raw').then((module) => module.default),
  codex: () => import('../../../content/guides-v2/codex.md?raw').then((module) => module.default),
  'claude-code': () =>
    import('../../../content/guides-v2/claude-code.md?raw').then((module) => module.default),
  opencode: () =>
    import('../../../content/guides-v2/opencode.md?raw').then((module) => module.default),
  openclaw: () =>
    import('../../../content/guides-v2/openclaw.md?raw').then((module) => module.default),
  'chatbox-mobile': () =>
    import('../../../content/guides-v2/chatbox-mobile.md?raw').then((module) => module.default),
  'cherry-studio-image': () =>
    import('../../../content/guides-v2/cherry-studio-image.md?raw').then(
      (module) => module.default,
    ),
  troubleshooting: () =>
    import('../../../content/guides-v2/troubleshooting.md?raw').then((module) => module.default),
})

const freezeMeta = (meta: GuideV2Meta): GuideV2Meta =>
  Object.freeze({
    ...meta,
    platforms: Object.freeze([...meta.platforms]),
  })

export const GUIDE_V2_REGISTRY: readonly GuideV2RegistryEntry[] = Object.freeze(
  manifest.entries.map((entry) => {
    const meta = freezeMeta(entry.meta as GuideV2Meta)
    return Object.freeze({
      meta,
      path: entry.path,
      source: entry.source,
      contentHash: entry.contentHash,
      load: contentLoaders[meta.slug],
    })
  }),
)

export const getGuideV2Entry = (slug: string): GuideV2RegistryEntry | undefined =>
  GUIDE_V2_REGISTRY.find((entry) => entry.meta.slug === slug)

export const getGuideV2Navigation = (slug: string): GuideV2Navigation | undefined => {
  const index = GUIDE_V2_REGISTRY.findIndex((entry) => entry.meta.slug === slug)
  if (index < 0) return undefined

  return Object.freeze({
    ...(index > 0 ? { previous: GUIDE_V2_REGISTRY[index - 1] } : {}),
    ...(index < GUIDE_V2_REGISTRY.length - 1
      ? { next: GUIDE_V2_REGISTRY[index + 1] }
      : {}),
  })
}
