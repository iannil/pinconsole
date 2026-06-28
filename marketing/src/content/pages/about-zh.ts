import type { PageContent } from '../../content/types';

export const zhAbout: PageContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: '关于 PinConsole — 开源自托管访客监控与共浏览平台',
    description: 'PinConsole 是一款开源、自托管的实时访客监控与运营互动平台，AGPL-3.0 许可。了解我们的使命、价值观与开源承诺。',
    ogTitle: '关于 PinConsole — 开源自托管访客监控平台',
    ogDescription: 'PinConsole 是一款开源（AGPL-3.0）自托管的实时访客监控、共浏览与录像回放平台。数据主权第一。',
  },
  hero: {
    eyebrow: 'About',
    h1: '关于 PinConsole',
    subtitle: '开源、自托管、数据主权优先。构建竞品的开源替代品。',
  },
  sections: [
    {
      heading: '我们的使命',
      body: 'PinConsole 的使命很简单：为 ToB 实时访客监控、运营互动和录像回放领域提供一个真正开源、自托管的替代方案。\n\n我们相信数据主权——你的访客数据应该属于你，而不是 SaaS 供应商的云。我们相信开源——社区应该能够审计、修改和信任他们依赖的基础设施。我们相信可持续性——通过商业授权为开源开发提供资金，而不牺牲 AGPL-3.0 的核心理念。',
    },
    {
      heading: '技术栈',
      body: '后端：Go + Gin + coder/websocket + PostgreSQL + Redis + MinIO\n前端：Vue 3 + TypeScript + Vite + Pinia\n访客 SDK：TypeScript + rrweb（DOM 全量采集）\n部署：单二进制（Go embed），Docker Compose 一键启动',
    },
    {
      heading: '核心功能',
      body: '• 实时访客监控：看到访客在你网站上的一举一动\n• 双向共浏览：运营可直接操作访客浏览器（点击、滚动、代填）\n• 会话录像回放：完整 DOM 记录，支持快进/倒放/速度控制\n• 主动弹窗聊天：在正确的时间与访客互动\n• 隐私保护：选择性脱敏、GDPR 合规、被遗忘权\n• 纵深防御：四层反爬虫体系',
    },
    {
      heading: '开源承诺',
      body: 'PinConsole 在 AGPL-3.0 许可下发布。这意味着：\n\n• 你可以自由自托管使用\n• 你可以审计每一行代码\n• 你可以 Fork 和维护自己的分支\n• 如果你将修改版本作为 SaaS 提供服务，必须将修改以相同许可发布\n\n对于需要在专有产品中嵌入 PinConsole 的团队，我们提供标准的商业授权。',
    },
    {
      heading: '联系维护者',
      body: 'PinConsole 由 Rong Zhu 独立开发与维护。\n\n• GitHub：https://github.com/iannil/pinconsole\n• 技术咨询、合规评估、商业授权：请通过下方表单联系',
    },
  ],
  cta: {
    title: '开始使用 PinConsole',
    subtitle: '自托管部署只需 5 分钟。AGPL-3.0。你的数据，你的服务器。',
    primary: { label: 'GitHub 上开始', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: '联系维护者', href: '#consult' },
  },
};
