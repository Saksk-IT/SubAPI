# Open Claw 配置教程

> API base_url：`https://sakai.my/`

从注册中转账户、兑换额度、创建 API Key 开始，完整接入 Open Claw。支持腾讯云在线配置，也支持本地 ~/.openclaw 配置。

## 教程要点

- 注册中转
- 腾讯云在线配置
- 本地配置
- 模型测试

## Open Claw 完整接入流程

### 开始前准备

开始前请先完成 API Key 准备：Open Claw 的 OpenAI-compatible 地址通常填写 `https://sakai.my/v1`。如果“使用密钥”弹窗给出的地址不同，请以弹窗为准。

### 1. 从第一步开始：注册、兑换、创建 Key

1. 打开 [中转注册页](https://sakai.my/register)，填写邮箱、验证码和密码，完成账户注册。
2. 使用质保补发兑换码、站内兑换码，或通过 [链动小铺卡密地址](https://catfk.com/shop/92O8CR0C) 购买额度包并兑换。
3. 兑换后打开 [额度查询页](https://sakai.my/profile)，确认余额或订阅权益到账。
4. 进入 [API 密钥页面](https://sakai.my/keys)，点击“创建密钥”，按兑换码来源选择“质保补偿”或 GPT / GPT-Plus 分组。
5. 创建后点击“使用密钥”，复制 Open Claw 或 OpenAI-compatible 配置中的 `base_url` 和 `api_key`。

### 2. 方式 A：腾讯云在线配置（推荐新手）

**适用场景：**你已经在腾讯云开通 Open Claw / 龙虾服务器，希望直接在云端面板中接入中转模型。

1. 登录腾讯云，进入你的 Open Claw / 龙虾服务器控制台。
2. 进入“应用管理”或模型相关页面，找到模型配置入口。
3. 选择“自定义模型”或“JSON 输入”。不同面板版本文字可能略有差异，核心是添加自定义 OpenAI-compatible provider。
4. 把下方 JSON 粘贴进去，将 `api_key` 替换成自己的真实 API Key，再点击“添加并应用”。
5. 回到对话或模型测试页面，选择刚添加的 GPT-5.5 模型并发起一次测试。

```json
{
  "provider": "openai",
  "base_url": "https://sakai.my/v1",
  "api": "openai-completions",
  "api_key": "填写你的 API 密钥",
  "model": {
    "id": "gpt-5.5",
    "name": "GPT-5.5"
  }
}
```

注意：`api_key` 字段务必替换为自己的 API Key，不要提交占位符；模型 ID 以中转后台可用模型清单为准。

### 3. 方式 B：本地配置（Windows / macOS / Linux）

如果你使用本地 Open Claw，可在配置目录中新增中转 provider。官方文档中的 provider 配置使用 `models.providers` 结构；本文示例保留一个 `sakms` provider，便于和默认模型区分。

| 系统 | 配置目录 | 打开方式 |
| --- | --- | --- |
| **Windows** | `%USERPROFILE%\.openclaw` | 按 `Win` + `R`，输入 `%USERPROFILE%\.openclaw` 并回车；不存在时先新建。 |
| **macOS** | `~/.openclaw` | 终端执行 `mkdir -p ~/.openclaw && open ~/.openclaw`。 |
| **Linux** | `~/.openclaw` | 终端执行 `mkdir -p ~/.openclaw && cd ~/.openclaw`。 |

进入上述目录，编辑或新建 `openclaw.json`。如果已有配置，请只合并 `models.providers.sakms` 这一段，避免覆盖原来的本地设置。

```json
{
  "models": {
    "providers": {
      "sakms": {
        "apiKey": "填写你的 API 密钥",
        "baseURL": "https://sakai.my/v1",
        "api": "openai-responses",
        "models": [
          {
            "id": "gpt-5.5",
            "name": "GPT-5.5"
          }
        ]
      }
    }
  }
}
```

提示：腾讯云端示例使用 `openai-completions`，本地 Open Claw 示例使用 `openai-responses`。如果你的客户端版本明确要求另一种 API 类型，请以客户端提示为准。

### 4. 验证与快速检查

- `base_url` / `baseURL` 是否为 `https://sakai.my/v1`，或是否与“使用密钥”弹窗一致。
- `api_key` / `apiKey` 是否已替换成自己的真实 API Key，没有保留“填写你的 API 密钥”。
- 云端配置使用 `openai-completions`，本地配置使用 `openai-responses`，除非客户端版本另有提示。
- 出现 `401` 时重新复制 Key；出现 `404` 时检查 `/v1`；出现额度不足时打开 [额度查询页](https://sakai.my/profile)。
