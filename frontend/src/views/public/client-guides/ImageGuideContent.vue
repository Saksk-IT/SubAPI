<script setup lang="ts">
import { Icon } from '@/components/icons'
</script>

<template>
  <h1>Cherry Studio 图像生成流程</h1>
  <section id="imageStart" class="codex-callout codex-callout--important">
    <p><strong>开始前准备：</strong>请先完成<a href="/registration-key-guide">父教程《中转注册、兑换与 API 密钥配置教程》</a>，并从“使用密钥”弹窗复制自己的 API 地址和 API Key。</p>
  </section>
  <section id="imageGuideIntro" class="codex-callout codex-callout--important">
    <p><strong>当前图像生成路径：</strong>生图请走 <code>/imagegen</code> 路径。除“GPT Image 生图专用”分组外，其他分组暂无法在 Codex 中直接调用 <code>image2</code> 生图模型。</p>
    <p>建议按本教程配置 Cherry Studio，图像生成流程更专业，也更方便选择模型和管理提示词。</p>
  </section>

  <h2 id="imageDownload">1. 下载 Cherry Studio</h2>
  <p>打开 Cherry Studio 官网 <a href="https://cherryai.com.cn/" target="_blank" rel="noopener noreferrer">https://cherryai.com.cn/</a>，下载并安装适合你系统的版本。</p>
  <figure class="codex-figure codex-config-result">
    <img src="/img/image-guide/image.png" alt="Cherry Studio 官网下载页面" loading="lazy">
    <figcaption>图 1：进入 Cherry Studio 官网下载客户端。</figcaption>
  </figure>

  <h2 id="imageService">2. 配置模型服务</h2>
  <ol class="codex-steps-list">
    <li>打开 Cherry Studio，点击右上角设置按钮。</li>
    <li>在“模型服务”中找到并选择 <strong>New API</strong>。</li>
    <li>填写父教程“使用密钥”弹窗给出的 API 地址和自己的中转 API Key。</li>
  </ol>
  <div class="codex-image-grid">
    <figure class="codex-figure">
      <img src="/img/image-guide/image-3.png" alt="Cherry Studio 右上角设置按钮" loading="lazy">
      <figcaption>图 2：点击右上角设置按钮。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/image-guide/image-5.png" alt="Cherry Studio 模型服务 New API 入口" loading="lazy">
      <figcaption>图 3：在模型服务中找到 New API。</figcaption>
    </figure>
  </div>
  <section class="codex-callout">
    <p><strong>填写提醒：</strong>API 地址和 API Key 均以自己的“使用密钥”弹窗为准，不要复制教程截图中的旧地址或示例密钥。</p>
  </section>

  <h2 id="imageModel">3. 配置图像生成模型</h2>
  <ol class="codex-steps-list">
    <li>点击“获取模型列表”，拉取当前账号可用模型。</li>
    <li>找到 <code>gpt-image-2</code>，点击右侧加号添加模型。</li>
    <li>在端点类型中选择“图像生成”，然后点击“添加模型”。</li>
    <li>返回模型列表，确认 <code>gpt-image-2</code> 已经以图像生成端点保存。</li>
  </ol>
  <div class="codex-image-grid">
    <figure class="codex-figure">
      <img src="/img/image-guide/image-8.png" alt="Cherry Studio 添加 gpt-image-2 模型" loading="lazy">
      <figcaption>图 4：获取模型列表后，找到 gpt-image-2 并点击右侧加号。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/image-guide/image-9.png" alt="Cherry Studio 选择图像生成端点类型" loading="lazy">
      <figcaption>图 5：端点类型选择图像生成后添加模型。</figcaption>
    </figure>
  </div>

  <h2 id="imageGenerate">4. 开始生图</h2>
  <ol class="codex-steps-list">
    <li>点击上方加号，新建任务。</li>
    <li>选择“绘画”。</li>
    <li>选择刚配置的模型提供商和 <code>gpt-image-2</code> 模型。</li>
    <li>输入提示词并发送，等待图片生成完成。</li>
  </ol>
  <div class="codex-image-grid">
    <figure class="codex-figure">
      <img src="/img/image-guide/image-12.png" alt="Cherry Studio 选择绘画入口" loading="lazy">
      <figcaption>图 6：点击上方加号后，选择“绘画”。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/image-guide/image-13.png" alt="Cherry Studio 选择模型提供商和模型发送提示词" loading="lazy">
      <figcaption>图 7：选择模型提供商和模型后发送提示词。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/image-guide/image-14.png" alt="Cherry Studio 图像生成成功示例" loading="lazy">
      <figcaption>图 8：图片生成成功。</figcaption>
    </figure>
  </div>

  <h2 id="imageCheck">5. 完成检查</h2>
  <ul class="codex-checklist">
    <li><Icon name="checkCircle" class="codex-icon" /><span>模型服务 API 地址与父教程“使用密钥”弹窗一致。</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span>API 密钥来自你自己的中转后台账号，没有复制教程截图或他人密钥。</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span><code>gpt-image-2</code> 已添加，并且端点类型为“图像生成”。</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span>生图入口选择“绘画”，并在发送前确认模型提供商和模型都已切换到刚配置的服务。</span></li>
  </ul>
</template>
