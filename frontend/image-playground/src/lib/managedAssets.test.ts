import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it, vi } from 'vitest'

import {
  prepareManagedServiceWorker,
  removeManagedServiceWorker,
} from './managedServiceWorker'

const dir = dirname(fileURLToPath(import.meta.url))

describe('managed static runtime', () => {
  it('does not load external font styles', () => {
    const css = readFileSync(resolve(dir, '../index.css'), 'utf8')
    expect(css).not.toMatch(/@import\s+url\(['"]https?:\/\//)
  })

  it('pins the DOM sanitizer above the audited vulnerable range', () => {
    const packageJson = JSON.parse(readFileSync(resolve(dir, '../../package.json'), 'utf8'))

    expect(packageJson.overrides?.dompurify).toBe('3.4.12')
  })

  it('uses parent-controlled class dark mode', () => {
    const tailwind = readFileSync(resolve(dir, '../../tailwind.config.js'), 'utf8')
    const css = readFileSync(resolve(dir, '../index.css'), 'utf8')
    expect(tailwind).toContain("darkMode: 'class'")
    expect(css).not.toContain('@media (prefers-color-scheme: dark)')
  })

  it('builds directly into the backend image-playground asset directory', () => {
    const vite = readFileSync(resolve(dir, '../../vite.config.ts'), 'utf8')

    expect(vite).toContain("base: '/image-playground/'")
    expect(vite).toContain("outDir: '../../backend/internal/web/dist/image-playground'")
    expect(vite).toContain('emptyOutDir: true')
    expect(vite).toContain('port: 5174')
    expect(vite).toContain('strictPort: true')
  })

  it('opens IndexedDB with the configured user-scoped database name', () => {
    const db = readFileSync(resolve(dir, './db.ts'), 'utf8')
    expect(db).toContain('indexedDB.open(getManagedDatabaseName(), DB_VERSION)')
    expect(db).not.toContain("const DB_NAME = 'gpt-image-playground'")
  })

  it('partitions persisted UI state with the same managed user scope', () => {
    const store = readFileSync(resolve(dir, '../store.ts'), 'utf8')

    expect(store).toContain('name: getManagedStorageName()')
  })

  it('does not register or enumerate service workers from the entrypoint', () => {
    const main = readFileSync(resolve(dir, '../main.tsx'), 'utf8')
    expect(main).not.toContain('serviceWorker.register')
    expect(main).not.toContain('serviceWorker.getRegistrations')
    expect(main).toContain('prepareManagedServiceWorker')
  })

  it('does not ship a service worker or PWA registration surface', () => {
    const html = readFileSync(resolve(dir, '../../index.html'), 'utf8')
    const header = readFileSync(resolve(dir, '../components/Header.tsx'), 'utf8')

    expect(existsSync(resolve(dir, '../../public/sw.js'))).toBe(false)
    expect(existsSync(resolve(dir, '../../public/manifest.webmanifest'))).toBe(false)
    expect(html).not.toContain('rel="manifest"')
    expect(html).not.toContain('apple-mobile-web-app')
    expect(html).not.toContain('apple-touch-icon')
    expect(header).not.toContain('beforeinstallprompt')
    expect(header).not.toContain('安装为应用')
  })

  it('waits for managed configuration before importing the app and store', () => {
    const main = readFileSync(resolve(dir, '../main.tsx'), 'utf8')
    expect(main).not.toContain("import App from './App'")
    expect(main).toContain("import('./App')")
    expect(main).toContain("import('./store')")
    expect(main).toContain('runAfterWindowLoad')
    expect(main).toContain('startSub2ApiBridge')
    expect(main).toContain('requireManagedRuntime')
    expect(main).toContain('请从 Sub2API 侧边栏进入生图功能')
    expect(main).toContain('enforceManagedSettings(appModules.useStore.getState().settings)')
  })

  it('lets the managed entrypoint initialize the store exactly once', () => {
    const app = readFileSync(resolve(dir, '../App.tsx'), 'utf8')
    const main = readFileSync(resolve(dir, '../main.tsx'), 'utf8')

    expect(app).not.toContain('initStore')
    expect(app).not.toContain('buildSettingsFromUrlParams')
    expect(app).not.toContain('loadCustomProviderSettingsFromUrl')
    expect(main.match(/appModules\.initStore\(\)/g)).toHaveLength(1)
  })

  it('handles presentation-only configure updates without reinitializing or rerendering', () => {
    const main = readFileSync(resolve(dir, '../main.tsx'), 'utf8')

    expect(main).toContain('hasSameManagedCredentialIdentity(config)')
    expect(main).toContain('updateManagedPresentationConfig(config)')
  })

  it('does not call GitHub from the version-check hook', () => {
    const hook = readFileSync(resolve(dir, '../hooks/useVersionCheck.ts'), 'utf8')
    expect(hook).not.toContain('api.github.com')
    expect(hook).not.toMatch(/\bfetch\(/)
  })

  it('routes all authenticated OpenAI requests through managedFetch', () => {
    for (const file of ['agentApi.ts', 'openaiCompatibleImageApi.ts']) {
      const source = readFileSync(resolve(dir, file), 'utf8')
      expect(source).toContain("import { managedFetch } from './managedMode'")
      expect(source).not.toMatch(/await fetch\(buildApiUrl/)
    }
  })

  it('only unregisters a stale service worker inside the image playground scope', async () => {
    const unregister = vi.fn().mockResolvedValue(true)
    const getRegistration = vi.fn().mockResolvedValue({
      scope: 'https://sub2api.example/image-playground/',
      unregister,
    })
    const getRegistrations = vi.fn()

    await removeManagedServiceWorker({ getRegistration, getRegistrations } as any, '/image-playground/')

    expect(getRegistration).toHaveBeenCalledWith('/image-playground/')
    expect(getRegistrations).not.toHaveBeenCalled()
    expect(unregister).toHaveBeenCalledOnce()
  })

  it('does not unregister service workers outside the managed scope', async () => {
    const unregister = vi.fn()
    const container = {
      getRegistration: vi.fn().mockResolvedValue({
        scope: 'https://sub2api.example/',
        unregister,
      }),
    }

    await removeManagedServiceWorker(container as any, '/image-playground/')

    expect(unregister).not.toHaveBeenCalled()
  })

  it('reloads before bridge startup when a legacy playground worker still controls the page', async () => {
    const unregister = vi.fn().mockResolvedValue(true)
    const reload = vi.fn()
    const storage = new Map<string, string>()
    const sessionStorage = {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
    }
    const container = {
      controller: { scriptURL: 'https://sub2api.example/image-playground/sw.js' },
      getRegistration: vi.fn().mockResolvedValue({
        scope: 'https://sub2api.example/image-playground/',
        unregister,
      }),
    }

    await expect(prepareManagedServiceWorker(container, '/image-playground/', {
      location: { origin: 'https://sub2api.example', reload },
      sessionStorage,
    })).resolves.toBe('reloading')

    expect(unregister).toHaveBeenCalledOnce()
    expect(reload).toHaveBeenCalledOnce()
    expect([...storage.values()]).toEqual(['1'])
  })

  it('fails closed instead of looping when a retired worker still controls the reloaded page', async () => {
    const reload = vi.fn()
    const sessionStorage = {
      getItem: vi.fn().mockReturnValue('1'),
      setItem: vi.fn(),
      removeItem: vi.fn(),
    }
    const container = {
      controller: { scriptURL: 'https://sub2api.example/image-playground/sw.js' },
      getRegistration: vi.fn().mockResolvedValue(undefined),
    }

    await expect(prepareManagedServiceWorker(container, '/image-playground/', {
      location: { origin: 'https://sub2api.example', reload },
      sessionStorage,
    })).rejects.toThrow('still controlled')

    expect(reload).not.toHaveBeenCalled()
  })

  it('reloads before bridge startup when a root service worker controls the child page', async () => {
    const reload = vi.fn()
    const container = {
      controller: { scriptURL: 'https://sub2api.example/sw.js' },
      getRegistration: vi.fn().mockResolvedValue(undefined),
    }

    await expect(prepareManagedServiceWorker(container, '/image-playground/', {
      location: { origin: 'https://sub2api.example', reload },
      sessionStorage: { getItem: vi.fn(), setItem: vi.fn(), removeItem: vi.fn() },
    })).resolves.toBe('reloading')

    expect(reload).toHaveBeenCalledOnce()
  })

  it('clears the reload marker when no playground worker controls the page', async () => {
    const removeItem = vi.fn()
    const container = {
      controller: null,
      getRegistration: vi.fn().mockResolvedValue(undefined),
    }

    await expect(prepareManagedServiceWorker(container, '/image-playground/', {
      location: { origin: 'https://sub2api.example', reload: vi.fn() },
      sessionStorage: { getItem: vi.fn(), setItem: vi.fn(), removeItem },
    })).resolves.toBe('ready')

    expect(removeItem).toHaveBeenCalledOnce()
  })
})
