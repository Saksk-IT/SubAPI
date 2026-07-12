import { getCurrentScope, onScopeDispose, readonly, shallowRef } from 'vue'

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
  readonly subscribe: (listener: () => void) => () => void
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
  if (!storage) return createEmptyEnvelope()
  try {
    const raw = storage.getItem(GUIDE_V2_PROGRESS_STORAGE_KEY)
    return raw === null ? createEmptyEnvelope() : (parseEnvelope(raw) ?? createEmptyEnvelope())
  } catch {
    return createEmptyEnvelope()
  }
}

interface ProgressChannel {
  readonly current: () => ProgressEnvelope
  readonly write: (next: ProgressEnvelope) => void
  readonly clear: () => void
  readonly subscribe: (listener: () => void) => () => void
}

const createProgressChannel = (storage?: Storage): ProgressChannel => {
  let envelope = loadEnvelope(storage)
  const listeners = new Set<() => void>()
  const notify = (): void => [...listeners].forEach((listener) => listener())

  const write = (next: ProgressEnvelope): void => {
    envelope = next
    try {
      storage?.setItem(GUIDE_V2_PROGRESS_STORAGE_KEY, JSON.stringify(next))
    } catch {
      // The channel keeps the immutable snapshot when browser storage is disabled.
    }
    notify()
  }

  const clear = (): void => {
    envelope = createEmptyEnvelope()
    try {
      storage?.removeItem(GUIDE_V2_PROGRESS_STORAGE_KEY)
    } catch {
      // The channel snapshot is already cleared.
    }
    notify()
  }

  const subscribe = (listener: () => void): (() => void) => {
    listeners.add(listener)
    return () => listeners.delete(listener)
  }

  return Object.freeze({ current: () => envelope, write, clear, subscribe })
}

const storageChannels = new WeakMap<Storage, ProgressChannel>()
const memoryOnlyChannel = createProgressChannel()

const getProgressChannel = (storage: Storage | null | undefined): ProgressChannel => {
  if (!storage) return memoryOnlyChannel
  const existing = storageChannels.get(storage)
  if (existing) return existing

  const created = createProgressChannel(storage)
  storageChannels.set(storage, created)
  return created
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
  storage: Storage | null | undefined = getDefaultStorage(),
  now: Clock = () => new Date().toISOString(),
): GuideV2ProgressStore => {
  const channel = getProgressChannel(storage)

  const get = (guide: GuideV2Slug): GuideProgress => {
    assertGuide(guide)
    return channel.current().guides[guide] ?? createEmptyProgress()
  }

  const update = (
    guide: GuideV2Slug,
    updater: (current: GuideProgress) => Omit<GuideProgress, 'updatedAt'>,
  ): GuideProgress => {
    assertGuide(guide)
    const envelope = channel.current()
    const nextProgress = freezeProgress({ ...updater(get(guide)), updatedAt: readNow(now) })
    const nextEnvelope = Object.freeze({
      version: SCHEMA_VERSION,
      guides: Object.freeze({ ...envelope.guides, [guide]: nextProgress }),
    })
    channel.write(nextEnvelope)
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
    const envelope = channel.current()
    const { [guide]: _removed, ...remainingGuides } = envelope.guides
    const nextEnvelope = Object.freeze({
      version: SCHEMA_VERSION,
      guides: Object.freeze(remainingGuides),
    })
    channel.write(nextEnvelope)
    return createEmptyProgress()
  }

  const clearAll = (): void => channel.clear()

  return Object.freeze({
    get,
    completeStep,
    uncompleteStep,
    setPlatform,
    setLastAnchor,
    clear,
    clearAll,
    subscribe: channel.subscribe,
  })
}

export const useGuideV2Progress = (
  guide: GuideV2Slug,
  storage?: Storage,
  now?: Clock,
) => {
  const store = createGuideV2Progress(storage, now)
  const progress = shallowRef(store.get(guide))
  const sync = (): void => {
    progress.value = store.get(guide)
  }
  const unsubscribe = store.subscribe(sync)
  if (getCurrentScope()) onScopeDispose(unsubscribe)

  return Object.freeze({
    progress: readonly(progress),
    completeStep: (stepId: string) => store.completeStep(guide, stepId),
    uncompleteStep: (stepId: string) => store.uncompleteStep(guide, stepId),
    setPlatform: (platform: string | null) => store.setPlatform(guide, platform),
    setLastAnchor: (anchor: string | null) => store.setLastAnchor(guide, anchor),
    clear: () => store.clear(guide),
    clearAll: store.clearAll,
  })
}
