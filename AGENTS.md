# 项目规则（AI）

- 本文件用于约束 AI 助手在此仓库内的工作方式，专注于工程规则与交付质量。请将标注为 **MUST/禁止** 的内容视为硬约束。
- 本项目现在处于开发阶段，在本机运行的方式是开发模式。
- 本文件包含项目的简略信息。每当用户提出需求，都要先阅读本文件已包含的项目简略信息，找到核心文件，寻求到核心问题之后再开始改动。
- 当用户提出某需求或某功能时，请你先参考业内最佳实践，然后再进行实现或改动。

## 0) 不可协商（MUST）

- 始终使用简体中文进行响应。
- 每条助手消息必须以以下两行开头：
  1) `【（必须填写本次实际使用的模型名称）】`
  2) `亲爱的 Wang`
- 结尾做简单本轮总结。
- 本轮任务完成后：在做完最小验证后，自动提交和同步本次相关改动（只包含本次任务；提交信息需能概括改动，如果没有远程仓库，则先提交，可以不同步。所有改动推送到主分支，不要新增分支。

## 1) 指令范围与工作方式

- 本文件适用于整个仓库。进入子目录工作时，还必须读取该路径上更近的 `AGENTS.md`；目前 `frontend/image-playground/AGENTS.md` 对生图子应用提供补充约束。若发生冲突，除本文件第 0 节的 MUST 规则外，以更接近目标文件的规则为准。
- 每次动手前先检查 `git status --short --branch`，从用户指定的文件、路由、错误或测试开始定位，并阅读相邻实现和测试。只解决已确认的根因，不顺手重构无关区域。
- 工作区可能包含用户未提交或未跟踪的改动。必须保留它们；不得擅自删除、覆盖、暂存、格式化或回滚，不得使用 `git reset --hard`、`git clean`、强制 checkout 或未经允许的 stash。
- 文档与配置冲突时，以可执行配置为准：`backend/go.mod`、各 `package.json`、锁文件、`Makefile`、Docker Compose 和 `.github/workflows/` 是版本与命令的权威来源。
- 修改前优先复用仓库已有模式、公共契约和工具；仅在确有必要时新增依赖、抽象或配置。

## 2) 项目概况

Sub2API 是一个 AI API 网关与订阅配额管理平台，负责鉴权、计费、账号调度、并发/速率控制和上游请求转发。

- 后端：Go（当前由 `backend/go.mod` 固定为 1.26.5）、Gin、Ent、Wire、PostgreSQL、Redis。
- 主前端：Vue 3 + TypeScript + Vite + Pinia + Vue Router + Vue I18n，使用 `pnpm@9.15.9`。
- 生图子应用：React 19 + TypeScript + Vite + Zustand，位于 `frontend/image-playground/`，独立使用 npm 与 `package-lock.json`。
- 本地完整环境：`deploy/docker-compose.dev.yml` 从当前源码构建应用，并启动 PostgreSQL、Redis 和 Sub2API。

## 3) 核心目录与入口

- `backend/cmd/server/`：后端入口与 Wire 依赖注入。
- `backend/internal/server/`：HTTP 服务、路由和中间件装配。
- `backend/internal/handler/`：HTTP 边界、参数校验与 DTO 转换。
- `backend/internal/service/`：业务逻辑与上游网关编排。
- `backend/internal/repository/`：数据库、Redis 和外部持久化实现。
- `backend/ent/schema/`：Ent Schema 源文件；`backend/ent/` 其余大部分文件为生成代码。
- `backend/migrations/`：按文件名顺序执行的不可变、前向 SQL 迁移。
- `frontend/src/api/`：前端 API 契约；`views/`、`components/`、`composables/`、`stores/` 分别承载页面、组件、复用逻辑和状态。
- `frontend/src/router/`：路由、守卫和页面元信息；路由变化要同步检查相邻测试。
- `frontend/src/i18n/`：国际化资源；已国际化页面不得新增孤立硬编码文案。
- `frontend/image-playground/`：独立 React 生图应用；修改前先读该目录自己的 `AGENTS.md`。
- `frontend/src/content/guides-v2/`：V2 教程正文和媒体清单的源数据。
- `docs/static-guides/`：教程源稿、导出说明与生成成品；维护流程见其 `README.md`。
- `deploy/`：本地开发和生产部署配置；`tools/`：审计与教程导出工具。

## 4) 环境、开发与构建

首次安装依赖：

```bash
pnpm --dir frontend install --frozen-lockfile
npm --prefix frontend/image-playground ci
cd backend && go mod download
```

常用命令：

```bash
# 从本地源码构建并启动完整开发栈；先在 deploy/.env 配置必需值
cd deploy && docker compose -f docker-compose.dev.yml up --build -d

# 同时启动 Vue 主站和 React 生图应用的热更新服务
make dev-frontend

# 已有 PostgreSQL、Redis 和后端配置时，可直接运行后端
cd backend && go run ./cmd/server

# 按 Vue -> React -> Go 的顺序构建
make build
```

- Vue 默认端口为 `3000`，生图应用固定为 `5174`，API 默认代理到 `http://localhost:8080`；按需使用 `VITE_DEV_PORT`、`VITE_DEV_PROXY_TARGET` 和 `VITE_IMAGE_PLAYGROUND_DEV_TARGET` 覆盖。
- 完整栈启动后，以 `docker compose ps`、`http://127.0.0.1:8080/health` 和真实页面/API 行为为准，不以“进程存在”代替可用性验证。
- 不得覆盖现有 `.env`，不得提交 `.env`、`config.yaml`、运行数据、日志、数据库目录或备份。

