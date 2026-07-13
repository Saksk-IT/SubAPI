# Image Playground New-Tab Launcher Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an admin-controlled image-generation entry switch and replace the embedded iframe with a current-page key picker that securely launches the React playground in a new tab.

**Architecture:** A public opt-out flag controls the sidebar and compatibility route. A global Vue launcher owns key selection and a short-lived popup bridge; the React tab exchanges its opener for a one-time MessagePort, receives the key only in memory, then severs `window.opener`.

**Tech Stack:** Go/Gin, Vue 3, Pinia, Vitest, React 19, TypeScript, MessageChannel, Vite, Docker.

---

### Task 1: Backend public feature switch

**Files:**
- Modify: `backend/internal/service/domain_constants.go`
- Modify: `backend/internal/service/setting_parse.go`
- Modify: `backend/internal/service/setting_update.go`
- Modify: `backend/internal/service/settings_view.go`
- Modify: `backend/internal/service/setting_public.go`
- Modify: `backend/internal/handler/dto/settings.go`
- Modify: `backend/internal/handler/admin/setting_handler_update.go`
- Modify: `backend/internal/handler/admin/setting_handler_audit.go`
- Test: `backend/internal/handler/dto/public_settings_injection_schema_test.go`
- Test: `backend/internal/server/api_contract_test.go`
- Test: `backend/internal/service/setting_service_public_test.go`
- Test: `backend/internal/service/setting_service_update_test.go`
- Create: `backend/internal/handler/admin/setting_handler_image_generation_test.go`

- [ ] **Step 1: Write failing contract tests**

Require `image_generation_enabled` in admin settings, public settings and injected settings, with a default value of `true`:

```go
require.Equal(t, true, payload["image_generation_enabled"])
require.Contains(t, publicSettingsJSON, `"image_generation_enabled":true`)
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run:

```bash
cd backend
go test ./internal/handler/dto ./internal/server -run 'PublicSettings|Settings' -count=1
```

Expected: FAIL because the field is absent.

- [ ] **Step 3: Implement the setting end to end**

Add the constant and typed fields:

```go
const SettingKeyImageGenerationEnabled = "image_generation_enabled"

ImageGenerationEnabled bool  `json:"image_generation_enabled"`
ImageGenerationEnabled *bool `json:"image_generation_enabled"`
```

Parse missing values as `true`, persist only when the optional update field is present, include the value in admin/public/injected views, and add the field name to audit diffs.

- [ ] **Step 4: Verify GREEN**

Run the focused tests, then:

```bash
cd backend
go test ./internal/service ./internal/handler/admin ./internal/handler/dto ./internal/server -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the backend switch**

```bash
git add backend/internal/service backend/internal/handler backend/internal/server
git commit -m "feat: add image generation entry switch"
```

