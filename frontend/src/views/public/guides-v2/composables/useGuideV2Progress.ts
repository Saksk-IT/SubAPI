import { readonly, shallowRef } from 'vue'

import { GUIDE_V2_SLUGS, type GuideV2Slug } from '../guide-v2.types'

export const GUIDE_V2_PROGRESS_STORAGE_KEY = 'sub2api:guides:v2:progress:v1'

const SCHEMA_VERSION = 1 as const
const MAX_VALUE_LENGTH = 200
const ANCHOR_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/
const GUIDE_SLUG_SET = new Set<string>(GUIDE_V2_SLUGS)

export interface GuideProgress {
  readonly completedStepIds: readonly string[]
  readonly platform: string | null
  readonly lastAnchor: string | null
  readonly updatedAt: string | null
}

interface ProgressEnvelope {
  readonly version: typeof SCHEMA_VERSION
  readonly guides: Readonly<Partial<Record<GuideV2Slug, GuideProgress>>>
}

export interface GuideV2ProgressStore {
  readonly get: (guide: GuideV2Slug) => GuideProgress
  readonly completeStep: (guide: GuideV2Slug, stepId: string) => GuideProgress
  readonly uncompleteStep: (guide: GuideV2Slug, stepId: string) => GuideProgress
  readonly setPlatform: (guide: GuideV2Slug, platform: string | null) => GuideProgress
  readonly setLastAnchor: (guide: GuideV2Slug, anchor: string | null) => GuideProgress
  readonly clear: (guide: GuideV2Slug) => GuideProgress
  readonly clearAll: () => void
}

type Clock = () => string

const createEmptyProgress = (): GuideProgress =>
  Object.freeze({
    completedStepIds: Object.freeze([] as string[]),
    platform: null,
    lastAnchor: null,
    updatedAt: null,
  })

const createEmptyEnvelope = (): ProgressEnvelope =>
  Object.freeze({ version: SCHEMA_VERSION, guides: Object.freeze({}) })

let memoryEnvelope = createEmptyEnvelope()
const failedStorages = new WeakSet<Storage>()

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

const isSafeText = (value: unknown): value is string =>
  typeof value === 'string' && value.length > 0 && value.length <= MAX_VALUE_LENGTH

const isNullableText = (value: unknown): value is string | null =>
  value === null || isSafeText(value)

const isAnchor = (value: unknown): value is string =>
  isSafeText(value) && ANCHOR_PATTERN.test(value)

const isNullableAnchor = (value: unknown): value is string | null =>
  value === null || isAnchor(value)

const isTimestamp = (value: unknown): value is string =>
  isSafeText(value) && !Number.isNaN(Date.parse(value))

const isNullableTimestamp = (value: unknown): value is string | null =>
  value === null || isTimestamp(value)

const freezeProgress = (progress: GuideProgress): GuideProgress =>
  Object.freeze({
    ...progress,
    completedStepIds: Object.freeze([...progress.completedStepIds]),
  })

const parseProgress = (value: unknown): GuideProgress | undefined => {
  if (!isRecord(value)) return undefined

  const { completedStepIds, platform, lastAnchor, updatedAt } = value
  if (
    !Array.isArray(completedStepIds) ||
    !completedStepIds.every(isAnchor) ||
    new Set(completedStepIds).size !== completedStepIds.length ||
    !isNullableText(platform) ||
    !isNullableAnchor(lastAnchor) ||
    !isNullableTimestamp(updatedAt)
  ) {
    return undefined
  }

  return freezeProgress({ completedStepIds, platform, lastAnchor, updatedAt })
}

const parseEnvelope = (raw: string): ProgressEnvelope | undefined => {
  let value: unknown
  try {
    value = JSON.parse(raw)
  } catch {
    return undefined
  }

  if (!isRecord(value) || value.version !== SCHEMA_VERSION || !isRecord(value.guides)) {
    return undefined
  }

  const entries = Object.entries(value.guides)
  if (entries.some(([guide]) => !GUIDE_SLUG_SET.has(guide))) return undefined

  const parsedEntries = entries.map(([guide, progress]) => [guide, parseProgress(progress)] as const)
  if (parsedEntries.some(([, progress]) => progress === undefined)) return undefined

  const guides = Object.fromEntries(parsedEntries) as Partial<Record<GuideV2Slug, GuideProgress>>
  return Object.freeze({ version: SCHEMA_VERSION, guides: Object.freeze(guides) })
}

const getDefaultStorage = (): Storage | undefined => {
  try {
    return typeof globalThis.localStorage === 'undefined' ? undefined : globalThis.localStorage
  } catch {
    return undefined
  }
}

const loadEnvelope = (storage: Storage | undefined): ProgressEnvelope => {
  if (!storage) return memoryEnvelope
  if (failedStorages.has(storage)) return memoryEnvelope

  try {
    const raw = storage.getItem(GUIDE_V2_PROGRESS_STORAGE_KEY)
    const loaded = raw === null ? createEmptyEnvelope() : (parseEnvelope(raw) ?? createEmptyEnvelope())
    memoryEnvelope = loaded
    return loaded
  } catch {
    failedStorages.add(storage)
    return memoryEnvelope
  }
}

