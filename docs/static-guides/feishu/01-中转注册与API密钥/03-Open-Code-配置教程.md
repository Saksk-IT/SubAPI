# Open Code 配置教程

> API base_url：`https://sakai.my/`

前置步骤：请先完成父教程《中转注册、兑换与 API 密钥配置教程》，准备好自己的 `base_url` 和 API Key。本文只讲客户端配置，不再重复注册、兑换和创建密钥。

## 教程要点

- 安装并首次启动 Open Code
- 配置 `opencode.json`
- 使用 `/connect` 临时切换
- 验证 provider 和模型

## Open Code 配置流程

### 1. 安装并首次启动 Open Code

1. 按你的系统安装 Open Code CLI；如果已经安装，可以直接进入下一步。
2. 打开一个新终端，输入 `opencode` 启动一次，确认命令可用。
3. 首次启动后退出 Open Code，再写入配置文件；这样可以避免配置目录不存在。

提示：Open Code 配置格式会随版本演进，字段细节可对照 [Open Code 官方配置文档](https://opencode.ai/docs/config)。本文示例使用官方支持的自定义 provider 写法。

### 2. 定位 Open Code 配置目录

| 系统 | 配置目录 | 配置文件 |
| --- | --- | --- |
| **Windows** | `%USERPROFILE%\.config\opencode\` | `opencode.json` 或 `opencode.jsonc` |
| **macOS** | `~/.config/opencode/` | `opencode.json` 或 `opencode.jsonc` |
| **Linux** | `~/.config/opencode/` | `opencode.json` 或 `opencode.jsonc` |

提示：目录不存在时先手动创建。Windows 用户记得显示文件扩展名，避免实际文件名变成 `opencode.json.txt`。

### 3. 方式 A：写入 `opencode.json`（推荐，长期生效）

把下面配置写入 `opencode.json`。其中 `baseURL`、`apiKey`、模型 ID 都应以“使用密钥”弹窗或中转后台模型清单为准。

```json
{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "sakms": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "SAKMS",
      "options": {
        "baseURL": "https://sakai.my/v1",
        "apiKey": "填写你的 API 密钥"
      },
      "models": {
        "gpt-5.5": { "name": "GPT-5.5" },
        "gpt-5.4": { "name": "GPT-5.4" }
      }
    }
  }
}
```

- `provider` 下的 `sakms` 是自定义名称，后续在 Open Code 内选择对应 provider 即可。
- `baseURL` 对大小写敏感，Open Code 示例中使用的是 `baseURL`，不要写成 `base_url`。
- 如果你的账户只支持某些模型，就只保留后台可用的模型条目，避免出现 `model not found`。

### 4. 方式 B：客户端内 `/connect` 临时切换

1. 在终端输入 `opencode` 启动客户端。
2. 在交互界面输入 `/connect`。
3. 按提示选择 OpenAI-compatible / custom provider，并填写 `baseURL`：`https://sakai.my/v1`。
4. 继续填写自己的 `apiKey`，选择模型后发起一次测试对话。

```text
/connect
```

此方式适合临时测试、多账号切换或不方便编辑配置文件的环境。

## 验证与排错

- 任意终端输入 `opencode`，选择 `sakms` provider 后发起一次对话，能正常返回即成功。
- 返回 `404` 时，优先检查 `baseURL` 是否包含 `/v1`。
- 返回 `401` 时，重新复制自己的 API Key，确认没有多余空格，也没有复制教程示例。
- 返回额度不足或限流时，打开 [额度查询页](https://sakai.my/profile) 检查余额、订阅日额度和密钥分组。
