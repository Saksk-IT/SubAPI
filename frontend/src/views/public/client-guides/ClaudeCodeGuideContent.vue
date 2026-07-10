<script setup lang="ts">
import { Icon } from '@/components/icons'
</script>

<template>
  <h1>Claude Code 配置流程</h1>
  <section id="claudeStart" class="codex-callout codex-callout--important">
    <p><strong>开始前准备：</strong>请先完成<a href="/registration-key-guide">父教程《中转注册、兑换与 API 密钥配置教程》</a>，再从“使用密钥”弹窗复制 Claude Code 对应的真实 <code>base_url</code> 和 <code>api_key</code>。教程截图和示例里的密钥不能直接复制。</p>
  </section>
  <figure class="codex-figure codex-config-result">
    <img src="/img/codex-guide/image-22.png" alt="Claude Code 配置弹窗示例，密钥已脱敏" loading="lazy">
    <figcaption>Claude Code 配置示例，截图中的 API Key 已脱敏。请以自己的“使用密钥”弹窗为准。</figcaption>
  </figure>

  <h2 id="claudeManual">1. 手动配置 Claude Code</h2>
  <p>按“使用密钥”弹窗中的 Claude Code 配置，手动写入环境变量。Claude Code 支持在 <code>settings.json</code> 里通过 <code>env</code> 为每次会话注入变量；也可以直接写到系统环境变量中。</p>

  <h3 id="claudePath">1.1 定位 Claude 配置目录</h3>
  <div class="codex-doc-table-wrap">
    <table class="codex-doc-table">
      <thead><tr><th>系统</th><th>配置目录</th><th>打开方式</th></tr></thead>
      <tbody>
        <tr><td><strong>Windows</strong></td><td><code>%userprofile%\.claude</code></td><td>按 <kbd>Win</kbd> + <kbd>R</kbd>，输入 <code>%userprofile%\.claude</code> 并回车；目录不存在时可手动新建。</td></tr>
        <tr><td><strong>macOS</strong></td><td><code>~/.claude</code></td><td>终端执行 <code>mkdir -p ~/.claude && open ~/.claude</code>。</td></tr>
        <tr><td><strong>Linux</strong></td><td><code>~/.claude</code></td><td>终端执行 <code>mkdir -p ~/.claude && cd ~/.claude</code>。</td></tr>
      </tbody>
    </table>
  </div>

  <h3 id="claudeSettings">1.2 方式 A：写入 <code>settings.json</code>（推荐）</h3>
  <p>在 <code>~/.claude/settings.json</code> 中写入下面结构。<code>ANTHROPIC_BASE_URL</code> 和 <code>ANTHROPIC_AUTH_TOKEN</code> 请复制“使用密钥”弹窗里的真实值；如果弹窗给出的地址带 <code>/v1</code>，就照弹窗填写。</p>
  <pre class="codex-code-block"><code>{
  "env": {
    "ANTHROPIC_BASE_URL": "https://sakai.my",
    "ANTHROPIC_AUTH_TOKEN": "填写你的 API 密钥",
    "ANTHROPIC_MODEL": "gpt-5.5"
  }
}</code></pre>
  <section class="codex-callout">
    <p><strong>提示：</strong>如果文件里已经有其他设置，只新增或合并 <code>env</code> 字段，不要覆盖原有 <code>permissions</code>、<code>hooks</code> 等配置。</p>
  </section>

  <h3 id="claudeEnv">1.3 方式 B：配置系统环境变量</h3>
  <div class="codex-doc-table-wrap">
    <table class="codex-doc-table">
      <thead><tr><th>系统</th><th>设置方法</th></tr></thead>
      <tbody>
        <tr><td><strong>Windows PowerShell</strong></td><td><code>setx ANTHROPIC_BASE_URL "https://sakai.my"</code><br><code>setx ANTHROPIC_AUTH_TOKEN "填写你的 API 密钥"</code></td></tr>
        <tr><td><strong>macOS / zsh</strong></td><td>在 <code>~/.zshrc</code> 末尾追加 <code>export ANTHROPIC_BASE_URL="https://sakai.my"</code> 和 <code>export ANTHROPIC_AUTH_TOKEN="填写你的 API 密钥"</code>，保存后执行 <code>source ~/.zshrc</code>。</td></tr>
        <tr><td><strong>Linux</strong></td><td>在 <code>~/.bashrc</code> 或 <code>~/.zshrc</code> 追加同样的 <code>export</code> 语句，再重新打开终端。</td></tr>
      </tbody>
    </table>
  </div>

  <h2 id="claudeVerify">2. 验证与排错</h2>
  <ul class="codex-checklist">
    <li><Icon name="checkCircle" class="codex-icon" /><span>打开新终端窗口，输入 <code>claude</code>，能进入交互并发起一次对话即配置成功。</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span>如果提示认证失败，回到中转站重新复制 Claude Code 配置，并确认没有复制教程截图里的脱敏密钥。</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span>如果配置后仍无效，先退出 Claude Code，再关闭旧终端，重新打开终端后再次输入 <code>claude</code>。</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span>如果提示额度或限流，打开 <a href="https://sakai.my/profile" target="_blank" rel="noopener noreferrer">额度查询页</a> 检查余额、订阅日额度和分组是否正确。</span></li>
  </ul>
</template>
