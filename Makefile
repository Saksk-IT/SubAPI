.PHONY: build build-backend build-frontend build-frontend-vue build-frontend-image-playground dev-frontend \
	test test-backend test-frontend test-frontend-critical test-image-playground audit-frontend

# 审计例外检查脚本使用 Python 3.10+ 语法；优先选择本机可用的新版本。
PYTHON ?= $(shell for candidate in python3.13 python3.12 python3.11 python3.10 python3; do \
	command -v $$candidate 2>/dev/null && break; \
done)

FRONTEND_CRITICAL_VITEST := \
	src/views/auth/__tests__/LinuxDoCallbackView.spec.ts \
	src/views/auth/__tests__/WechatCallbackView.spec.ts \
	src/views/user/__tests__/PaymentView.spec.ts \
	src/views/user/__tests__/PaymentResultView.spec.ts \
	src/components/user/profile/__tests__/ProfileInfoCard.spec.ts \
	src/views/admin/__tests__/SettingsView.spec.ts \
	src/composables/__tests__/useImageGenerationAccess.spec.ts \
	src/features/imagePlayground/__tests__/bridge.spec.ts \
	src/features/imagePlayground/__tests__/viteDevProxy.spec.ts \
	src/views/user/__tests__/ImageGenerationView.spec.ts \
	src/components/layout/__tests__/AppSidebar.spec.ts

# 一键编译前后端。后端嵌入前端产物，因此必须先完成两个前端构建。
build: build-backend

# 编译后端（复用 backend/Makefile）
build-backend: build-frontend
	@$(MAKE) -C backend build

# 编译 Vue 主站，再编译 React 生图应用；React 的 Vite 配置直接输出到 Go 嵌入目录。
build-frontend: build-frontend-image-playground

build-frontend-vue:
	@pnpm --dir frontend run build:vue

build-frontend-image-playground: build-frontend-vue
	@npm --prefix frontend/image-playground run build

# 同时启动 Vue 主站和 React 生图子应用，保证 /image-playground/ 开发代理可用。
dev-frontend:
	@pnpm --dir frontend run dev:all

# 运行测试（后端 + 前端）
test: test-backend test-frontend

test-backend:
	@$(MAKE) -C backend test

test-frontend:
	@pnpm --dir frontend exec eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts --ignore-pattern image-playground/
	@pnpm --dir frontend run typecheck
	@$(MAKE) test-frontend-critical
	@$(MAKE) test-image-playground

test-frontend-critical:
	@pnpm --dir frontend exec vitest run $(FRONTEND_CRITICAL_VITEST)

test-image-playground:
	@npm --prefix frontend/image-playground test

audit-frontend:
	@audit_file=$$(mktemp); \
		pnpm --dir frontend audit --prod --audit-level=high --json > $$audit_file || true; \
		$(PYTHON) tools/check_pnpm_audit_exceptions.py --audit $$audit_file --exceptions .github/audit-exceptions.yml; \
		status=$$?; rm -f $$audit_file; exit $$status
	@npm --prefix frontend/image-playground audit --omit=dev --audit-level=high
