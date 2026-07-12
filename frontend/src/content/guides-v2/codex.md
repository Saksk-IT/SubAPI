---
title: Codex API 配置
slug: codex
summary: 初始化 Codex，按平台定位配置目录，写入配置并使用 API Key 登录验证。
duration: 12 分钟
platforms:
  - Windows
  - macOS
  - Linux
difficulty: 入门
updatedAt: 2026-07-13
version: v2
---
# Codex API 配置

Codex CLI、编辑器插件与桌面应用使用同一套用户级 `.codex` 配置。本篇采用先初始化、再完全关闭、最后编辑 `config.toml` 与 `auth.json` 的稳定流程。

## 第 1 步：下载安装并先打开一次 {#install-and-initialize}

从[Codex 官方页面](https://openai.com/codex/)选择适合当前系统的安装方式。安装后先打开一次，让客户端创建用户配置目录；看到初始界面后退出。

![首次打开 Codex 完成配置目录初始化](/img/guides/v2/codex/initialize.webp "安装后先打开一次，再退出并编辑配置。")

## 第 2 步：完全关闭 Codex {#close-codex}

退出当前会话并关闭所有 Codex 窗口、终端进程和编辑器插件宿主。编辑时仍有进程运行，旧配置可能在退出时覆盖新文件。

> [!WARNING]
> 不要只关闭一个窗口。若不确定，先退出编辑器和终端，再编辑配置文件。

## 第 3 步：定位平台配置目录 {#locate-config}

### Windows {#windows}

目录通常是 `%USERPROFILE%\.codex`。在资源管理器中启用“隐藏的项目”和“文件扩展名”，避免创建出 `config.toml.txt`。

### macOS {#macos}

目录是 `~/.codex`。访达中按 `Command` + `Shift` + `.` 可显示隐藏目录，也可从终端打开用户目录。

### Linux {#linux}

目录是 `~/.codex`。可在终端中创建目录后，用文本编辑器打开其中的配置文件。

![三个系统中的 .codex 目录与两个配置文件](/img/guides/v2/codex/config-folder.webp "不同系统使用同一个用户级 .codex 目录概念。")

## 第 4 步：写入 config.toml 与 auth.json {#write-config-files}

在“使用密钥”弹窗选择 Codex，按弹窗给出的字段写入 `config.toml`。示例仅展示结构，地址与模型以弹窗为准：

```toml
model = "<当前可用模型 ID>"
model_provider = "custom"

[model_providers.custom]
name = "Custom API"
base_url = "<使用密钥弹窗中的完整地址>"
wire_api = "responses"
```

尖括号内容是不可直接使用的占位符。请从“使用密钥”弹窗复制完整地址，并从当前模型清单复制模型 ID。

若弹窗要求 `auth.json`，使用同一弹窗提供的字段名。示例密钥只能写成 `sk-example-not-a-real-key`；实际配置时替换为你自己的密钥，并确保 JSON 逗号与引号正确。

## 第 5 步：重新打开并使用 API 登录 {#api-login}

重新打开 Codex，选择其他登录方式，再选择 API Key 登录，粘贴自己的密钥并确认。不要使用教程示例值。

![Codex 从其他登录方式进入 API Key 登录](/img/guides/v2/codex/api-login.webp "选择 API Key 登录并粘贴自己的密钥。")

## 第 6 步：发送测试并处理专属问题 {#test-codex}

新建会话并发送“回复 OK”。若失败，先检查：两个文件是否位于同一个 `.codex` 目录、扩展名是否正确、`wire_api` 是否与弹窗一致、修改前是否完全关闭客户端。

> [!TIP]
> `config.toml.txt`、旧进程覆盖配置、在错误用户目录下编辑，是 Codex 最常见的三类专属问题。其他状态码请查看[统一排错指南](/guides/v2/troubleshooting)。