### Task 2: Frontend flag registry and admin toggle

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/admin/settings.ts`
- Modify: `frontend/src/stores/app.ts`
- Modify: `frontend/src/utils/featureFlags.ts`
- Create: `frontend/src/utils/__tests__/featureFlags.spec.ts`
- Modify: `frontend/src/views/admin/SettingsView.vue`
- Modify: `frontend/src/views/admin/__tests__/SettingsView.spec.ts`
- Modify: `frontend/src/i18n/locales/zh/admin/settings.ts`
- Modify: `frontend/src/i18n/locales/en/admin/settings.ts`

- [ ] **Step 1: Write failing frontend flag and settings tests**

```ts
expect(FeatureFlags.imageGeneration).toEqual({
  key: 'image_generation_enabled',
  mode: 'opt-out',
  label: 'Image Generation',
})
expect(wrapper.text()).toContain('生图功能')
expect(updateSettings).toHaveBeenCalledWith(expect.objectContaining({
  image_generation_enabled: false,
}))
```

- [ ] **Step 2: Verify RED**

```bash
cd frontend
pnpm exec vitest run src/utils/__tests__/featureFlags.spec.ts src/views/admin/__tests__/SettingsView.spec.ts
```

Expected: FAIL because the flag, form field and toggle are missing.

- [ ] **Step 3: Implement types, defaults, registry and UI**

Add `image_generation_enabled: boolean` to public/admin types, use `true` in fallback public settings, register an opt-out flag, add a feature card and bind the setting form/save payload:

```ts
imageGeneration: defineFlag({
  key: 'image_generation_enabled',
  mode: 'opt-out',
  label: 'Image Generation',
})
```

- [ ] **Step 4: Verify GREEN and type safety**

```bash
cd frontend
pnpm exec vitest run src/utils/__tests__/featureFlags.spec.ts src/views/admin/__tests__/SettingsView.spec.ts
pnpm run typecheck
```

Expected: PASS.

### Task 3: Pure popup bridge state machine

**Files:**
- Create: `frontend/src/features/imagePlayground/popupBridge.ts`
- Create: `frontend/src/features/imagePlayground/__tests__/popupBridge.spec.ts`
- Modify: `frontend/src/features/imagePlayground/bridge.ts`
- Modify: `frontend/src/features/imagePlayground/__tests__/bridge.spec.ts`

- [ ] **Step 1: Write failing protocol tests**

Cover popup blocking, exact ready source/origin/nonce/schema checks, `connected` before configure, exact configured ACK, timeout, popup close, duplicate and out-of-order messages, independent concurrent sessions, and cleanup. The intended API is:

```ts
const session = openImagePlaygroundPopup({
  apiKey,
  apiKeyId,
  apiKeyName,
  storageScope,
  locale,
  theme,
  timeoutMs: 8_000,
  openWindow: window.open.bind(window),
})
await session.configured
session.abort()
```

Assert that the URL and window name never contain the key:

```ts
expect(openWindow).toHaveBeenCalledWith('/image-playground/', expect.stringMatching(/^sub2api-image-playground:/))
expect(JSON.stringify(openWindow.mock.calls)).not.toContain(apiKey)
```

- [ ] **Step 2: Verify RED**

```bash
cd frontend
pnpm exec vitest run src/features/imagePlayground/__tests__/popupBridge.spec.ts src/features/imagePlayground/__tests__/bridge.spec.ts
```

Expected: FAIL because `popupBridge.ts` and connected messages do not exist.

- [ ] **Step 3: Implement the immutable session state machine**

Use one closure per launch with its own nonce, `WindowProxy`, timer, MessageChannel and requestId. Send configure only after an exact connected message, resolve only after the matching configured ACK, and reject with stable error codes:

```ts
type PopupLaunchErrorCode =
  | 'popup_blocked'
  | 'connection_timeout'
  | 'popup_closed'
  | 'configuration_failed'
