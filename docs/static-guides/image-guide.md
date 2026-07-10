# Cherry Studio 图像生成教程

> API base_url：`https://sakai.my/`

前置步骤：请先完成父教程《中转注册、兑换与 API 密钥配置教程》，准备好自己的 `base_url` 和 API Key。本文只讲客户端配置，不再重复注册、兑换和创建密钥。

## 教程要点

- 安装 Cherry Studio
- 配置 New API 模型服务
- 添加 `gpt-image-2` 图像生成模型
- 使用绘画入口生成图片

## Cherry Studio 图像生成流程

### 当前图像生成路径说明

当前生图请走 `/imagegen` 路径。除“GPT Image 生图专用”分组外，其他分组暂无法在 Codex 中直接调用 `image2` 生图模型。

建议按本教程配置 Cherry Studio，图像生成流程更专业，也更方便选择模型和管理提示词。

### 1. 下载 Cherry Studio

打开 Cherry Studio 官网 <https://cherryai.com.cn/>，下载并安装适合你系统的版本。

![Cherry Studio 官网下载页面](../../frontend/public/img/image-guide/image.png)

图 1：进入 Cherry Studio 官网下载客户端。

### 2. 配置模型服务

1. 打开 Cherry Studio，点击右上角设置按钮。
2. 在“模型服务”中找到并选择 **New API**。
3. 填写 API 地址和密钥，具体值以父教程“使用密钥”弹窗显示的真实配置为准。

![Cherry Studio 右上角设置按钮](../../frontend/public/img/image-guide/image-3.png)

图 2：点击右上角设置按钮。

![Cherry Studio 模型服务 New API 入口](../../frontend/public/img/image-guide/image-5.png)

图 3：在模型服务中找到 New API。

填写提醒：API 地址和 API 密钥都以父教程“使用密钥”弹窗为准，不要复制教程截图中的示例值。

### 3. 配置图像生成模型

1. 点击“获取模型列表”，拉取当前账号可用模型。
2. 找到 `gpt-image-2`，点击右侧加号添加模型。
3. 在端点类型中选择“图像生成”，然后点击“添加模型”。
4. 返回模型列表，确认 `gpt-image-2` 已经以图像生成端点保存。

![Cherry Studio 添加 gpt-image-2 模型](../../frontend/public/img/image-guide/image-8.png)

图 4：找到 gpt-image-2 并点击右侧加号。

![Cherry Studio 选择图像生成端点类型](../../frontend/public/img/image-guide/image-9.png)

图 5：端点类型选择图像生成后添加模型。

### 4. 开始生图

1. 点击上方加号，新建任务。
2. 选择“绘画”。
3. 选择刚配置的模型提供商和 `gpt-image-2` 模型。
4. 输入提示词并发送，等待图片生成完成。

![Cherry Studio 选择绘画入口](../../frontend/public/img/image-guide/image-12.png)

图 6：选择绘画。

![Cherry Studio 选择模型提供商和模型发送提示词](../../frontend/public/img/image-guide/image-13.png)

图 7：选择模型提供商和模型后发送提示词。

![Cherry Studio 图像生成成功示例](../../frontend/public/img/image-guide/image-14.png)

图 8：图片生成成功。

### 5. 完成检查

- 模型服务 API 地址与父教程“使用密钥”弹窗显示的地址一致。
- API 密钥来自你自己的中转后台账号，没有复制教程截图或他人密钥。
- `gpt-image-2` 已添加，并且端点类型为“图像生成”。
- 生图入口选择“绘画”，并在发送前确认模型提供商和模型都已切换到刚配置的服务。
