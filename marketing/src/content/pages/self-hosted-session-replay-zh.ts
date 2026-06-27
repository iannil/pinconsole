import type { SeoPageContent } from './types';

export const selfHostedSessionReplayZh: SeoPageContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: '私有化录像回放：开源替代 FullStory 和 Hotjar',
    description: '使用 PinConsole 实现私有化录像回放。开源、AGPL-3.0，部署在你自己的基础设施上。录制、存储、回放每个访客会话——数据从不出你的服务器。',
    ogTitle: '私有化录像回放 —— PinConsole 开源解决方案',
    ogDescription: '自托管录像回放平台。AGPL-3.0，5 分钟部署。基于 rrweb 录制每个访客会话，存储在自有 MinIO，随时回放。数据不出域。',
  },
  hero: {
    h1: '私有化录像回放：在你的基础设施上录制和回放',
    subtitle: '5 分钟在你的服务器上部署开源录像回放。PinConsole 使用 rrweb 录制每个访客会话，存储在自有 MinIO，通过标准 rrweb player 回放——数据从不出你的网络。',
  },
  comparison: {
    rows: [
      { label: '部署模式', pinconsole: '自托管（你的服务器）', competitor: 'SaaS 云' },
      { label: '数据存储', pinconsole: '你的 MinIO / S3 存储', competitor: '厂商的云端' },
      { label: '保留期控制', pinconsole: '可配置（默认 30 天）', competitor: '受厂商套餐限制' },
      { label: '录制引擎', pinconsole: 'rrweb（开源 DOM 采集）', competitor: '闭源' },
      { label: '选择性截图', pinconsole: '1fps WebP（canvas/WebGL/iframe）', competitor: '全页面截图' },
      { label: '隐私控制', pinconsole: 'GDPR consent + 被遗忘权 + IP 截断', competitor: '各厂商不同' },
      { label: '共浏览', pinconsole: '已包含（双向）', competitor: '不含（需独立工具）' },
      { label: '实时监控', pinconsole: '实时 DOM 流', competitor: '仅历史回放' },
      { label: '许可协议', pinconsole: 'AGPL-3.0（开源）', competitor: '闭源商业' },
      { label: '价格', pinconsole: '免费。商用授权可谈。', competitor: '每年 30-100 万+' },
    ],
  },
  content: {
    sections: [
      {
        heading: '什么是私有化录像回放？',
        body: '录像回放（Session Replay）捕捉访客在网站上的每一次交互——鼠标移动、点击、滚动、表单输入——并将其重建为视频般的播放。私有化录像回放意味着所有数据存储和处理都在你的自有基础设施上，而非第三方分析云端。这对于在 GDPR、等保、个保法下处理敏感客户数据的公司至关重要。',
      },
      {
        heading: '为什么自托管而非 SaaS（FullStory、Hotjar、LogRocket）？',
        body: 'FullStory、Hotjar、LogRocket 等 SaaS 录像回放工具按月页面浏览量或录制量收费。规模扩大后，年费可达六位数。更重要的是，它们将访客的行为数据存储在第三方服务器上——每一次点击、滚动和表单输入。对于服务金融、医疗或政务客户的 ToB SaaS 公司，这构成了不可接受的数据驻留风险。私有化录像回放同时解决了成本膨胀和数据驻留两个问题。',
      },
      {
        heading: 'PinConsole：录像回放 + 共浏览，一个自托管栈',
        body: '与仅提供历史分析的 FullStory 或 Hotjar 不同，PinConsole 将录像回放与实时共浏览结合。你可以实时观察访客，然后无缝切换为共浏览会话协助他们——无需切换工具。会话数据存储在自有 MinIO，保留期可配置（默认 30 天），支持 GDPR 删除和 IP 截断。',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'PinConsole 的录像回放与 FullStory 相比如何？',
        answer: 'PinConsole 使用 rrweb 进行 DOM 采集，通过标准 rrweb player 提供像素级回放。主要区别在部署：PinConsole 是自托管的，使用你自己的存储；FullStory 是 SaaS。PinConsole 还包括共浏览和访客监控——FullStory 没有这些功能。',
      },
      {
        question: '录像回放需要多少存储空间？',
        answer: '一个典型会话录制（rrweb 事件流 + 选择性截图）约为 100-500 KB。按 1,000 个会话/天计算，每月需要约 3-15 GB 的 MinIO 存储空间。保留期可配置。',
      },
      {
        question: '可以从 PinConsole 导出会话数据吗？',
        answer: '会话录制以标准 rrweb 事件数据格式存储在你的 MinIO 存储桶中。你可以使用任何 S3 兼容工具访问、导出或备份。元数据（会话、访客）存储在你的 PostgreSQL 数据库中。',
      },
    ],
  },
  cta: {
    title: '5 分钟部署私有化录像回放',
    subtitle: '无注册，无销售，数据不离开你的服务器。',
    primary: { label: '开始部署', href: '#top' },
    secondary: { label: '联系 maintainer', href: '#consult' },
  },
};
