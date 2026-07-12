---
title: Cherry Studio 图像生成配置
slug: cherry-studio-image
summary: 添加兼容模型服务与图像模型，从绘画入口完成一次生成验证。
duration: 10 分钟
platforms:
  - Windows
  - macOS
  - Linux
difficulty: 入门
updatedAt: 2026-07-13
version: v2
---
# Cherry Studio 图像生成配置

本篇把图像生成服务添加到 Cherry Studio，并从“绘画”入口验证。当前推荐模型示例为 `gpt-image-2`，该建议核验于 2026-07-13；实际可用模型以控制台实时清单为准。

## 第 1 步：安装 Cherry Studio {#install-cherry-studio}

从[Cherry Studio 官方网站](https://cherry-ai.com/)下载适合系统的版本。安装后打开一次，确认能进入设置页面。

### Windows {#windows}

使用官网提供的 Windows 安装包，完成安装后从开始菜单打开。

### macOS {#macos}

使用与芯片架构匹配的安装包；首次打开若被系统拦截，请核对下载来源与签名。

### Linux {#linux}

选择官网当前提供的 Linux 包格式，并按发行版要求授予运行权限。

## 第 2 步：添加模型服务 {#add-model-service}

进入设置中的“模型服务”，选择支持 OpenAI-compatible 或 New API 的服务项。填写“使用密钥”弹窗给出的 API 地址与自己的密钥。

![Cherry Studio 添加兼容模型服务](/img/guides/v2/cherry-studio/add-service.webp "添加服务时核对 API 地址、密钥和连通性。")

## 第 3 步：添加图像模型 {#add-image-model}

点击获取模型列表，找到当前可用的图像模型。若使用 `gpt-image-2`，将端点类型明确设为“图像生成”，再保存模型。

> [!NOTE]
> 模型名称相同不代表端点类型一定正确。必须确认它被保存为图像生成端点，而不是普通对话模型。

## 第 4 步：从绘画入口生成 {#generate-from-painting}

新建任务并选择“绘画”，切换到刚添加的服务和图像模型。输入一个不含个人信息的简单提示词，然后发送。

![Cherry Studio 从绘画入口发送图像生成任务](/img/guides/v2/cherry-studio/generate-image.webp "进入绘画，选择图像模型并发送提示词。")

## 第 5 步：检查生成结果 {#verify-image-result}

等待结果完整显示，确认没有模型、端点或媒体加载错误。若请求成功但图片无法显示，先保留错误文字，不要反复提交同一任务。

模型不可用、媒体失败或状态码问题请查看[统一排错指南](/guides/v2/troubleshooting)。
