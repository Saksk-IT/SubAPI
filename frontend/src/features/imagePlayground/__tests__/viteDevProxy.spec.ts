import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import packageJson from '../../../../package.json'

const testDirectory = dirname(fileURLToPath(import.meta.url))
const viteConfigSource = readFileSync(resolve(testDirectory, '../../../../vite.config.ts'), 'utf8')

describe('image playground development proxy', () => {
  it('provides one command that starts both frontend development servers', () => {
    expect(packageJson.scripts['dev:all']).toBe('node scripts/dev-all.mjs')
  })

  it('keeps the image playground in the standard frontend build output', () => {
    expect(packageJson.scripts.build).toContain('build:vue')
    expect(packageJson.scripts.build).toContain('npm --prefix image-playground run build')
  })
  it('keeps the React dev server behind the Vue same-origin path', () => {
    expect(viteConfigSource).toContain('VITE_IMAGE_PLAYGROUND_DEV_TARGET')
    expect(viteConfigSource).toContain("'/image-playground'")
    expect(viteConfigSource).toContain('target: imagePlaygroundDevTarget')
    expect(viteConfigSource).toContain('ws: true')
  })
})
