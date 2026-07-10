<script setup lang="ts">
import { Icon } from '@/components/icons'
</script>

<template>
  <h1>移动端 Chatbox 配置流程</h1>
  <section id="mobileStart" class="codex-callout codex-callout--important">
    <p><strong>开始前准备：</strong>请先完成<a href="/registration-key-guide">父教程《中转注册、兑换与 API 密钥配置教程》</a>，准备好自己的 API Key 和“使用密钥”弹窗给出的 API 主机。</p>
  </section>

  <h2 id="mobileDownload">1. 前往官网下载 Chatbox</h2>
  <p>浏览器打开 Chatbox 官网 <a href="https://chatboxai.app/zh" target="_blank" rel="noopener noreferrer">https://chatboxai.app/zh</a>，选择适配你设备的版本。该应用支持 iOS、Android 等移动端平台。</p>
  <figure class="codex-figure">
    <img src="/img/codex-guide/image-32.png" alt="Chatbox 官网下载页面" loading="lazy">
    <figcaption>图 1：前往 Chatbox 官网下载移动端应用。</figcaption>
  </figure>

  <h2 id="mobileHome">2. 确认安装后的应用界面</h2>
  <p>下载并安装完成后，打开应用，确认界面与下图类似，再开始添加自定义模型提供方。</p>
  <figure class="codex-figure">
    <img src="/img/codex-guide/image-33.png" alt="移动端 Chatbox 应用首页" loading="lazy">
    <figcaption>图 2：安装完成后的 Chatbox 首页示意。</figcaption>
  </figure>

  <h2 id="mobileConfig">3. 配置步骤</h2>
  <ol class="codex-steps-list">
    <li>打开应用后，点击左上角菜单按钮。</li>
    <li>进入“设置”。</li>
    <li>打开“模型提供方”。</li>
    <li>点击“添加”，新建一个自定义提供方。</li>
    <li>API 模式选择 <strong>OpenAI response API 兼容</strong>。</li>
    <li>按下方说明填写 API 主机、API 密钥，并获取模型列表。</li>
  </ol>

  <div class="codex-image-grid codex-image-grid--three">
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-34.png" alt="点击移动端左上角菜单" loading="lazy">
      <figcaption>图 3：打开左上角菜单。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-35.png" alt="点击设置入口" loading="lazy">
      <figcaption>图 4：进入设置页。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-36.png" alt="选择模型提供方" loading="lazy">
      <figcaption>图 5：打开模型提供方配置。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-37.png" alt="添加新的模型提供方" loading="lazy">
      <figcaption>图 6：点击添加。</figcaption>
    </figure>
  </div>

  <section class="codex-callout">
    <p><strong>填写规则：</strong>API 模式选择 <strong>OpenAI response API 兼容</strong>；API 主机和 API 密钥以父教程“使用密钥”弹窗为准，不要根据旧截图手动填写。</p>
  </section>

  <figure class="codex-figure codex-config-result">
    <img src="/img/codex-guide/image-40.png" alt="从中转后台复制 API 密钥" loading="lazy">
    <figcaption>图 7：在中转后台复制自己的 API 密钥后粘贴到移动端配置里。</figcaption>
  </figure>

  <h3 id="mobileModels">3.1 获取并选择模型</h3>
  <ol class="codex-steps-list">
    <li>点击“获取”拉取当前可用模型列表。</li>
    <li>在模型列表中点击右侧加号，添加你要使用的主流模型。</li>
    <li>回到配置页检查 API 主机、API 密钥和模型是否都已保存。</li>
  </ol>

  <div class="codex-image-grid">
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-42.png" alt="在模型列表中添加主流模型" loading="lazy">
      <figcaption>图 8：获取模型列表后，点击右侧加号添加模型。</figcaption>
    </figure>
  </div>

  <h2 id="mobileCheck">4. 完成检查</h2>
  <ul class="codex-checklist">
    <li><Icon name="checkCircle" class="codex-icon" /><span>API 模式已选择 <code>OpenAI response API 兼容</code>。</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span>API 主机与父教程“使用密钥”弹窗给出的地址一致。</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span>API 密钥来自你自己的 <a href="https://sakai.my/keys" target="_blank" rel="noopener noreferrer">中转后台</a>，不是教程示例。</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span>至少已添加一个可用模型，避免新建对话后没有模型可选。</span></li>
  </ul>

  <h2 id="mobileUse">5. 开始使用</h2>
  <p>配置完成后，新建一个对话，在右下角入口中切换到刚添加的模型即可开始使用。</p>
  <div class="codex-image-grid">
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-44.png" alt="打开移动端新对话并点击右下角" loading="lazy">
      <figcaption>图 9：新建对话后点击右下角模型入口。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-45.png" alt="选择自己配置的移动端模型" loading="lazy">
      <figcaption>图 10：下滑找到自己配置的模型并点击使用。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-46.png" alt="移动端模型配置成功后的界面" loading="lazy">
      <figcaption>图 11：配置成功后即可正常使用。</figcaption>
    </figure>
  </div>
</template>
