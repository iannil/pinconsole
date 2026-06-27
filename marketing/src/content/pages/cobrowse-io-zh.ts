import type { SeoPageContent } from './types';

export const cobrowseIoZh: SeoPageContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: 'PinConsole vs Cobrowse.io：开源、自托管的共浏览替代方案',
    description: '对比 PinConsole（开源、自托管、AGPL-3.0）与 Cobrowse.io。两者都提供共浏览，只有 PinConsole 给你完整的数据主权、录像回放、和无按坐席收费。',
    ogTitle: 'PinConsole vs Cobrowse.io —— 开源共浏览替代方案',
    ogDescription: '自托管 Cobrowse.io 替代品。AGPL-3.0，你的数据在你的服务器上。共浏览、录像回放、访客监控、反爬虫——全功能。',
  },
  hero: {
    h1: 'PinConsole vs Cobrowse.io：开源替代方案',
    subtitle: 'Cobrowse.io 是领先的共浏览 SDK，但它是闭源且仅 SaaS 的。PinConsole 提供同样的共浏览能力——加上录像回放和访客监控——全部自托管，AGPL-3.0。',
  },
  comparison: {
    rows: [
      { label: '部署模式', pinconsole: '自托管（你的基础设施）', competitor: 'SaaS 云 + 企业本地部署' },
      { label: '许可协议', pinconsole: 'AGPL-3.0（开源）', competitor: '商业闭源 SDK' },
      { label: '数据主权', pinconsole: '数据从不出你的服务器', competitor: '数据经云端转发（企业版可本地）' },
      { label: '共浏览', pinconsole: '完整双向（点击/滚动/代填/跳转）', competitor: '完整双向共浏览' },
      { label: '录像回放', pinconsole: '已包含（基于 rrweb）', competitor: '不包含（需额外工具）' },
      { label: '访客监控', pinconsole: '实时 DOM + 选择性截图', competitor: '不包含' },
      { label: '弹窗 & 聊天', pinconsole: '弹窗推送 + 双向聊天', competitor: '不包含' },
      { label: '反爬虫', pinconsole: '限流 + UA + 行为 + fingerprint', competitor: '不包含' },
      { label: '价格', pinconsole: '免费（AGPL-3.0）。商用授权可谈。', competitor: '按坐席月付或企业年付' },
      { label: '部署方式', pinconsole: 'docker compose，5 分钟', competitor: 'SaaS SDK 代码片段或本地部署' },
    ],
  },
  content: {
    sections: [
      {
        heading: '为什么从 Cobrowse.io 迁移到 PinConsole？',
        body: 'Cobrowse.io 是打磨精良的共浏览 SDK，但其 SaaS 模式意味着你的客户屏幕数据经过他们的云转发。如果你的合规框架（GDPR、等保、个保法）要求数据在境内或本地，Cobrowse.io 的企业本地部署方案可以——但价格不菲。PinConsole 免费且开源，提供同等共浏览能力，加上录像回放和访客监控，无需额外付费。',
      },
      {
        heading: '不止是共浏览——完整的客户运营平台',
        body: 'Cobrowse.io 只聚焦共浏览。PinConsole 是一个完整的客户运营平台：实时访客监控（看每个访客在你的网站上做什么）、录像回放（录制和回放任何会话）、弹窗聊天（主动推送消息）、和反爬虫保护。全部打包在一个自托管的单二进制中。',
      },
      {
        heading: '开源意味着无厂商锁定',
        body: '使用 Cobrowse.io，你被绑定在它的定价、它的路线图和它的合规认证上。使用 PinConsole，AGPL-3.0 许可保证你永远可以自托管、审计代码，需要时 fork。商用授权可用于闭源嵌入场景。',
      },
    ],
  },
  faq: {
    items: [
      {
        question: 'PinConsole 的集成方式与 Cobrowse.io 相同吗？',
        answer: 'PinConsole 使用访客 SDK 代码片段（类似 Cobrowse.io 的 snippet），添加到你的网站即可。v1 支持所有前端框架，暂不支持原生移动端 SDK。',
      },
      {
        question: '可以同时使用 PinConsole 和 Cobrowse.io 吗？',
        answer: '技术上讲可以，但没有必要——PinConsole 在一个自托管栈中覆盖了共浏览、录像回放、监控和聊天。同时运行两者只会增加复杂度而无额外收益。',
      },
      {
        question: 'PinConsole 的共浏览可靠性与 Cobrowse.io 相比如何？',
        answer: 'PinConsole 使用 rrweb 进行 DOM 采集，与许多生产级共浏览工具使用相同的底层技术。双向共浏览（光标同步、点击转发、表单代填、页面导航）经过 65+ 个端到端测试验证。',
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
