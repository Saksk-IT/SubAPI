# Static Guides V2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在完全冻结 7 个 V1 教程入口的前提下，交付 `/guides/v2` 教程中心、快速开始、6 个客户端教程和公共排错中心，并让网站与 Word/飞书共享同一套 Markdown 正文。

**Architecture:** V2 放入独立路由、组件、样式、内容和媒体目录。9 份 Markdown 是唯一正文；Node 构建脚本解析 YAML Frontmatter、校验链接和媒体，并生成带内容哈希的轻量 manifest。Vue 按路由懒加载单篇 Markdown，Python 导出器读取同一 manifest 与源稿并验证哈希。网站媒体使用 WebP，导出通过媒体清单映射到同尺寸 PNG fallback。

**Tech Stack:** Vue 3、Vue Router 4、TypeScript、Vite 5、Vitest、Vue Test Utils、marked、DOMPurify、yaml、Python 3、python-docx、Browser 工具。

---

## 实施约束

- 当前 `main` 比 `origin/main` 超前 2 个提交：`06fb8812` 删除 14 份旧生成物，`9f64064a` 为 V2 设计规格。最终推送会一并推送 `06fb8812`；不得改写历史、reset 或擅自恢复旧产物。
- 当前工作区存在并行“图像生成”开发。唯一必然重叠文件是 `frontend/src/router/index.ts`，其中 `/image-generation` 未提交路由必须保留。
- 不修改当前已脏的 `Makefile`、Dockerfile、CI workflow、AppSidebar、i18n、生图功能和后端文件；V2 验证通过显式命令执行。
- 禁止 `git add .`、`git add frontend` 或目录级暂存。每个提交都用精确文件清单或 `git add -p`。
- V2 hash 定位在 V2 View 内处理，不改全局 `scrollBehavior`，避免改变 V1 与其他页面行为。
- V2 新增核心模块的 statements、branches、functions、lines 覆盖率均不低于 80%；解析器和本地进度分支覆盖率目标为 90%。

## Markdown 与媒体契约

- Frontmatter 必填：`title`、`slug`、`summary`、`duration`、`platforms`、`difficulty`、`updatedAt`、`version`。
- 每篇必须且只能有一个 H1。
- 步骤标题：`## 第 N 步：标题 {#ascii-anchor}`；只有以“第 N 步”开头的 H2 进入进度。
- 普通章节：`## 标题 {#ascii-anchor}`，不进入进度。
- 平台分支：`### Windows {#windows}`、`### macOS {#macos}`、`### Linux {#linux}`；无 JavaScript 与导出文档中全部展开。
- 提示块：`> [!TIP]`、`> [!WARNING]`、`> [!SUCCESS]`、`> [!NOTE]`。
- 图片：`![替代文字](/img/guides/v2/...webp "图片说明")`。
- 视频：普通链接 `[视频：标题](/img/guides/v2/...mp4)`；网页渐进增强为播放器，导出保留封面、说明和链接。
- 禁止 Vue 私有标签、脚本、事件属性、`javascript:` 与 `data:text/html` URL。
- `media-manifest.json` 为素材元数据，不是第二份正文；每张 WebP 必须登记同尺寸 PNG `exportPath`。

### Task 1: 冻结 V1 契约并更新规格状态

**Files:**
- Modify: `docs/superpowers/specs/2026-07-13-static-guides-v2-design.md`
- Modify: `frontend/src/router/__tests__/guide-routes.spec.ts`
- Modify: `frontend/src/router/__tests__/guards.spec.ts`
- Test: `frontend/src/views/public/__tests__/guide-hierarchy.spec.ts`

- [ ] **Step 1: 将设计状态改为实施中**

将规格头部更新为：

```markdown
**状态：** 已确认，实施中
```

- [ ] **Step 2: 写 V1 精确冻结测试**

在 `guide-routes.spec.ts` 增加完整矩阵：

```ts
const frozenV1Routes = [
  ['RegistrationKeyGuide', '/registration-key-guide', 'registration'],
  ['CodexGuide', '/codex-guide', 'codex'],
  ['ClaudeCodeGuide', '/claude-code-guide', 'claude'],
  ['OpenCodeGuide', '/open-code-guide', 'openCode'],
  ['OpenClawGuide', '/open-claw-guide', 'openClaw'],
  ['MobileGuide', '/mobile-guide', 'mobile'],
  ['ImageGuide', '/image-guide', 'image'],
] as const

it.each(frozenV1Routes)('freezes %s at %s', async (name, path, guideKey) => {
  const { default: router } = await import('@/router')
  const route = router.getRoutes().find((record) => record.name === name)
  expect(route).toMatchObject({
    path,
    meta: { requiresAuth: false, guideKey },
  })
})
```

