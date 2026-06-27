import type { SeoPageContent } from './types';

export const selfHostedCoBrowsingZh: SeoPageContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: '私有化共浏览：开源、自托管的屏幕共享解决方案',
    description: '使用 PinConsole 实现私有化共浏览。开源、AGPL-3.0，部署在你自己的基础设施上。双向共浏览、录像回放、访客监控——数据从不出你的服务器。',
    ogTitle: '私有化共浏览 —— PinConsole 开源解决方案',
    ogDescription: '开源、自托管共浏览平台。AGPL-3.0，5 分钟部署。完整双向共浏览、录像回放、访客监控。你的数据，你的服务器。',
  },
  hero: {
    h1: '私有化共浏览：在你的基础设施上安全共享屏幕',
    subtitle: '5 分钟在你的服务器上部署开源共浏览。PinConsole 提供双向共浏览、录像回放和实时访客监控——全部基于 AGPL-3.0，数据从不出你的网络。',
  },
  comparison: {
    rows: [
      { label: '部署方式', pinconsole: '你的自有服务器（Docker compose）', competitor: 'SaaS 云或企业本地部署' },
      { label: '数据驻留', pinconsole: '100% 在你的基础设施上', competitor: '第三方服务器（除非企业本地版）' },
      { label: '共浏览', pinconsole: '双向：点击、滚动、代填、跳转', competitor: '各厂商不同' },
      { label: '录像回放', pinconsole: '已包含（基于 rrweb）', competitor: '通常是独立工具/附加功能' },
      { label: '访客监控', pinconsole: '实时 DOM + 选择性截图', competitor: '通常不包含' },
      { label: '许可协议', pinconsole: 'AGPL-3.0（开源）', competitor: '闭源商业' },
      { label: '价格', pinconsole: '免费。商用授权可谈。', competitor: '每年 30-100 万+' },
      { label: '部署时间', pinconsole: '5 分钟（docker compose）', competitor: 'SDK 集成 + 厂商入驻' },
    ],
  },
  content: {
    sections: [
      {
        heading: '什么是私有化共浏览？',
        body: '私有化共浏览（co-browsing）让客服或运营人员实时查看并与访客的浏览器交互——高亮元素、点击按钮、代填表单、跳转页面——全部运行在公司的自有基础设施上，而非第三方云端。这对于满足 GDPR、等保、个保法等要求客户数据在境内或本地处理存储的合规场景至关重要。',
      },
      {
        heading: '为什么企业选择自托管而非 SaaS 共浏览？',
        body: 'SaaS 共浏览工具（Upscope、Cobrowse.io、Surfly）使用方便，但要求客户屏幕数据经过第三方服务器。对于金融、保险、医疗、政务行业，这是硬卡点。私有化共浏览通过完全运行在你的内网上解决这个问题。PinConsole 让私有化变得切实可行——一条 docker compose 命令即可，不需要 K8s、不需要云服务、没有厂商依赖。',
      },
      {
        heading: 'PinConsole：功能完备的开源共浏览',
        body: 'PinConsole 不是功能受限的"开源核心"版本。AGPL-3.0 版本包含一切：双向共浏览（光标同步、点击转发、表单代填、页面导航）、录像回放（录制和回放任何会话）、实时访客监控（观察每个访客的 DOM 变更）、弹窗聊天（主动推送）、和反爬虫保护（限流、UA 屏蔽、行为分析、fingerprint）。全部打包在一个自托管的单二进制中。',
      },
    ],
  },
  faq: {
    items: [
      {
        question: '自托管 PinConsole 需要什么基础设施？',
        answer: '需要 Docker（或带 Go 运行时的 Linux 服务器）、PostgreSQL 16+、Redis 7+、和 MinIO（或 S3 兼容存储）。仓库中的 docker-compose.yml 文件自动设置所有依赖。',
      },
      {
        question: '私有化共浏览能在防火墙后工作吗？',
        answer: '可以。PinConsole 专为内网和离线环境设计。访客 SDK 通过 WebSocket 连接你的服务器——只要你的网站可访问，共浏览就能工作。无需外部服务调用。',
      },
      {
        question: '可以定制共浏览体验吗？',
        answer: 'PinConsole 是开源（AGPL-3.0）的，你可以定制任何东西。代码库是干净的 Go + TypeScript monorepo。定制开发服务可通过咨询联系。',
      },
    ],
  },
  cta: {
    title: '5 分钟部署私有化共浏览',
    subtitle: '无注册，无销售，数据不离开你的服务器。',
    primary: { label: '开始部署', href: '#top' },
    secondary: { label: '联系 maintainer', href: '#consult' },
  },
};