function assertGuide(guide: string): asserts guide is GuideV2Slug {
  if (!GUIDE_SLUG_SET.has(guide)) throw new TypeError(`无效的 V2 教程标识：${guide}`)
}

const assertAnchor = (value: string, field: string): void => {
  if (!isAnchor(value)) throw new TypeError(`${field} 必须是有效的 ASCII 锚点`)
}

const assertNullableText = (value: string | null, field: string): void => {
  if (!isNullableText(value)) throw new TypeError(`${field} 必须是非空短文本或 null`)
}

const assertNullableAnchor = (value: string | null, field: string): void => {
  if (!isNullableAnchor(value)) throw new TypeError(`${field} 必须是有效的 ASCII 锚点或 null`)
}

const readNow = (now: Clock): string => {
  const timestamp = now()
  if (!isTimestamp(timestamp)) throw new TypeError('时钟必须返回有效时间字符串')
  return timestamp
}

export const createGuideV2Progress = (
  storage: Storage | undefined = getDefaultStorage(),
  now: Clock = () => new Date().toISOString(),
): GuideV2ProgressStore => {
  let envelope = loadEnvelope(storage)

  const persist = (next: ProgressEnvelope): void => {
    envelope = next
    memoryEnvelope = next
    try {
      storage?.setItem(GUIDE_V2_PROGRESS_STORAGE_KEY, JSON.stringify(next))
      if (storage) failedStorages.delete(storage)
    } catch {
      if (storage) failedStorages.add(storage)
      // The immutable module snapshot remains available when browser storage is disabled.
    }
  }

  const get = (guide: GuideV2Slug): GuideProgress => {
    assertGuide(guide)
    return envelope.guides[guide] ?? createEmptyProgress()
  }

  const update = (
    guide: GuideV2Slug,
    updater: (current: GuideProgress) => Omit<GuideProgress, 'updatedAt'>,
  ): GuideProgress => {
    assertGuide(guide)
    const nextProgress = freezeProgress({ ...updater(get(guide)), updatedAt: readNow(now) })
    const nextEnvelope = Object.freeze({
      version: SCHEMA_VERSION,
      guides: Object.freeze({ ...envelope.guides, [guide]: nextProgress }),
    })
    persist(nextEnvelope)
    return nextProgress
  }

  const completeStep = (guide: GuideV2Slug, stepId: string): GuideProgress => {
    assertAnchor(stepId, '步骤标识')
    return update(guide, (current) => ({
      ...current,
      completedStepIds: current.completedStepIds.includes(stepId)
        ? [...current.completedStepIds]
        : [...current.completedStepIds, stepId],
    }))
  }

  const uncompleteStep = (guide: GuideV2Slug, stepId: string): GuideProgress => {
    assertAnchor(stepId, '步骤标识')
    return update(guide, (current) => ({
      ...current,
      completedStepIds: current.completedStepIds.filter((id) => id !== stepId),
    }))
  }

  const setPlatform = (guide: GuideV2Slug, platform: string | null): GuideProgress => {
    assertNullableText(platform, '平台')
    return update(guide, (current) => ({ ...current, platform }))
  }

  const setLastAnchor = (guide: GuideV2Slug, lastAnchor: string | null): GuideProgress => {
    assertNullableAnchor(lastAnchor, '最后锚点')
    return update(guide, (current) => ({ ...current, lastAnchor }))
  }

  const clear = (guide: GuideV2Slug): GuideProgress => {
    assertGuide(guide)
    const { [guide]: _removed, ...remainingGuides } = envelope.guides
    const nextEnvelope = Object.freeze({
      version: SCHEMA_VERSION,
      guides: Object.freeze(remainingGuides),
    })
    persist(nextEnvelope)
    return createEmptyProgress()
  }

  const clearAll = (): void => {
    envelope = createEmptyEnvelope()
    memoryEnvelope = envelope
    try {
      storage?.removeItem(GUIDE_V2_PROGRESS_STORAGE_KEY)
      if (storage) failedStorages.delete(storage)
    } catch {
      if (storage) failedStorages.add(storage)
      // The in-memory snapshot is already cleared.
    }
  }

  return Object.freeze({
    get,
    completeStep,
    uncompleteStep,
    setPlatform,
    setLastAnchor,
    clear,
    clearAll,
  })
}

export const useGuideV2Progress = (
  guide: GuideV2Slug,
  storage?: Storage,
  now?: Clock,
) => {
  const store = createGuideV2Progress(storage, now)
  const progress = shallowRef(store.get(guide))
  const sync = (next: GuideProgress): GuideProgress => {
    progress.value = next
    return next
  }

  return Object.freeze({
    progress: readonly(progress),
    completeStep: (stepId: string) => sync(store.completeStep(guide, stepId)),
    uncompleteStep: (stepId: string) => sync(store.uncompleteStep(guide, stepId)),
    setPlatform: (platform: string | null) => sync(store.setPlatform(guide, platform)),
    setLastAnchor: (anchor: string | null) => sync(store.setLastAnchor(guide, anchor)),
    clear: () => sync(store.clear(guide)),
  })
}