在 `guards.spec.ts` 增加 7 个路径的 Backend Mode 逐项放行断言。

- [ ] **Step 3: 运行冻结门禁**

Run:

```bash
pnpm --dir frontend exec vitest run \
  src/router/__tests__/guide-routes.spec.ts \
  src/router/__tests__/guards.spec.ts \
  src/views/public/__tests__/guide-hierarchy.spec.ts
```

Expected: PASS。该任务是 characterization gate；若失败，说明 V1 当前已漂移，先定位并恢复测试预期对应的现状，不进入 V2 实现。

- [ ] **Step 4: 提交冻结门禁**

```bash
git add -f docs/superpowers/specs/2026-07-13-static-guides-v2-design.md
git add frontend/src/router/__tests__/guide-routes.spec.ts frontend/src/router/__tests__/guards.spec.ts
git diff --cached --name-only
git commit -m "test: freeze static guide v1 contracts"
```

### Task 2: 建立 manifest 生成器、内容类型与安全解析层

**Files:**
- Create: `frontend/scripts/build-guide-v2-manifest.mjs`
- Create: `frontend/src/views/public/guides-v2/guide-v2.types.ts`
- Create: `frontend/src/views/public/guides-v2/guide-v2.parser.ts`
- Create: `frontend/src/views/public/guides-v2/__tests__/guide-v2.parser.spec.ts`
- Create: `frontend/src/views/public/guides-v2/__tests__/guide-v2.manifest.spec.ts`
- Modify: `frontend/package.json`
- Modify: `frontend/pnpm-lock.yaml`

- [ ] **Step 1: 写 parser RED 测试**

覆盖缺字段、非法平台、错误日期、非 `v2`、重复锚点、危险 URL、脚本/事件属性、缺失媒体和合法 Markdown：

```ts
it('rejects unsafe links with source context', () => {
  expect(() => parseGuideMarkdown({
    sourceName: 'unsafe.md',
    body: '# 标题\n\n[危险](javascript:alert(1))',
    metadata: validMeta,
    media: [],
  })).toThrow('unsafe.md: 不允许的链接协议')
})

it('creates stable step and platform sections', () => {
  const document = parseGuideMarkdown({
    sourceName: 'codex.md',
    metadata: validMeta,
    media: [],
    body: '# 配置 Codex\n\n## 第 1 步：初始化 {#initialize}\n\n### Windows {#windows}\n\n完成。',
  })
  expect(document.steps.map((step) => step.id)).toEqual(['initialize'])
  expect(document.platforms).toContain('Windows')
})
```

- [ ] **Step 2: 运行 parser RED**

```bash
pnpm --dir frontend exec vitest run src/views/public/guides-v2/__tests__/guide-v2.parser.spec.ts
```

Expected: FAIL，提示 `guide-v2.parser` 不存在。

- [ ] **Step 3: 写 manifest 生成器 RED 测试**

```ts
it('builds a deterministic manifest from a temporary content directory', async () => {
  const manifest = await buildManifest({ contentDir: fixtureDirectory })
  expect(manifest.entries).toHaveLength(9)
  expect(manifest.entries[0]).toMatchObject({
    slug: 'index',
    path: '/guides/v2',
    version: 'v2',
  })
  expect(manifest.entries[0].contentHash).toMatch(/^[a-f0-9]{64}$/)
})
```

- [ ] **Step 4: 运行 manifest RED**

```bash
pnpm --dir frontend exec vitest run src/views/public/guides-v2/__tests__/guide-v2.manifest.spec.ts
```

Expected: FAIL，提示 manifest builder 不存在。

- [ ] **Step 5: 添加 `yaml` 与脚本命令**

```bash
pnpm --dir frontend add yaml
```

在 `frontend/package.json` 只增加生成与检查命令；此时不要修改 `build`，避免在真实 9 篇源稿尚未创建时破坏现有构建：

```json
{
  "scripts": {
    "guides:v2:manifest": "node scripts/build-guide-v2-manifest.mjs",
    "guides:v2:check": "node scripts/build-guide-v2-manifest.mjs --check"
  }
}
```

- [ ] **Step 6: 实现不可变类型与生成 manifest**

核心类型：

