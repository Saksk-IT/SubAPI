import { readFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDirectory = dirname(fileURLToPath(import.meta.url))
const stylesDirectory = resolve(testDirectory, '../styles')

const ruleHasMinimumHeight = (source: string, selector: string, height: string): boolean => {
  const rulePattern = /([^{}]+)\{([^{}]*)\}/g

  return Array.from(source.matchAll(rulePattern)).some(([, selectorList, declarations]) => {
    const selectors = selectorList.split(',').map((candidate) => candidate.trim())
    return selectors.includes(selector) && declarations.includes(`min-height: ${height}`)
  })
}

describe('Guide V2 responsive touch targets', () => {
  it('keeps mobile header and copy controls at least 44px tall', async () => {
    const [layoutCss, responsiveCss] = await Promise.all([
      readFile(resolve(stylesDirectory, 'layout.css'), 'utf8'),
      readFile(resolve(stylesDirectory, 'responsive.css'), 'utf8'),
    ])
    const mobileCss = responsiveCss.split('@media (max-width: 620px) {')[1]

    expect(mobileCss).toBeDefined()
    expect(ruleHasMinimumHeight(mobileCss, '.guide-v2-header__brand', '44px')).toBe(true)
    expect(ruleHasMinimumHeight(mobileCss, '.guide-v2-code__toolbar button', '44px')).toBe(true)
    expect(ruleHasMinimumHeight(layoutCss, '.guide-v2-header__nav a', '44px')).toBe(true)
  })
})
