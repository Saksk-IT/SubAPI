import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar v0.1.149 merge', () => {
  it('contains no unresolved merge markers', () => {
    expect(componentSource).not.toMatch(/^(<<<<<<<|=======|>>>>>>>)/m)
  })

  it('adds a gated image generation entry without replacing batch image', () => {
    expect(componentSource).toContain("import { useImageGenerationAccess } from '@/composables/useImageGenerationAccess'")
    expect(componentSource).toContain("{ path: '/image-generation', label: t('nav.imageGeneration'), icon: ImageGenerationIcon")
    expect(componentSource).toContain('featureFlag: flagImageGenerationAccess')
    expect(componentSource).toContain('refreshImageGenerationAccess(true)')
    expect(componentSource).toContain("{ path: '/batch-image', label: t('nav.batchImage'), icon: BatchImageIcon")

    expect(componentSource.indexOf("path: '/image-generation'")).toBeLessThan(
      componentSource.indexOf("path: '/batch-image'")
    )
  })

  it('adds the upstream batch image entry behind its access gate', () => {
    expect(componentSource).toContain("import { useBatchImageAccess } from '@/composables/useBatchImageAccess'")
    expect(componentSource).toContain('const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()')
    expect(componentSource).toContain("{ path: '/batch-image', label: t('nav.batchImage'), icon: BatchImageIcon")
    expect(componentSource).toContain('featureFlag: flagBatchImageAccess')
  })

  it('keeps the locally added admin data, activity, and channel pages', () => {
    expect(componentSource).toContain("{ path: '/admin/data-dashboard', label: t('nav.dataDashboard')")
    expect(componentSource).toContain("{ path: '/admin/channels/status', label: t('nav.channelStatus')")
    expect(componentSource).toContain("path: '/admin/activities'")
    expect(componentSource).toContain("{ path: '/admin/activities/first-recharge', label: t('nav.firstRechargeManagement')")
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar custom menu new-tab behavior', () => {
  it('keeps normal custom menu items on router links by default', () => {
    expect(componentSource).toContain("openInNewTab: item.open_in_new_tab === true")
    expect(componentSource).toContain("openInNewTab: cm.open_in_new_tab === true")
  })

  it('opens enabled custom menu items with embedded query forwarding', () => {
    expect(componentSource).toContain('@click.capture="handleMenuItemClick(item, $event)"')
    expect(componentSource).toContain('function resolveMenuItemOpenUrl(item: NavItem): string')
    expect(componentSource).toContain('buildEmbeddedUrl(')
    expect(componentSource).toContain("window.open(resolveMenuItemOpenUrl(item), '_blank', 'noopener,noreferrer')")
  })
})