```ts
export type GuideV2Slug =
  | 'index' | 'get-started' | 'codex' | 'claude-code'
  | 'opencode' | 'openclaw' | 'chatbox-mobile'
  | 'cherry-studio-image' | 'troubleshooting'

export type GuideV2Meta = Readonly<{
  title: string
  slug: GuideV2Slug
  summary: string
  duration: string
  platforms: readonly string[]
  difficulty: '新手' | '入门'
  updatedAt: string
  version: 'v2'
}>

export type GuideV2ManifestEntry = Readonly<GuideV2Meta & {
  path: string
  source: string
  contentHash: string
}>
```

`build-guide-v2-manifest.mjs` 必须：

1. 精确读取 9 个 `.md`。
2. 使用 `yaml.parse` 解析 Frontmatter。
3. 校验字段、slug、唯一 H1、锚点、内部链接和 media manifest。
4. 计算去除 Frontmatter 后正文的 SHA-256。
5. 接受可注入的内容目录供测试使用；CLI 默认指向真实 V2 内容目录。
6. 原子写入 `manifest.generated.json`；`--check` 只比较，不写文件。

- [ ] **Step 7: 实现 parser**

parser 使用 `marked.lexer` 生成 typed blocks；段落与表格的 HTML 通过 DOMPurify 白名单净化，代码块与媒体保持结构化 token，不能把不受控 HTML直接交给 Vue。registry 和真实 manifest 在 Task 3 创建，以保证 Task 2 的每个提交仍可正常构建。

- [ ] **Step 8: 运行 GREEN 与定向覆盖率**

```bash
pnpm --dir frontend exec vitest run \
  src/views/public/guides-v2/__tests__/guide-v2.parser.spec.ts \
  src/views/public/guides-v2/__tests__/guide-v2.manifest.spec.ts
```

Expected: PASS。

- [ ] **Step 9: 提交内容契约**

```bash
git add frontend/package.json frontend/pnpm-lock.yaml \
  frontend/scripts/build-guide-v2-manifest.mjs \
  frontend/src/views/public/guides-v2/guide-v2.types.ts \
  frontend/src/views/public/guides-v2/guide-v2.parser.ts \
  frontend/src/views/public/guides-v2/__tests__/guide-v2.parser.spec.ts \
  frontend/src/views/public/guides-v2/__tests__/guide-v2.manifest.spec.ts
git commit -m "feat: add static guide v2 content contract"
```

### Task 3: 编写 9 份源稿并制作双格式素材

**Files:**
- Create: `frontend/src/content/guides-v2/index.md`
- Create: `frontend/src/content/guides-v2/get-started.md`
- Create: `frontend/src/content/guides-v2/codex.md`
- Create: `frontend/src/content/guides-v2/claude-code.md`
- Create: `frontend/src/content/guides-v2/opencode.md`
- Create: `frontend/src/content/guides-v2/openclaw.md`
- Create: `frontend/src/content/guides-v2/chatbox-mobile.md`
- Create: `frontend/src/content/guides-v2/cherry-studio-image.md`
- Create: `frontend/src/content/guides-v2/troubleshooting.md`
- Create: `frontend/src/content/guides-v2/media-manifest.json`
- Create: `frontend/src/content/guides-v2/manifest.generated.json`
- Create: `frontend/src/views/public/guides-v2/guide-v2.registry.ts`
- Create: `frontend/src/views/public/guides-v2/__tests__/guide-v2.registry.spec.ts`
- Create: `frontend/src/views/public/guides-v2/__tests__/guide-v2.content.spec.ts`
- Create: `frontend/public/img/guides/v2/**`
- Modify: `frontend/package.json`

- [ ] **Step 1: 写内容完整性 RED 测试**

```ts
it('contains the exact nine source documents', () => {
  expect(manifest.map((entry) => entry.slug)).toEqual([
    'index', 'get-started', 'codex', 'claude-code', 'opencode',
    'openclaw', 'chatbox-mobile', 'cherry-studio-image', 'troubleshooting',
  ])
})

it('keeps all media simulated and exportable', () => {
  for (const media of mediaManifest) {
    expect(media.simulated).toBe(true)
    expect(media.webPath).toMatch(/\.webp$/)
    expect(media.exportPath).toMatch(/\.png$/)
    expect(media.width).toBeGreaterThan(0)
    expect(media.height).toBeGreaterThan(0)
  }
})
```

另断言：首页含两条入口和 6 客户端；排错中心含 401/404/429/模型不可用/配置未生效；每个客户端链接排错中心；无疑似真实 Key、账号、余额、用量和旧域名。

- [ ] **Step 2: 运行内容 RED**

```bash
pnpm --dir frontend exec vitest run src/views/public/guides-v2/__tests__/guide-v2.content.spec.ts
```

