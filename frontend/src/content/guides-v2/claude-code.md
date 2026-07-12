---
title: Claude Code 配置
slug: claude-code
summary: 安装 Claude Code，在 settings.json 与系统环境变量之间二选一并完成终端验证。
duration: 10 分钟
platforms:
  - Windows
  - macOS
  - Linux
difficulty: 入门
updatedAt: 2026-07-13
version: v2
---
# Claude Code 配置

Claude Code 可以从用户级 `settings.json` 或系统环境变量读取自定义服务配置。两种方式二选一，避免同名变量互相覆盖。

## 第 1 步：按官方方式安装 {#install-claude-code}

先阅读[Claude Code 官方安装文档](https://docs.anthropic.com/en/docs/claude-code/overview)，按当前系统完成安装。在新终端运行 `claude --version`，确认命令可用后再配置。

## 第 2 步：定位配置目录 {#locate-claude-config}

### Windows {#windows}

用户配置目录通常是 `%USERPROFILE%\.claude`，配置文件为 `settings.json`。目录不存在时可以新建。

### macOS {#macos}

用户配置目录是 `~/.claude`。可以先执行 `mkdir -p ~/.claude`，再用文本编辑器创建 `settings.json`。

### Linux {#linux}

用户配置目录同样是 `~/.claude`。确认文件归当前用户所有，并避免用管理员身份创建后导致无法保存。

## 第 3 步：在两种方式中二选一 {#choose-configuration}

方式 A 适合只让 Claude Code 使用这些变量；方式 B 适合多个终端会话共享。不要同时配置两套不同值。

![Claude Code 的 settings.json 与系统环境变量二选一](/img/guides/v2/claude-code/settings.webp "两种配置方式只选一种，避免变量覆盖。")

## 第 4 步：配置 settings.json {#configure-settings}

把“使用密钥”弹窗中的真实地址与密钥填入 `env`。已有文件时只合并 `env`，不要覆盖 `permissions` 或 `hooks`：

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "<使用密钥弹窗中的完整地址>",
    "ANTHROPIC_AUTH_TOKEN": "sk-example-not-a-real-key"
  }
}
```

尖括号地址是不可直接使用的占位符，请完整复制“使用密钥”弹窗中的真实值。如果选择系统环境变量，则在 Windows 用户环境变量或 macOS/Linux 的 shell 配置中设置相同的两个变量，并移除 `settings.json` 中的冲突值。

## 第 5 步：重开终端并验证 {#restart-and-verify}

完全关闭旧终端，打开新终端运行 `claude`。进入交互后发送一个简单请求；如果仍读取旧值，检查 shell 配置是否加载、变量名拼写以及 JSON 格式。

> [!SUCCESS]
> 能进入交互并收到回复即完成。认证、地址、限流或模型错误请按[统一排错指南](/guides/v2/troubleshooting)处理。
