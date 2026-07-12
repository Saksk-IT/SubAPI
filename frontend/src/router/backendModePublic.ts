const BACKEND_MODE_ALLOWED_PATHS = [
  '/login',
  '/key-usage',
  '/setup',
  '/payment/result',
  '/payment/airwallex',
  '/legal',
  '/registration-key-guide',
  '/codex-guide',
  '/claude-code-guide',
  '/open-code-guide',
  '/open-claw-guide',
  '/mobile-guide',
  '/image-guide',
] as const

const BACKEND_MODE_CALLBACK_PATHS = [
  '/auth/callback',
  '/auth/linuxdo/callback',
  '/auth/dingtalk/callback',
  '/auth/dingtalk/email-completion',
  '/auth/oidc/callback',
  '/auth/wechat/callback',
  '/auth/wechat/payment/callback',
] as const

const BACKEND_MODE_PENDING_AUTH_PATHS = ['/register', '/email-verify'] as const

export const isGuideV2Path = (path: string): boolean =>
  path === '/guides/v2' || path.startsWith('/guides/v2/')

export const isBackendModePublicRouteAllowed = (
  path: string,
  hasPendingAuthSession: boolean,
): boolean => {
  if (isGuideV2Path(path)) return true

  if (
    BACKEND_MODE_ALLOWED_PATHS.some(
      (allowedPath) => path === allowedPath || path.startsWith(allowedPath),
    )
  ) {
    return true
  }

  if (BACKEND_MODE_CALLBACK_PATHS.some((callbackPath) => path === callbackPath)) {
    return true
  }

  return (
    hasPendingAuthSession &&
    BACKEND_MODE_PENDING_AUTH_PATHS.some((allowedPath) => path === allowedPath)
  )
}