Expected: FAIL，9 份源稿和素材不存在。

- [ ] **Step 3: 编写 9 份完整源稿**

固定章节范围：

| 文件 | 必须包含的正文 |
| --- | --- |
| `index.md` | 完整流程、从零开始、已有 Key、6 客户端卡片文案、排错入口 |
| `get-started.md` | 注册、获取权益、创建 Key、选择分组、复制配置、安全提示 |
| `codex.md` | 下载初始化、Windows/macOS/Linux、config.toml、auth.json、API 登录、测试 |
| `claude-code.md` | 安装、settings.json/环境变量二选一、各平台路径、重开终端、验证 |
| `opencode.md` | 官方安装、配置目录、opencode.json、/connect、验证 |
| `openclaw.md` | 云端/本地二选一、配置、模型测试、专属错误 |
| `chatbox-mobile.md` | iOS/Android、提供方、API 主机、Key、获取模型、测试对话 |
| `cherry-studio-image.md` | 模型服务、当前推荐配置、图像模型、绘画入口、验证 |
| `troubleshooting.md` | 401、404、429、模型不可用、配置未生效、文件扩展名、联系支持 |

所有示例 Key 使用 `sk-example-not-a-real-key`，易变模型与域名集中在“当前推荐配置”章节并标注核验日期。

- [ ] **Step 4: 制作素材**

使用 imagegen 仅生成教程中心装饰图、流程图和非精确 UI 插图；需要精准按钮文字的教程图使用确定性模拟界面。至少交付以下 13 组 WebP+PNG：

1. `common/setup-flow`
2. `common/create-key-group`
3. `codex/initialize`
4. `codex/config-folder`
5. `codex/api-login`
6. `claude-code/settings`
7. `opencode/config`
8. `openclaw/mode-choice`
9. `chatbox/add-provider`
10. `chatbox/select-model`
11. `cherry-studio/add-service`
12. `cherry-studio/generate-image`
13. `troubleshooting/status-map`

每组登记：`id`、`guide`、`step`、`kind`、`webPath`、`exportPath`、`posterPath`、`alt`、`caption`、`width`、`height`、`source`、`clientVersion`、`updatedAt`、`simulated`。

- [ ] **Step 5: 生成 manifest 并运行 GREEN**

先在 `frontend/package.json` 把现有 `build` 改为：

```json
"build": "pnpm guides:v2:check && vue-tsc -b && vite build"
```

创建 registry，从生成 manifest 读取元数据，并为 9 篇正文登记显式 `?raw` 动态 import：

```ts
const loaders: Readonly<Record<GuideV2Slug, () => Promise<string>>> = {
  index: () => import('@/content/guides-v2/index.md?raw').then((module) => module.default),
  'get-started': () => import('@/content/guides-v2/get-started.md?raw').then((module) => module.default),
  codex: () => import('@/content/guides-v2/codex.md?raw').then((module) => module.default),
  'claude-code': () => import('@/content/guides-v2/claude-code.md?raw').then((module) => module.default),
  opencode: () => import('@/content/guides-v2/opencode.md?raw').then((module) => module.default),
  openclaw: () => import('@/content/guides-v2/openclaw.md?raw').then((module) => module.default),
  'chatbox-mobile': () => import('@/content/guides-v2/chatbox-mobile.md?raw').then((module) => module.default),
  'cherry-studio-image': () => import('@/content/guides-v2/cherry-studio-image.md?raw').then((module) => module.default),
  troubleshooting: () => import('@/content/guides-v2/troubleshooting.md?raw').then((module) => module.default),
}
```

```bash
pnpm --dir frontend run guides:v2:manifest
pnpm --dir frontend run guides:v2:check
pnpm --dir frontend exec vitest run \
  src/views/public/guides-v2/__tests__/guide-v2.content.spec.ts \
  src/views/public/guides-v2/__tests__/guide-v2.registry.spec.ts
```

Expected: PASS。

- [ ] **Step 6: 提交正文和素材**

使用精确清单暂存 9 个 Markdown、2 个 manifest、内容测试和 `frontend/public/img/guides/v2/**`，提交：

```text
docs: add static guide v2 source content
```

### Task 4: 接入 V2 路由、公开访问与专用 404

**Files:**
- Create: `frontend/src/views/public/guides-v2/GuideV2HubView.vue`
- Create: `frontend/src/views/public/guides-v2/GuideV2View.vue`
- Create: `frontend/src/views/public/guides-v2/GuideV2NotFoundView.vue`
- Create: `frontend/src/router/__tests__/guide-v2-routes.spec.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/router/__tests__/guards.spec.ts`

