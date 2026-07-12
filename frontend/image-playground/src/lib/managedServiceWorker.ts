interface RegistrationLike {
  scope: string
  unregister(): Promise<boolean>
}

interface ServiceWorkerContainerLike {
  controller?: { scriptURL: string } | null
  getRegistration(scope?: string): Promise<RegistrationLike | undefined>
}

interface ManagedServiceWorkerEnvironment {
  location: {
    origin: string
    reload(): void
  }
  sessionStorage: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>
}

const RETIRE_RELOAD_MARKER = 'sub2api:image-playground:retire-service-worker'

export async function removeManagedServiceWorker(container: ServiceWorkerContainerLike, baseUrl: string) {
  const registration = await container.getRegistration(baseUrl)
  if (!registration) return
  const expectedPath = new URL(baseUrl, registration.scope).pathname
  const scopePath = new URL(registration.scope).pathname
  if (!scopePath.startsWith(expectedPath)) return
  await registration.unregister()
}

export async function prepareManagedServiceWorker(
  container: ServiceWorkerContainerLike,
  baseUrl: string,
  environment: ManagedServiceWorkerEnvironment = window,
): Promise<'ready' | 'reloading'> {
  await removeManagedServiceWorker(container, baseUrl)

  if (!container.controller) {
    environment.sessionStorage.removeItem(RETIRE_RELOAD_MARKER)
    return 'ready'
  }

  if (environment.sessionStorage.getItem(RETIRE_RELOAD_MARKER) === '1') {
    throw new Error('Image playground is still controlled by a retired service worker')
  }

  environment.sessionStorage.setItem(RETIRE_RELOAD_MARKER, '1')
  environment.location.reload()
  return 'reloading'
}
