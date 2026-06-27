import type { BlogContent } from './types';

export const fullstoryAlternativeZh: BlogContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: '如何构建一个自托管的 FullStory 替代品：PinConsole 架构深度解析',
    description: '从技术角度深度解析 PinConsole 的架构设计——Go 后端、rrweb DOM 采集、MinIO 事件存储、单二进制部署。一个完整的自托管会话回放和共浏览平台的构建实录。',
    ogTitle: '如何构建一个自托管的 FullStory 替代品 —— PinConsole 架构深度解析',
    ogDescription: 'Go + rrweb + MinIO：PinConsole 的开源自托管 FullStory 替代方案架构实录。',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-27',
    readingTime: '10 分钟',
    tags: ['技术架构', '自托管', '开源', 'Session Replay'],
  },
  hero: {
    h1: '如何构建一个自托管的 FullStory 替代品：PinConsole 架构深度解析',
    subtitle: '为什么要自建、技术选型背后的思考、以及如何 5 分钟部署上线。',
  },
  content: {
    sections: [
      {
        heading: 'SaaS 会话回放工具的痛点',
        body: `FullStory 是一个非常优秀的产品。它的会话回放功能打磨精良，AI 搜索令人印象深刻，热力图和愤怒点击分析也是业界标杆。但对许多团队来说，FullStory（以及 Hotjar、LogRocket、Smartlook 等类似工具）有一个根本性的限制：纯 SaaS。\n\n这意味着你的会话数据——用户的每一次点击、滚动和输入——都被传输并存放在第三方云端。对于有严格数据主权要求的组织（GDPR、等保、个保法），这是一个合规难题。企业本地部署方案是存在的，但伴随而来的是企业级定价：FullStory Business 版起步价 $500/月，用量上去后费用迅速攀升。\n\n对中国用户来说还有一层额外的痛苦：FullStory 的 SaaS 服务部署在美国，国内访问速度极慢。SDK 加载经常超时，管理后台打开需要几十秒。这不是工具的问题，是网络架构决定的。\n\n除了成本和合规，还有一个功能缺口：这些工具都不提供共浏览或实时访客监控。如果你想实时观察访客行为并主动提供帮助，你需要 Cobrowse.io 或 Upscope——又一个订阅，又一个集成。\n\n我们构建 PinConsole 就是为了在一个自托管的二进制中解决所有这些问题。`,
      },
      {
        heading: '为什么选 Go？',
        body: `选择 Go 作为后端语言，基于三个核心需求：\n\n1. 单二进制部署 —— 我们希望用户只需 Docker Compose 就能部署 PinConsole。Go 编译为静态链接的单一二进制，零运行时依赖。\n\n2. 并发能力 —— 会话回放涉及大量并发的 WebSocket 连接。Go 的 goroutine 模型优雅地处理了这个问题，无需线程池或事件循环的复杂性。\n\n3. 性能 —— 事件处理管道（接收 rrweb 事件 → 存储到 MinIO → 广播给运营端）需要快速且可预测。Go 的运行时在负载下提供一致的延迟表现。\n\n我们的基线目标是每个房间支持 500 个并发 WebSocket 连接——这足以让一个运营人员实时服务中等规模电商网站的流量。`,
      },
      {
        heading: '整体架构',
        body: `PinConsole 采用中心化的 hub-and-spoke 架构。所有流量经过中心服务器——无 P2P 连接，无 WebRTC，无第三方转发。\n\n技术栈非常直接：\n\n• PostgreSQL —— 元数据存储（站点、会话、用户、运营账号）\n• Redis —— 在线状态跟踪、限流、热缓存\n• MinIO —— rrweb 事件流存储 + 选择性截图\n• Go 服务器 —— Gin HTTP 路由 + coder/websocket hub\n\n管理前端是一个 Vue 3 SPA，通过 Go 的 //go:embed 指令嵌入。访客 SDK 是一个 TypeScript 库，从同源提供服务。一切——服务器、管理界面、SDK——都在一个二进制中。`,
        code: `docker compose up -d\n# PostgreSQL、Redis 和 MinIO 作为 sidecar 启动\n# PinConsole 二进制绑定到 :8080\n# 管理后台：https://app.yourdomain.com\n# SDK：https://app.yourdomain.com/sdk.js`,
        codeLanguage: 'bash',
      },
      {
        heading: '用 rrweb 做 DOM 采集',
        body: `会话录制方面，我们使用 rrweb——这个开源库驱动了许多生产级会话回放工具。rrweb 将 DOM 变更捕获为序列化的事件流：开始时是全量 DOM 快照，然后记录增量变更（点击、滚动、输入、尺寸变化等）。\n\n为什么选 rrweb：\n\n• 经过生产验证——被数百万会话量级的项目使用\n• 事件格式规范，可序列化（JSON）\n• 支持 iframe、Shadow DOM、Canvas/WebGL 快照（通过采集插件）\n• MIT 许可——与我们的 AGPL-3.0 栈兼容\n\n我们围绕 rrweb 构建了一些自定义基础设施。一个繁忙站点上的原始事件流可能每秒产生数百个事件。我们不将每个事件存储为独立对象，而是将事件按时间窗口批量打包，存储为 MinIO 对象。这使存储操作减少了 10-100 倍，并使回放的顺序读取速度更快。\n\n隐私方面，我们实现了选择性采集模式。运营人员可以配置 CSS 选择器来屏蔽敏感元素（密码字段、信用卡输入框、PII 容器）。被屏蔽的数据永远不会发送到服务器——在 SDK 层面就已被剥离。`,
      },
      {
        heading: '实时 WebSocket Hub',
        body: `实时层是 PinConsole 共浏览和实时监控功能的核心。我们选择 coder/websocket 而非其他库，原因很简单：它是极简的、符合 Go 风格的库，无全局状态、无隐式 goroutine 启动、API 干净。\n\nHub 模式的工作方式：\n\n• 每个站点有一个由站点 ID 标识的"房间"\n• 访客 SDK 连接到其站点的房间\n• 运营人员连接到同一房间\n• 消息按路由转发：访客 → hub → 运营（共浏览时反向同样）\n\n实时监控方面，SDK 每 100ms 发送一次 DOM 快照（已节流）加上增量事件。运营者的管理面板将其渲染为实时预览。共浏览时，运营操作（点击、滚动、表单填写）被序列化为 rrweb 变更事件并转发到访客浏览器——带有 300ms 的去抖延迟以防止冲突。\n\n一个重要的设计细节：我们不向所有运营者广播所有事件。每个访客流与一个运营者 1:1 锁定（claim/release 模式）。这避免了扇出开销，并确保清晰的所有权——不会有两个运营者争夺同一个共浏览会话的控制权。`,
      },
      {
        heading: '用 MinIO 存储事件',
        body: `会话回放数据的特征是写入密集且只追加。一个 10 分钟的会话可能产生 50,000+ 个 rrweb 事件（约 2-5MB 序列化 JSON）。把这些存在 PostgreSQL 中不是不行，但并非最优——对象存储更便宜、顺序读取更快、且不会与 OLTP 业务争抢资源。\n\nMinIO（兼容 S3 的对象存储）是自然的选择：\n\n• 每个会话的事件存储为 "session" 前缀下的一系列对象\n• 第一个对象是全量 DOM 快照（snapshot.json）\n• 后续对象是 5 秒时间窗口的事件批次（events-{timestamp}.json）\n• 截图模式：检测到 Canvas/WebGL/跨域 iframe 时，以 1fps WebP（质量 70）存储截图\n\n回放时，服务器按顺序读取事件对象并流式传输到管理界面。间隙检测优雅地处理缺失的批次——如果某个批次不可用，回放会显示"缓冲中"指示并跳过。\n\n我们还实现了保留策略系统：运营者可以为每个站点设置 TTL（7/30/90 天或永久）。MinIO 的生命周期策略负责实际删除——我们只需设置对象标签。`,
      },
      {
        heading: 'SDK：小巧、模块化、自托管',
        body: `访客 SDK 使用 TypeScript + Vite 构建。它作为一个 /sdk.js 文件从 PinConsole 管理后台的同源提供服务。无 CDN，无第三方脚本加载器——只要你的服务器可达，SDK 就能工作。\n\nSDK 压缩后约 15KB（相比之下 FullStory 的 SDK 约 50KB+）。我们通过以下方式实现这个体积：\n\n• 仅打包 rrweb 核心录制器（不带回放器——那是服务器端的任务）\n• 使用轻量的 WebSocket 客户端而非 HTTP 长轮询\n• 功能通过 SDK 配置对象选择启用\n\nSDK 生命周期：\n\n1. 页面加载 → SDK 初始化并建立 WebSocket 连接\n2. rrweb 开始录制 DOM 变更，附带选择性屏蔽\n3. 事件在本地缓冲，每 200ms 刷新一次\n4. 如果连接断开，事件被排队并在重连后重放\n5. 访客离开时，发送 "session-end" 标记\n\n我们刻意避免任何形式的用户标识（除了反爬虫所需的最小指纹外）。会话由服务器生成的 UUID 标识，不在站点间追踪访客。`,
      },
      {
        heading: '安全与反爬虫设计',
        body: `由于 PinConsole 自托管且面向生产流量，我们从第一天起就构建了纵深防御。\n\n反爬虫系统在四个层面运作：\n\n1. 速率限制 —— 每个 IP 和每个会话的事件速率上限（可配置）\n2. User-Agent 黑名单 —— 已知爬虫 UA 在 HTTP 层面被拒绝\n3. 行为分析 —— 服务器监控事件模式。每秒产生 1000 次点击的会话很可能是爬虫而非真人\n4. TLS 指纹 —— 对进入的 WebSocket 连接进行被动 JA3 指纹识别\n\n管理后台方面，所有认证路由都需要 HttpOnly 会话 cookie。运营人员的 WebSocket 连接使用同一 cookie 进行身份验证——URL 中无 token，查询字符串中无 token。\n\n我们还实现了严格默认的内容安全策略（CSP）：仅同源可加载脚本，仅允许 challenges.cloudflare.com 用于 Turnstile（如启用），内联样式使用哈希值验证。`,
      },
      {
        heading: '与 FullStory 的功能对标：已有 vs 路线图',
        body: `我们诚实地说明现在的定位：\n\n✓ 会话回放（基于 rrweb，像素级精度）\n✓ 实时访客监控（DOM + 截图）\n✓ 双向共浏览（点击、滚动、表单代填、页面导航）\n✓ 主动弹窗聊天\n✓ 反爬虫保护\n✓ 自托管（你的服务器，你的数据）\n✓ AGPL-3.0 免费，商用授权可谈\n\n△ 热力图和愤怒点击 —— 路线图中（预计 2026 Q3）\n△ AI 驱动的会话搜索 —— 路线图中\n△ 漏斗和转化分析 —— 路线图中\n\n✗ 原生移动端 SDK —— v1 范围之外，PinConsole v1 仅面向 Web\n✗ 站点自定义域名 —— post-v1 待办\n\n如果你今天就需要 FullStory 的高级分析层，PinConsole 可以作为核心回放和监控平台，同时继续使用 FullStory 的热力图功能。但对于重视数据主权、成本和开源精神的团队——PinConsole 已经覆盖了关键用例。`,
      },
      {
        heading: '生产环境部署',
        body: `部署 PinConsole 有意保持简单：\n\n1. 克隆仓库\n2. 配置 .env 文件\n3. docker compose up -d\n4. 将 SDK 代码片段添加到你的网站\n\n生产环境建议：\n\n• PostgreSQL 15+，配连接池（100+ 并发站点推荐 PgBouncer）\n• Redis 7+，用于在线状态和限流\n• MinIO 分布式模式，用于高可用对象存储\n• 反向代理（Nginx/Caddy）用于 TLS 终止\n• 定期备份 PostgreSQL 和 MinIO 存储桶\n\n单二进制模型意味着升级是原子操作：拉取新镜像，重启，完成。数据库迁移在启动时自动运行。无包管理器，无运行时升级，无依赖冲突。`,
        code: `git clone https://github.com/iannil/pinconsole\ncd pinconsole\ncp .env.example .env\n# 编辑 .env，填入你的密钥\ndocker compose up -d`,
        codeLanguage: 'bash',
      },
    ],
  },
  cta: {
    title: '立即试用 PinConsole',
    subtitle: '5 分钟自托管部署。AGPL-3.0。你的数据，你的服务器。',
    primary: { label: 'GitHub 上开始', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: '联系 maintainer', href: '#consult' },
  },
};
