---
title: OpenClaw 在线与本地配置
slug: openclaw
summary: 先选择腾讯云在线模式或本地模式，再分别配置 provider 并完成模型测试。
duration: 12 分钟
platforms:
  - Windows
  - macOS
  - Linux
difficulty: 入门
updatedAt: 2026-07-13
version: v2
---
# OpenClaw 在线与本地配置

OpenClaw 的云端面板与本地客户端配置位置不同。开始前先选定一种模式，不要把两套字段混在同一个文件中。

## 第 1 步：选择腾讯云在线或本地模式 {#choose-openclaw-mode}

- 已开通腾讯云 OpenClaw 或相关在线实例：走在线面板路径。
- 在自己的 Windows、macOS 或 Linux 设备运行：走本地配置路径。

![OpenClaw 在线面板与本地配置的模式选择](/img/guides/v2/openclaw/mode-choice.webp "先选择运行位置，再使用对应的独立配置路径。")

## 第 2 步：配置腾讯云在线路径 {#configure-cloud}

登录腾讯云控制台，进入实例的应用管理或模型配置页，选择自定义模型或 JSON 输入。不同面板版本的按钮文字可能不同，核心字段如下：

```json
{
  "provider": "openai",
  "base_url": "<使用密钥弹窗中的完整地址>",
  "api": "openai-completions",
  "api_key": "sk-example-not-a-real-key",
  "model": {
    "id": "<当前可用模型 ID>",
    "name": "当前可用模型"
  }
}
```

尖括号内容是不可直接使用的占位符。请复制“使用密钥”弹窗中的完整地址和当前模型清单中的真实 ID；保存并应用后等待实例更新。不要把示例密钥当作真实密钥。

## 第 3 步：定位本地配置目录 {#locate-local-config}

### Windows {#windows}

目录通常是 `%USERPROFILE%\.openclaw`，配置文件为 `openclaw.json`。

### macOS {#macos}

目录通常是 `~/.openclaw`。目录不存在时先创建，再编辑 `openclaw.json`。

### Linux {#linux}

目录通常也是 `~/.openclaw`。确保文件权限允许当前用户读取。

## 第 4 步：配置本地路径 {#configure-local}

已有文件时只合并 `models.providers.custom`，不要覆盖其他设置：

```json
{
  "models": {
    "providers": {
      "custom": {
        "apiKey": "sk-example-not-a-real-key",
        "baseURL": "<使用密钥弹窗中的完整地址>",
        "api": "openai-responses",
        "models": [
          { "id": "<当前可用模型 ID>", "name": "当前可用模型" }
        ]
      }
    }
  }
}
```

> [!NOTE]
> 在线示例使用 `openai-completions`，本地示例使用 `openai-responses`。如果当前客户端明确提示另一种 API 类型，以客户端与“使用密钥”弹窗为准。

## 第 5 步：完成模型测试 {#test-openclaw}

在线模式回到面板的模型测试页，本地模式重启 OpenClaw；选择刚添加的 provider 和模型，发送一个简单请求。先确认模型被正确选中，再判断返回结果。

若地址、认证、模型或限流报错，请进入[统一排错指南](/guides/v2/troubleshooting)。
