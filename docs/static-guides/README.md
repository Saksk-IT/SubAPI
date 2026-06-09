# 静态教程 Markdown 源稿索引

本目录用于反补当前静态教程页的 Markdown 源稿，方便管理员后续先修改 Markdown，再把修改后的 Markdown 发给 AI，由 AI 同步修改对应 Vue 静态页面。

## 维护流程

1. 管理员先修改本目录中的 Markdown 源稿。
2. 将修改后的 Markdown 发给 AI，并说明要同步到哪个静态页面。
3. AI 按 Markdown 的文字、章节、表格、代码块、图片说明，同步修改对应 Vue 文件。
4. 同步后至少运行前端类型检查或相关路由测试，并用浏览器检查页面。

## 页面映射

| Markdown 源稿 | 线上路由 | Vue 源文件 | 说明 |
| --- | --- | --- | --- |
| [codex-guide.md](./codex-guide.md) | `/codex-guide` | `frontend/src/views/public/CodexGuideView.vue` | Codex 总教程：注册、兑换、创建 Key、Codex 配置、排错 FAQ |
| [claude-code-guide.md](./claude-code-guide.md) | `/claude-code-guide` | `frontend/src/views/public/client-guides/ClaudeCodeGuideContent.vue` | Claude Code 配置教程 |
| [open-code-guide.md](./open-code-guide.md) | `/open-code-guide` | `frontend/src/views/public/client-guides/OpenCodeGuideContent.vue` | Open Code 配置教程 |
| [open-claw-guide.md](./open-claw-guide.md) | `/open-claw-guide` | `frontend/src/views/public/client-guides/OpenClawGuideContent.vue` | Open Claw 配置教程 |
| [mobile-guide.md](./mobile-guide.md) | `/mobile-guide` | `frontend/src/views/public/client-guides/MobileGuideContent.vue` | 移动端 Chatbox 配置教程 |

公共页面外壳、目录、顶部互跳卡片来自：

- `frontend/src/views/public/ClientGuideView.vue`
- `frontend/src/views/public/client-guide-data.ts`
- `frontend/src/styles/codex-guide.css`

公共图片目录：

- `frontend/public/img/codex-guide/`

## 给 AI 的同步要求

同步 Markdown 到 Vue 页面时，请遵循以下规则：

- 保留现有路由、CSS 类名、响应式布局和图标组件用法。
- Markdown 中的图片路径按 Vue 页面写法转换，例如 `../../frontend/public/img/codex-guide/image.png` 对应页面内 `/img/codex-guide/image.png`。
- 不要复制示例中的脱敏密钥；页面中所有密钥都必须继续保持占位或脱敏。
- 如果只改某一个教程，只提交对应 Markdown 和对应 Vue 文件，避免带入无关改动。
- 对 JSON、TOML、Shell 命令等代码块保持原格式。
- 修改表格或排错清单时，同步检查移动端布局不要溢出。
