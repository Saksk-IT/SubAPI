# Claude Code 配置教程

> API base_url：`https://sakai.my/`

前置步骤：请先完成父教程《中转注册、兑换与 API 密钥配置教程》，准备好自己的 `base_url` 和 API Key。本文只讲客户端配置，不再重复注册、兑换和创建密钥。

## 教程要点

- 定位 Claude 配置目录
- 写入 `settings.json`
- 可选的系统环境变量配置
- 启动 Claude Code 验证

## 开始前准备

在父教程中点击“使用密钥”，切换到 Claude Code 配置区域，复制弹窗里的真实 `base_url` 和 `api_key`。

![Claude Code 配置弹窗示例，密钥已脱敏](../../frontend/public/img/codex-guide/image-22.png)

图：Claude Code 配置示例。截图中的 API Key 已脱敏，请以自己的弹窗为准。
## Claude Code 配置流程

按“使用密钥”弹窗中的 Claude Code 配置，手动写入环境变量。Claude Code 支持在 `settings.json` 里通过 `env` 为每次会话注入变量；也可以直接写到系统环境变量中。

### 1. 定位 Claude 配置目录

| 系统 | 配置目录 | 打开方式 |
| --- | --- | --- |
| **Windows** | `%userprofile%\.claude` | 按 `Win` + `R`，输入 `%userprofile%\.claude` 并回车；目录不存在时可手动新建。 |
| **macOS** | `~/.claude` | 终端执行 `mkdir -p ~/.claude && open ~/.claude`。 |
| **Linux** | `~/.claude` | 终端执行 `mkdir -p ~/.claude && cd ~/.claude`。 |

### 2. 方式 A：写入 `settings.json`（推荐）

在 `~/.claude/settings.json` 中写入下面结构。`ANTHROPIC_BASE_URL` 和 `ANTHROPIC_AUTH_TOKEN` 请复制“使用密钥”弹窗里的真实值；如果弹窗给出的地址带 `/v1`，就照弹窗填写。

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "https://sakai.my",
    "ANTHROPIC_AUTH_TOKEN": "填写你的 API 密钥",
    "ANTHROPIC_MODEL": "gpt-5.5"
  }
}
```

提示：如果文件里已经有其他设置，只新增或合并 `env` 字段，不要覆盖原有 `permissions`、`hooks` 等配置。

### 3. 方式 B：配置系统环境变量

| 系统 | 设置方法 |
| --- | --- |
| **Windows PowerShell** | `setx ANTHROPIC_BASE_URL "https://sakai.my"`<br>`setx ANTHROPIC_AUTH_TOKEN "填写你的 API 密钥"` |
| **macOS / zsh** | 在 `~/.zshrc` 末尾追加 `export ANTHROPIC_BASE_URL="https://sakai.my"` 和 `export ANTHROPIC_AUTH_TOKEN="填写你的 API 密钥"`，保存后执行 `source ~/.zshrc`。 |
| **Linux** | 在 `~/.bashrc` 或 `~/.zshrc` 追加同样的 `export` 语句，再重新打开终端。 |

## 验证与排错

- 打开新终端窗口，输入 `claude`，能进入交互并发起一次对话即配置成功。
- 如果提示认证失败，回到中转站重新复制 Claude Code 配置，并确认没有复制教程截图里的脱敏密钥。
- 如果配置后仍无效，先退出 Claude Code，再关闭旧终端，重新打开终端后再次输入 `claude`。
- 如果提示额度或限流，打开 [额度查询页](https://sakai.my/profile) 检查余额、订阅日额度和分组是否正确。
