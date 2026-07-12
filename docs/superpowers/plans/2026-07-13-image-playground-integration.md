# Sub2API 同源生图应用实施计划

> **执行要求：** 严格按 TDD 的 RED → GREEN → IMPROVE 顺序逐项完成；每个阶段只修改相关文件并保留用户现有工作区内容。

**目标：** 在 Sub2API 中交付完整的同源 React 生图子应用，并通过安全消息桥接复用用户的 OpenAI 生图密钥与现有网关。

**架构：** Vue 负责认证、入口、权限、密钥选择与错误壳；React 负责完整生图工作区；Go 统一托管两套静态产物并按路径应用 CSP。

**技术栈：** Vue 3、Vitest、React 19、TypeScript、Vite、Zustand、IndexedDB、Go/Gin、Docker。

---

## Task 1：宿主权限与侧边栏入口

**文件：**
- 新建：`frontend/src/composables/useImageGenerationAccess.ts`
- 新建：`frontend/src/composables/__tests__/useImageGenerationAccess.spec.ts`
- 修改：`frontend/src/components/layout/AppSidebar.vue`
- 修改：`frontend/src/components/layout/__tests__/AppSidebar.spec.ts`
- 修改：`frontend/src/locales/zh-CN.ts` 及现有语言文件

**步骤：**
1. 先写失败测试，覆盖活跃状态、OpenAI 平台、生图权限和分页行为。
2. 写侧边栏源结构测试，要求新增独立 `/image-generation` 入口且不替换 `/batch-image`。
3. 实现访问 composable 与入口图标/文案。
4. 运行目标 Vitest 并确认通过。

## Task 2：Vue 宿主页与安全桥协议

**文件：**
- 新建：`frontend/src/features/imagePlayground/bridge.ts`
- 新建：`frontend/src/features/imagePlayground/__tests__/bridge.spec.ts`
- 新建：`frontend/src/views/user/ImageGenerationView.vue`
- 新建：`frontend/src/views/user/__tests__/ImageGenerationView.spec.ts`
- 修改：`frontend/src/router/index.ts`

**步骤：**
1. 先写失败测试，覆盖 origin/source/version/nonce 校验、凭据筛选、受管 profile 派生和消息结构。
2. 写视图测试，覆盖加载、无权限、错误、ready/configure、超时、卸载清理与重新加载。
3. 实现纯函数协议层，再实现视图壳。
4. iframe 使用严格 sandbox/allow，nonce 放在 `name` 而非 URL。
5. 运行目标 Vitest 与类型检查。

## Task 3：移植 React 应用并建立嵌入模式

**文件：**
- 新建：`frontend/image-playground/**`
- 新建：`frontend/image-playground/src/lib/sub2apiBridge.ts`
- 新建：`frontend/image-playground/src/lib/__tests__/sub2apiBridge.test.ts`
- 新建：`frontend/image-playground/src/lib/sanitizeEmbeddedSettings.ts`
- 新建：`frontend/image-playground/src/lib/__tests__/sanitizeEmbeddedSettings.test.ts`
- 修改：React store、设置页、导入导出、入口、Vite 与 Service Worker 注册相关文件

**步骤：**
1. 从固定上游提交导入源码、测试与 MIT LICENSE，不导入构建产物和 node_modules。
2. 先写失败测试，证明密钥会从 persist/export/URL 设置中剥离，且恶意跨来源/错误 source/错误 nonce 消息被拒绝。
3. 实现嵌入模式桥接与受管 profile 注入；配置完成前不初始化主 store。
4. 嵌入模式锁定同源 `/v1`，禁用自定义 Provider、fal.ai、URL 导入、密钥编辑与外部资源。
5. 禁止注册 Service Worker；收窄或移除会影响父应用的全局副作用。
6. 保留图库、编辑、蒙版、Responses、Agent、历史、收藏及安全导入导出。
7. 运行 React 目标测试、全量测试、类型检查与生产构建。

## Task 4：静态托管、CSP 与构建链

**文件：**
- 修改：`backend/internal/server/middleware/security_headers.go`
- 修改：`backend/internal/server/middleware/security_headers_test.go`
- 修改：`backend/internal/config/config.go`
- 修改：`deploy/config.example.yaml`
- 修改：`Makefile`
- 修改：`Dockerfile`
- 修改：`deploy/Dockerfile`
- 修改：`.dockerignore`
- 修改：`.github/workflows/backend-ci.yml`
- 修改：`.github/workflows/release.yml`
- 修改：`.github/workflows/security-scan.yml`

**步骤：**
1. 先写 Go 失败测试：父页面 `frame-src 'self'`；仅 React 子路径允许 `SAMEORIGIN`/`frame-ancestors 'self'`；其他页面保持 DENY/none。
2. 实现路径级安全头策略与收紧的 React CSP。
3. 调整 Vue→React 构建顺序，确保 React 构建不清空 Vue 产物。
4. 将两套依赖安装、测试、审计与构建接入 Docker/CI/release。
5. 构建后验证 `/image-playground/` 与其 hash asset 均存在。

## Task 5：图片入口边界硬化

**文件：**
- 修改：`backend/internal/service/openai_images.go`
- 修改：对应 Go 测试
- 修改：`deploy/config.example.yaml`

**步骤：**
1. 先写失败测试，覆盖 multipart 图片超过上限时拒绝而非静默截断。
2. 为数量、局部图数量、压缩参数补明确上限与用户可读错误。
3. 对齐示例配置中的上游响应读取上限与代码默认值。
4. 运行图片 handler/service/routes 的目标测试。

## Task 6：集成验证与审查

**步骤：**
1. 运行 Vue lint、类型检查与全量 Vitest。
2. 运行 React 全量测试、类型检查、production build 与 production dependency audit。
3. 运行 Go 相关测试，再运行仓库允许范围内的全量测试。
4. 启动本地开发环境，用浏览器验证桌面和移动端：入口权限、加载、生成、编辑、Responses、Agent、历史、导出与错误恢复。
5. 检查 DevTools：URL/storage/export 不含 API key，网络请求仅同源，CSP 无非预期报错。
6. 执行安全审查、需求一致性审查与代码质量审查，修复 CRITICAL/HIGH 及合理的 MEDIUM 问题。
7. 检查 git diff，只暂存本次相关文件；提交约定式提交并推送当前 `main`。
