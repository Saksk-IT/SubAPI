---
title: Chatbox 移动端配置
slug: chatbox-mobile
summary: 在 iOS 或 Android 添加自定义提供方，获取模型并完成首次测试对话。
duration: 8 分钟
platforms:
  - iOS
  - Android
difficulty: 新手
updatedAt: 2026-07-13
version: v2
---
# Chatbox 移动端配置

本篇适用于 iOS 与 Android。开始前请准备“使用密钥”弹窗中的 API 主机、自己的 API Key 和至少一个可用模型名称。

## 第 1 步：安装并打开 Chatbox {#install-chatbox}

从[Chatbox 官方网站](https://chatboxai.app/zh)进入与你设备对应的官方商店页面。安装完成后打开应用，确认能进入主界面。

### iOS {#ios}

在系统商店确认开发者信息和应用名称，再完成安装。首次启动时按需选择本地权限。

### Android {#android}

优先从官网指向的官方渠道安装。若系统提示安装来源风险，先核对下载来源，不要从陌生链接继续。

## 第 2 步：进入模型提供方设置 {#open-provider-settings}

打开左上角菜单，进入“设置”与“模型提供方”，点击添加自定义提供方。不同版本的入口位置可能略有变化。

![Chatbox 添加自定义模型提供方](/img/guides/v2/chatbox/add-provider.webp "从设置进入模型提供方，再点击添加。")

## 第 3 步：填写 API 主机与密钥 {#configure-host-and-key}

API 模式选择 OpenAI response API 兼容。API 主机按“使用密钥”弹窗填写，通常需要完整的 HTTPS 地址；API Key 粘贴你自己的密钥，不使用教程示例。

> [!WARNING]
> 手机截图容易同步到相册云端。密钥输入完成后不要截屏，也不要在录屏中打开该页面。

## 第 4 步：获取并选择模型 {#fetch-and-select-model}

点击“获取”拉取模型列表，在列表中添加至少一个当前可用模型，并保存提供方。若拉取失败，先核对地址末尾是否按弹窗要求包含 `/v1`。

![Chatbox 获取模型并添加到可选列表](/img/guides/v2/chatbox/select-model.webp "获取模型列表，添加目标模型并保存。")

## 第 5 步：创建测试对话 {#test-chatbox}

返回首页新建对话，从模型选择器切换到刚添加的模型，发送“回复 OK”。能收到回复并且顶部显示目标模型，即完成配置。

认证、地址、限流或模型选择问题请查看[统一排错指南](/guides/v2/troubleshooting)。
