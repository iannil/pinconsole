import type { SeoPageContent } from './types';

export const fullstoryZh: SeoPageContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: 'PinConsole vs FullStory：开源、自托管的 Session Replay 替代方案',
    description: '对比 PinConsole（开源、自托管、AGPL-3.0）与 FullStory。两者都提供会话回放和分析，只有 PinConsole 给你自托管部署、共浏览、访客实时监控，以及无需按回放付费的定价模式。',
    ogTitle: 'PinConsole vs FullStory —— 开源会话回放替代方案',
    ogDescription: '自托管 FullStory 替代品。AGPL-3.0，你的数据在你的服务器上。会话回放、共浏览、实时访客监控、反爬虫——全功能单二进制。',
  },
  hero: {
    h1: 'PinConsole vs FullStory：开源替代方案',
    subtitle: 'FullStory 是会话回放和数字体验分析领域的行业标准——但它是纯 SaaS、闭源的，企业版起价 $500+/月。PinConsole 提供会话回放、共浏览、实时访客监控和反爬虫保护，全部打包在一个自托管二进制中，AGPL-3.0 免费使用。',
  },
  comparison: {
    rows: [
      { label: '部署模式', pinconsole: '自托管（你的基础设施）', competitor: '纯 SaaS 云' },
      { label: '许可协议', pinconsole: 'AGPL-3.0（开源）', competitor: '商业闭源' },
      { label: '数据主权', pinconsole: '数据从不出你的服务器', competitor: '数据存储在 FullStory 云端' },
      { label: '会话回放', pinconsole: '基于 rrweb，自托管存储', competitor: '全功能，含 AI 搜索' },
      { label: '共浏览', pinconsole: '已包含（双向点击/滚动/代填/跳转）', competitor: '不包含（需额外工具）' },
      { label: '访客监控', pinconsole: '实时 DOM + 选择性截图', competitor: '仅历史数据分析' },
      { label: '热力图 / 愤怒点击', pinconsole: '计划中（路线图）', competitor: '全功能' },
      { label: '弹窗 & 主动聊天', pinconsole: '推送通知 + 双向聊天', competitor: '不包含' },
      { label: '反爬虫保护', pinconsole: '限流 + UA + 行为 + fingerprint', competitor: '不包含' },
      { label: 'SDK 大小', pinconsole: '轻量（~15KB gzip）', competitor: '~50KB+（较重）' },
      { label: '价格', pinconsole: '免费（AGPL-3.0）。商用授权可谈。', competitor: '从 $500/月（Business 版）' },
      { label: '部署方式', pinconsole: 'docker compose，5 分钟', competitor: '仅 SaaS 代码片段' },
    ],
  },
  content: {
    sections: [
      {
        heading: '为什么从 FullStory 迁移到 PinConsole？',
        body: 'FullStory 是功能强大的数字体验平台，拥有业界领先的会话回放、AI 搜索和分析功能。但对许多团队来说，纯 SaaS 模式带来了挑战：你的会话数据存储在 FullStory 的服务器上，成本随用量快速上升（Business 版起价 $500/月），而且没有共浏览或实时访客监控——你需要额外购买其他工具来弥补。PinConsole 是一个自托管替代方案，在一个二进制中覆盖了会话回放、共浏览、实时监控和反爬虫保护，无需按会话量付费。',
      },
      {
        heading: '自托管意味着数据在你的控制之下',
        body: '使用 FullStory，每一次会话录制都被传输并存储在他们的云端基础设施上。对于需要遵守 GDPR、等保、个保法的组织来说，这可能是一个合规障碍。PinConsole 完全运行在你的基础设施上——会话数据、DOM 快照和用户交互从不出你的服务器。你可以自主控制保留策略、备份计划和数据删除——无厂商锁定，无突发策略变更。',
      },
      {
        heading: '不止是回放——完整的客户运营套件',
        body: 'FullStory 擅长会话回放和分析，但它不涵盖共浏览、实时访客监控或客户互动工具。PinConsole 被设计为一个完整的客户运营平台：回放历史会话以调试问题，实时观看访客行为，通过弹窗聊天主动联系访客，并通过共浏览协助支持——全部来自同一个管理界面。此外，内置的反爬虫保护可确保你的分析数据不受自动化流量的干扰。',
      },
      {
        heading: '在中国市场的独特优势',
        body: 'FullStory 在中国大陆访问速度极慢，其 SaaS 服务需要跨国网络连接，经常导致 SDK 加载超时或录制失败。PinConsole 完全自托管部署，SDK 与你的应用部署在同一网络环境中——国内团队无需翻墙即可流畅使用所有功能。结合数据境内存储的合规优势，PinConsole 是中国企业进行访客分析和客户运营的理想选择。',
      },
      {
        heading: '开源，可审计，免费',
        body: 'FullStory 是闭源的——你无法审计代码、自托管部署或影响产品路线图。PinConsole 是 AGPL-3.0 开源项目：完整源代码在 GitHub 上，你可以审计每一行代码、fork 项目、无限期自托管。无需许可费用，无需按坐席付费，无突发涨价。商用授权可用于需要闭源集成的场景。',
      },
    ],
  },
  evidence: {
    blocks: [
      {
        quote: 'FullStory Business 版起价 $599/月——10,000 次会话已包含，超出部分 $0.04/次。需年付承诺。不提供自托管部署选项。',
        attribution: 'FullStory 定价页面（2026 年 6 月数据）',
        sourceUrl: 'https://www.fullstory.com/plans/',
      },
      {
        quote: 'FullStory 的 JavaScript SDK 压缩后约 50KB+，可能影响页面加载性能。官方文档建议异步加载以缓解这一问题。在中国大陆，SDK 加载经常因网络延迟而超时。',
        attribution: 'FullStory 开发者文档 & 实际使用反馈',
        sourceUrl: 'https://developer.fullstory.com/',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'PinConsole 能完全替代 FullStory 的会话回放功能吗？',
        answer: '可以——PinConsole 使用 rrweb 进行基于 DOM 的会话采集和回放，与许多生产工具使用相同的底层技术。它会记录所有用户交互（点击、滚动、输入），并以像素级精度回放。FullStory 拥有更高级的 AI 搜索和分析功能，但 PinConsole 覆盖了核心回放场景，外加 FullStory 不具备的共浏览和实时监控能力。',
      },
      {
        question: 'PinConsole 有热力图和愤怒点击检测吗？',
        answer: '尚未支持——热力图和愤怒点击分析已在 PinConsole 路线图中。如果团队需要这些功能，可以暂时将 PinConsole 作为核心回放和监控平台，同时继续使用 FullStory 的免费版或专注分析工具来获得热力图数据。',
      },
      {
        question: 'PinConsole 如何处理 GDPR 和数据隐私合规？',
        answer: '由于 PinConsole 是自托管的，所有会话数据保留在你的基础设施上——无第三方数据处理，无跨境数据传输。这使得 GDPR 合规比纯 SaaS 方案（如 FullStory）简单得多，因为 FullStory 的数据是在其美国云端处理的。对于中国用户，数据境内存储也直接满足个保法要求。',
      },
      {
        question: 'PinConsole 适合大规模会话录制吗？',
        answer: '是的——PinConsole 基于 Go 构建，使用 PostgreSQL、Redis 和 MinIO 存储，为生产环境设计。架构支持并发 WebSocket 连接（500+/房间）和基于 MinIO 的高效事件流存储。通过 Docker Compose 即可快速部署。',
      },
    ],
  },
  cta: {
    title: '5 分钟自托管 PinConsole',
    subtitle: '无注册门槛。无销售电话。clone，compose，部署。',
    primary: { label: '开始部署', href: '#top' },
    secondary: { label: '联系 maintainer', href: '#consult' },
  },
};
