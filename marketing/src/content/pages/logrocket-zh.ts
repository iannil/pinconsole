import type { SeoPageContent } from './types';

export const logrocketZh: SeoPageContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: 'PinConsole vs LogRocket：开源、自托管的 Session Replay 替代方案',
    description: '对比 PinConsole（开源、自托管、AGPL-3.0）与 LogRocket。两者都提供会话回放和前端监控，只有 PinConsole 给你自托管部署、共浏览、实时访客监控，以及无需按会话付费的定价模式。',
    ogTitle: 'PinConsole vs LogRocket —— 开源会话回放替代方案',
    ogDescription: '自托管 LogRocket 替代品。AGPL-3.0，你的数据在你的服务器上。会话回放、共浏览、实时访客监控、反爬虫——零会话上限。',
  },
  hero: {
    h1: 'PinConsole vs LogRocket：开源替代方案',
    subtitle: 'LogRocket 将会话回放与前端监控（控制台日志、网络请求、JS 错误）结合——对开发者来说是一个强大的组合。但它是纯 SaaS 的，价格随会话量增长，且不提供自托管选项。PinConsole 提供会话回放、共浏览、实时监控和反爬虫保护，全部在一个自托管二进制中，AGPL-3.0 免费使用，零会话上限。',
  },
  comparison: {
    rows: [
      { label: '部署模式', pinconsole: '自托管（你的基础设施）', competitor: '纯 SaaS 云' },
      { label: '许可协议', pinconsole: 'AGPL-3.0（开源）', competitor: '商业闭源' },
      { label: '数据主权', pinconsole: '数据从不出你的服务器', competitor: '数据存储在 LogRocket 云端' },
      { label: '会话回放', pinconsole: '基于 rrweb，无限会话', competitor: '免费 1k 次/月，5k 次($99)，15k 次($249)' },
      { label: '前端监控', pinconsole: '计划中（路线图）', competitor: '控制台日志、网络、JS 错误、性能指标' },
      { label: '共浏览', pinconsole: '已包含（双向点击/滚动/代填/跳转）', competitor: '不包含' },
      { label: '访客监控', pinconsole: '实时 DOM + 选择性截图', competitor: '仅历史回放' },
      { label: '弹窗 & 主动聊天', pinconsole: '推送通知 + 双向聊天', competitor: '不包含' },
      { label: '反爬虫保护', pinconsole: '限流 + UA + 行为 + fingerprint', competitor: '不包含' },
      { label: '价格', pinconsole: '免费（AGPL-3.0）。商用授权可谈。', competitor: '免费（1k 次/月）。$99/月（5k 次），$249/月（15k 次）' },
      { label: '部署方式', pinconsole: 'docker compose，5 分钟', competitor: '仅 SaaS 代码片段' },
    ],
  },
  content: {
    sections: [
      {
        heading: '为什么从 LogRocket 迁移到 PinConsole？',
        body: 'LogRocket 是一款对开发者友好的会话回放工具，具有独特的前端监控能力——控制台日志、网络请求和 JS 错误都与会话回放同步记录。这对调试确实非常有用。但 LogRocket 纯 SaaS 模式意味着你的会话数据存储在云端，价格按会话量计算：每月免费 1,000 次，然后 $99 获得 5,000 次，$249 获得 15,000 次。高流量团队的月度费用可能相当可观。PinConsole 提供无限会话回放，无按会话付费，加上 LogRocket 没有的共浏览和实时访客监控——全部在一个自托管二进制中。',
      },
      {
        heading: '从会话上限到无限录制',
        body: 'LogRocket 的免费版（每月 1,000 次会话）对小项目还算合理，但有一定流量的生产站点很快会超出限制。$249/月购买 15,000 次会话——如果网站每月有 50,000 访客，你需要多个套餐或承受高额成本。而 PinConsole 没有会话上限——每一次访客交互都会被录制。无论你每月有 1,000 还是 100,000 次会话，成本是一样的：零许可费用。唯一成本是你的基础设施（小站点 VPS 每月仅需几美元）。',
      },
      {
        heading: 'LogRocket 缺乏实时互动能力',
        body: 'LogRocket 擅长记录"发生了什么"——控制台错误、慢网络请求、愤怒点击。但它本质上是一个历史工具。你可以回放会话来调试问题，但无法实时看到网站正在发生什么，也无法与访客互动。PinConsole 用三项实时能力填补了这个空白：实时访客监控（实时观看每个访客的会话）、主动弹窗聊天（在访客离开前联系他们）、和双向共浏览（接管会话以引导用户）。这些功能将会话回放从调试工具转变为客户运营平台。',
      },
      {
        heading: '在中国市场的独特优势',
        body: 'LogRocket 的服务部署在美国云端，对中国大陆用户来说访问延迟较高。SDK 加载和会话数据上传经常受到跨国网络连接的影响。PinConsole 完全自托管部署，SDK 与你的应用部署在同一网络中——国内团队无需翻墙即可流畅使用所有功能。所有数据境内存储，直接满足个保法的数据本地化要求。',
      },
      {
        heading: '开源，开发者友好，免费',
        body: 'LogRocket 是闭源的——你无法审计录制逻辑、扩展平台或控制路线图。PinConsole 是 AGPL-3.0 开源项目：完整源代码在 GitHub 上，你可以审计每一行代码、fork 项目、无限期自托管。对于重视代码透明度和自主权的开发者团队，PinConsole 提供与 LogRocket 同等的会话回放能力，加上实时互动功能——无需按会话付费或厂商锁定。',
      },
    ],
  },
  evidence: {
    blocks: [
      {
        quote: 'LogRocket 免费版：每月 1,000 次会话，3 天数据保留。专业版：$99/月——5,000 次会话，7 天保留。企业版：$249/月——15,000 次会话，30 天保留。所有套餐包含会话回放、控制台日志和网络监控。',
        attribution: 'LogRocket 定价页面（2026 年 6 月数据）',
        sourceUrl: 'https://logrocket.com/pricing/',
      },
      {
        quote: 'LogRocket 会捕获 console.log、console.warn、console.error、未处理的 JS 异常、fetch/XHR 网络请求和响应、以及性能指标——所有数据与会话回放时间线同步。',
        attribution: 'LogRocket 官方文档',
        sourceUrl: 'https://docs.logrocket.com/',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'PinConsole 能替代 LogRocket 的前端调试功能吗？',
        answer: '会话回放方面——可以，PinConsole 提供基于 rrweb 的无限录制，与 LogRocket 同级别的 DOM 采集能力。前端监控方面（控制台日志、网络请求、JS 错误）——尚未支持，这些功能在路线图中。过渡期间可以使用 PinConsole 进行无限会话回放，同时保留 LogRocket 免费版进行前端监控。',
      },
      {
        question: 'PinConsole 像 LogRocket 一样捕获控制台日志和网络请求吗？',
        answer: 'v1 不支持——PinConsole v1 聚焦于通过 rrweb 进行 DOM 级别的会话采集。控制台日志和网络请求监控在路线图中。不过，PinConsole 提供了 LogRocket 不具备的功能：共浏览、实时访客监控和主动聊天。根据你的主要用例，PinConsole 可能已经覆盖了你的核心需求。',
      },
      {
        question: 'PinConsole 的会话存储成本与 LogRocket 相比如何？',
        answer: 'PinConsole 使用 MinIO（兼容 S3 的对象存储）存储事件流，成本远低于 LogRocket 的按会话定价。一次典型的会话在 S3 兼容存储中的成本不到一分钱。对于高流量站点，自托管 PinConsole 的基础设施成本通常远低于 LogRocket 的订阅费用。',
      },
      {
        question: 'PinConsole 适合需要调试能力的开发者团队吗？',
        answer: '适合——PinConsole 的会话回放基于 rrweb，与许多生产调试工具使用相同的底层技术。开发者可以回放会话、检查任意时间点的 DOM 状态、使用时间线理解用户行为。对于既需要会话回放又需要前端监控的团队，PinConsole 覆盖回放侧，同时可以用浏览器的 DevTools 或轻量监控工具处理控制台日志。',
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
