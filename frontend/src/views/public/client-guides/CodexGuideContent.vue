<script setup lang="ts">
import { Icon } from '@/components/icons'
</script>

<template>
  <h1>Codex 客户端配置流程</h1>
  <section id="codexStart" class="codex-callout codex-callout--important">
    <p><strong>开始前准备：</strong>请先完成<a href="/registration-key-guide">父教程《中转注册、兑换与 API 密钥配置教程》</a>，创建自己的 API Key，并从“使用密钥”弹窗复制 Codex 对应的真实 <code>base_url</code> 和 <code>api_key</code>。</p>
  </section>
  <p>适用范围：<strong>Codex CLI、Codex VSCode 插件、Codex APP</strong>，三者共用同一份 <code>.codex</code> 配置。</p>
  <div class="codex-alert codex-alert--important codex-alert--standout">
    <Icon name="exclamationTriangle" class="codex-icon" />
    <div>
      <strong>下载后必须先打开一次 Codex。</strong>
      <span>先让 Codex 初始化配置目录，然后退出登录并完全关闭 Codex，再编辑配置文件。</span>
    </div>
  </div>
  <a class="codex-link-card" href="https://openai.com/zh-Hans-CN/codex/" target="_blank" rel="noopener noreferrer">
    <Icon name="download" class="codex-icon" />
    <span>打开 Codex 官方下载页面</span>
  </a>
  <figure class="codex-figure">
    <img src="/img/codex-guide/image-6.png" alt="OpenAI Codex 下载页面截图" loading="lazy">
    <figcaption>图 1：点击页面中的下载入口获取 Codex。</figcaption>
  </figure>

  <h2 id="codexManual">1. 手动配置 Codex</h2>
  <p>按父教程“使用密钥”弹窗中的 Codex CLI 配置，手动修改 <code>config.toml</code> 和 <code>auth.json</code>。请直接复制自己弹窗里的完整配置，不要根据教程截图手动拼接地址或密钥。</p>
  <div class="codex-alert codex-alert--important codex-alert--standout">
    <Icon name="exclamationTriangle" class="codex-icon" />
    <div>
      <strong>编辑配置前必须完全关闭 Codex。</strong>
      <span>确认 Codex 进程没有运行，否则刚写入的配置可能被旧进程覆盖。</span>
    </div>
  </div>

  <h3 id="codexWindows">1.1 Windows</h3>
  <ol class="codex-steps-list">
    <li>确认 Codex 已启动过一次，然后退出登录并完全关闭。</li>
    <li>打开用户目录；如看不到隐藏目录，请在资源管理器中启用“显示隐藏项目”。</li>
    <li>进入当前用户的 <code>.codex</code> 目录。</li>
    <li>编辑或新建 <code>config.toml</code> 与 <code>auth.json</code>，文件名不要带隐藏的 <code>.txt</code> 后缀。</li>
    <li>把父教程“使用密钥”弹窗里的 Codex 配置分别写入两个文件并保存。</li>
  </ol>

  <h3 id="codexMac">1.2 macOS</h3>
  <ol class="codex-steps-list">
    <li>在用户目录中找到 <code>.codex</code>；如果看不到，请按 <kbd>Command</kbd> + <kbd>Shift</kbd> + <kbd>.</kbd> 显示隐藏文件。</li>
    <li>确认目录内存在 <code>config.toml</code> 和 <code>auth.json</code>，不存在时按原扩展名新建。</li>
    <li>复制父教程弹窗给出的 Codex 配置，保存后再重新打开 Codex。</li>
  </ol>
  <div class="codex-image-grid">
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-7.png" alt="Mac 访达中找到 .codex 文件夹" loading="lazy">
      <figcaption>图 2：在用户目录中找到 <code>.codex</code>。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-8.png" alt=".codex 文件夹中的 config.toml 和 auth.json" loading="lazy">
      <figcaption>图 3：确认存在 <code>config.toml</code> 和 <code>auth.json</code>。</figcaption>
    </figure>
  </div>

  <h2 id="codexLogin">2. 重新打开 Codex 并使用 API 登录</h2>
  <p>完成文件配置后重新打开 Codex，选择“换种方式登录”，再选择 API 登录并粘贴自己的 API Key。密钥应从父教程中的 <a href="https://sakai.my/keys" target="_blank" rel="noopener noreferrer">API 密钥页面</a>创建并复制。</p>
  <div class="codex-image-grid codex-image-grid--three">
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-10.png" alt="Codex 选择换种方式登录" loading="lazy">
      <figcaption>图 4：选择“换种方式登录”。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-11.png" alt="Codex API 登录输入框" loading="lazy">
      <figcaption>图 5：选择 API 登录并粘贴密钥。</figcaption>
    </figure>
    <figure class="codex-figure">
      <img src="/img/codex-guide/image-12.png" alt="中转服务 API 密钥复制位置，密钥已脱敏" loading="lazy">
      <figcaption>图 6：只复制自己中转后台生成的 API Key。</figcaption>
    </figure>
  </div>

  <h2 id="codexVerify">验证与排错</h2>
  <h3 id="codexRemote">1. 联系群主远程（推荐）</h3>
  <section class="codex-callout codex-callout--important">
    <p>如果仍无法登录或无法判断配置问题，请返回<a href="/registration-key-guide#usageNotes">父教程“使用前说明”</a>，扫描交流群二维码后联系群主远程协助。</p>
  </section>

  <h3>2. 登录失败时快速检查</h3>
  <ul class="codex-checklist">
    <li><Icon name="checkCircle" class="codex-icon" /><span>编辑配置前是否已经完全关闭 Codex？</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span><code>config.toml</code> 和 <code>auth.json</code> 是否位于同一个 <code>.codex</code> 目录？</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span>是否使用了自己的 API Key，而不是教程截图中的示例？</span></li>
    <li><Icon name="checkCircle" class="codex-icon" /><span>创建密钥时是否按计费模式选择了对应订阅分组，或 <code>OpenAI-Plus</code>、<code>Pro</code>、<code>Claude</code> 等额度分组？</span></li>
  </ul>

  <h3>3. 常见报错对照表</h3>
  <div class="codex-doc-table-wrap">
    <table class="codex-doc-table">
      <thead>
        <tr><th>报错 / 现象</th><th>原因</th><th>处理方式</th></tr>
      </thead>
      <tbody>
        <tr>
          <td><code>401 Unauthorized</code> / Incorrect API key</td>
          <td>密钥错误、已删除，或配置时 Codex 仍在运行。</td>
          <td>关闭 Codex，回到<a href="/registration-key-guide#createApiKey">父教程</a>重新创建或复制密钥，再配置并登录。</td>
        </tr>
        <tr>
          <td><code>404 Not Found</code></td>
          <td><code>base_url</code> 写错，或缺少客户端要求的 <code>/v1</code>。</td>
          <td>检查是否与“使用密钥”弹窗完全一致。</td>
        </tr>
        <tr>
          <td><code>quota exceeded</code> / <code>429 Too Many Requests</code></td>
          <td>余额不足、订阅额度耗尽或触发频率限制。</td>
          <td><a href="https://sakai.my/profile" target="_blank" rel="noopener noreferrer">查看余额和额度</a>，必要时等待刷新或补充额度。</td>
        </tr>
        <tr>
          <td><code>model not found</code></td>
          <td>模型 ID 拼错，或当前分组不支持该模型。</td>
          <td>按中转后台当前可用模型清单重新填写。</td>
        </tr>
        <tr>
          <td>客户端启动后无反应</td>
          <td>Codex 仍在使用修改前的配置。</td>
          <td>完全关闭 Codex 后重新打开，再发起一次测试。</td>
        </tr>
        <tr>
          <td><code>config.toml.txt</code> / <code>auth.json.txt</code></td>
          <td>Windows 隐藏文件扩展名，配置被保存成文本文件。</td>
          <td>显示文件扩展名后，把文件名改为 <code>config.toml</code> 或 <code>auth.json</code>。</td>
        </tr>
        <tr>
          <td>原 Codex 聊天记录不见</td>
          <td>切换 API 中转后，本地 provider 不一致。</td>
          <td>参考 <a href="https://github.com/Dailin521/codex-provider-sync/releases" target="_blank" rel="noopener noreferrer">codex-provider-sync releases</a> 恢复。</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
