# 教程源稿与飞书上传文件

本目录保留教程的可编辑 Markdown 源稿，并生成飞书上传成品。源稿本身已经按“一个父教程 + 六个子教程”整理完成，不再依赖导出脚本切割或改写正文。由于飞书导入 Markdown 时无法稳定识别 `data:` 图片，当前**优先使用 Word 成品**；每张教程截图都直接存放在 `.docx` 文件内部的 `word/media/`，无需额外上传图片目录。

## 推荐维护流程（Word）

1. 修改本目录中的 Markdown 源稿。
2. 执行 `python3 tools/export_word_guides.py` 重新生成 Word 成品。
3. 执行 `python3 tools/export_word_guides.py --check` 确认成品与源稿一致。
4. 先把父教程导入飞书，再将其余六份教程导入为父教程下的子页面。

运行 Word 导出工具需要 `python-docx>=1.2`。如果系统 Python 没有该依赖，可使用项目已配置的 Codex 文档运行时。

## Word 上传目录

`feishu-word/01-中转注册与API密钥/` 中固定包含七份文件：

1. `00-中转注册与API密钥配置教程.docx`（父教程）
2. `01-Codex-API-登录对接教程.docx`
3. `02-Claude-Code-配置教程.docx`
4. `03-Open-Code-配置教程.docx`
5. `04-Open-Claw-配置教程.docx`
6. `05-移动端-Chatbox-配置教程.docx`
7. `06-Cherry-Studio-图像生成教程.docx`

不要直接编辑生成的 `.docx`；需要调整内容时，修改对应 Markdown 源稿后重新生成。

## Markdown 兼容成品

`feishu/01-中转注册与API密钥/` 保留同样层级的七份自包含 Markdown，供归档或其他支持 Base64 图片的平台使用。生成与检查命令：

```bash
python3 tools/export_feishu_guides.py
python3 tools/export_feishu_guides.py --check
```

飞书导入时仍以 Word 成品为准。

## 源稿与网站静态页面映射

| Markdown 源稿 | 线上路由 | Vue 源文件 | 生成内容 |
| --- | --- | --- | --- |
| [registration-key-guide.md](./registration-key-guide.md) | `/registration-key-guide` | `frontend/src/views/public/client-guides/RegistrationKeyGuideContent.vue` | 父教程：注册、兑换、创建 API 密钥 |
| [codex-guide.md](./codex-guide.md) | `/codex-guide` | `frontend/src/views/public/client-guides/CodexGuideContent.vue` | 子教程：Codex 客户端配置 |
| [claude-code-guide.md](./claude-code-guide.md) | `/claude-code-guide` | `frontend/src/views/public/client-guides/ClaudeCodeGuideContent.vue` | 子教程：Claude Code 配置 |
| [open-code-guide.md](./open-code-guide.md) | `/open-code-guide` | `frontend/src/views/public/client-guides/OpenCodeGuideContent.vue` | 子教程：Open Code 配置 |
| [open-claw-guide.md](./open-claw-guide.md) | `/open-claw-guide` | `frontend/src/views/public/client-guides/OpenClawGuideContent.vue` | 子教程：Open Claw 配置 |
| [mobile-guide.md](./mobile-guide.md) | `/mobile-guide` | `frontend/src/views/public/client-guides/MobileGuideContent.vue` | 子教程：移动端 Chatbox 配置 |
| [image-guide.md](./image-guide.md) | `/image-guide` | `frontend/src/views/public/client-guides/ImageGuideContent.vue` | 子教程：Cherry Studio 图像生成 |

公共页面外壳、目录、顶部互跳卡片来自：

- `frontend/src/views/public/ClientGuideView.vue`
- `frontend/src/views/public/client-guide-data.ts`
- `frontend/src/styles/codex-guide.css`

公共图片目录：

- `frontend/public/img/codex-guide/`
- `frontend/public/img/image-guide/`

## 源稿维护要求

- 源稿继续使用 `../../frontend/public/img/...` 图片路径，由导出工具统一内嵌。
- 父教程只保留中转注册、兑换、创建密钥流程；客户端配置统一放入六份子教程，每份子教程都明确引用父教程。
- Markdown 源稿是最终正文，飞书 Markdown 导出器只负责把本地 PNG 转成 Base64，Word 导出器只负责排版并将图片写入 `.docx`。
- 教程截图不得包含可用的 API Key；已知未脱敏或过时截图必须从源稿中移除。
- 对 JSON、TOML、Shell 命令等代码块保持原格式。
- 保留必要的外部网址；文档之间使用父教程、子教程名称描述关系，不写网站内部路由。
