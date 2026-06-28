import type { BlogContent } from './types';

export const securityZh: BlogContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: '会话回放的纵深防御：用 Go 构建反爬虫基础设施',
    description: 'PinConsole 如何实现多层反爬虫纵深防御——速率限制、行为分析、SDK 指纹、WebSocket 认证和 URL 校验。构建自托管会话回放平台的安全技术指南。',
    ogTitle: '会话回放的纵深防御：反爬虫基础设施构建实录',
    ogDescription: '速率限制、行为分析、SDK 指纹、WebSocket 认证——PinConsole 的四层反爬虫纵深防御体系。',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-28',
    readingTime: '11 分钟',
    tags: ['安全', '反爬虫', '技术架构', '自托管'],
  },
  hero: {
    h1: '会话回放的纵深防御：用 Go 构建反爬虫基础设施',
    subtitle: '速率限制、行为分析、SDK 指纹、WebSocket 认证——PinConsole 如何保护自托管会话回放免受爬虫和攻击。',
  },
  content: {
    sections: [
      {
        heading: '为什么会话回放工具是爬虫的重点目标',
        body: `会话回放工具处于一个独特的安全位置。它们向每个访客提供 JavaScript SDK，维持持久的 WebSocket 连接，并将 DOM 数据流式传回服务器。对爬虫操作者来说，这是一个金矿——SDK 是公开的，WebSocket 协议可以被逆向，服务器接受来自任何人的事件数据。\n\n一个简单的会话回放部署可能以多种方式被滥用：\n\n• 事件注入：爬虫发送虚假事件以污染分析数据\n• 数据窃取：爬虫利用 SDK 大规模抓取页面内容\n• 凭证填充：如果速率限制薄弱，登录端点会被暴力破解\n• 会话劫持：如果 WebSocket 认证薄弱，攻击者可以监听实时访客流\n\n因为 PinConsole 是自托管的，安全态势与 SaaS 工具有所不同。没有供应商管理的 WAF，前面没有 Cloudflare，没有安全团队监控流量模式。安全必须内建于应用本身。\n\n以下是我们构建纵深防御的方式——四层防护，每层捕获前一层遗漏的攻击。`,
      },
      {
        heading: '第一层：HTTP 速率限制（每 IP 固定窗口）',
        body: `最外层的防御是 HTTP 级别的速率限制。每个到达 PinConsole API 的请求都经过一个 Gin 中间件，强制执行每 IP 的速率上限。\n\n实现很直接：对于每个请求，我们对一个 Redis 键（ratelimit:{ip}:{window}）执行 INCR 并设置 EXPIRE 为窗口时长（默认 60 秒）。如果计数超过限制（默认每分钟 60 次请求），我们返回 429 Too Many Request，附带标准的速率限制响应头。\n\n几个值得注意的设计决策：\n\n• Redis 故障时优雅处理。如果 Redis 不可达，中间件放行请求。我们宁愿短暂地出现不限速的窗口，也不愿因 Redis 抖动导致整个管理后台离线。\n\n• 速率限制仅在生产环境生效。在开发模式下禁用，以便 e2e 测试不会触发速率上限。\n\n• 健康检查端点（GET /healthz、GET /readyz）完全绕过速率限制。\n\n• 响应中包含 Retry-After 和 X-RateLimit-Remaining 头，行为良好的客户端可以在遇到错误前主动降速。\n\n这一层可以阻止简单的爬虫和配置错误的机器人。但无法阻止使用轮换 IP 的分布式攻击——那需要更深层的分析。`,
        code: `// 简化的 HTTP 速率限制中间件
func RateLimitMiddleware(rdb *redis.Client) gin.HandlerFunc {
  return func(c *gin.Context) {
    ip := c.ClientIP()
    now := time.Now().Unix() / 60  // 1 分钟窗口
    key := fmt.Sprintf("ratelimit:%s:%d", ip, now)

    count, err := rdb.Incr(ctx, key).Result()
    if err != nil {
      c.Next()  // Redis 故障：放行
      return
    }
    if count == 1 {
      rdb.Expire(ctx, key, 60*time.Second)
    }
    if count > 60 {
      c.AbortWithStatusJSON(429, gin.H{"error": "rate_limit_exceeded"})
      return
    }
    c.Next()
  }
}`,
        codeLanguage: 'go',
      },
      {
        heading: '第二层：WebSocket 速率限制（每会话滑动窗口）',
        body: `HTTP 速率限制不适用于 WebSocket 连接——一旦升级成功，连接是持久的，消息自由流动。因此我们需要一个独立的 WebSocket 消息流速率限制器。\n\nPinConsole 对每个会话应用两个滑动窗口限制：\n\n• 每 10 秒窗口最多 500 条消息\n• 每 10 秒窗口最多 50 MiB 数据\n\n两个限制通过一个 Redis Lua 脚本原子性执行，同时 INCR（消息计数）和 INCRBY（字节计数）并设置 EXPIRE。键的作用域是会话 ID，而非 IP——这防止了爬虫通过轮换 IP 绕过限制。\n\n当任一限制被超出时，服务器：\n\n1. 通过 antiscrape.FlagSession() 标记会话（持久化到 Redis，TTL 10 分钟）\n2. 立即关闭 WebSocket 连接，返回 1011（Internal Error）状态码\n3. 记录违规日志，附带会话 ID 供运营人员审查\n\n标记会传播到管理面板——任何监控实时流的运营人员都会看到该会话因可疑活动被标记的警告。\n\n关键的是，速率检查在事件写入事件流之前执行。被标记的消息永远不会持久化到 MinIO 或 Redis Stream——它在进入管道之前就被拒绝了。这防止了即使限制器在流中间触发时事件数据被污染。`,
        code: `-- Lua：原子化 WebSocket 速率检查
local countKey = KEYS[1]       -- ws:rate:count:{session}
local bytesKey = KEYS[2]       -- ws:rate:bytes:{session}
local window = ARGV[1]         -- 10（秒）
local maxMsg = ARGV[2]         -- 500
local maxBytes = ARGV[3]       -- 52428800（50 MiB）
local msgSize = ARGV[4]

local msgCount = redis.call('INCR', countKey)
if msgCount == 1 then
  redis.call('EXPIRE', countKey, window)
end
if msgCount > tonumber(maxMsg) then
  return {0, "rate_exceeded"}
end

local byteCount = redis.call('INCRBY', bytesKey, msgSize)
if byteCount == msgSize then
  redis.call('EXPIRE', bytesKey, window)
end
if byteCount > tonumber(maxBytes) then
  return {0, "rate_exceeded"}
end

return {1, "ok"}`,
        codeLanguage: 'lua',
      },
      {
        heading: '第三层：SDK 指纹与行为分析',
        body: `HTTP 速率限制捕获简单的爬虫。WebSocket 速率限制捕获激进的事件注入。但一个精密的爬虫——以人类般的速度发送人类般的事件模式——可以同时通过这两层。\n\n为此，我们需要行为分析。\n\n当访客 SDK 初始化时，它通过 Canvas 哈希、WebGL 供应商/渲染器、屏幕分辨率和时区收集浏览器指纹。这个指纹不用于跨站追踪——它用于会话识别并作为行为分析的输入。\n\n服务端的 BehaviorTracker 维护每个会话的统计数据：\n\n• 鼠标事件计数\n• 点击位置分布\n• 事件类型分布（鼠标、滚动、输入、调整大小等）\n• 事件间隔统计（最小、最大、首次、最后）\n\n每 100 个事件，跟踪器运行三种启发式检查：\n\n1. 总事件 > 50 且鼠标事件数为 0 → 纯脚本生成。人类总会产生至少一些鼠标移动。\n\n2. 同一 (x, y) 坐标点击超过 20 次 → 机器模式。人类不会在同一像素上点击 20 次。\n\n3. 总事件 > 100 且最大间隔/最小间隔 < 2.0 → 机器生成的时间。人类事件时间是不规则的；机器是均匀的。\n\n如果任何启发式触发，会话在 Redis 中被标记（flagged:session:{id}，TTL 10 分钟）。管理面板展示被标记的会话，附带可见的指示器和标记原因。\n\n这些启发式有意保持简单。它们对真实访客几乎不会产生误报（人类总会移动鼠标并改变时间间隔），同时捕获最常见的爬虫模式。我们不需要 99.9% 的爬虫检测准确率——我们需要让爬虫操作的代价明显高于在竞争对手平台上操作。`,
        code: `// 行为分析：三种启发式检查
func (bt *BehaviorTracker) CheckAndFlag(sessionID string) (flagged bool, reason string) {
  bt.mu.Lock()
  defer bt.mu.Unlock()

  // 启发式 1：无鼠标移动
  if bt.totalEvents > 50 && bt.mouseEvents == 0 {
    return true, "no_mouse_movement"
  }

  // 启发式 2：同一位置重复点击
  for _, count := range bt.clickPositions {
    if count > 20 {
      return true, "repeated_clicks"
    }
  }

  // 启发式 3：均匀的事件时间间隔
  if bt.totalEvents > 100 && bt.minInterval > 0 {
    ratio := float64(bt.maxInterval) / float64(bt.minInterval)
    if ratio < 2.0 {
      return true, "uniform_timing"
    }
  }

  return false, ""
}`,
        codeLanguage: 'go',
      },
      {
        heading: '第四层：WebSocket 认证——URL 中无令牌',
        body: `WebSocket 连接特别容易受到认证泄露的影响。最常见的模式——将 JWT 令牌作为查询参数传递——会将令牌泄露到服务器日志、referrer 头和浏览器历史中。\n\nPinConsole 对运营人员 WebSocket 连接采取了不同的方法。运营人员通过标准的表单 POST 登录，收到一个 HttpOnly 会话 cookie（mm_session）。这个 cookie：\n\n• SameSite=Lax——防止来自外部来源的 CSRF\n• HttpOnly——JavaScript 无法访问，防止 XSS 窃取\n• 生产环境设置 Secure——仅在 HTTPS 下发送\n• 24 小时 MaxAge——自动会话过期\n\n当管理 SPA 打开 WebSocket 连接时，mm_session cookie 由浏览器自动包含（同源）。服务器在接受 WebSocket 升级之前验证 cookie 是否有效（通过 Redis）。如果无效，连接在建立任何数据交换之前被拒绝。\n\n访客 WebSocket 连接使用不同的机制。SDK 调用 POST /api/session/init 后，服务器返回一个 session_id。这个 ID 在 WebSocket 连接建立后的 hello 消息中发送。服务器在接受事件之前验证会话存在性和访客-会话绑定。\n\n对于两种路径，关键原则是：认证在 WebSocket 连接建立之前完成。不存在未认证连接可以观察流量的窗口。`,
      },
      {
        heading: 'User-Agent 黑名单：坦诚面对其局限性',
        body: `我们包含一个 User-Agent 黑名单作为轻量级的第一层过滤。它能捕获最低成本的爬虫：\n\n• curl/、wget/、python-requests/、Go-http-client/\n• HeadlessChrome、PhantomJS、jsdom\n• scrapy、bot、crawler、spider（子串匹配）\n• 空的 User-Agent\n\n检查是不区分大小写的子串匹配——有意设计得简单以避免因创意 UA 格式导致的漏报。被阻止的请求返回 403，附带描述性的错误码（blocked_user_agent 或 empty_user_agent）。\n\n但代码注释很坦诚："UA 黑名单只能阻挡低水平爬虫；现代 Puppeteer/Playwright headless=new 的 UA 与真实浏览器几乎一致。"这一层捕获的是低端爬虫。高级爬虫需要更深层的防御。\n\n值得注意的是，UA 中间件甚至在开发模式下也运行。它是唯一这样做的安全层——因为如果你用 curl 脚本开发 API 调用，你应该知道不设置真实的浏览器 UA 是无法成功的。`,
      },
      {
        heading: 'URL 安全：防止运营人员发起的攻击',
        body: `共浏览赋予运营人员导航访客浏览器和显示弹窗的能力。这很强大——如果不加约束，也很危险。\n\n导航指令经过 isURLAllowed() 检查：\n\n• 同源 URL 始终允许\n• localhost URL 允许（用于开发和内部工具）\n• 跨源 URL 被阻止，除非运营人员显式将其加入白名单\n\n弹窗 URL 经过 isURLSchemeAllowed() 检查：\n\n• 只允许 https: 和相对 URL\n• javascript:、data:、vbscript:、file: 被显式拒绝\n\n这些检查在服务端指令处理程序中执行。即使一个被攻破的管理客户端发送恶意指令，服务器也会在到达访客 SDK 之前将其拒绝。\n\n指令类型本身也被白名单限制——只接受 8 种类型。任何未知类型在 API 层被静默丢弃。`,
      },
      {
        heading: '登录暴力破解防护与密码策略',
        body: `认证系统有自己的专用防护层。\n\n登录尝试按（email，IP）对进行速率限制：15 分钟内 5 次失败触发锁定。计数器通过 Redis 原子 Lua 脚本维护，因此同一凭证的并发请求无法突破限制。\n\nRedis 故障时也优雅处理——如果 Redis 宕机，登录请求绕过速率检查。我们宁愿出现短暂的无限制登录窗口，也不愿将所有运营人员永久锁定在管理面板之外。\n\n密码使用 bcrypt 以成本 12 进行哈希。最低成本在启动时强制执行——如果有人将 BCRYPT_COST 设置为低于 12，服务器拒绝启动。这防止了在生产环境中意外削弱哈希成本。\n\n管理员账号创建流程要求操作者在首次登录时更改默认密码。SYSTEM_EMAIL 和 SYSTEM_PASSWORD 环境变量在启动时被验证，以确保在生产模式下它们不是默认值。`,
      },
      {
        heading: '配置 Fail-Secure：防止部署错误',
        body: `安全配置错误是导致泄露的最常见原因。我们在服务器绑定到任何端口之前构建了一个启动验证系统，检查关键配置：\n\n• SERVER_ENV 必须是以下之一：development、staging、production。像"produciton"这样的拼写错误会被捕获。\n• 在生产模式下，SYSTEM_PASSWORD 不能是默认值。\n• 在生产模式下，MinIO 凭证不能是默认的 minioadmin/minioadmin。\n• 如果 PostgreSQL 不在 localhost，必须启用 SSL 模式。\n• 如果 MinIO 端点不是 localhost，必须启用 TLS（https://）。\n\n每次检查都会产生特定的错误信息："SERVER_ENV=produciton is invalid; did you mean production?"——而不是通用的"配置错误"。这减少了部署时的调试时间。\n\n有些检查是环境有条件的。例如，本地开发可以使用不带 SSL 的普通 PostgreSQL，但部署到远程数据库的生产环境必须启用 SSL。服务器根据部署上下文调整其严格程度。\n\n我们还在代码层面强制执行发布构建标志。绕过机制（如开发模式认证跳过）由 //go:build !release 构建标签控制。即使有人将 SERVER_ENV=development 设置在生产环境中，发布二进制文件的绕过函数始终返回 false。编译器强制执行这一点——这不是一个可能配置错误的运行时检查。`,
      },
      {
        heading: '我们没有做什么（以及为什么）',
        body: `对安全的坦诚很重要。以下是我们刻意不实现的部分：\n\n没有 TLS 指纹（JA3）。对 WebSocket 连接进行被动 JA3 指纹识别是一种强大的反爬虫技术，但它需要维护指纹数据库并处理浏览器版本变化导致的误报。我们选择了行为分析——更简单、更透明，且更难在不根本改变爬虫行为的情况下绕过。\n\n没有内容安全策略（CSP）。因为管理面板、访客 SDK 和 API 共享同源（单二进制部署），CSP 增加了复杂性却没有成比例的好处。CSP 本可防止的主要攻击向量——XSS——已经被 HttpOnly cookie 和输入验证所缓解。\n\n没有 WAF。自托管部署有自己的反向代理（Nginx、Caddy），添加 WAF 会增加部署摩擦。安全应该易于正确实现。\n\n这些缺口由部署环境填补：反向代理处的 TLS 终结、防火墙处的网络级 IP 封锁、以及基础设施层面的监控。\n\n对于自托管工具来说，"与部署环境协同防御"比"单个应用内的防御"更加现实。我们构建的四层处理应用层面的威胁。基础设施层面的威胁由运营者已有的安全栈处理——因为他们比我们更了解自己的网络。`,
      },
      {
        heading: '通过审计确保安全：我们如何发现并修复漏洞',
        body: `没有安全系统在第一天就是完美的。我们的擦除 API（DELETE /api/privacy/visitor/:fingerprint）最初没有角色检查——任何已认证的运营人员都可以触发 GDPR 擦除。这在一个常规代码审计中（切片 1ac）被捕获。\n\n修复是一个单一的 guard：在处理擦除请求之前要求 admin 角色。提交信息说明："role check: only admin can delete visitors。"\n\n这是我们刻意遵循的模式。不是试图一开始就构建一个完美的安全模型然后冻结它，而是早期发布，使用合理的保护措施，并持续审计。我们自己的修复的审计日志是透明的——没有通过隐蔽手段实现的安全。\n\n如果你正在部署 PinConsole，你可以在 git 日志中看到每个安全修复。你可以运行测试套件来验证修复是否有效。如果你发现了漏洞，你可以提交修复——因为它是开源的。`,
      },
    ],
  },
  relatedPosts: [
    { title: '构建生产级 WebSocket Hub：支撑 500 条并发连接', url: '/blog/websocket-hub-500-concurrent-zh/', description: 'PinConsole 高性能 WebSocket Hub 架构。' },
    { title: '如何构建一个自托管的 FullStory 替代品', url: '/blog/self-hosted-fullstory-alternative/', description: 'PinConsole 架构决策解析。' },
  ],
  cta: {
    title: '体验自托管的会话回放',
    subtitle: '纵深防御内建其中。AGPL-3.0。你的数据，你的服务器。',
    primary: { label: '在 GitHub 上开始', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: '联系维护者', href: '#consult' },
  },
};
