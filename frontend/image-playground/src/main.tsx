import 'core-js/actual/array/at'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import 'streamdown/styles.css'
import 'katex/dist/katex.min.css'
import './index.css'
import { DEFAULT_SETTINGS } from './lib/apiProfiles'
import {
  activateManagedConfig,
  applyManagedPresentation,
  clearManagedConfig,
  enforceManagedSettings,
  getManagedSnapshot,
  hasSameManagedCredentialIdentity,
  redactSettingsSecrets,
  requireManagedRuntime,
  updateManagedPresentationConfig,
} from './lib/managedMode'
import { prepareManagedServiceWorker } from './lib/managedServiceWorker'
import { runAfterWindowLoad, startSub2ApiBridge } from './lib/sub2apiBridge'
import { installMobileViewportGuards } from './lib/viewport'

installMobileViewportGuards()

const root = createRoot(document.getElementById('root')!)
const releaseManagedRuntime = requireManagedRuntime()
let bridge: ReturnType<typeof startSub2ApiBridge> | null = null
let appModules: Awaited<ReturnType<typeof loadAppModules>> | null = null
let appInitialized = false

function StatusView(props: { title: string, detail: string }) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-gray-50 px-6 text-gray-900 dark:bg-gray-950 dark:text-gray-100">
      <section className="w-full max-w-md rounded-2xl border border-gray-200 bg-white p-6 text-center shadow-sm dark:border-white/10 dark:bg-gray-900">
        <h1 className="text-lg font-semibold">{props.title}</h1>
        <p className="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">{props.detail}</p>
      </section>
    </main>
  )
}

function renderStatus(title: string, detail: string) {
  root.render(<StatusView title={title} detail={detail} />)
}

async function loadAppModules() {
  const [app, store] = await Promise.all([import('./App'), import('./store')])
  return { App: app.default, ...store }
}

async function configureApp(config: Parameters<typeof activateManagedConfig>[0]) {
  const currentScope = getManagedSnapshot()?.config.storageScope
  if (currentScope && currentScope !== config.storageScope) {
    clearManagedConfig()
    renderStatus('正在切换账号', '正在重新载入当前用户的独立生图空间…')
    window.location.reload()
    return
  }

  if (hasSameManagedCredentialIdentity(config)) {
    updateManagedPresentationConfig(config)
    applyManagedPresentation(config)
    return
  }

  const previousSettings = appModules?.useStore.getState().settings ?? DEFAULT_SETTINGS
  activateManagedConfig(config, previousSettings)
  applyManagedPresentation(config)
  appModules ??= await loadAppModules()
  const settings = enforceManagedSettings(appModules.useStore.getState().settings)
  appModules.useStore.getState().setSettings(settings)

  if (!appInitialized) {
    await appModules.initStore()
    appInitialized = true
  }

  root.render(
    <StrictMode>
      <appModules.App />
    </StrictMode>,
  )
}

function clearApp() {
  clearManagedConfig()
  if (appModules) {
    const state = appModules.useStore.getState()
    state.setSettings(redactSettingsSecrets(state.settings))
  }
  root.render(<StatusView title="连接已断开" detail="API 密钥已从生图页面内存清除，请重新选择密钥后继续。" />)
}

async function startManagedBridge() {
  if ('serviceWorker' in navigator) {
    const serviceWorkerState = await prepareManagedServiceWorker(navigator.serviceWorker, import.meta.env.BASE_URL)
    if (serviceWorkerState === 'reloading') {
      renderStatus('正在清理旧离线缓存', '完成后页面会自动重新连接 Sub2API。')
      return
    }
  }

  bridge = startSub2ApiBridge({
    window: window as any,
    onConfigure: configureApp,
    onClear: clearApp,
  })

  if (bridge.mode === 'direct') {
    renderStatus('请从 Sub2API 侧边栏进入生图功能', '此页面需要由 Sub2API 安全传入当前用户选择的 API 密钥，不能直接独立使用。')
  }
}

renderStatus('正在连接 Sub2API', '正在等待安全配置，请稍候…')
const cancelLoadStart = runAfterWindowLoad(document, window, () => {
  void startManagedBridge().catch((error) => {
    console.error('Failed to prepare the managed image playground:', error)
    renderStatus('无法安全启动生图功能', '旧版离线缓存仍在控制页面，请关闭此标签页后重新从侧边栏进入。')
  })
})

window.addEventListener('beforeunload', () => {
  cancelLoadStart()
  bridge?.dispose()
  clearManagedConfig()
  releaseManagedRuntime()
}, { once: true })