- [ ] **Step 1: 写路由 RED 测试**

断言 9 个路径可解析、`requiresAuth: false`、标题正确、无效 slug 显示专用 404、Backend Mode 精确允许 `/guides/v2` 与 `/guides/v2/` 前缀但拒绝 `/guides/v2evil`。

```ts
it.each(expectedV2Routes)('registers %s as a public route', async (_label, path) => {
  const { default: router } = await import('@/router')
  const resolved = router.resolve(path)
  expect(resolved.meta.requiresAuth).toBe(false)
})
```

- [ ] **Step 2: 运行路由 RED**

```bash
pnpm --dir frontend exec vitest run src/router/__tests__/guide-v2-routes.spec.ts
```

Expected: FAIL，V2 路由不存在。

- [ ] **Step 3: 在当前工作树做小块路由补丁**

增加两个路由记录：

```ts
{
  path: '/guides/v2',
  name: 'GuideV2Hub',
  component: () => import('@/views/public/guides-v2/GuideV2HubView.vue'),
  meta: { requiresAuth: false, title: 'AI 客户端使用指南' },
},
{
  path: '/guides/v2/:slug',
  name: 'GuideV2Detail',
  component: () => import('@/views/public/guides-v2/GuideV2View.vue'),
  meta: { requiresAuth: false, title: 'AI 客户端使用指南' },
},
```

公共判断使用边界明确的纯函数：

```ts
const isGuideV2Path = (path: string): boolean =>
  path === '/guides/v2' || path.startsWith('/guides/v2/')
```

无效 slug 由 `GuideV2View` 渲染 `GuideV2NotFoundView`，因此仍保持公开访问。

- [ ] **Step 4: 核对并行 `/image-generation` 路由仍存在**

```bash
git diff -- frontend/src/router/index.ts
rg -n "path: '/image-generation'|path: '/guides/v2'" frontend/src/router/index.ts
```

Expected: 两类路由同时存在，未覆盖并行 hunk。

- [ ] **Step 5: 运行 GREEN**

```bash
pnpm --dir frontend exec vitest run \
  src/router/__tests__/guide-routes.spec.ts \
  src/router/__tests__/guide-v2-routes.spec.ts \
  src/router/__tests__/guards.spec.ts
```

Expected: PASS。

- [ ] **Step 6: 分块暂存并提交**

```bash
git add frontend/src/views/public/guides-v2/GuideV2HubView.vue \
  frontend/src/views/public/guides-v2/GuideV2View.vue \
  frontend/src/views/public/guides-v2/GuideV2NotFoundView.vue \
  frontend/src/router/__tests__/guide-v2-routes.spec.ts \
  frontend/src/router/__tests__/guards.spec.ts
git add -p frontend/src/router/index.ts
git diff --cached -- frontend/src/router/index.ts
git commit -m "feat: expose static guide v2 routes"
```

暂存的 router hunk 只能包含 V2；并行 `/image-generation` hunk必须保持未暂存。

### Task 5: 实现不可变本地进度状态

**Files:**
- Create: `frontend/src/views/public/guides-v2/composables/useGuideV2Progress.ts`
- Create: `frontend/src/views/public/guides-v2/composables/__tests__/useGuideV2Progress.spec.ts`

- [ ] **Step 1: 写进度 RED 测试**

```ts
it('updates progress without mutating the previous state', () => {
  const progress = createGuideV2Progress(storage)
  const before = progress.get('codex')
  const after = progress.completeStep('codex', 'initialize')
  expect(after).not.toBe(before)
  expect(before.completedStepIds).toEqual([])
  expect(after.completedStepIds).toEqual(['initialize'])
})
```

继续覆盖刷新恢复、损坏 JSON、schema 版本错误、`getItem`/`setItem` 抛错、单篇清除、全部清除和不删除其他业务 key。

- [ ] **Step 2: 运行 RED**

```bash
pnpm --dir frontend exec vitest run \
  src/views/public/guides-v2/composables/__tests__/useGuideV2Progress.spec.ts
```

Expected: FAIL，模块不存在。

- [ ] **Step 3: 实现版本化不可变状态**

```ts
const STORAGE_KEY = 'sub2api:guides:v2:progress:v1'

type GuideProgress = Readonly<{
  completedStepIds: readonly string[]
  platform: string | null
  lastAnchor: string | null
  updatedAt: string | null
}>
```

所有更新使用新对象和新数组；localStorage 不可用时保存到模块内存副本。

