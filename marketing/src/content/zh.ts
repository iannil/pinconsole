import type { PageContent } from './types';
import { zhNavLinks, zhNavCta, zhLocaleSwitch } from './nav-shared';

export const zh: PageContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: 'PinConsole — 你的访客，你的数据。',
    description:
      '开源 ToB 实时访客监控 + 运营互动 + 录像回放平台。AGPL-3.0，自托管，数据从不出门。竞品 SaaS 的开源替代。PinConsole 是开源的自托管共浏览和录像回放方案。',
    ogTitle: 'PinConsole — 你的访客，你的数据。',
    ogDescription:
      '开源 ToB 实时访客监控 + 运营互动 + 录像回放平台。自托管，AGPL-3.0，数据从不出门。',
  },
  nav: { links: [...zhNavLinks], cta: zhNavCta, localeSwitch: { ...zhLocaleSwitch } },
  hero: {
    eyebrow: 'AGPL-3.0 · 自托管 · 数据主权',
    h1: '你的访客，\n你的数据。',
    h2:
      'PinConsole 是开源、自托管的实时访客监控、共浏览（co-browsing）和录像回放平台。年费 30-100 万的 SaaS 锁住你的数据、锁住你的功能、年年涨价。',
    cta: {
      primary: { label: '预约咨询', href: '#consult' },
      secondary: { label: '5 分钟自部署', href: '#data-sovereignty' },
      tertiary: { label: 'GitHub ★', href: 'https://github.com/iannil/pinconsole' },
    },
  },
  features: {
    eyebrow: '现已交付',
    title: '商业竞品能做的，这里都齐了',
    subtitle: '90+ commits，65 e2e 测试全绿。下面五项核心能力，每一项都已经端到端验证，可以现在就跑在你的服务器上。',
    items: [
      {
        icon: 'eye',
        title: '实时访客监控',
        description: 'rrweb 全量采集 DOM 变更 + 鼠标 + 点击 + 滚动 + 失焦表单值。1fps WebP 选择性截图（canvas / WebGL / 跨域 iframe）。',
        bullets: [
          'DOM 全量 + 选择性截图',
          '行为序列化（rrweb 节点 ID 稳定）',
          '访客 SDK 同源分发，单 JS 文件',
        ],
        screenshot: '/screenshots/dashboard.webp',
      },
      {
        icon: 'arrows-out-cardinal',
        title: '双向协同 (co-browsing)',
        description: '运营可高亮 / 点击 / 滚动 / 代填 / 跳转。防抖 300ms 平衡体验与流量。访客随时紧急退出。',
        bullets: [
          '节点 ID 选择器（不用 CSS/XPath）',
          '代填防抖动 300ms',
          '跳转接管 + 跨页面会话续接',
        ],
        screenshot: '/screenshots/cobrowse-active.webp',
      },
      {
        icon: 'video',
        title: '录像回放',
        description: 'MinIO 归档事件流 + 选择性截图。默认 30 天可配，GDPR 删除接口。rrweb-player 标准 replayer。',
        bullets: [
          '事件流压缩归档（MessagePack）',
          '默认保留 30 天，可配置',
          'GDPR 被遗忘权一键删除',
        ],
        screenshot: '/screenshots/replay.webp',
      },
      {
        icon: 'chat-circle-dots',
        title: '弹窗 + 双向聊天',
        description: '消息通道持久化到 PostgreSQL。弹窗推送支持文本 / 链接 / 跳转。聊天记录与 session 绑定。',
        bullets: [
          '弹窗 + 聊天共用通道',
          '历史消息按 session 检索',
          '运营 1:1 锁定避免冲突',
        ],
        screenshot: '/screenshots/chat.webp',
      },
      {
        icon: 'shield-check',
        title: '反爬虫 + GDPR',
        description: 'rate limit + UA 黑名单 + 行为分析 + canvas/WebGL fingerprint。consent opt-in + IP 截断 + co-browse 横幅。',
        bullets: [
          'Redis 滑动窗口限流',
          'GDPR consent opt-in 默认',
          'IP 截断 /24，行为日志保留可配',
        ],
        screenshot: '/screenshots/privacy.webp',
      },
    ],
  },
  dataSovereignty: {
    eyebrow: '数据主权可验证',
    title: '为什么你的数据，真的在你手里',
    subtitle:
      '三个层面的设计决策，让"数据主权"从一句营销口号变成可验证的工程事实。',
    pillars: [
      {
        icon: 'scale',
        title: 'AGPL-3.0 强 copyleft',
        description:
          '任何对 PinConsole 的修改必须开源。云厂商不能拿去做 SaaS——这是 license 层的硬保护。你用的是真正的开源，不是"source available"。',
      },
      {
        icon: 'stack',
        title: '标准栈，无锁定',
        description:
          'PostgreSQL 16 · Redis 7 · MinIO · Go 1.22 · Vue 3。每一层都是行业标准，Schema 在你手里，数据可随时导出迁出。',
      },
      {
        icon: 'certificate',
        title: '合规就绪',
        description:
          'GDPR consent opt-in + 被遗忘权 + IP 截断；HttpOnly cookie + bcrypt；命令授权 + popup URL 白名单；WS trace_id 端到端可观测。',
      },
    ],
    architectureAlt: 'PinConsole 架构图：访客 SDK → pinconsole-server → PostgreSQL/Redis/MinIO，全部在你的基础设施内',
  },
  selfHost: {
    eyebrow: '5 分钟跑起来',
    title: '一行 docker compose，\n跑在你自己的服务器',
    subtitle:
      '没有云厂商依赖，没有 marketplace 集成，没有"激活"流程。clone 下来，跑 docker compose，单二进制就起来了。',
    code: `git clone https://github.com/iannil/pinconsole
cd pinconsole
cp .env.example .env

make docker-up build-frontend build
./server/bin/pinconsole-server

# 访客落地页 → http://localhost:8080/
# 运营后台  → http://localhost:8080/admin`,
    docsLink: { label: '完整部署文档 →', href: 'https://github.com/iannil/pinconsole#readme' },
  },
  roadmap: {
    eyebrow: '立场透明',
    title: '现已交付。下一步去哪。',
    subtitle:
      '我们公开承诺做什么、不做什么。立场不会因为商业压力改变——这是 OSS 替代品的意义。',
    columns: [
      {
        status: 'shipped',
        title: '现已交付',
        items: [
          '实时访客监控 (rrweb 全量)',
          '双向协同 (cursor/click/scroll/fill/navigate)',
          '录像归档 + 历史回放',
          '弹窗推送 + 双向聊天',
          '认证 + 多运营 claim/release 锁',
          '反爬虫 (rate limit + UA + 行为 + fingerprint)',
          'GDPR consent + 被遗忘权 + IP 截断',
          '可观测性 (LifecycleTracker + trace_id)',
          '中英双语 i18n',
          'Docker Compose 一键部署',
        ],
      },
      {
        status: 'coming',
        title: '计划中',
        items: [
          '自定义域名 (DNS 验证 + ACME)',
          '页面编辑器 (低代码 / 拖拽)',
          'Tauri 桌面端 (Win + Mac)',
          'SSO / SAML / OIDC (企业)',
          '反爬加固 (CAPTCHA + honeypot)',
          '分析仪表盘 (漏斗 / 热力图)',
          'Redis Pub/Sub 多实例 hub',
        ],
      },
      {
        status: 'out-of-scope',
        title: '明确不做',
        items: [
          '多租户 SaaS',
          '订阅计费',
          '注册流 / self-signup',
          '云托管服务',
          '立场不变',
        ],
      },
    ],
  },
  faq: {
    eyebrow: '常见疑问',
    title: '决策者想问的',
    subtitle: '如果你在这里没找到答案，表单留言，我们 48h 内回复。',
    items: [
      {
        question: 'AGPL-3.0 我们公司商用合规吗？',
        answer:
          'AGPL 要求"对外提供服务"时必须开源修改。公司内部使用（不对外服务）不触发。你用 PinConsole 服务你自己的访客，不对外销售 PinConsole，没有任何合规问题。云厂商拿去做 SaaS 才需要开源——这正是 license 的保护机制。',
      },
      {
        question: '单人开发，能撑多久？',
        answer:
          'v1 90+ commits 已完成端到端切片，PLAN.md §8 公开 post-v1 路线。咨询收入支撑持续维护。如果项目对你关键，建议购买咨询/定制服务——你的付费是项目可持续的最好保障。',
      },
      {
        question: '能不能定制开发？',
        answer:
          '能，这是咨询的核心。表单里写明需求，48h 内回复评估。定制开发可单独计费，所有代码按 AGPL-3.0 归还给项目（你的私有 deployment 不需要开源）。',
      },
      {
        question: '能不能托管我们的部署？',
        answer:
          'v1 不做托管，长期立场也是不做。避免与 OSS 用户竞争。可推荐合作的部署伙伴（你自选），或协助你搭建自有部署。',
      },
      {
        question: '500 并发不够怎么办？',
        answer:
          'v1 是单实例 hub（process-local map）。多实例需要 Redis Pub/Sub 总线，是 post-v1 切片。如果你的房间并发需求 > 500，建议咨询时说明，我们可以优先排期或定制。',
      },
      {
        question: '能不能过等保 / ISO27001？',
        answer:
          '产品层合规就绪（GDPR / bcrypt / 审计日志 / consent）。等保 / ISO27001 需结合你的部署环境评估。咨询可协助合规文档梳理、架构对照、第三方评估配合。',
      },
    ],
  },
  finalCTA: {
    eyebrow: '聊聊你的场景',
    title: '我们不卖订阅。我们讨论你的场景。',
    subtitle:
      '提交后数据存于 maintainer 自托管 Cloudflare D1（亚洲 region），不分享第三方。我们不会群发邮件，只在你的场景上下文回复。',
    form: {
      nameLabel: '姓名 *',
      namePlaceholder: '张三',
      companyLabel: '公司 *',
      companyPlaceholder: 'XX 科技有限公司',
      contactLabel: '联系方式（手机或邮箱）*',
      contactPlaceholder: '+86 138xxxx 或 you@company.com',
      purposeLabel: '咨询用途 *',
      purposes: [
        { value: 'evaluate', label: '评估替代现有 SaaS' },
        { value: 'self-host', label: '自托管部署咨询' },
        { value: 'custom', label: '定制开发' },
        { value: 'compliance', label: '合规咨询（GDPR/等保/ISO）' },
        { value: 'other', label: '其他' },
      ],
      messageLabel: '留言（可选）',
      messagePlaceholder: '当前用什么平台？想解决什么问题？期望什么时候上线？',
      submitLabel: '提交咨询',
      privacyNote:
        '提交即同意我们将留言用于咨询回复。数据存于 maintainer 自托管 Cloudflare D1，不分享第三方，可邮件申请删除。',
      successMessage: '已收到。我们 48 小时内回复到你留下的联系方式。',
      errorMessage: '提交失败，请稍后再试或直接邮件 contact@pinconsole.com。',
    },
  },
  footer: {
    tagline: '你的访客，你的数据。',
    columns: [
      {
        title: '产品',
        links: [
          { label: '能力', href: '#features' },
          { label: '数据主权', href: '#data-sovereignty' },
          { label: '路线图', href: '#roadmap' },
        ],
      },
      {
        title: '对比',
        links: [
          { label: 'vs Upscope', href: '/alternatives/upscope' },
          { label: 'vs Cobrowse.io', href: '/alternatives/cobrowse-io' },
        ],
      },
      {
        title: '资源',
        links: [
          { label: '私有化共浏览', href: '/co-browsing/self-hosted' },
          { label: '私有化录像回放', href: '/session-replay/self-hosted' },
          { label: 'GitHub 仓库', href: 'https://github.com/iannil/pinconsole' },
          { label: '部署文档', href: 'https://github.com/iannil/pinconsole#readme' },
          { label: 'FAQ', href: '#faq' },
          { label: '咨询', href: '#consult' },
        ],
      },
      {
        title: '法律',
        links: [
          { label: 'AGPL-3.0 License', href: 'https://github.com/iannil/pinconsole/blob/master/LICENSE' },
          { label: '隐私说明', href: '#consult' },
        ],
      },
    ],
    license: 'AGPL-3.0-or-later',
    sourceNote: '© 2026 PinConsole. Built with Calm Crafted design system.',
  },
};
