# Sub2API 同源生图应用集成设计

**日期：** 2026-07-13
**目标：** 将 `CookSleep/gpt_image_playground` 的完整生图体验作为 Sub2API 内置、同源、独立构建的 React 应用接入，在侧边栏提供“生图”入口，同时复用 Sub2API 用户密钥、权限、计费与网关能力。

## 1. 方案结论

采用“Vue 宿主页 + 同源 iframe + 独立 React 构建产物”的结构：

- Vue 路由：`/image-generation`
- React 静态入口：`/image-playground/`
- React 构建目录：`backend/internal/web/dist/image-playground/`
- 网关请求仍走当前站点的 `/v1/images/generations`、`/v1/images/edits` 与 `/v1/responses`
- Vue 与 React 通过严格校验的 `postMessage` 协议传递内存态配置

React 应用保持自己的依赖、全局样式、Zustand 状态、IndexedDB 历史与组件体系，避免把超大 React 页面硬改成 Vue 或污染现有 Vue 依赖图。

## 2. 用户体验

1. 有可用 OpenAI 生图密钥的已登录用户在侧边栏看到“生图”。
2. 点击后进入 `/image-generation`，宿主页加载同源 React 应用。
3. 宿主页分页读取当前用户的活跃 API 密钥，只选择：
   - `status === active`
   - `group.platform === openai`
   - `group.allow_image_generation === true`
4. React 应用收到受管配置后可使用图库、文生图、图片编辑、蒙版、流式局部图、收藏、历史、导入导出与 Agent 多轮工作流。
5. 无可用密钥、密钥加载失败、子应用加载失败时显示可操作的中文错误状态。
6. 桌面端使用完整工作区；移动端 iframe 占满可用视口并提供触控友好布局。

## 3. 信任与安全边界

同源 iframe 配合 `allow-scripts` 与 `allow-same-origin` 不是安全隔离。移植后的 React 代码必须视为 Sub2API 的一等可信代码并接受同等审查。采用以下控制：

- 消息双方同时校验 `event.origin`、`event.source`、消息类型、协议版本与一次性会话 nonce。
- nonce 通过 iframe 的 `name` 传入，不进入 URL、日志或 Referer。
- 只向子应用发送符合权限条件的 Sub2API API 密钥；绝不发送 JWT、refresh token 或管理员凭据。
- API 密钥只保存在 React 运行时内存：禁止写入 Zustand persist、localStorage、IndexedDB、导出 ZIP、URL 参数或错误日志。
- 嵌入模式禁用自定义 Provider、fal.ai、任意 base URL、URL 配置导入与密钥编辑/复制。
- 子应用 API base URL 固定为当前站点 `/v1`，网络请求仅允许同源。
- 禁用上游 Service Worker，避免其清理或缓存同源其他路径资源。
- 移除外部字体/脚本/样式依赖；React 路径使用收紧的独立 CSP。
- 仅 `/image-playground/` 响应允许 `SAMEORIGIN` 与 `frame-ancestors 'self'`；其余页面继续使用 `DENY` 与 `frame-ancestors 'none'`。
- Vue 宿主页的 CSP 明确加入 `frame-src 'self'`。

## 4. 构建与静态托管

构建顺序固定为：

1. Vue 构建并清空 `backend/internal/web/dist`。
2. React 以 `/image-playground/` 为 production base 构建到上述 dist 子目录。
3. Go 的现有 `embed` 静态文件系统一并嵌入两个应用。

根 Makefile、Dockerfile、deploy Dockerfile、CI release 和 security scan 均安装/构建/审计两个前端。React 子应用保留上游 MIT LICENSE，并记录固定上游提交与后续同步方法。

## 5. 状态与配置模型

Vue 宿主提供只读 credential profiles。每个可用 Sub2API 密钥派生两个受管 profile：

- Images profile：用于 Images API 的生成与编辑。
- Responses profile：用于 Responses image generation 与 Agent 文本阶段。

React 内部可选择 profile、模型、尺寸、质量、背景、格式、压缩、流式与局部图等业务参数；凭据、base URL 与 Provider 由宿主锁定。凭据刷新时以不可变更新替换内存态 profiles。

## 6. 失败与恢复

- 宿主密钥请求失败：保留重试按钮，不启动带空配置的生成请求。
- iframe 超时或协议错误：显示连接失败并允许重新加载。
- 子应用运行中密钥失效：沿用现有 API 错误展示，并允许宿主重新同步。
- 页面离开：销毁消息监听、定时器与内存中的 credential profiles。
- 历史、收藏和图片数据仍使用独立 IndexedDB；导出数据只包含非敏感设置。

## 7. 验收标准

- 有权限用户可见并进入“生图”，无权限用户不显示入口且路由内有二次保护。
- 文生图、编辑/蒙版、Responses 流式、Agent、多图历史、收藏、ZIP 导入导出可用。
- 密钥不会出现在 URL、localStorage、IndexedDB、导出包、控制台或网络日志中。
- React 子路径能被开发环境、Go embed、Docker 与 release 构建正确托管。
- CSP 允许宿主嵌入子应用，但不放宽其他页面的防嵌入策略。
- Vue/React 单元测试、Go 测试、类型检查、生产构建与浏览器关键流程通过。
