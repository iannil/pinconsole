import type { SeoPageContent } from './types';

export const upscopeZh: SeoPageContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: 'PinConsole vs Upscope：开源、自托管的共浏览替代方案',
    description: '对比 PinConsole（开源、自托管、AGPL-3.0）与 Upscope。PinConsole 提供共浏览、录像回放、访客监控——全部运行在你的自有基础设施上。',
    ogTitle: 'PinConsole vs Upscope —— 开源共浏览替代方案',
    ogDescription: '自托管 Upscope 替代品。AGPL-3.0 许可，数据不出域。共浏览、录像回放、访客监控——全在你自己手里。',
  },
  hero: {
    h1: 'PinConsole vs Upscope：开源替代方案',
    subtitle: '两者都提供共浏览和录像回放。区别在于：Upscope 是 SaaS，按坐席收费。PinConsole 是开源、自托管的，AGPL-3.0——你的数据从不出你的基础设施。',
  },
  comparison: {
    rows: [
      { label: '部署模式', pinconsole: '自托管（你的基础设施）', competitor: 'SaaS 云' },
      { label: '许可协议', pinconsole: 'AGPL-3.0（开源）', competitor: '商业闭源' },
      { label: '数据主权', pinconsole: '数据从不出你的服务器', competitor: '数据存储在 Upscope 云端' },
      { label: '共浏览', pinconsole: '完整双向（点击/滚动/代填/跳转）', competitor: '双向共浏览' },
      { label: '录像回放', pinconsole: '已包含（基于 rrweb）', competitor: '附加功能' },
      { label: '实时访客监控', pinconsole: '实时 DOM + 选择性截图', competitor: '仅基础在线状态' },
      { label: 'GDPR 合规', pinconsole: '内置：consent、被遗忘权、IP 截断', competitor: '依赖厂商' },
      { label: '反爬虫', pinconsole: '限流 + UA + 行为 + fingerprint', competitor: '不支持' },
      { label: '价格', pinconsole: '免费（AGPL-3.0）。商用授权可谈。', competitor: '按坐席月付' },
      { label: '部署方式', pinconsole: '一行 docker compose，5 分钟', competitor: '仅 SaaS，无法自托管' },
    ],
  },
  content: {
    sections: [
      {
        heading: '为什么从 Upscope 迁移到 PinConsole？',
        body: 'Upscope 是能力完善的共浏览工具，但其 SaaS 模式意味着你的屏幕共享数据、访客行为和录像会话全部存放在第三方服务器上。对于合规敏感的行业——金融、保险、医疗、政务——这是不可逾越的障碍。PinConsole 提供同样的共浏览和录像回放能力，但全部部署在你的自有基础设施上，数据从不出你的内网。',
      },
      {
        heading: '自托管共浏览，合规无忧',
        body: '当合规团队说"禁止使用 SaaS 屏幕共享工具"时，你只有两个选择：自研（3-4 个月工程投入）或部署 PinConsole（5 分钟）。PinConsole 基于 rrweb 构建——与许多商业共浏览工具使用的技术同源——因此你获得生产级的 DOM 采集、双向互动和录像回放，而不需要写一行基础设施代码。',
      },
      {
        heading: '从免费 AGPL 版本开始，需要时升级',
        body: 'PinConsole 的 AGPL-3.0 版本功能完整——实时访客监控、双向共浏览、录像回放、弹窗和反爬虫。如果你需要将 PinConsole 嵌入商业产品中，可购买商用授权。联系 maintainer 讨论你的场景。',
      },
    ],
  },
  evidence: {
    blocks: [
      {
        quote: 'Upscope Essentials 版：$39/坐席/月，Pro 版：$79/坐席/月，企业版定制报价。所有套餐按坐席月付。仅共浏览——无会话回放、无访客监控、无聊天功能。',
        attribution: 'Upscope 定价页面（2026 年 6 月数据）',
        sourceUrl: 'https://upscope.com/pricing',
      },
      {
        quote: 'Upscope 是共浏览专精工具——单一功能做得很好。但如果团队还需要会话回放、访客监控或主动聊天，必须采购和集成额外工具，增加了成本和运维复杂度。',
        attribution: 'Upscope 功能对比',
        sourceUrl: 'https://upscope.com/features',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'PinConsole 能完全替代 Upscope 吗？',
        answer: 'PinConsole 覆盖了 Upscope 的核心共浏览和录像回放场景。主要区别在于部署模式：Upscope 是 SaaS，PinConsole 是自托管。迁移需要将 PinConsole 部署在你的基础设施上，并将访客 SDK 集成到你的网站或应用中。',
      },
      {
        question: '可以迁移 Upscope 的会话数据到 PinConsole 吗？',
        answer: 'v1 不支持直接数据迁移。但迁移后新产生的会话将被正常采集。如有定制迁移需求，可联系 maintainer 评估。',
      },
      {
        question: 'PinConsole 支持单页应用（React/Vue/Angular）吗？',
        answer: '支持。PinConsole 的访客 SDK 无框架依赖，可与任何 SPA 或 MPA 配合使用。底层使用 rrweb 进行 DOM 采集，能可靠处理动态 DOM 变更。',
      },
    ],
  },
  cta: {
    title: '今天就试试 PinConsole',
    subtitle: '5 分钟自托管部署。无注册、无销售电话、数据不离开你的服务器。',
    primary: { label: '5 分钟自部署', href: '#top' },
    secondary: { label: '预约咨询', href: '#consult' },
  },
};
