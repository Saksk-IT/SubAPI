---
title: OpenCode 配置
slug: opencode
summary: 安装 OpenCode，配置长期 provider 或用 /connect 临时切换并验证模型。
duration: 10 分钟
platforms:
  - Windows
  - macOS
  - Linux
difficulty: 入门
updatedAt: 2026-07-13
version: v2
---
# OpenCode 配置

OpenCode 支持在 `opencode.json` 中保存 OpenAI-compatible provider，也支持在客户端内用 `/connect` 临时连接。本篇同时给出两条路径。

## 第 1 步：按官方方式安装 {#install-opencode}

打开[OpenCode 官方安装文档](https://opencode.ai/docs/)，按当前系统选择安装命令。安装后运行一次 `opencode`，确认命令可用并让程序创建配置目录，然后退出。

## 第 2 步：定位配置目录 {#locate-opencode-config}

### Windows {#windows}

配置目录通常为 `%USERPROFILE%\.config\opencode`，文件名使用 `opencode.json` 或当前版本支持的 `opencode.jsonc`。启用文件扩展名显示。

### macOS {#macos}

配置目录为 `~/.config/opencode`。如果不存在，先创建目录，再创建 `opencode.json`。

### Linux {#linux}

配置目录同样为 `~/.config/opencode`。确认你编辑的是当前登录用户的目录。

## 第 3 步：写入长期配置 {#write-opencode-config}

以下结构展示自定义 provider 的核心字段。地址、密钥和模型 ID 以“使用密钥”弹窗及当前模型清单为准：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "custom": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Custom API",
      "options": {
        "baseURL": "<使用密钥弹窗中的完整地址>",
        "apiKey": "sk-example-not-a-real-key"
      },
      "models": {
        "<当前可用模型 ID>": { "name": "当前可用模型" }
      }
    }
  }
}
```

尖括号内容是不可直接使用的占位符。保存前必须替换为“使用密钥”弹窗中的完整地址和当前模型清单中的真实 ID。

![OpenCode 自定义 provider 配置结构](/img/guides/v2/opencode/config.webp "重点核对 baseURL、apiKey 与 models 三个区域。")

> [!WARNING]
> 字段名是 `baseURL`，不要写成 `base_url`。已有配置时合并 `provider`，不要删除其他 provider。

## 第 4 步：需要时用 connect 临时切换 {#connect-temporarily}

启动 OpenCode，在交互界面输入 `/connect`，选择 OpenAI-compatible 或自定义 provider，然后按提示填写 API 主机、API Key 和模型。这条路径适合临时测试；长期使用仍建议保存配置文件。

```text
/connect
```

## 第 5 步：选择模型并验证 {#verify-opencode}

选择刚添加的 `custom` provider 和一个当前可用模型，发送“回复 OK”。出现 `404` 时先核对 `/v1`，出现 `401` 时重新复制自己的密钥，模型列表不匹配时以控制台为准。

更多情况请查看[统一排错指南](/guides/v2/troubleshooting)。
