import { effectScope } from 'vue'
import { describe, expect, it, vi } from 'vitest'

import {
  GUIDE_V2_PROGRESS_STORAGE_KEY,
  createGuideV2Progress,
  useGuideV2Progress,
} from '../useGuideV2Progress'

const FIXED_NOW = '2026-07-13T08:30:00.000Z'

const createStorage = (initial: Readonly<Record<string, string>> = {}): Storage => {
  const values = new Map(Object.entries(initial))

  return {
    get length() {
      return values.size
    },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => values.delete(key),
    setItem: (key, value) => values.set(key, value),
  }
}

const now = () => FIXED_NOW

describe('createGuideV2Progress', () => {
  it('updates progress without mutating the previous state', () => {
    const progress = createGuideV2Progress(createStorage(), now)
    const before = progress.get('codex')
    const after = progress.completeStep('codex', 'initialize')

    expect(after).not.toBe(before)
    expect(after.completedStepIds).not.toBe(before.completedStepIds)
    expect(before.completedStepIds).toEqual([])
    expect(after).toEqual({
      completedStepIds: ['initialize'],
      platform: null,
      lastAnchor: null,
      updatedAt: FIXED_NOW,
    })
  })

  it('deduplicates completed steps and can mark them incomplete', () => {
    const progress = createGuideV2Progress(createStorage(), now)

    progress.completeStep('codex', 'initialize')
    const completed = progress.completeStep('codex', 'initialize')
    const incomplete = progress.uncompleteStep('codex', 'initialize')

    expect(completed.completedStepIds).toEqual(['initialize'])
    expect(incomplete.completedStepIds).toEqual([])
    expect(completed.completedStepIds).toEqual(['initialize'])
  })

  it('restores persisted progress and isolates guides', () => {
    const storage = createStorage()
    const firstVisit = createGuideV2Progress(storage, now)
    firstVisit.completeStep('codex', 'initialize')
    firstVisit.completeStep('opencode', 'install')

    const refreshed = createGuideV2Progress(storage, now)

    expect(refreshed.get('codex').completedStepIds).toEqual(['initialize'])
    expect(refreshed.get('opencode').completedStepIds).toEqual(['install'])
    expect(refreshed.get('claude-code').completedStepIds).toEqual([])
  })

  it('merges writes from sibling stores for the same guide', () => {
    const storage = createStorage()
    const first = createGuideV2Progress(storage, now)
    const second = createGuideV2Progress(storage, now)

    first.completeStep('codex', 'initialize')
    second.setPlatform('codex', 'macOS')

    expect(first.get('codex')).toMatchObject({
      completedStepIds: ['initialize'],
      platform: 'macOS',
    })
    expect(second.get('codex')).toMatchObject({
      completedStepIds: ['initialize'],
      platform: 'macOS',
    })
  })

  it('preserves different guides written from sibling stores', () => {
    const storage = createStorage()
    const first = createGuideV2Progress(storage, now)
    const second = createGuideV2Progress(storage, now)

    first.completeStep('codex', 'initialize')
    second.completeStep('opencode', 'install')

    const refreshed = createGuideV2Progress(storage, now)
    expect(refreshed.get('codex').completedStepIds).toEqual(['initialize'])
    expect(refreshed.get('opencode').completedStepIds).toEqual(['install'])
  })

  it('does not let a stale sibling store revive data after clearAll', () => {
    const storage = createStorage()
    const first = createGuideV2Progress(storage, now)
    first.completeStep('codex', 'initialize')
    const staleSibling = createGuideV2Progress(storage, now)

    first.clearAll()
    staleSibling.setPlatform('codex', 'macOS')

    expect(staleSibling.get('codex')).toMatchObject({
      completedStepIds: [],
      platform: 'macOS',
    })
    expect(createGuideV2Progress(storage, now).get('codex').completedStepIds).toEqual([])
  })

  it.each([
    ['broken JSON', '{not-json'],
    ['wrong schema version', JSON.stringify({ version: 2, guides: {} })],
    [
      'invalid fields',
      JSON.stringify({
        version: 1,
        guides: {
          codex: {
            completedStepIds: ['ok', 3],
            platform: null,
            lastAnchor: null,
            updatedAt: null,
          },
        },
      }),
    ],
    [
      'invalid guide collection',
      JSON.stringify({ version: 1, guides: ['codex'] }),
    ],
    [
      'non-object guide progress',
      JSON.stringify({ version: 1, guides: { codex: [] } }),
    ],
    [
      'duplicate completed steps',
      JSON.stringify({
        version: 1,
        guides: {
          codex: {
            completedStepIds: ['initialize', 'initialize'],
            platform: null,
            lastAnchor: null,
            updatedAt: null,
          },
        },
      }),
    ],
    [
      'invalid platform field',
      JSON.stringify({
        version: 1,
        guides: {
          codex: {
            completedStepIds: [],
            platform: 3,
            lastAnchor: null,
            updatedAt: null,
          },
        },
      }),
    ],
    [
      'invalid anchor field',
      JSON.stringify({
        version: 1,
        guides: {
          codex: {
            completedStepIds: [],
            platform: null,
            lastAnchor: 'Bad Anchor',
            updatedAt: null,
          },
        },
      }),
    ],
    [
      'invalid timestamp field',
      JSON.stringify({
        version: 1,
        guides: {
          codex: {
            completedStepIds: [],
            platform: null,
            lastAnchor: null,
            updatedAt: 'not-a-date',
          },
        },
      }),
    ],
  ])('safely falls back for %s', (_label, storedValue) => {
    const storage = createStorage({ [GUIDE_V2_PROGRESS_STORAGE_KEY]: storedValue })

    expect(createGuideV2Progress(storage, now).get('codex')).toEqual({
      completedStepIds: [],
      platform: null,
      lastAnchor: null,
      updatedAt: null,
    })
  })

  it('keeps working from memory when storage reads or writes throw', () => {
    const readErrorStorage = createStorage()
    readErrorStorage.getItem = vi.fn(() => {
      throw new Error('read denied')
    })
    readErrorStorage.setItem = vi.fn(() => {
      throw new Error('write denied')
    })

    const progress = createGuideV2Progress(readErrorStorage, now)
    const completed = progress.completeStep('codex', 'initialize')

    expect(completed.completedStepIds).toEqual(['initialize'])
    expect(progress.get('codex').completedStepIds).toEqual(['initialize'])
    expect(readErrorStorage.setItem).toHaveBeenCalled()
  })

  it('restores the module snapshot in a new store after setItem throws', () => {
    const storage = createStorage()
    storage.setItem = vi.fn(() => {
      throw new Error('write denied')
    })

    createGuideV2Progress(storage, now).completeStep('codex', 'initialize')

    expect(createGuideV2Progress(storage, now).get('codex').completedStepIds).toEqual([
      'initialize',
    ])
  })

  it('isolates fallback snapshots for different Storage instances', () => {
    const storageA = createStorage()
    storageA.setItem = vi.fn(() => {
      throw new Error('A write denied')
    })
    const storageB = createStorage()

    createGuideV2Progress(storageA, now).completeStep('codex', 'initialize')
    createGuideV2Progress(storageB, now).completeStep('opencode', 'install')

    const restoredA = createGuideV2Progress(storageA, now)
    expect(restoredA.get('codex').completedStepIds).toEqual(['initialize'])
    expect(restoredA.get('opencode').completedStepIds).toEqual([])
  })

  it('shares a dedicated memory-only channel when no Storage is supplied', () => {
    const first = createGuideV2Progress(null, now)
    first.clearAll()
    const second = createGuideV2Progress(null, now)

    first.completeStep('codex', 'initialize')

    expect(second.get('codex').completedStepIds).toEqual(['initialize'])
    second.clearAll()
  })

  it('sets platform, anchor, and deterministic update timestamps', () => {
    const progress = createGuideV2Progress(createStorage(), now)

    const platform = progress.setPlatform('codex', 'macOS')
    const anchored = progress.setLastAnchor('codex', 'api-login')
    const cleared = progress.setPlatform('codex', null)

    expect(platform).toMatchObject({ platform: 'macOS', updatedAt: FIXED_NOW })
    expect(anchored).toMatchObject({
      platform: 'macOS',
      lastAnchor: 'api-login',
      updatedAt: FIXED_NOW,
    })
    expect(cleared).toMatchObject({ platform: null, lastAnchor: 'api-login' })
    expect(platform).toMatchObject({ platform: 'macOS', lastAnchor: null })
  })

  it('clears one guide without touching another guide or business storage', () => {
    const storage = createStorage({ auth_token: 'keep-me' })
    const progress = createGuideV2Progress(storage, now)
    progress.completeStep('codex', 'initialize')
    progress.completeStep('opencode', 'install')

    const cleared = progress.clear('codex')

    expect(cleared.completedStepIds).toEqual([])
    expect(progress.get('opencode').completedStepIds).toEqual(['install'])
    expect(storage.getItem('auth_token')).toBe('keep-me')
  })

  it('clears all guide progress using only its own storage key', () => {
    const storage = createStorage({ auth_token: 'keep-me' })
    const removeItem = vi.spyOn(storage, 'removeItem')
    const progress = createGuideV2Progress(storage, now)
    progress.completeStep('codex', 'initialize')

    progress.clearAll()

    expect(progress.get('codex').completedStepIds).toEqual([])
    expect(removeItem).toHaveBeenCalledTimes(1)
    expect(removeItem).toHaveBeenCalledWith(GUIDE_V2_PROGRESS_STORAGE_KEY)
    expect(storage.getItem('auth_token')).toBe('keep-me')
  })

  it('clears memory even when removeItem throws', () => {
    const storage = createStorage()
    const progress = createGuideV2Progress(storage, now)
    progress.completeStep('codex', 'initialize')
    storage.removeItem = vi.fn(() => {
      throw new Error('remove denied')
    })

    expect(() => progress.clearAll()).not.toThrow()
    expect(progress.get('codex').completedStepIds).toEqual([])
    expect(createGuideV2Progress(storage, now).get('codex').completedStepIds).toEqual([])
  })

  it.each([
    ['dangerous guide key', '__proto__', 'step'],
    ['unknown guide key', 'unknown-guide', 'step'],
    ['empty step id', 'codex', ''],
    ['oversized step id', 'codex', 'x'.repeat(201)],
  ])('rejects %s at the boundary', (_label, guide, step) => {
    const progress = createGuideV2Progress(createStorage(), now)

    expect(() => progress.completeStep(guide as 'codex', step)).toThrow()
  })

  it('rejects invalid setter values and clocks at the boundary', () => {
    const progress = createGuideV2Progress(createStorage(), () => 'not-a-date')

    expect(() => progress.setPlatform('codex', '' as 'macOS')).toThrow('平台')
    expect(() => progress.setLastAnchor('codex', 'Bad Anchor')).toThrow('最后锚点')
    expect(() => progress.completeStep('codex', 'initialize')).toThrow('时钟')
  })

  it('rejects dangerous persisted guide keys instead of merging them', () => {
    const malicious = `{"version":1,"guides":{"__proto__":{"completedStepIds":[],"platform":null,"lastAnchor":null,"updatedAt":null}}}`
    const storage = createStorage({ [GUIDE_V2_PROGRESS_STORAGE_KEY]: malicious })
    const progress = createGuideV2Progress(storage, now)

    expect(progress.get('codex').completedStepIds).toEqual([])
    expect(({} as { polluted?: boolean }).polluted).toBeUndefined()
  })
})

