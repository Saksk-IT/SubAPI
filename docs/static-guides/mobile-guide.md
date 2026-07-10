# 移动端 Chatbox 配置教程

> API base_url：`https://sakai.my/`

前置步骤：请先完成父教程《中转注册、兑换与 API 密钥配置教程》，准备好自己的 `base_url` 和 API Key。本文只讲客户端配置，不再重复注册、兑换和创建密钥。

## 教程要点

- 下载并安装 Chatbox
- 添加 OpenAI response API 兼容提供方
- 获取并选择模型
- 新建对话完成验证

## Chatbox 配置流程

### 1. 前往官网下载 Chatbox

浏览器打开 Chatbox 官网 <https://chatboxai.app/zh>，选择适配你设备的版本。该应用支持 iOS、Android 等移动端平台。

![Chatbox 官网下载页面](../../frontend/public/img/codex-guide/image-32.png)

图 1：前往 Chatbox 官网下载移动端应用。

### 2. 确认安装后的应用界面

下载并安装完成后，打开应用，确认界面与下图类似，再开始添加自定义模型提供方。

![移动端 Chatbox 应用首页](../../frontend/public/img/codex-guide/image-33.png)

图 2：安装完成后的 Chatbox 首页示意。

### 3. 配置步骤

1. 打开应用后，点击左上角菜单按钮。
2. 进入“设置”。
3. 打开“模型提供方”。
4. 点击“添加”，新建一个自定义提供方。
5. API 模式选择 **OpenAI response API 兼容**。
6. 按下方说明填写 API 主机、API 密钥，并获取模型列表。

![点击移动端左上角菜单](../../frontend/public/img/codex-guide/image-34.png)

图 3：打开左上角菜单。

![点击设置入口](../../frontend/public/img/codex-guide/image-35.png)

图 4：进入设置页。

![选择模型提供方](../../frontend/public/img/codex-guide/image-36.png)

图 5：打开模型提供方配置。

![添加新的模型提供方](../../frontend/public/img/codex-guide/image-37.png)

图 6：点击添加。

填写规则：API 主机和 API 密钥请以父教程“使用密钥”弹窗显示的真实配置为准。

![从中转后台复制 API 密钥](../../frontend/public/img/codex-guide/image-40.png)

图 7：在中转后台复制自己的 API 密钥后粘贴到移动端配置里。

#### 3.1 获取并选择模型

1. 点击“获取”拉取当前可用模型列表。
2. 在模型列表中点击右侧加号，添加你要使用的主流模型。
3. 回到配置页检查 API 主机、API 密钥和模型是否都已保存。

![在模型列表中添加主流模型](../../frontend/public/img/codex-guide/image-42.png)

图 8：点击右侧加号添加模型。

### 4. 完成检查

- API 模式已选择 `OpenAI response API 兼容`。
- API 主机与父教程“使用密钥”弹窗显示的地址一致。
- API 密钥来自你自己的 [中转后台](https://sakai.my/keys)，不是教程示例。
- 至少已添加一个可用模型，避免新建对话后没有模型可选。

### 5. 开始使用

配置完成后，新建一个对话，在右下角入口中切换到刚添加的模型即可开始使用。

![打开移动端新对话并点击右下角](../../frontend/public/img/codex-guide/image-44.png)

图 9：新建对话后点击右下角模型入口。

![选择自己配置的移动端模型](../../frontend/public/img/codex-guide/image-45.png)

图 10：下滑找到自己配置的模型并点击使用。

![移动端模型配置成功后的界面](../../frontend/public/img/codex-guide/image-46.png)

图 11：配置成功后即可正常使用。
