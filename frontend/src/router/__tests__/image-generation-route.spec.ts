import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const routerPath = resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts')
const routerSource = readFileSync(routerPath, 'utf8')

describe('image generation route', () => {
  it('registers an authenticated lazy-loaded user route', () => {
    expect(routerSource).toContain("path: '/image-generation'")
    expect(routerSource).toContain("name: 'ImageGeneration'")
    expect(routerSource).toContain("component: () => import('@/views/user/ImageGenerationView.vue')")
    expect(routerSource).toMatch(/path: '\/image-generation'[\s\S]*?requiresAuth: true/)
  })
})