```

Never store the key in module globals, Pinia or a browser storage API.

- [ ] **Step 4: Verify GREEN**

Run the two focused bridge suites and ensure all listeners, timers and ports are closed in success, error and abort paths.

### Task 4: Global Vue launcher and compatibility route

**Files:**
- Create: `frontend/src/stores/imageGenerationLauncher.ts`
- Modify: `frontend/src/stores/index.ts`
- Create: `frontend/src/components/image-generation/ImageGenerationLauncher.vue`
- Create: `frontend/src/components/image-generation/__tests__/ImageGenerationLauncher.spec.ts`
- Modify: `frontend/src/App.vue`
- Replace: `frontend/src/views/user/ImageGenerationView.vue`
- Replace: `frontend/src/views/user/__tests__/ImageGenerationView.spec.ts`

- [ ] **Step 1: Write failing component tests**

Cover loading, retry, no-key, selected key, blocked popup, connecting, success close, timeout, abort, feature disable and user change. Assert the store contains no key field:

```ts
expect(useImageGenerationLauncherStore()).toMatchObject({ isOpen: false })
expect(useImageGenerationLauncherStore()).not.toHaveProperty('apiKey')
```

- [ ] **Step 2: Verify RED**

```bash
cd frontend
pnpm exec vitest run src/components/image-generation/__tests__/ImageGenerationLauncher.spec.ts src/views/user/__tests__/ImageGenerationView.spec.ts
```

Expected: FAIL because the global launcher is missing and the old view still embeds an iframe.

- [ ] **Step 3: Implement the global modal**

Mount one launcher in `App.vue`; the Pinia store exposes only:

```ts
export const useImageGenerationLauncherStore = defineStore('image-generation-launcher', () => {
  const isOpen = ref(false)
  const open = () => { isOpen.value = true }
  const close = () => { isOpen.value = false }
  return { isOpen, open, close }
})
```

The component keeps eligible keys and the active bridge session locally, uses a Teleport modal with focus/escape handling, and synchronously opens the popup from the confirm button.

- [ ] **Step 4: Replace the old route view with a compatibility launcher**

On mount, call `launcher.open()` and replace the route with `/admin/dashboard` for admins or `/dashboard` for users. Do not render an iframe.

- [ ] **Step 5: Verify GREEN and mobile semantics**

Run the focused suites and verify the modal uses a scrollable max-height, full-width mobile controls and at least 44px touch targets.

### Task 5: Sidebar action and route guard

**Files:**
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/components/layout/__tests__/AppSidebar.spec.ts`
- Modify: `frontend/src/router/meta.d.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/router/__tests__/image-generation-route.spec.ts`
- Modify: `frontend/src/router/__tests__/feature-access.spec.ts`

- [ ] **Step 1: Write failing navigation tests**

Require sidebar visibility to combine the global flag and key access, require click prevention plus `launcher.open()`, and require explicit false to redirect the legacy route without affecting `/v1`:

```ts
expect(FeatureFlags.imageGeneration.mode).toBe('opt-out')
expect(event.preventDefault).toHaveBeenCalled()
expect(launcher.open).toHaveBeenCalledOnce()
```

- [ ] **Step 2: Verify RED**

Run the three focused sidebar/router suites.

- [ ] **Step 3: Implement action navigation and guard**

Add `action?: 'image-generation'` to `NavItem`, use a combined flag getter, prevent router navigation for the action, close the mobile sidebar, add `requiresImageGeneration` route meta, load public settings in the guard and redirect only when the loaded value is explicitly `false`.

- [ ] **Step 4: Verify GREEN**

Run the focused suites and `pnpm run typecheck`.

### Task 6: React opener bridge

**Files:**
- Modify: `frontend/image-playground/src/lib/sub2apiBridge.ts`
- Modify: `frontend/image-playground/src/lib/sub2apiBridge.test.ts`
- Modify: `frontend/image-playground/src/main.tsx`

- [ ] **Step 1: Write failing child bridge tests**

Require direct access rejection, opener-only source checking, exact connect schema, one transferred port, clearing `window.name`, nulling `window.opener`, connected-before-configure, configure-once, safe error ACKs, and no secret in ready/connected/error messages.

- [ ] **Step 2: Verify RED**

```bash
npm --prefix frontend/image-playground test -- src/lib/sub2apiBridge.test.ts
```

Expected: FAIL because the child currently communicates through `window.parent` and has no connected state.

- [ ] **Step 3: Implement opener-to-port handoff**

Choose only a non-null opener as the host, post the secret-free ready message to the exact origin, accept one valid transferred port, clear name/opener, post connected through the port, accept one valid configure request, and close the port after the result ACK.

- [ ] **Step 4: Verify GREEN and the managed-runtime suites**

```bash
npm --prefix frontend/image-playground test -- src/lib/sub2apiBridge.test.ts src/lib/managedMode.test.ts src/lib/managedAssets.test.ts
```

Expected: PASS.

### Task 7: Full verification, review and delivery

**Files:**
- Verification only; no new production file is planned in this task.

- [ ] **Step 1: Run complete automated verification**