describe('useGuideV2Progress', () => {
  it('keeps a readonly Vue ref synchronized with store operations', () => {
    const guide = useGuideV2Progress('codex', createStorage(), now)

    guide.completeStep('initialize')
    guide.setPlatform('macOS')
    guide.setLastAnchor('api-login')

    expect(guide.progress.value).toEqual({
      completedStepIds: ['initialize'],
      platform: 'macOS',
      lastAnchor: 'api-login',
      updatedAt: FIXED_NOW,
    })

    guide.uncompleteStep('initialize')
    expect(guide.progress.value.completedStepIds).toEqual([])
    guide.clear()
    expect(guide.progress.value.updatedAt).toBeNull()
  })

  it('synchronizes sibling refs and exposes clearAll', () => {
    const storage = createStorage()
    const first = useGuideV2Progress('codex', storage, now)
    const second = useGuideV2Progress('codex', storage, now)

    first.completeStep('initialize')
    expect(second.progress.value.completedStepIds).toEqual(['initialize'])

    second.clearAll()
    expect(first.progress.value.completedStepIds).toEqual([])
    expect(second.progress.value.completedStepIds).toEqual([])
  })

  it('unsubscribes a wrapper when its Vue effect scope stops', () => {
    const storage = createStorage()
    const scope = effectScope()
    const scoped = scope.run(() => useGuideV2Progress('codex', storage, now))!
    const active = useGuideV2Progress('codex', storage, now)

    scope.stop()
    active.completeStep('initialize')

    expect(scoped.progress.value.completedStepIds).toEqual([])
    expect(active.progress.value.completedStepIds).toEqual(['initialize'])
  })
})
