---
title: API 客户端配置指南
slug: index
summary: 从创建 API Key 到选择客户端、完成验证和排查问题的一站式入口。
duration: 3 分钟
platforms:
  - Windows
  - macOS
  - Linux
  - iOS
  - Android
difficulty: 新手
updatedAt: 2026-07-13
version: v2
---
# API 客户端配置指南

这里是全部 V2 教程的入口。你可以从零开始完成账户与密钥准备，也可以带着已有 API Key 直接选择客户端。页面不依赖固定网站品牌，配置值以“使用密钥”弹窗显示为准。

![从准备密钥到完成验证的三段式流程](/img/guides/v2/common/setup-flow.webp "先准备连接信息，再配置客户端，最后完成一次最小验证。")

## 第 1 步：选择你的起点 {#choose-start}

**从零开始**：先阅读[注册、获取权益与创建 API Key](/guides/v2/get-started)，完成密钥创建和分组选择，再返回本页选择客户端。

**已有 API Key**：确认你同时知道 API 主机地址、密钥适用分组和可用模型，然后直接进入下一步。

> [!TIP]
> 建议为每台设备或每个客户端创建独立密钥。以后需要吊销时，不会影响其他设备。

## 第 2 步：选择客户端 {#choose-client}

以下六条路径互相独立，只需阅读与你当前工具对应的一篇：

- [Codex](/guides/v2/codex)：适合 Codex CLI、编辑器插件与桌面应用，需要配置 `.codex` 目录。
- [Claude Code](/guides/v2/claude-code)：适合终端开发工作流，可选 `settings.json` 或系统环境变量。
- [OpenCode](/guides/v2/opencode)：适合自定义 OpenAI-compatible provider，配置文件、credential 与模型选择步骤相互独立。
- [OpenClaw](/guides/v2/openclaw)：先在腾讯云在线模式与本地模式之间二选一。
- [Chatbox 移动端](/guides/v2/chatbox-mobile)：适用于 iOS 与 Android 的对话客户端。
- [Cherry Studio 图像生成](/guides/v2/cherry-studio-image)：用于添加图像模型并从绘画入口生成图片。

## 第 3 步：完成一次最小验证 {#verify-once}

配置完成后，只发送一个不含隐私的简单请求，例如“回复 OK”。确认客户端能正常返回、模型名称正确，再开始正式使用。

> [!SUCCESS]
> 能返回内容只证明当前地址、密钥、模型组合可用。仍请妥善保存密钥，不要放进截图、聊天记录或公开仓库。

## 第 4 步：从症状进入排错 {#open-troubleshooting}

如果遇到 `401`、`404`、`429`、`model not found`、配置未生效、文件扩展名或媒体加载失败，请直接打开[统一排错指南](/guides/v2/troubleshooting)。先记录完整错误文字和发生步骤，再逐项检查，通常比反复重装更快。
