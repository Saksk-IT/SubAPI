# Sub2API 生图入口开关与新标签页启动器设计

**日期：** 2026-07-13
**状态：** 已确认
**目标：** 为管理员提供生图入口显示开关，并将现有同源 iframe 改为“当前页选钥、新标签页运行”的完整 React 生图体验。

## 1. 方案结论

采用“公开功能开关 + 全局 Vue Launcher + 同源新标签页 + 每次加载独立 MessageChannel”的结构：

- 新增公开设置 `image_generation_enabled`，默认 `true`。
- `App.vue` 挂载唯一 `ImageGenerationLauncher`；极小 Pinia store 只保存弹窗开关，不保存 API 密钥。
- 侧边栏“生图”点击后阻止路由跳转，在当前页面弹出密钥选择框。
- 用户确认后同步调用 `window.open('/image-playground/', uniqueWindowName)` 创建新标签页。
- React 子应用通过 `window.opener` 完成每次页面加载的发现和端口转移，单次配置过程只使用独立 `MessagePort`。
- 旧 `/image-generation` 路由保留为兼容入口：启用时唤起同一个 Launcher，关闭时由路由守卫重定向。
- 功能开关只控制 Sub2API Web 入口，不改变 `/v1/images/*`、`/v1/responses` 等网关 API 权限。

## 2. 功能开关

管理员在“系统设置 → 功能开关 → 生图功能”中控制入口：

- 默认开启，保持升级兼容。
- 开启时，只有拥有至少一个合格生图密钥的用户显示侧边栏入口。
- 关闭时，普通用户、管理员个人区和简单模式中的入口均隐藏。
- 关闭时，访问旧 `/image-generation` 路由会被重定向到相应仪表盘。
- 已成功打开的生图标签页继续运行，不强制清除其内存态配置。
- `/image-playground/` 静态入口仍可达，但缺少合法 opener、nonce 和受管配置时继续显示直达拒绝页。

该字段完整进入设置常量、默认值、解析、管理员读写、审计、公开设置、HTML 注入、前端类型、feature flag registry 与设置 UI，使用 `opt-out` 语义避免公开设置尚未加载时入口闪烁消失。

## 3. Launcher 交互

1. 用户点击侧边栏“生图”。
2. 全局弹窗分页读取当前用户密钥，只保留：
   - `status === active`
   - `group.platform === openai`
   - `group.allow_image_generation === true`
3. 加载成功后默认选中第一项，用户也可以改选其他密钥。
4. 用户点击“打开生图”时同步创建新标签页，避免异步调用触发浏览器 popup blocker。
5. 弹窗保持连接状态，直至收到受管配置 ACK；成功后关闭。
6. 关闭移动端侧边栏，主页面保持在用户原来的路由，不跳到中间宿主页。

无可用密钥时提供“前往密钥管理”；加载失败提供重试；弹窗被浏览器拦截、标签页提前关闭、握手超时或配置失败时提供明确错误和重新打开操作。

## 4. 新标签页安全协议

每次启动创建独立会话，每次初次加载或刷新都执行一轮状态机：

`pending-ready → pending-connected → pending-configured → done | failed`

协议要求：

- 每次会话使用新的 UUID nonce；window name 仅包含固定前缀与 nonce，不包含密钥。
- 新标签页地址固定为同源 `/image-playground/`，不使用 query、hash、BroadcastChannel、localStorage 或 sessionStorage 传参。
- 父页严格校验 `event.origin`、`event.source === popup WindowProxy`、popup 当前路径、协议版本、nonce、精确字段集合和消息顺序。
- 子页严格校验 `event.origin`、`event.source === window.opener`、协议版本、nonce、精确字段集合和唯一端口。
- 子页收到 connect 后接管当前页面生命周期内唯一的 `MessagePort`，再发送 connected；`window.name` 只保留高熵 nonce，`window.opener` 只保留同源主站引用，用于刷新后重新发现。
- 父页只在 connected 后通过端口发送 configure；API 密钥只存在于父页受管会话内存、MessagePort 队列和子应用受管运行时内存。
- 子页成功或失败时回传相同正整数 requestId；双方随后移除 window listener 并关闭不再需要的端口。
- 重复、越序、来源错误、nonce 错误、额外字段、无端口或多端口消息全部忽略并最终 fail-closed；每次刷新使用递增 requestId 和新的 MessageChannel。
- 8 秒未完成、popup 被关闭或 opener 导航时清理会话；错误信息不得包含 API 密钥或内部异常详情。

`noopener`/`noreferrer` 不能用于该受管标签页，因为它们会移除刷新恢复所需的 opener；安全性通过固定同源 URL、高熵 nonce、严格 source/origin 校验、每轮独立端口以及不持久化密钥获得。主站页面关闭或刷新、账号切换后，内存会话失效，子页后续刷新必须重新从侧边栏进入。

## 5. 组件边界

- `ImageGenerationLauncher.vue`：弹窗 UI、密钥加载、选择、启动状态与错误状态。
- `imageGenerationLauncher` store：仅提供 `open/close` 与可观察弹窗状态，不保存密钥或 popup 引用。
- `popupBridge.ts`：纯协议和会话控制，不依赖 Vue，可单元测试。
- `AppSidebar.vue`：仅把生图菜单标记为 action 并调用 Launcher，不承载弹窗或密钥逻辑。
- `ImageGenerationView.vue`：降级为兼容路由壳，只唤起 Launcher 并返回合适仪表盘，不再渲染 iframe。
- React `sub2apiBridge.ts`：从只支持 `window.parent` 扩展为只接受合法 `window.opener` 的顶层受管模式。

## 6. 生命周期与失败恢复

- Launcher 取消或失败时移除消息监听、清理定时器、关闭端口并丢弃局部密钥引用；成功后配置进入标签页对应的父页内存会话，直到标签页关闭、主站卸载或用户切换。
- 登录用户变化、登出或功能开关变为关闭时，立即关闭尚未完成的 Launcher 会话。
- 成功配置的新标签页在当前页面生命周期内不依赖主页面继续存在；只要原主站页面及其内存会话仍在，刷新子页会自动重新握手并恢复配置。
- 主站页面关闭或刷新、账号切换后不再提供恢复配置；已加载的子页可继续使用，但再次刷新会停留在连接提示，需要从侧边栏重新进入。
- 多次启动使用独立 nonce、WindowProxy、端口与 requestId，禁止共享“当前 popup”全局凭据状态。
- 子应用既有 IndexedDB 历史和收藏继续按 `storageScope` 隔离，但任何凭据均不得进入持久化或导出内容。

## 7. 验收标准

- 管理员可在功能开关中保存、读取并即时应用生图入口状态。
- 关闭时所有侧边栏入口消失，旧路由被拦截，网关 API 不受影响。
- 点击入口不改变当前路由，弹窗在桌面和移动端均可用。
- 选择密钥后打开真正的新标签页，完整 React 生图功能可用。
- popup blocker、无密钥、加载错误、握手错误、超时和提前关闭均有可恢复提示。
- 已正常打开的生图工作台在主站内存会话仍有效时，用户主动刷新后可恢复同一密钥配置。
- origin/source/nonce/schema/顺序/requestId 校验与多标签隔离测试通过。
- API 密钥不出现在 URL、window name、Storage、IndexedDB、导出包、控制台、错误消息或长期 store 中。
- Vue、React、Go 单元与集成测试、生产构建、Docker 构建和真实 HTTP/浏览器关键流程通过。
