import type { BlogContent } from './types';

export const agplVsMitZh: BlogContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: 'AGPL-3.0 vs MIT：为什么我们的开源项目选择 AGPL 许可协议',
    description: 'PinConsole 选择 AGPL-3.0 而非 MIT 或 Apache 2.0 的真实原因——战略考量、取舍、以及商用授权如何运作。',
    ogTitle: 'AGPL-3.0 vs MIT —— 为什么 PinConsole 选择 AGPL',
    ogDescription: '对于一个定位于"开源 SaaS 替代品"的项目，AGPL-3.0 是正确选择。原因如下。',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-27',
    readingTime: '7 分钟',
    tags: ['开源', '许可协议', 'AGPL', '商业'],
  },
  hero: {
    h1: 'AGPL-3.0 vs MIT：为什么我们的开源项目选择 AGPL 许可协议',
    subtitle: '关于 PinConsole 许可协议选择的透明说明——以及这对用户、贡献者和商业采用者分别意味着什么。',
  },
  content: {
    sections: [
      {
        heading: '简短的回答',
        body: '我们选择 AGPL-3.0 是因为 PinConsole 的定位是 FullStory、Hotjar 和 LogRocket 等商业 SaaS 工具的开源替代品。如果我们使用 MIT 或 Apache 2.0，云服务商可以拿去直接封装成 SaaS 服务，与开源项目竞争——且无需回馈任何代码。AGPL-3.0 通过一个关键条款阻止了这种情况：任何修改并以网络形式提供服务的人，都必须以相同协议发布其修改。\n\n简而言之：AGPL 保护项目不被云服务商商品化，同时对自托管用户保持完全开源和免费。',
      },
      {
        heading: '为什么不用 MIT？',
        body: 'MIT 是最宽松的许可协议——几乎可以用 MIT 代码做任何事情，包括将其用于闭源商业产品。这对库和框架来说是好事（React 是 MIT，Vue 是 MIT），但对于以服务形式销售的服务器端应用来说，这可能很危险。\n\n看看 MongoDB 和 Elasticsearch 的经历。它们最初都是 Apache 2.0。云服务商（AWS、Azure）将它们打包为托管服务，与构建它们的公司直接竞争——既不分担收入也不共享代码改进。最终这两个项目都转向了源代码可用许可（SSPL、Elastic License）。\n\n我们希望从一开始就避免这种局面。PinConsole 是一个完整的应用，不是一个库。如果我们使用 MIT，某云服务商明天就可以推出"PinConsole as a Service"，削弱我们提供商用授权和资助持续开发的能力。\n\nAGPL-3.0 堵住了这个漏洞：如果你修改了 PinConsole 并以网络服务形式提供，你必须分发你的修改。这不影响自托管使用——它防止的是 SaaS 商品化。',
      },
      {
        heading: '为什么不用 Apache 2.0？',
        body: 'Apache 2.0 在宽松程度上与 MIT 类似，外加一个专利授权条款。对于 PinConsole 来说，专利条款没有必要（我们没有需要保护或授权的专利组合）。而且和 MIT 一样，Apache 2.0 没有解决 SaaS 漏洞——云服务商可以提供服务而不回馈代码。\n\n我们曾短暂考虑过 Apache 2.0，但结论是 AGPL-3.0 的网络使用条款对我们的商业模式至关重要：软件对自托管用户完全免费，商用授权覆盖闭源嵌入或 SaaS 分发场景。',
      },
      {
        heading: 'AGPL-3.0 对你意味着什么',
        body: '如果你是自托管用户，AGPL-3.0 不会影响你的日常使用：\n\n• 你可以在自己的基础设施上免费、永久部署 PinConsole\n• 你可以为内部使用修改代码，无需分享你的修改\n• 你可以审计每一行代码的安全性和合规性\n• 你可以 fork 项目并维护自己的分支\n\n唯一的限制：如果你修改了 PinConsole 并向第三方提供 SaaS 服务，你必须以 AGPL-3.0 发布你的修改。如果你只是内部使用来分析自己网站的流量，你完全自由。\n\n对于需要将 PinConsole 嵌入到闭源产品中的团队，我们提供标准的商用授权。请联系 maintainer 了解详情。',
      },
      {
        heading: '贡献者怎么办？',
        body: '我们要求所有贡献者签署贡献者许可协议（CLA），授予 PinConsole 永久、免版税的权利，以在 AGPL-3.0 和商业条款下使用其贡献。这是 AGPL 项目的标准做法（MySQL/MariaDB、SugarCRM 等），确保：\n\n1. 项目可以为商业授权用户重新许可贡献的代码\n2. 贡献者保留其代码的完全所有权\n3. 项目长期可持续\n\n如果你提交的是 bug 修复或小的改进，CLA 只需一次签署。对于较大的功能贡献，我们会与你协商确保许可清晰。',
      },
      {
        heading: 'AGPL vs SSPL vs BSL——为什么我们不更进一步',
        body: '我们评估了更严格的源代码可用许可，如 SSPL（MongoDB）和 BSL（Business Source License，MariaDB 和 CockroachDB 使用）。这些许可比 AGPL 更进一步，明确限制云服务商将软件作为服务提供。\n\n我们选择 AGPL-3.0 的理由：\n\n• 它是 OSI 批准的知名开源许可——不存在"是否开源"的争议\n• 网络使用条款对我们的需求已经足够——我们不担心有人分发修改版，我们担心的是 SaaS 商品化\n• 与更广泛的开源生态系统兼容——AGPL 代码可以与 Apache 2.0 和 MIT 代码组合\n• 企业法务团队熟悉——与 SSPL 不同，AGPL 有成熟的判例\n\n如果将来遇到具体的 SaaS 滥用行为，我们可以随时添加 Commons Clause 或切换到不同的许可，但 AGPL-3.0 是正确的起点。',
      },
      {
        heading: '结论',
        body: 'AGPL-3.0 将我们的激励与用户的激励对齐。自托管用户获得免费、开放、可审计的平台。商业用户获得闭源使用的授权路径。项目受到保护，不会陷入"开源核心但 AWS 拿去卖"的困境——这个困境已经伤害了太多开源公司。\n\n我们相信，对于一个定位于"开源替代品"的项目来说，这是最可持续的模式——希望你也认同。',
      },
    ],
  },
  relatedPosts: [
    { title: '可持续开源：PinConsole 的 AGPL-3.0 与商业授权实践', url: '/blog/open-source-business-model-zh/', description: 'AGPL-3.0 与商业授权如何协同为开源开发提供资金。' },
    { title: '如何构建一个自托管的 FullStory 替代品', url: '/blog/self-hosted-fullstory-alternative/', description: 'PinConsole 的完整架构——Go、rrweb、MinIO 和单二进制部署。' },
  ],
  cta: {
    title: '自托管 PinConsole——AGPL-3.0 免费',
    subtitle: '无需注册。零会话上限。你的数据，你的服务器。',
    primary: { label: 'GitHub 上开始', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: '了解商用授权', href: '#consult' },
  },
};
