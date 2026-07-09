import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppHeader.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AppHeader v0.1.149 merge', () => {
  it('contains no unresolved merge markers', () => {
    expect(componentSource).not.toMatch(/^(<<<<<<<|=======|>>>>>>>)/m)
  })

  it('keeps the locally added scheduled rate notice', () => {
    expect(componentSource).toContain('v-if="rateScheduleNotice"')
    expect(componentSource).toContain('const rateScheduleActivePercent = ref<number | null>(null)')
    expect(componentSource).toContain('const rateScheduleNotice = computed(() => {')
    expect(componentSource).toContain('refreshRateScheduleNotice()')
  })

  it('includes the upstream available, frozen, and total balance breakdown', () => {
    expect(componentSource).toContain('const availableBalance = computed(')
    expect(componentSource).toContain('const frozenBalance = computed(')
    expect(componentSource).toContain('const totalBalance = computed(')
    expect(componentSource).toContain('{{ balanceAvailableText }}')
    expect(componentSource).toContain('{{ balanceFrozenText }}')
    expect(componentSource).toContain('{{ balanceTotalText }}')
  })
})