- [ ] **Step 4: 运行 GREEN 与覆盖率**

```bash
pnpm --dir frontend exec vitest run \
  src/views/public/guides-v2/composables/__tests__/useGuideV2Progress.spec.ts \
  --coverage \
  --coverage.include='src/views/public/guides-v2/composables/useGuideV2Progress.ts'
```

Expected: PASS，四项覆盖率均 ≥ 80%，branches 目标 ≥ 90%。

- [ ] **Step 5: 提交**

```text
feat: persist static guide v2 progress locally
```

### Task 6: 实现云端蓝图 UI 与关键交互

**Files:**
- Create: `frontend/src/views/public/guides-v2/components/GuideV2Header.vue`
- Create: `frontend/src/views/public/guides-v2/components/GuideV2Hero.vue`
- Create: `frontend/src/views/public/guides-v2/components/GuideV2Sidebar.vue`
- Create: `frontend/src/views/public/guides-v2/components/GuideV2MobileToc.vue`
- Create: `frontend/src/views/public/guides-v2/components/GuideV2Renderer.vue`
- Create: `frontend/src/views/public/guides-v2/components/GuideV2Step.vue`
- Create: `frontend/src/views/public/guides-v2/components/GuideV2Media.vue`
- Create: `frontend/src/views/public/guides-v2/components/GuideV2CodeBlock.vue`
- Create: `frontend/src/views/public/guides-v2/components/GuideV2Notice.vue`
- Create: `frontend/src/views/public/guides-v2/components/GuideV2Support.vue`
- Create: `frontend/src/views/public/guides-v2/styles/tokens.css`
- Create: `frontend/src/views/public/guides-v2/styles/layout.css`
- Create: `frontend/src/views/public/guides-v2/styles/content.css`
- Create: `frontend/src/views/public/guides-v2/styles/responsive.css`
- Create: `frontend/src/views/public/guides-v2/components/__tests__/GuideV2Renderer.spec.ts`
- Create: `frontend/src/views/public/guides-v2/components/__tests__/GuideV2CodeBlock.spec.ts`
- Create: `frontend/src/views/public/guides-v2/components/__tests__/GuideV2Media.spec.ts`
- Create: `frontend/src/views/public/guides-v2/components/__tests__/GuideV2MobileToc.spec.ts`
- Create: `frontend/src/views/public/guides-v2/__tests__/GuideV2HubView.spec.ts`
- Create: `frontend/src/views/public/guides-v2/__tests__/GuideV2View.spec.ts`
- Modify: `frontend/src/views/public/guides-v2/GuideV2HubView.vue`
- Modify: `frontend/src/views/public/guides-v2/GuideV2View.vue`
- Modify: `frontend/src/views/public/guides-v2/GuideV2NotFoundView.vue`

- [ ] **Step 1: 按组件逐个写 RED 测试**

必须覆盖：

- 网站品牌读取 `useAppStore().siteName`，不硬编码 SAK AI。
- 每页只有一个 H1。
- Hub 首屏只有流程、6 客户端和“两条入口”。
- 桌面粘性目录、移动目录位于 Hero 后；Esc 关闭并恢复焦点。
- 平台标签可键盘操作，选择刷新后保留。
- 代码复制成功显示反馈；复制失败显示“请手动选择并复制”。
- 图片包含 alt、width、height 和 lazy；失败后显示说明、重试和 PNG fallback。
- 视频不自动播放，使用 `preload="metadata"`、poster 和文字链接。
- 进度、清除进度、上一篇/下一篇和公共排错入口。
- `prefers-reduced-motion` 下不依赖动画表达状态。

示例：

```ts
it('shows a manual fallback when clipboard copy fails', async () => {
  vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValue(new Error('denied'))
  const wrapper = mount(GuideV2CodeBlock, { props: { code: 'example', language: 'text' } })
  await wrapper.get('button').trigger('click')
  expect(wrapper.text()).toContain('请手动选择并复制')
})
```

- [ ] **Step 2: 每个测试文件先确认 RED**

```bash
pnpm --dir frontend exec vitest run \
  src/views/public/guides-v2/components \
  src/views/public/guides-v2/__tests__/GuideV2HubView.spec.ts \
  src/views/public/guides-v2/__tests__/GuideV2View.spec.ts
```

Expected: FAIL，组件尚未实现。

- [ ] **Step 3: 实现最小 GREEN，再按职责拆分**

