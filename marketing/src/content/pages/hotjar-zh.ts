import type { SeoPageContent } from './types';

export const hotjarZh: SeoPageContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: 'PinConsole vs Hotjar：开源、自托管的 Session Replay 和热力图替代方案',
    description: '对比 PinConsole（开源、自托管、AGPL-3.0）与 Hotjar。两者都提供会话回放和热力图，只有 PinConsole 给你自托管部署、共浏览、零会话上限和实时访客监控。',
    ogTitle: 'PinConsole vs Hotjar —— 开源会话回放替代方案',
    ogDescription: '自托管 Hotjar 替代品。AGPL-3.0，你的数据在你的服务器上。会话回放、共浏览、实时访客监控、反爬虫——零会话上限。',
  },
  hero: {
    h1: 'PinConsole vs Hotjar：开源替代方案',
    subtitle: 'Hotjar 是最受欢迎的会话回放和热力图工具之一——但免费版每天仅限 35 次录制，付费版起价 $32/月（约 ¥230/月），不提供自托管选项。PinConsole 提供会话回放、共浏览、实时监控和反爬虫保护，全部在一个自托管二进制中，AGPL-3.0 免费使用，零会话上限。',
  },
  comparison: {
    rows: [
      { label: '部署模式', pinconsole: '自托管（你的基础设施）', competitor: '纯 SaaS 云' },
      { label: '许可协议', pinconsole: 'AGPL-3.0（开源）', competitor: '商业闭源' },
      { label: '数据主权', pinconsole: '数据从不出你的服务器', competitor: '数据存储在 Hotjar 云端（欧盟/美国）' },
      { label: '会话回放', pinconsole: '基于 rrweb，无限会话', competitor: '免费版 35 次/天，付费版按额度' },
      { label: '热力图', pinconsole: '计划中（路线图 2026 Q3）', competitor: '全功能点击/移动/滚动热力图' },
      { label: '共浏览', pinconsole: '已包含（双向点击/滚动/代填/跳转）', competitor: '不包含' },
      { label: '访客监控', pinconsole: '实时 DOM + 选择性截图', competitor: '仅历史分析' },
      { label: '弹窗 & 调查', pinconsole: '推送通知 + 双向聊天', competitor: '包含问卷和反馈小组件' },
      { label: '反爬虫保护', pinconsole: '限流 + UA + 行为 + fingerprint', competitor: '不包含' },
      { label: 'SDK 大小', pinconsole: '~15KB gzip', competitor: '~35KB（追踪 + 识别 + 事件）' },
      { label: '价格', pinconsole: '免费（AGPL-3.0）。商用授权可谈。', competitor: '免费版（35 次/天）。Plus 版 $32/月，Business 版 $83/月，Scale 版 $171/月' },
      { label: '部署方式', pinconsole: 'docker compose，5 分钟', competitor: '仅 SaaS 代码片段' },
    ],
  },
  content: {
    sections: [
      {
        heading: '为什么从 Hotjar 迁移到 PinConsole？',
        body: 'Hotjar 以其直观的热力图和会话回放功能深受喜爱——它通常是团队采用的第一个分析工具。但随着网站流量增长，限制变得明显：免费版每天仅录制 35 次会话，付费版价格上涨迅速（Scale 版 $171/月），而且仅限于历史分析。没有共浏览，没有实时访客监控，也没有自托管选项。PinConsole 是一个自托管替代方案，在一个二进制中覆盖了会话回放、共浏览、实时监控和反爬虫保护——零会话上限，零重复费用。',
      },
      {
        heading: '从会话上限到无限录制',
        body: 'Hotjar 的免费版对初创公司还算友好——但每天 35 次录制意味着你只看到流量的极小一部分。Plus 版（$32/月）提升到 100 次/天，Business 版（$83/月）500 次，Scale 版（$171/月）2,000 次。如果你的网站每天有 10,000 访客，根据你选择的套餐，你只能看到 0.35% 到 20% 的用户行为。而 PinConsole 没有会话上限——每一次访客交互都会被录制并可回放。你的存储限制仅由 MinIO 和 PostgreSQL 的容量决定，而不是订阅套餐。',
      },
      {
        heading: 'Hotjar 不具备的实时能力',
        body: 'Hotjar 专为历史分析设计——你今天录制会话，明天分析它们。它在这方面很出色，但它无法让你实时看到网站上正在发生的事情。PinConsole 增加了三项 Hotjar 不具备的实时能力：实时访客监控（实时查看每个访客的 DOM）、主动弹窗聊天（在访客离开前主动联系）、和双向共浏览（接管访客会话以提供帮助）。这些功能将 PinConsole 从一个分析工具转变为客户运营平台。',
      },
      {
        heading: '在中国市场的独特优势',
        body: 'Hotjar 的服务部署在欧盟和美国的云端，对中国大陆用户来说访问速度极慢。SDK 经常因为网络延迟而加载超时，管理后台打开需要数十秒。此外，数据存储在境外服务器上，对于需要遵守个保法的中国企业来说存在合规风险。PinConsole 完全自托管，SDK 与你的应用部署在同一网络中——国内团队无需翻墙即可流畅使用所有功能，所有数据境内存储。',
      },
      {
        heading: '开源，无厂商锁定',
        body: 'Hotjar 于 2021 年被 Contentsquare 收购——产品方向、定价和技术路线图现在由一家更大的企业分析公司控制。使用 PinConsole，AGPL-3.0 许可保证你永远可以自托管、审计代码、fork 项目，自主控制路线图。无突发价格变更，无功能移除，无强制迁移。商用授权可用于需要闭源集成的场景。',
      },
    ],
  },
  evidence: {
    blocks: [
      {
        quote: 'Hotjar Plus 版：$32/月——每天 100 次录制，1 年数据保留。Business 版：$83/月——每天 500 次。Scale 版：$171/月——每天 2,000 次。所有套餐包含热力图、会话回放和问卷功能。',
        attribution: 'Hotjar 定价页面（2026 年 6 月数据）',
        sourceUrl: 'https://www.hotjar.com/pricing/',
      },
      {
        quote: 'Hotjar 的追踪脚本压缩后约 35KB，包含页面浏览、会话录制、热力图和表单分析四个独立脚本组件。在中国大陆，这些脚本加载经常因跨境网络延迟而超时。',
        attribution: 'Hotjar 官方文档 & 实际使用反馈',
        sourceUrl: 'https://help.hotjar.com/hc/en-us/articles/360019339974',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'PinConsole 能替代 Hotjar 的会话回放和热力图功能吗？',
        answer: '会话回放方面——可以，PinConsole 提供基于 rrweb 的无限录制，像素级精度回放。热力图方面——尚未支持；热力图和愤怒点击分析已在 PinConsole 路线图中。如果你两者都需要，PinConsole 可以作为核心回放和监控平台，同时继续使用 Hotjar 免费版的热力图功能作为过渡。',
      },
      {
        question: 'PinConsole 支持像 Hotjar 那样的问卷和反馈小组件吗？',
        answer: 'v1 不支持——PinConsole 聚焦于会话回放、共浏览和实时监控。对于问卷和反馈功能，你可以继续使用 Hotjar 免费版或专门的工具。PinConsole 的弹窗聊天系统可以处理主动消息推送，但它专为实时互动而非异步问卷设计。',
      },
      {
        question: 'PinConsole 适合刚开始做分析的小团队吗？',
        answer: '非常适合——PinConsole 设计和部署都很简单（docker compose，5 分钟），设置成本为零：无月费、无按会话付费、无承诺。团队可以从第一天起就进行无限会话录制，随着流量增长无需担忧套餐限制。',
      },
      {
        question: 'PinConsole 对页面性能的影响与 Hotjar 相比如何？',
        answer: 'PinConsole 的 SDK 压缩后约 15KB——大约是 Hotjar 组合追踪脚本的一半大小。两者都使用异步加载以最小化页面影响。使用 PinConsole，SDK 从你自己的域名提供服务（无需额外 DNS 查询），在中国大陆的加载速度和稳定性远优于 Hotjar。',
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
