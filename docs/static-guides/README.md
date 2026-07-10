# 教程 Markdown 源稿与飞书上传文件

本目录保留教程的可编辑 Markdown 源稿，并生成可直接导入飞书的自包含 Markdown。飞书成品中的图片已编码为 `data:image/png;base64,...`，每份文件都不依赖仓库外部的图片目录。

## 维护流程

1. 管理员先修改本目录中的 Markdown 源稿。
2. 执行 `python3 tools/export_feishu_guides.py` 重新生成上传文件。
3. 执行 `python3 tools/export_feishu_guides.py --check` 确认生成稿与源稿一致。
4. 按文件名前的序号，将 `feishu/` 下六份 Markdown 上传到飞书。

## 飞书上传成品

`feishu/` 目录只包含下列六份待上传文件：

1. `01-Codex-API-登录对接教程.md`
2. `02-Claude-Code-配置教程.md`
3. `03-Open-Code-配置教程.md`
4. `04-Open-Claw-配置教程.md`
5. `05-移动端-Chatbox-配置教程.md`
6. `06-Cherry-Studio-图像生成教程.md`

不要直接编辑 `feishu/` 中的生成文件；需要调整内容时，修改上级目录的对应源稿后重新生成。

## 旧静态页面映射

| Markdown 源稿 | 线上路由 | Vue 源文件 | 说明 |
| --- | --- | --- | --- |
| [codex-guide.md](./codex-guide.md) | `/codex-guide` | `frontend/src/views/public/CodexGuideView.vue` | Codex 总教程：注册、兑换、创建 Key、Codex 配置、排错 FAQ |
| [claude-code-guide.md](./claude-code-guide.md) | `/claude-code-guide` | `frontend/src/views/public/client-guides/ClaudeCodeGuideContent.vue` | Claude Code 配置教程 |
| [open-code-guide.md](./open-code-guide.md) | `/open-code-guide` | `frontend/src/views/public/client-guides/OpenCodeGuideContent.vue` | Open Code 配置教程 |
| [open-claw-guide.md](./open-claw-guide.md) | `/open-claw-guide` | `frontend/src/views/public/client-guides/OpenClawGuideContent.vue` | Open Claw 配置教程 |
| [mobile-guide.md](./mobile-guide.md) | `/mobile-guide` | `frontend/src/views/public/client-guides/MobileGuideContent.vue` | 移动端 Chatbox 配置教程 |
| [image-guide.md](./image-guide.md) | `/image-guide` | `frontend/src/views/public/client-guides/ImageGuideContent.vue` | Cherry Studio 图像生成教程 |

公共页面外壳、目录、顶部互跳卡片来自：

- `frontend/src/views/public/ClientGuideView.vue`
- `frontend/src/views/public/client-guide-data.ts`
- `frontend/src/styles/codex-guide.css`

公共图片目录：

- `frontend/public/img/codex-guide/`
- `frontend/public/img/image-guide/`

## 源稿维护要求

修改 Markdown 源稿时，请遵循以下规则：

- 源稿继续使用 `../../frontend/public/img/...` 图片路径，由导出工具统一转为内嵌图片。
- 不要复制示例中的脱敏密钥；页面中所有密钥都必须继续保持占位或脱敏。
- 对 JSON、TOML、Shell 命令等代码块保持原格式。
- 保留外部网址；将网站内部路由改写为飞书文档名称。