所有 CSS 选择器以 `.guide-v2-` 开头；单文件控制在 200–400 行，禁止继续扩展 `codex-guide.css`。详情页监听 route slug/hash，在 `nextTick` 后滚动到目标元素，并使用 `scroll-margin-top` 处理固定页眉。

- [ ] **Step 4: 运行 UI GREEN 与定向覆盖率**

```bash
pnpm --dir frontend exec vitest run \
  src/views/public/guides-v2/components \
  src/views/public/guides-v2/__tests__/GuideV2HubView.spec.ts \
  src/views/public/guides-v2/__tests__/GuideV2View.spec.ts

pnpm --dir frontend exec vitest run \
  src/views/public/guides-v2 \
  --coverage \
  --coverage.include='src/views/public/guides-v2/**/*.{ts,vue}' \
  --coverage.thresholds.statements=80 \
  --coverage.thresholds.branches=80 \
  --coverage.thresholds.functions=80 \
  --coverage.thresholds.lines=80
```

Expected: PASS，四项阈值达标。

- [ ] **Step 5: 提交**

```text
feat: build static guide v2 experience
```

### Task 7: 扩展飞书 Markdown 与 Word 导出

**Files:**
- Modify: `tools/export_feishu_guides.py`
- Modify: `tools/export_feishu_guides_test.py`
- Modify: `tools/export_word_guides.py`
- Modify: `tools/export_word_guides_test.py`
- Create: `tools/export_guides_v2_test.py`
- Modify: `docs/static-guides/README.md`

- [ ] **Step 1: 写导出 RED 测试**

保留 V1 默认 7 份，并新增：

```py
def test_v2_exports_exact_nine_guides(self) -> None:
    result = self.run_export("--edition", "v2")
    self.assertEqual(result.returncode, 0, result.stderr)
    self.assertEqual(len(tuple(self.output_dir.rglob("*.md"))), 9)

def test_v2_replaces_webp_with_png_fallback(self) -> None:
    result = self.run_export("--edition", "v2")
    self.assertEqual(result.returncode, 0, result.stderr)
    content = next(self.output_dir.rglob("02-Codex.md")).read_text("utf-8")
    self.assertIn("data:image/png;base64,", content)
    self.assertNotIn(".webp", content)
```

Word 测试还要断言 DOCX 无外部图片关系、Frontmatter 不进入正文、全部平台分支保留、视频降级为封面/说明/链接、`--check` 可发现缺失/额外/过期文件。

- [ ] **Step 2: 运行导出 RED**

```bash
python3 -m unittest tools.export_guides_v2_test
```

Expected: FAIL，CLI 不识别 `--edition v2`。

- [ ] **Step 3: 实现兼容 CLI 与 manifest 哈希检查**

新增 `--edition v1|v2`，默认 `v1`。V2 默认输出：

```text
docs/static-guides/feishu-v2/02-AI客户端使用指南/
docs/static-guides/feishu-word-v2/02-AI客户端使用指南/
```

两套目录精确包含：`00-教程中心`、`01-快速开始`、`02-Codex`、`03-Claude-Code`、`04-OpenCode`、`05-OpenClaw`、`06-Chatbox-移动端`、`07-Cherry-Studio-生图`、`08-公共排错中心`。

V2 导出前计算正文 SHA-256 并与 `manifest.generated.json` 比较；不一致则失败并提示先运行 `pnpm --dir frontend guides:v2:manifest`。

- [ ] **Step 4: 使用可用文档运行时跑 GREEN**

先调用工作区依赖加载器取得捆绑 Python。若不可用，使用 `/tmp` 临时环境：

```bash
python3 -m venv /tmp/subapi-guides-v2-venv
/tmp/subapi-guides-v2-venv/bin/pip install 'python-docx>=1.2,<2' 'Pillow>=10,<12'
/tmp/subapi-guides-v2-venv/bin/python -m unittest \
  tools.export_feishu_guides_test \
  tools.export_word_guides_test \
  tools.export_guides_v2_test
```

Expected: PASS，且现有 V1 默认导出测试保持通过。

- [ ] **Step 5: 生成与检查 V2 成品**

```bash
python3 tools/export_feishu_guides.py --edition v2
python3 tools/export_feishu_guides.py --edition v2 --check
/tmp/subapi-guides-v2-venv/bin/python tools/export_word_guides.py --edition v2
/tmp/subapi-guides-v2-venv/bin/python tools/export_word_guides.py --edition v2 --check
```

生成目录继续作为本地交付成品，不强制加入 Git；`06fb8812` 已删除上一版生成物。

- [ ] **Step 6: 渲染 9 份 DOCX 做视觉检查**

