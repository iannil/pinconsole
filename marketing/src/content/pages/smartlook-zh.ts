import type { SeoPageContent } from './types';

export const smartlookZh: SeoPageContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: 'PinConsole vs Smartlook：开源、自托管的 Session Replay 替代方案',
    description: '对比 PinConsole（开源、自托管、AGPL-3.0）与 Smartlook。两者都提供会话回放和分析，只有 PinConsole 给你自托管部署、共浏览、实时访客监控，以及无需按会话付费的定价模式。',
    ogTitle: 'PinConsole vs Smartlook —— 开源会话回放替代方案',
    ogDescription: '自托管 Smartlook 替代品。AGPL-3.0，你的数据在你的服务器上。会话回放、共浏览、实时访客监控、反爬虫——零会话上限。',
  },
  hero: {
    h1: 'PinConsole vs Smartlook：开源替代方案',
    subtitle: 'Smartlook 提供会话回放、热力图和产品分析，免费版每月高达 3,000 次会话——这是业界最慷慨的免费额度之一。但它是纯 SaaS 的，付费版按会话量计费。PinConsole 提供会话回放、共浏览、实时监控和反爬虫保护，全部在一个自托管二进制中，AGPL-3.0 免费使用，零会话上限。',
  },
  comparison: {
    rows: [
      { label: '部署模式', pinconsole: '自托管（你的基础设施）', competitor: '纯 SaaS 云' },
      { label: '许可协议', pinconsole: 'AGPL-3.0（开源）', competitor: '商业闭源' },
      { label: '数据主权', pinconsole: '数据从不出你的服务器', competitor: '数据存储在 Smartlook 云端（欧盟）' },
      { label: '会话回放', pinconsole: '基于 rrweb，无限会话', competitor: '免费 3k 次/月，付费版按量' },
      { label: '热力图', pinconsole: '计划中（路线图）', competitor: '点击/移动/滚动热力图' },
      { label: '共浏览', pinconsole: '已包含（双向点击/滚动/代填/跳转）', competitor: '不包含' },
      { label: '访客监控', pinconsole: '实时 DOM + 选择性截图', competitor: '仅历史回放' },
      { label: '弹窗 & 主动聊天', pinconsole: '推送通知 + 双向聊天', competitor: '不包含' },
      { label: '反爬虫保护', pinconsole: '限流 + UA + 行为 + fingerprint', competitor: '不包含' },
      { label: 'SDK 大小', pinconsole: '~15KB gzip', competitor: '~25KB+（全功能）' },
      { label: '价格', pinconsole: '免费（AGPL-3.0）。商用授权可谈。', competitor: '免费（3k 次/月）。付费版按会话量计价' },
      { label: '部署方式', pinconsole: 'docker compose，5 分钟', competitor: '仅 SaaS 代码片段' },
    ],
  },
  content: {
    sections: [
      {
        heading: '为什么从 Smartlook 迁移到 PinConsole？',
        body: 'Smartlook 是一款优秀的会话回放和产品分析工具，免费版每月 3,000 次会话是业界最大方的额度之一。它在产品团队和移动端开发者中特别受欢迎。但 Smartlook 是纯 SaaS 的——你的会话数据存储在欧盟云端，付费版按会话量计价。没有共浏览，没有实时访客监控，也没有自托管选项。PinConsole 是一个自托管替代方案，在一个二进制中覆盖了会话回放、共浏览、实时监控和反爬虫保护——零会话上限，零重复费用。',
      },
      {
        heading: '从会话配额到无限录制',
        body: 'Smartlook 免费版每月 3,000 次会话是业界最大方的额度之一——适合小项目和早期创业公司。但成长中的网站会很快超出这个限制：月会话量 10,000 的网站需要付费版，价格随量增长。而 PinConsole 没有会话配额——每一次访客交互都会被录制，无需计算配额。唯一成本是你的基础设施。',
      },
      {
        heading: 'Smartlook 不具备的实时能力',
        body: 'Smartlook 擅长历史会话回放和产品分析——你事后观看录制记录。但它无法让你实时看到访客行为或与用户互动。PinConsole 增加了三项 Smartlook 没有的能力：实时访客监控（实时观看会话）、主动弹窗聊天（在访客离开前联系他们）、和双向共浏览（引导遇到困难的访客完成结账流程）。这些功能将会话回放从分析工具转变为实时客户运营平台。',
      },
      {
        heading: '在中国市场的独特优势',
        body: 'Smartlook 的服务部署在欧盟云端，对中国大陆用户访问速度较慢。SDK 加载和数据上传经常受到跨国网络连接的影响。PinConsole 完全自托管，SDK 与你的应用部署在同一网络中——国内团队无需翻墙即可流畅使用所有功能。所有数据境内存储，直接满足个保法要求。',
      },
      {
        heading: '开源，无厂商依赖',
        body: 'Smartlook 是闭源的——你的分析工作流依赖于他们的持续运营、定价和功能决策。使用 PinConsole，AGPL-3.0 许可保证你永远可以自托管、审计代码、fork 项目并自主控制。无突发价格变更，无功能下架，无强制迁移。对于重视长期独立性的产品团队，PinConsole 提供了核心会话回放能力加上额外的实时功能——全部自托管且免费。',
      },
    ],
  },
  evidence: {
    blocks: [
      {
        quote: 'Smartlook 免费版：每月 3,000 次会话，1 个月数据保留。付费版按会话量计价。高级套餐支持移动端 SDK。所有套餐仅 SaaS——不提供自托管部署选项。',
        attribution: 'Smartlook 定价页面（2026 年 6 月数据）',
        sourceUrl: 'https://www.smartlook.com/pricing/',
      },
      {
        quote: 'Smartlook 提供会话回放、热力图和产品分析，支持 iOS 和 Android 移动端 SDK。它不提供共浏览、实时访客监控或主动聊天——这些需要额外工具。',
        attribution: 'Smartlook 功能概览',
        sourceUrl: 'https://www.smartlook.com/features/',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'PinConsole 能替代 Smartlook 的会话回放功能吗？',
        answer: 'Web 会话回放方面——可以，PinConsole 提供基于 rrweb 的无限录制。Smartlook 在移动端 SDK（iOS/Android）方面有优势——PinConsole v1 仅支持 Web。产品分析和热力图方面——尚未支持，这些在路线图中。PinConsole 提供了 Smartlook 不具备的功能：共浏览、实时监控和主动聊天。',
      },
      {
        question: 'PinConsole 支持移动端会话录制吗？',
        answer: 'v1 不支持——PinConsole v1 仅面向 Web。移动端 SDK（iOS/Android）在 post-v1 待办中。对于纯移动端项目，Smartlook 仍然是更好的选择。对于以 Web 为主的团队，PinConsole 覆盖了会话回放加上 Smartlook 没有的实时功能。',
      },
      {
        question: 'PinConsole 的数据保留策略与 Smartlook 相比如何？',
        answer: 'PinConsole 的数据保留完全由你控制——你可以为每个站点设置 TTL（7/30/90 天或永久），由 MinIO 生命周期策略执行。Smartlook 免费版包含 1 个月保留，付费版提供更长保留期。使用 PinConsole，没有基于订阅套餐的保留限制。',
      },
      {
        question: 'PinConsole 适合需要分析能力的产品团队吗？',
        answer: '适合——PinConsole 覆盖会话回放和实时监控，这是产品团队的核心需求。如果你重度依赖 Smartlook 的产品分析仪表板（漏斗、趋势、用户路径），可以在过渡期同时使用两者。PinConsole 处理回放和监控侧，同时你可以评估分析功能是否满足需求。',
      },
    ],
  },
  cta: {
    title: '5 分钟自托管 PinConsole',
    subtitle: '无注册门槛。零会话上限。无销售电话。',
    primary: { label: '开始部署', href: '#top' },
    secondary: { label: '联系 maintainer', href: '#consult' },
  },
};
