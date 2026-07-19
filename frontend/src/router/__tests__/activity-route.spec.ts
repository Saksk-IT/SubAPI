import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const routerPath = resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts')
const routerSource = readFileSync(routerPath, 'utf8')

describe('user activity route', () => {
  it('keeps the activity center authenticated and independent from payment settings', () => {
    const routeStart = routerSource.indexOf("path: '/activities'")
    const nextRoute = routerSource.indexOf("path: '/redeem'", routeStart)
    const routeBlock = routerSource.slice(routeStart, nextRoute)

    expect(routeStart).toBeGreaterThan(-1)
    expect(routeBlock).toContain("component: () => import('@/views/user/ActivitiesView.vue')")
    expect(routeBlock).toContain('requiresAuth: true')
    expect(routeBlock).not.toContain('requiresPayment: true')
  })
})