使用文档 skill 提供的 `render_docx.py` 渲染到 `/tmp/subapi-guides-v2-render/`，逐页检查标题层级、代码块、表格、图片比例、分页和中文字体。

- [ ] **Step 7: 提交导出能力与 README**

```text
feat: export static guide v2 for feishu
```

### Task 8: 完整验证、代码审查、浏览器 QA 与推送

**Files:**
- Verify only: all V2 files and the exact staged diff
- Do not add: `.superpowers/`, `.venv/`, `deploy/`, `frontend/image-playground/`, unrelated backend/frontend files

- [ ] **Step 1: 运行 V1/V2 自动化测试**

```bash
pnpm --dir frontend exec vitest run \
  src/router/__tests__/guide-routes.spec.ts \
  src/router/__tests__/guide-v2-routes.spec.ts \
  src/router/__tests__/guards.spec.ts \
  src/views/public/__tests__/guide-hierarchy.spec.ts \
  src/views/public/guides-v2
```

Expected: PASS，0 failures。

- [ ] **Step 2: 运行定向覆盖率、静态检查与构建**

```bash
pnpm --dir frontend exec vitest run src/views/public/guides-v2 \
  --coverage \
  --coverage.include='src/views/public/guides-v2/**/*.{ts,vue}' \
  --coverage.thresholds.statements=80 \
  --coverage.thresholds.branches=80 \
  --coverage.thresholds.functions=80 \
  --coverage.thresholds.lines=80
pnpm --dir frontend run lint:check
pnpm --dir frontend run typecheck
pnpm --dir frontend run build
git diff --check
```

Expected: 全部 exit 0。

- [ ] **Step 3: 运行 Go 内嵌前端最小验证**

```bash
go test -tags embed ./internal/web
```

Run from: `backend/`

Expected: PASS。

- [ ] **Step 4: 启动开发站点并使用 Browser 工具 QA**

```bash
pnpm --dir frontend run dev -- --host 127.0.0.1 --port 4173
```

目标流程：`/guides/v2` → 选择 Codex → 从零开始/已有 Key → 切换平台 → 完成步骤 → 排错中心 → 返回教程。

分别检查 1440×900、768×1024、390×844：页面身份、非空内容、无框架错误层、控制台无相关 error/warn、单 H1、无横向溢出、移动目录不遮挡、44px 触控目标、进度刷新恢复、复制反馈、媒体失败 fallback、无效 slug 404 和 hash 定位。保存三档截图到仓库外临时目录。

- [ ] **Step 5: 进行代码审查与修复**

派发 code-reviewer 与 security-reviewer，提供设计规格、实施计划、起始 SHA 和当前 SHA。修复全部 CRITICAL/HIGH，尽量修复 MEDIUM；修复后重新运行受影响测试。

- [ ] **Step 6: 最终敏感信息与暂存边界检查**

```bash
rg -n 'sk-[A-Za-z0-9_-]{16,}|AKIA[0-9A-Z]{16}|api\.sakms\.top' \
  frontend/src/content/guides-v2 frontend/public/img/guides/v2
git status --short
git diff --cached --name-only
```

示例 `sk-example-not-a-real-key` 必须通过明确 allowlist 处理，不能把真实命中忽略掉。缓存区不得含生图功能、后端并行改动、`.superpowers`、`.venv` 或 deploy 运维残留。

- [ ] **Step 7: 最终提交并推送 main**

如果最后修复尚未提交：

```bash
git commit -m "fix: polish static guide v2 delivery"
```

确认远端未出现非快进后：

```bash
git push origin main
```

Expected: push 成功。若非快进，停止并报告；不要在当前脏工作区自动 rebase、stash 或覆盖并行改动。

## 最终验收清单

- [ ] V1 7 个路径、组件和 Backend Mode 公开访问全部冻结。
- [ ] V2 9 个页面、专用 404、hash 与深链刷新正常。
- [ ] 网站品牌读取部署站点名；导出品牌为“AI 客户端使用指南”。
- [ ] 9 份 Markdown 是唯一正文，manifest 哈希一致。
- [ ] 13 组 WebP/PNG 素材无真实 Key、账号、余额、用量或旧域名。
- [ ] Word/飞书 9 份成品生成、检查和 DOCX 渲染通过。
- [ ] V2 新增核心模块四项覆盖率均不低于 80%。
- [ ] 1440、768、390 三档 Browser QA 与关键流程通过。
- [ ] 无 V2 相关控制台错误、缺失媒体或重大视觉偏差。
- [ ] 最终提交仅包含 V2 文件；main 成功推送。