```bash
make test-frontend
cd frontend && pnpm run typecheck && pnpm run build
cd ../backend && go test -tags embed ./internal/web -count=1
go test ./internal/service ./internal/handler/admin ./internal/server -count=1
```

Expected: all commands PASS and both Vue and React indexes exist.

- [ ] **Step 2: Rebuild and verify the local development container**

```bash
cd deploy
docker compose -f docker-compose.dev.yml build sub2api
docker compose -f docker-compose.dev.yml up -d --no-deps --force-recreate sub2api
curl -fsS http://127.0.0.1:8080/health
curl -fsS http://127.0.0.1:8080/image-playground/ | grep 'GPT Image Playground'
```

Expected: healthy service and React index response.

- [ ] **Step 3: Browser QA**

Verify desktop and 390px mobile flows: toggle on/off, sidebar visibility, modal loading/no-key/error, key selection, new tab, settings and Agent controls, popup-blocked error, legacy-route redirect, and console/storage/URL secret checks. Do not send a paid generation request without explicit need.

- [ ] **Step 4: Parallel code/security/consistency review**

Resolve all CRITICAL/HIGH findings and reasonable MEDIUM findings. Confirm no API key appears in the diff, URL construction, Pinia state, local/session storage, IndexedDB, export code or logs.

- [ ] **Step 5: Commit and push only task files**

```bash
git diff --check
git status --short
git add -- \
  backend/internal/service/domain_constants.go \
  backend/internal/service/setting_parse.go \
  backend/internal/service/setting_update.go \
  backend/internal/service/settings_view.go \
  backend/internal/service/setting_public.go \
  backend/internal/service/setting_service_public_test.go \
  backend/internal/service/setting_service_update_test.go \
  backend/internal/handler/dto/settings.go \
  backend/internal/handler/dto/public_settings_injection_schema_test.go \
  backend/internal/handler/admin/setting_handler_update.go \
  backend/internal/handler/admin/setting_handler_audit.go \
  backend/internal/handler/admin/setting_handler_image_generation_test.go \
  backend/internal/server/api_contract_test.go \
  frontend/src/types/index.ts \
  frontend/src/api/admin/settings.ts \
  frontend/src/stores/app.ts \
  frontend/src/stores/index.ts \
  frontend/src/stores/imageGenerationLauncher.ts \
  frontend/src/utils/featureFlags.ts \
  frontend/src/utils/__tests__/featureFlags.spec.ts \
  frontend/src/views/admin/SettingsView.vue \
  frontend/src/views/admin/__tests__/SettingsView.spec.ts \
  frontend/src/i18n/locales/zh/admin/settings.ts \
  frontend/src/i18n/locales/en/admin/settings.ts \
  frontend/src/features/imagePlayground/bridge.ts \
  frontend/src/features/imagePlayground/popupBridge.ts \
  frontend/src/features/imagePlayground/__tests__/bridge.spec.ts \
  frontend/src/features/imagePlayground/__tests__/popupBridge.spec.ts \
  frontend/src/components/image-generation/ImageGenerationLauncher.vue \
  frontend/src/components/image-generation/__tests__/ImageGenerationLauncher.spec.ts \
  frontend/src/components/layout/AppSidebar.vue \
  frontend/src/components/layout/__tests__/AppSidebar.spec.ts \
  frontend/src/App.vue \
  frontend/src/views/user/ImageGenerationView.vue \
  frontend/src/views/user/__tests__/ImageGenerationView.spec.ts \
  frontend/src/router/meta.d.ts \
  frontend/src/router/index.ts \
  frontend/src/router/__tests__/image-generation-route.spec.ts \
  frontend/src/router/__tests__/feature-access.spec.ts \
  frontend/image-playground/src/lib/sub2apiBridge.ts \
  frontend/image-playground/src/lib/sub2apiBridge.test.ts \
  frontend/image-playground/src/main.tsx
git add -f -- docs/superpowers/plans/2026-07-13-image-playground-new-tab-launcher.md
git commit -m "feat: launch image playground in a new tab"
git push origin main
```
