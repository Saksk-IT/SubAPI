<script setup lang="ts">
import { Icon } from '@/components/icons'

import { guideLinks } from '../client-guide-data'
</script>

<template>
  <h1>中转注册、兑换与 API 密钥配置流程</h1>

  <h2 id="guideHierarchy">一、父子教程目录</h2>
  <p>本页是所有客户端配置的父教程。完成注册、兑换和 API 密钥创建后，再进入对应的客户端子教程；子教程不再重复公共流程。</p>
  <div class="codex-doc-table-wrap">
    <table class="codex-doc-table">
      <thead>
        <tr><th>客户端子教程</th><th>主要内容</th></tr>
      </thead>
      <tbody>
        <tr v-for="guide in guideLinks" :key="guide.key">
          <td><a :href="guide.path">{{ guide.title }}</a></td>
          <td>{{ guide.description }}</td>
        </tr>
      </tbody>
    </table>
  </div>

  <h2 id="usageNotes">二、使用前说明</h2>
  <p>本教程适用于小铺购买的额度或订阅兑换码、站内充值或订阅。</p>
  <div class="codex-doc-table-wrap">
    <table id="billingModes" class="codex-doc-table" aria-label="API 密钥分组与计费模式">
      <thead>
        <tr><th>计费模式</th><th>密钥分组</th><th>额度规则</th></tr>
      </thead>
      <tbody>
        <tr>
          <td><strong>订阅模式</strong></td>
          <td>选择相应订阅分组（订阅 xx 刀）</td>
          <td>每天有固定额度，通常在每天 0 点后刷新。</td>
        </tr>
        <tr>
          <td><strong>额度模式</strong></td>
          <td>选择相应消耗额度的分组（OpenAI-Plus、Pro、Claude 等）</td>
          <td>调用时按实际消耗从网站账户余额中扣费。</td>
        </tr>
      </tbody>
    </table>
  </div>
  <p>需要购买兑换码时，可打开 <a href="https://catfk.com/shop/92O8CR0C" target="_blank" rel="noopener noreferrer">小铺卡密自助购买地址</a>。</p>
  <figure class="codex-figure codex-support-qr">
    <img src="/img/codex-guide/image-16.png" alt="问题交流群二维码" loading="lazy">
    <figcaption>有任何问题可扫码加群，联系群主处理。</figcaption>
  </figure>

  <h2 id="registerAccount">三、注册中转账户</h2>
  <ol class="codex-steps-list">
    <li>浏览器打开 <a href="https://sakai.my/register" target="_blank" rel="noopener noreferrer">中转注册页</a>。</li>
    <li>填写邮箱、获取验证码并设置密码。</li>
    <li>输入验证码后完成账户创建。</li>
  </ol>
  <figure class="codex-figure">
    <img src="/img/codex-guide/image.png" alt="中转服务注册页面截图" loading="lazy">
    <figcaption>图 1：输入邮箱、密码和验证码后创建账户。</figcaption>
  </figure>

  <h2 id="redeemBenefits">四、获取并兑换权益</h2>
  <p>账户内还没有可用权益时，可通过站内充值或订阅、卡密自助购买，或兑换已发放的中转兑换码获得权益。</p>
  <ol class="codex-steps-list">
    <li>登录后打开 <a href="https://sakai.my/redeem" target="_blank" rel="noopener noreferrer">兑换页面</a>。</li>
    <li>输入中转兑换码或额度包兑换码。</li>
    <li>点击“兑换”，并等待成功提示。</li>
    <li>打开 <a href="https://sakai.my/profile" target="_blank" rel="noopener noreferrer">个人资料页</a>，确认余额或订阅权益已经到账。</li>
  </ol>
  <figure class="codex-figure">
    <img src="/img/codex-guide/image-1.png" alt="兑换成功后的页面截图" loading="lazy">
    <figcaption>图 2：兑换成功后，账户会获得对应余额或权益。</figcaption>
  </figure>

  <h2 id="createApiKey">五、创建 API 密钥</h2>
  <section class="codex-callout codex-callout--important">
    <p><strong>重要：</strong>本步是所有客户端子教程的共同前置步骤。</p>
  </section>
  <ol class="codex-steps-list">
    <li>登录后打开 <a href="https://sakai.my/keys" target="_blank" rel="noopener noreferrer">API 密钥页面</a>。</li>
    <li>点击“创建密钥”，按用途填写名称，例如 <code>codex</code>、<code>claude-mac</code> 或 <code>opencode-win</code>。</li>
    <li>根据计费模式选择对应的订阅分组，或 <code>OpenAI-Plus</code>、<code>Pro</code>、<code>Claude</code> 等额度分组；具体名称以中转后台实际显示为准。</li>
    <li>保存后回到密钥列表。</li>
  </ol>
  <div class="codex-key-flow" aria-label="API 密钥创建步骤">
    <article class="codex-key-step">
      <div class="codex-key-step__copy">
        <span class="codex-key-step__badge">1</span>
        <h3>创建密钥</h3>
        <p>进入 API 密钥页面后，点击“创建密钥”。</p>
      </div>
      <figure class="codex-key-step__media codex-key-step__media--wide">
        <img src="/img/codex-guide/image-18.png" alt="API 密钥页面暂无密钥并显示创建密钥按钮" loading="lazy">
        <figcaption>图 3：密钥列表为空时，点击页面中的“创建密钥”。</figcaption>
      </figure>
    </article>
    <article class="codex-key-step codex-key-step--focus">
      <div class="codex-key-step__copy">
        <span class="codex-key-step__badge">2</span>
        <h3>选择分组</h3>
        <p>按权益来源选择正确分组，避免后续出现可用额度异常。</p>
      </div>
      <figure class="codex-key-step__media codex-key-step__media--tall">
        <img src="/img/codex-guide/image-15.png" alt="创建密钥弹窗中选择 API 分组" loading="lazy">
        <figcaption>图 4：分组决定密钥使用的权益来源。</figcaption>
      </figure>
    </article>
    <article class="codex-key-step">
      <div class="codex-key-step__copy">
        <span class="codex-key-step__badge">3</span>
        <h3>使用密钥</h3>
        <p>创建后回到密钥列表，点击“使用密钥”。</p>
      </div>
      <figure class="codex-key-step__media codex-key-step__media--wide">
        <img src="/img/codex-guide/image-20.png" alt="密钥列表使用密钥入口" loading="lazy">
        <figcaption>图 5：点击“使用密钥”，进入客户端配置弹窗。</figcaption>
      </figure>
    </article>
  </div>

  <h2 id="clientConfig">六、查看客户端接入配置</h2>
  <ol class="codex-steps-list">
    <li>在密钥列表的操作列点击“使用密钥”。</li>
    <li>在弹窗中选择需要配置的客户端。</li>
    <li>记录弹窗给出的真实 <code>base_url</code> 和 <code>api_key</code>。</li>
    <li>不要复制教程截图中的示例或脱敏密钥。</li>
  </ol>
  <nav id="clientConfigEntries" class="codex-client-guide-grid codex-client-guide-grid--all codex-client-guide-grid--section" aria-label="选择客户端配置教程">
    <a
      v-for="guide in guideLinks"
      :key="guide.key"
      :href="guide.path"
      class="codex-client-guide-card"
    >
      <Icon :name="guide.icon" class="codex-icon" />
      <span><strong>{{ guide.title }}</strong><small>{{ guide.description }}</small></span>
    </a>
  </nav>

  <h2 id="parentFaq">七、常见问题</h2>
  <div class="codex-faq-list">
    <p><strong>Q1：一个账号能创建几把密钥？</strong><br>可创建多把，建议按客户端或设备分别命名，便于审计和单独吊销。</p>
    <p><strong>Q2：泄露密钥怎么办？</strong><br>立即在 API 密钥页面删除原密钥，重新创建一把，然后更新所有客户端配置。</p>
    <p><strong>Q3：Codex、Claude Code 和 Open Code 可以共用一把密钥吗？</strong><br>可以，但不建议。多把密钥可独立吊销，出现问题时更容易定位对应客户端。</p>
    <p><strong>Q4：充值余额和订阅额度如何消费？</strong><br>额度模式按调用量从余额扣款；订阅模式在订阅周期或每日额度内使用，具体规则以中转后台实时显示为准。</p>
    <p><strong>Q5：技术支持怎么联系？</strong><br>可扫描本文前面的交流群二维码，或登录中转站后在个人中心、站内公告、客服入口查看最新联系方式。</p>
  </div>
</template>
