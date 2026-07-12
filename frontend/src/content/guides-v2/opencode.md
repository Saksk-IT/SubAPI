---
title: OpenCode 配置
slug: opencode
summary: 安装 OpenCode，配置自定义 provider，保存 credential 并选择模型完成验证。
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

OpenCode 的自定义 provider 结构保存在 `opencode.json`；`/connect` 只保存 credential，不会替你填写 API 主机或模型。完成这两部分后，再用 `/models` 选择模型。

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

以下结构展示自定义 provider 的核心字段。地址与模型 ID 以“使用密钥”弹窗及当前模型清单为准；API Key 在第 4 步通过 `/connect` 单独保存：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "custom": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Custom API",
      "options": {
        "baseURL": "<使用密钥弹窗中的完整地址>"
      },
      "models": {
        "<当前可用模型 ID>": { "name": "当前可用模型" }
      }
    }
  }
}
```

尖括号内容是不可直接使用的占位符。保存前必须替换为“使用密钥”弹窗中的完整地址和当前模型清单中的真实 ID。

![OpenCode 自定义 provider 配置与凭据分工](/img/guides/v2/opencode/config.webp "provider 结构写入配置文件，credential 通过 connect 单独保存。")

> [!WARNING]
> 字段名是 `baseURL`，不要写成 `base_url`。已有配置时合并 `provider`，不要删除其他 provider。

## 第 4 步：用 connect 保存 credential {#save-provider-credential}

启动 OpenCode，在交互界面输入 `/connect`，选择 `Other`，输入与 `opencode.json` 完全相同的 provider ID `custom`，再粘贴自己的 API Key。`/connect` 只保存 credential 到 `~/.local/share/opencode/auth.json`；自定义 provider 仍必须在 `opencode.json` 配置，地址和模型也只在配置文件中维护。详情可对照[OpenCode 官方 provider 文档](https://opencode.ai/docs/providers/)。

```text
/connect
```

## 第 5 步：选择模型并验证 {#verify-opencode}

在交互界面输入 `/models`，从 `custom` provider 选择当前可用模型，再发送“回复 OK”。出现 `404` 时先核对配置文件中的完整地址，出现 `401` 时重新执行 `/connect` 保存 credential，模型列表不匹配时以当前清单为准。

```text
/models
```

更多情况请查看[统一排错指南](/guides/v2/troubleshooting#fix-404)。
