import type { BlogContent } from './types';

export const privacyZh: BlogContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: '开源会话回放的隐私设计：构建 GDPR 优先的访客监控系统',
    description: 'PinConsole 如何实现隐私优先的访客监控——选择加入同意、选择性脱敏、被遗忘权、数据保留。构建符合个保法/GDPR 的访客监控技术指南。',
    ogTitle: '开源会话回放的隐私设计：构建 GDPR 优先的访客监控',
    ogDescription: '选择加入同意、选择性脱敏、被遗忘权、30 天数据保留——PinConsole 隐私优先的访客监控系统实现。',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-28',
    readingTime: '10 分钟',
    tags: ['隐私', 'GDPR', '安全', '技术架构'],
  },
  hero: {
    h1: '开源会话回放的隐私设计：构建 GDPR 优先的访客监控系统',
    subtitle: '如何在构建访客监控时默认尊重隐私——同意、脱敏、擦除、保留，从第一天开始实现。',
  },
  content: {
    sections: [
      {
        heading: '会话回放的隐私悖论',
        body: `会话回放工具本质上是监控工具。它们记录访客的每一次点击、滚动、输入和导航。在产品团队手中，这些数据对 UX 优化、Bug 复现和转化分析来说价值连城。但在错误的环境下，这是一个隐私噩梦。\n\n每个主流会话回放工具都面对这种张力。FullStory、Hotjar、LogRocket——它们都默认采集，然后让你选择退出。数据离开你的基础设施，存储在跨司法管辖区的第三方服务器上。对于欧盟企业，GDPR 第 44 条（国际数据传输）立刻成为合规风险。\n\n我们以不同的方式构建了 PinConsole。因为是自托管的，我们的出发点是：数据永远不会离开你的基础设施。但这只是入门条件。真正的设计挑战是将隐私内建于监控管道本身——不是事后补充，不是合规复选框，而是一级的系统属性。\n\n以下是我们实现的方式。`,
      },
      {
        heading: '默认脱敏：SDK 层级的隐私保护',
        body: `第一道防线不是数据库权限或 API 守卫——而是访客 SDK。在任何数据离开访客浏览器之前，我们就决定了什么不应该被采集。\n\n默认情况下，每个输入字段都被脱敏。当 UNMASK_INPUTS 为 false（默认值）时，rrweb 将 <input> 和 <textarea> 的值记录为星号（***）。原始文本永远不会到达服务器——它在传输前就在 SDK 层面被剥离了。\n\n即使 UNMASK_INPUTS 设为 true，密码字段仍然无条件脱敏。这是硬编码的——没有配置可以移除它。这是安全底线，不是可选项。\n\n除了输入字段，我们还支持两种基于 CSS 的脱敏机制：\n\n• blockClass（'mm-block'）：带有此 class 的元素被完全排除在录制之外。没有快照、没有变化追踪、没有任何痕迹。用于永远不应该出现在任何录制中的元素——信用卡 iframe、医疗数据展示、内部管理面板。\n\n• ignoreClass（'mm-ignore'）：带有此 class 的元素被记录为占位符。它们的位置、大小和布局被保留以支持准确的回放，但内容被剥离。用于需要视觉上下文但不需要实际数据的动态内容面板。\n\n两种机制都在数据离开浏览器之前生效。没有可能意外暴露脱敏内容的服务端处理。SDK 就是隐私边界。`,
        code: `// SDK 配置：隐私优先的默认值
const config = {
  unmaskInputs: false,        // 默认：脱敏所有输入
  maskInputOptions: {
    password: true,            // 始终脱敏，不可配置
    email: true,               // 默认脱敏
    tel: true,                 // 默认脱敏
  },
  blockClass: 'mm-block',     // 完全排除在录制之外
  ignoreClass: 'mm-ignore',   // 显示为占位符，内容剥离
};`,
        codeLanguage: 'typescript',
      },
      {
        heading: '四种同意模式，一个框架',
        body: `不同的站点有不同的同意要求。一个由已登录员工使用的 SaaS 后台，与一个面向欧盟访客的公开电商网站，有着截然不同的隐私期望。我们没有采用单一的同意模型，而是构建了四种模式：\n\n• opt-in（默认）：在访客显式接受之前不采集任何数据。横幅在页面加载时立即显示。这是 GDPR 合规最安全的默认值。\n\n• opt-out：立即采集数据，但访客可以拒绝。如果访客拒绝，采集停止。适用于会话回放对运营至关重要但法律仍要求同意的场景。\n\n• always-on：不显示任何横幅直接开始采集。适用于内部工具或已认证的会话，此时同意隐含在雇佣合同或服务条款中。\n\n• always-off：永远不采集。适用于任何情况下都不应该被录制的页面或访客。\n\n同意状态在服务端持久化。当 SDK 初始化时，它调用 GET /api/privacy/consent?fingerprint=<hash> 检查现有的同意记录。如果没有记录且模式为 opt-in，则显示同意横幅。\n\n横幅本身是带有背景模糊的居中模态框——视觉上无可忽略。它包含：\n\n• 采集了什么数据的简短说明\n• 接受和拒绝按钮\n• 一个可选的隐私政策链接\n\n横幅文本可以通过管理后台自定义。你可以为每个站点设置不同的消息，使用任何语言。对于欧盟部署，我们建议按照 GDPR 第 13 条的要求说明数据处理目的（"我们记录您的会话以改进我们的网站"）。`,
      },
      {
        heading: '没有同意，SDK 不会启动',
        body: `这听起来显而易见，但大多数会话回放工具并非如此运作。许多 SDK 立即初始化录制器，只在传输前检查同意状态——这意味着 DOM 快照已经在内存中被捕获了。\n\nPinConsole 的 SDK 采取了更严格的方法。shouldCollectSurveillance() 函数是守门人：\n\n1. 检查同意模式\n2. 如果是 opt-in：仅在存在已接受的同意记录时才继续\n3. 如果是 opt-out：除非存在已拒绝的同意记录，否则继续\n4. 如果是 always-on：无条件继续\n5. 如果是 always-off：永远不继续\n\n当检查失败时，不会创建 rrweb 录制器。不会获取 DOM 快照。不会建立 WebSocket 连接。没有数据需要"被遗忘"，因为从未有过数据。\n\n这也有实际性能上的好处——未同意的访客不会下载 ~15KB 的 SDK，也不会产生 WebSocket 连接开销。隐私和性能在此达成一致。\n\n当访客撤回同意（通过 setConsent(false)）时，rrweb 录制器立即停止。任何尚未传输的缓冲事件被丢弃。WebSocket 连接被关闭。访客的隐私选择在同一个事件循环 tick 内被尊重。`,
      },
      {
        heading: '被遗忘权：跨所有存储层的级联删除',
        body: `GDPR 第 17 条赋予访客要求擦除其数据的权利。在自托管的会话回放工具中，这意味着要从三个存储系统中删除数据：PostgreSQL、MinIO 和 Redis。\n\n当运营人员提交删除请求（通过 /privacy 管理页面）时，服务器执行以下级联操作：\n\n1. 通过指纹查找访客，收集所有关联的会话 ID\n2. 从 event_blobs 表中列出所有 MinIO 对象键（在删除行之前——键是引用依据）\n3. 按依赖顺序删除 PostgreSQL 行：visitor_consents → chat_messages → co_browsing_commands → event_blobs → sessions → visitors\n4. 删除每个事件键对应的 MinIO 对象（尽力而为，不阻塞响应）\n5. 删除每个会话的 Redis 认领键（尽力而为）\n\n一个值得注意的设计选择：每个步骤独立提交。没有全局事务。理由是"偏向删除"——如果数据库删除成功但 MinIO 对象删除失败，访客的元数据仍然被删除了。孤立的事件对象无害，会被 GC worker 清理。\n\n如果访客已经被删除（重复请求），端点返回 200 OK 并附带说明——而不是 404。这防止了特定指纹是否存在的信息泄露。\n\n执行删除的运营人员需要 admin 角色——这是在代码审计发现初始实现允许任何运营人员触发擦除后添加的安全检查。`,
        code: `// 级联删除顺序（每个步骤独立提交）
func (r *ErasureRepo) DeleteVisitorByFingerprint(ctx context.Context, fp string) error {
  // 1. 访客同意记录
  r.db.Exec(ctx, "DELETE FROM visitor_consents WHERE fingerprint = $1", fp)
  // 2. 聊天消息（通过会话关联）
  r.db.Exec(ctx, "DELETE FROM chat_messages WHERE session_id IN (SELECT id FROM sessions WHERE visitor_id IN (SELECT id FROM visitors WHERE fingerprint = $1))", fp)
  // 3. 共浏览指令
  r.db.Exec(ctx, "DELETE FROM co_browsing_commands WHERE session_id IN (...)")
  // 4. 事件数据
  r.db.Exec(ctx, "DELETE FROM event_blobs WHERE session_id IN (...)")
  // 5. 会话
  r.db.Exec(ctx, "DELETE FROM sessions WHERE visitor_id IN (...)")
  // 6. 访客
  r.db.Exec(ctx, "DELETE FROM visitors WHERE fingerprint = $1", fp)
  return nil
}`,
        codeLanguage: 'go',
      },
      {
        heading: '数据保留：内建的 GC 机制',
        body: `会话回放数据积累得很快。一个中等流量的站点每周可以产生 GB 级别的事件数据。如果没有保留策略，存储成本会无限增长，隐私风险随着每个字节增加。\n\nPinConsole 运行一个每小时执行一次的垃圾回收（GC）worker。它的工作是删除已结束且超过保留期的会话。默认保留期为 30 天——与常见 SaaS 实践一致。\n\nGC 做得很彻底。它不只是删除 event_blobs。它按反向依赖顺序清理五个表，与擦除级联一致：\n\n1. event_blobs（MinIO 对象 + PG 行）\n2. chat_messages\n3. co_browsing_commands\n4. sessions\n5. visitors（last_seen_at < 阈值）\n\n每批最多处理 1000 条记录，防止长时间运行的事务。如果某批处理中途失败，下一个 GC 周期会从中断处继续。Worker 在启动时立即运行，然后按每小时定时器执行。\n\n保留期通过 RETENTION_DAYS 环境变量配置。运营人员可以为不同站点设置不同的保留期——高流量营销页面 7 天，关键引导流程 90 天，合规审计追踪永久保留。\n\n我们刻意没有使用 MinIO 存储桶生命周期策略进行删除。删除由应用逻辑驱动，而非存储层规则。这确保了 GC 尊重会话边界（部分清理会破坏回放），协调 PG 元数据删除，并记录其删除的内容。`,
        code: `// GC worker：每小时运行，分批处理
func (w *GCWorker) Run(ctx context.Context) {
  ticker := time.NewTicker(1 * time.Hour)
  defer ticker.Stop()

  // 启动时立即运行
  w.gcOnce(ctx)

  for {
    select {
    case <-ticker.C:
      w.gcOnce(ctx)
    case <-ctx.Done():
      return
    }
  }
}

func (w *GCWorker) gcOnce(ctx context.Context) {
  threshold := time.Now().Add(-w.cfg.RetentionPeriod)
  // event_blobs → chat_messages → co_browsing_commands
  // → sessions → visitors
  w.repo.DeleteSessionsEndedBefore(ctx, threshold, 1000)
}`,
        codeLanguage: 'go',
      },
      {
        heading: '共浏览隐私：先同意，后控制',
        body: `共浏览带来了一个完全不同的隐私问题。这不仅仅是记录发生了什么——而是让一个真实的人类运营人员实时访问访客的屏幕。隐私影响是直接且可感知的。\n\n当运营人员发起共浏览时，访客的浏览器会在任何运营操作到达页面之前显示一个横幅：\n\n• 一个眼睛图标（观察的视觉指示）\n• 运营人员的名字（谁在看的透明性）\n• 一个"退出"按钮（立即终止）\n\n这个横幅在整个共浏览会话期间持续显示。它不是一次性的接受——而是一个持续的知情指示器。如果访客在任何时候感到不适，他们可以一键终止，或者在一秒内按三次 Escape 键。\n\n对于最敏感的部署（金融服务、医疗保健），我们支持被动监控模式。在此模式下，运营人员可以看到访客的屏幕，但不能交互，直到访客显式点击"允许协助"。这确保了运营人员永远不会执行访客未明确授权的操作。\n\n每条共浏览指令都记录到 co_browsing_commands 表，包含 operator_id。如果出现隐私投诉，有完整的审计轨迹可以查询运营人员做了什么、何时做的、在哪个会话上。`,
      },
      {
        heading: 'IP 处理与数据最小化',
        body: `数据最小化（GDPR 第 5(1)(c) 条）意味着只采集你需要的数据。对于会话回放，你需要 DOM 事件——你通常不需要访客的 IP 地址，除了基本的速率限制。\n\nPinConsole 不会永久存储访客 IP 地址。服务器可能会使用 IP 进行：\n\n• 速率限制（内存计数器，不持久化）\n• 地理位置分析（仅国家级别，存储为 ISO 代码，非 IP）\n\n原始 IP 永远不会写入 sessions 表或任何日志。如果你需要更严格的控制，可以在反向代理层配置剥离 X-Forwarded-For 头，使其在到达应用之前就被移除。\n\n我们还最小化了会话标识符。每个会话由服务器生成的 UUID 标识——不是访客 ID cookie，不是指纹哈希。核心 SDK 中没有内建的跨会话追踪。如果你需要识别回访访客，这是一个需要显式配置的 opt-in 功能。`,
      },
      {
        heading: '自托管隐私：结构性的优势',
        body: `所有这些隐私功能在 SaaS 会话回放工具中也存在——但有一个根本区别：你的数据存储在它们的服务器上。\n\n使用 PinConsole，每个隐私保证都是结构性的。当数据从不离开你的基础设施：\n\n• 无需签署数据处理协议（DPA）\n• 无需进行国际传输影响评估（TIA）\n• 无需对数据处理者进行第三方供应商风险评估\n• 你的数据保留政策由你的存储成本强制执行，而非供应商的定价层级\n\n对于欧盟公司来说，这是"我们使用一家基于美国的 SaaS，数据存储在 AWS 美东区域"和"我们在自己的欧盟托管服务器上运行软件"之间的区别。后者所需的法律工作显著少于前者——尤其是在 GDPR 第五章（国际数据传输）的合规要求下。\n\n而且因为我们是开源的（AGPL-3.0），隐私保证是可审计的。你可以通过阅读代码来验证脱敏逻辑、同意流程和擦除级联。你不需要信任供应商的 SOC 2 报告——你可以准确看到数据如何从浏览器流向存储。`,
      },
      {
        heading: '隐私是功能，不是合规练习',
        body: `将会话回放中的隐私内建不仅仅是避免罚款。而是建立与访客之间的信任。\n\n当访客访问你的网站，看到一个清晰的同意对话框，解释了将要录制什么并赋予他们真正的控制权时，他们的信任会增加。当他们知道密码输入永远不会被捕获、表单提交默认脱敏、可以通过一个简单的流程请求删除时，会话回放的监控性质就变得不那么令人担忧了。\n\n对我们来说，隐私不是一个独立的工作流。它融入了每一层——从 SDK 的默认脱敏输入，到 GC worker 的每小时清理，再到管理后台的擦除界面。隐私是一级功能，与认证、回放和共浏览一样作为独立的纵向切片（1l）追踪。\n\n如果你正在评估会话回放工具，而隐私是一个关注点（它应该就是），PinConsole 提供了一个根本不同的方案：自托管、可审计、默认隐私优先。AGPL-3.0 免费使用。`,
      },
    ],
  },
  relatedPosts: [
    { title: '会话回放的纵深防御：用 Go 构建反爬虫基础设施', url: '/blog/defense-in-depth-zh/', description: '四层反爬虫纵深防御体系——速率限制、行为分析、SDK 指纹和 WebSocket 认证。' },
    { title: '如何构建一个自托管的 FullStory 替代品', url: '/blog/self-hosted-fullstory-alternative/', description: 'PinConsole 架构决策解析——Go、rrweb、MinIO、单二进制部署。' },
    { title: '开源共浏览实现', url: '/blog/building-co-browsing-zh/', description: 'PinConsole 如何实现隐私优先的共浏览。' },
  ],
  cta: {
    title: '体验隐私优先的会话回放',
    subtitle: '自托管。可审计。从第一天起就为 GDPR 就绪。',
    primary: { label: '在 GitHub 上开始', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: '联系维护者', href: '#consult' },
  },
};