## 5) 实现与生成约束

### 后端

- 保持 `handler -> service -> repository` 的分层。`handler` 和 `service` 不得直接依赖 repository、GORM 或 Redis；以 `backend/.golangci.yml` 的 depguard 规则及已有少量例外为准。
- Go 代码必须通过 `gofmt`；新增错误路径要显式处理错误，接口变化要同步所有 mock、stub、Wire provider 和调用方。
- 修改 `backend/ent/schema/` 后运行 `make -C backend generate`，检查并提交必要的 Ent/Wire 生成差异；禁止手工修改生成文件或 `backend/cmd/server/wire_gen.go`。
- 数据库变化必须新增前向迁移。已发布迁移受 checksum 保护，禁止修改、删除、重命名或重新排序；具体规则见 `backend/migrations/README.md`。
- 鉴权、管理员权限、数据范围和输入校验必须在后端落实，不能只依赖前端路由守卫或隐藏控件。

### 前端与文档

- Vue API 调用优先复用 `frontend/src/api/`，共享状态和逻辑优先复用现有 store/composable；避免在视图中重复请求契约。
- 用户可见文案按所在页面的既有国际化模式同步维护；保持现有组件、交互、响应式布局和无障碍模式。
- Vue 主站只使用 pnpm；生图子应用只使用 npm。依赖变化必须同步正确的锁文件，禁止混用包管理器。
- `pnpm --dir frontend run lint` 会自动修复文件；纯检查使用 `pnpm --dir frontend run lint:check`。
- 不直接编辑 `backend/internal/web/dist/`、`frontend/src/content/guides-v2/manifest.generated.json` 或生成的 `.docx`。修改源文件后运行对应生成命令。
- 修改 V2 教程正文或媒体清单后，先运行 `pnpm --dir frontend run guides:v2:manifest`，再运行 `pnpm --dir frontend run guides:v2:check`。

## 6) 分层验证

先运行与改动最接近的测试，再按风险扩大范围；不得声称未实际执行的验证已通过。

- 通用/纯文档：`git diff --check`，并检查最终 diff。
- Go 定向测试：`cd backend && go test ./受影响包/...`。
- 后端 CI 门禁：`make -C backend test-unit`、`make -C backend test-integration`、`cd backend && golangci-lint run --timeout=30m`。
- Vue 定向测试：`pnpm --dir frontend exec vitest run <相关 spec>`；再运行 `pnpm --dir frontend run typecheck` 和 `pnpm --dir frontend run lint:check`。
- Vue 全量测试：`pnpm --dir frontend run test:run`。根命令 `make test-frontend` 只运行关键 Vue 测试清单，但还会校验两套前端的 lint、类型和 React 测试。
- 生图子应用：`npm --prefix frontend/image-playground test` 和 `npm --prefix frontend/image-playground run build`。
- 跨前端/构建链路：`make test-frontend`、`make build-frontend`；发布级整体构建使用 `make build`。
- V2 教程：`pnpm --dir frontend run guides:v2:check`；涉及导出成品时按 `docs/static-guides/README.md` 运行相应 `--check` 命令，并做桌面端与移动端页面检查。
- Apple container 脚本：`/bin/bash -n deploy/apple-container.sh` 和 `/bin/bash deploy/tests/apple-container-test.sh`。
- 依赖变化：追加运行 `make audit-frontend`，不得静默降低阈值或放宽审计例外。

若因缺少服务、凭据、平台工具或时间成本未运行某项检查，交付时必须明确说明原因及已完成的替代验证。

## 7) 安全与数据边界

- 不得在代码、文档、终端输出、提交信息或回复中泄露 API Key、Token、Cookie、密码、私钥、真实用户数据或完整上游响应；示例必须使用明显占位符。
- 不读取或改动与任务无关的本地敏感文件和运行数据。日志排查只截取必要片段并先脱敏。
- 删除数据、覆盖配置、修改远程服务、执行生产部署、数据库恢复/迁移、支付或用户数据操作均需用户明确授权；自动提交代码不等于获准部署。
- 上游同步或大范围重构时必须保留本 Fork 的定制行为、部署默认值、国际化覆盖和兼容路径，并用实际 diff/测试证明没有误删。

## 8) Git 交付规则

- 始终在现有 `main` 分支工作，不创建新分支。禁止强推、改写历史或向 `upstream` 推送。
- 仅使用 `git add -- <本轮文件>` 显式暂存；禁止 `git add .` 和 `git add -A`。提交前检查 `git diff` 与 `git diff --cached`，确认没有夹带用户改动或敏感信息。
- 提交信息遵循仓库现有风格，优先使用 `feat:`、`fix:`、`docs:`、`test:`、`refactor:` 或 `chore:`，准确概括本轮改动。
- 最小验证通过后提交并推送到 `origin/main`。若远端更新导致非快进或冲突，停止并报告，禁止用强推覆盖。

## 9) 参考文档

- 项目与开发说明：`README_CN.md`、`DEV_GUIDE.md`（其中版本号可能滞后，以可执行配置为准）。
- 本地/生产部署：`deploy/README.md`、`deploy/DOCKER.md`。
- 数据库迁移：`backend/migrations/README.md`。
- V2 教程与导出：`docs/static-guides/README.md`。
