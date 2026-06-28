import type { BlogContent } from './types';

export const cobrowsingZh: BlogContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: '开源共浏览（Co-Browsing）实现：如何让两个浏览器共享一个会话',
    description: '深度解析 PinConsole 的双向共浏览架构——rrweb 节点 ID 选择器、1:1 运营锁定、300ms 表单防抖、以及 hub-and-spoke WebSocket 路由的核心实现。',
    ogTitle: '开源共浏览实现：让两个浏览器共享一个会话',
    ogDescription: 'rrweb 节点 ID、1:1 运营锁定、WebSocket hub 路由——PinConsole 开源共浏览的完整架构解析。',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-28',
    readingTime: '12 分钟',
    tags: ['技术架构', '共浏览', 'WebSocket', '开源'],
  },
  hero: {
    h1: '开源共浏览（Co-Browsing）实现：如何让两个浏览器共享一个会话',
    subtitle: 'PinConsole 双向共浏览的架构设计、技术权衡和安全机制——完全基于 rrweb 和 hub-and-spoke WebSocket 模型构建。',
  },
  content: {
    sections: [
      {
        heading: '为什么共浏览比会话回放难得多',
        body: `会话回放是一个已经解决的问题。你录制访客交互过程中的 DOM 变化，序列化为事件流，然后回放出来。它是"发后即焚"的——运营人员观看的是"刚才发生了什么"。\n\n共浏览正好反过来。运营人员需要伸入访客的实时浏览器，让事情发生——点击按钮、滚动到某个位置、填写表单字段。而且这一切必须感觉自然，没有半秒延迟，没有视觉闪烁，没有安全漏洞。\n\n这也是为什么大多数共浏览工具都是昂贵的 SaaS 产品。Cobrowse.io 每个运营人员每月收费 $149，Upscope 起步价 $99/月。而且没有一个开源的。\n\n我们构建 PinConsole 的共浏览就是为了改变这一点。它是免费的、自托管的，并且完全构建在与会话回放相同的 rrweb 基础之上。以下是它的底层工作原理。`,
      },
      {
        heading: '为什么用 rrweb 节点 ID 而非 CSS 选择器',
        body: `当运营人员在回放页面上点击时，服务器需要告诉访客浏览器："点击这个元素。"最直观的做法是 CSS 选择器——比如 div.container > button.btn-primary:nth-child(3)。\n\n我们没有这么做。原因如下：\n\nCSS 选择器是脆弱的。一个动态的 class 变化、一次导致 DOM 顺序改变的重新渲染、或者一次颠倒元素位置的 A/B 测试，都会破坏选择器。运营人员点了"立即购买"，但访客浏览器却高亮了页脚的链接。这不仅是个 bug——更是一场支持灾难。\n\nrrweb 优雅地解决了这个问题。rrweb 捕获的每个 DOM 节点都被分配了一个稳定的 data-rr-node-id 属性。这个 ID 对于相同的 DOM 树是确定性的——同一页面的两次快照会产生相同的节点 ID。而且 rrweb 会追踪节点的添加和删除，因此即使 DOM 发生变化，ID 仍然有效。\n\n当运营人员在 rrweb 回放画布上点击时，我们使用 elementFromPoint() 找到被点击的元素，沿着 DOM 树向上遍历直到找到带有 data-rr-node-id 的节点，然后将这个 ID 发送给访客的浏览器。访客 SDK 随后调用 document.querySelector([data-rr-node-id="<id>"]) 来定位目标并执行操作。\n\n这种方法经得起完整的页面重新渲染、CSS class 翻转和动态内容替换。只要 rrweb 的 DOM 快照逻辑是一致的（绝对是——它已经在数百万会话中经受住了考验），共浏览的目标就永远是正确的。`,
        code: `// Admin 覆盖层：查找被点击元素的 rrweb 节点 ID
function findRRWebNode(target: Element): number | null {
  let el: Element | null = target;
  while (el) {
    const id = el.getAttribute('data-rr-node-id');
    if (id !== null) return parseInt(id, 10);
    el = el.parentElement;
  }
  return null;
}`,
        codeLanguage: 'typescript',
      },
      {
        heading: '架构：Hub-and-Spoke WebSocket 路由',
        body: `共浏览流量全部经过一个中心化的 Go 服务器。没有 P2P、没有 WebRTC、没有第三方中继。每一条消息——访客的 DOM 变化、运营人员的鼠标移动、表单填写指令——都经过 hub。\n\n我们选择 coder/websocket 作为 WebSocket 层。它简洁、地道、无全局状态、没有隐式的 goroutine 启动，API 干净。Hub 的工作模式如下：\n\n• 每个站点是一个由站点 ID 标识的"房间"\n• 访客 SDK 连接到其站点所在的房间\n• 运营人员连接到同一个房间\n• 消息按类型路由：访客事件广播给所有订阅的运营人员，运营人员指令直接发送给目标访客\n\n一个关键的设计决策：我们不会将所有事件广播给所有运营人员。每个会话都是 1:1 锁定的。这避免了广播开销，也防止了两个运营人员争夺同一会话控制权的噩梦场景。\n\n对于反向路径（运营人员 → 访客），服务器维护了一个 visitorClients 映射：会话 ID → WebSocket 连接 ID。当运营人员提交一条指令时，服务器查找访客的连接并直接推送消息。`,
        code: `// 简化的 hub 路由
type Hub struct {
  sessions       map[uuid.UUID]*SessionChan  // session → 订阅者通道
  visitorClients map[uuid.UUID]uuid.UUID     // session → 访客客户端 ID
  clients       map[uuid.UUID]*Client        // 客户端 ID → WebSocket 连接
}

// 访客发送事件 → 所有订阅的运营人员
func (h *Hub) PublishEvent(sessionID uuid.UUID, msg []byte) {
  if ch, ok := h.sessions[sessionID]; ok {
    ch.publish(msg)  // 非阻塞扇出到所有订阅者
  }
}

// 运营人员发送指令 → 特定访客
func (h *Hub) SendCommandToVisitor(sessionID uuid.UUID, msg []byte) {
  if visitorID, ok := h.visitorClients[sessionID]; ok {
    if client, ok := h.clients[visitorID]; ok {
      client.Send(msg)
    }
  }
}`,
        codeLanguage: 'go',
      },
      {
        heading: '1:1 锁定：原子化认领协议',
        body: `多运营人员同时进行共浏览，如果没有所有权约束就是一场灾难。想象两个技术支持人员同时填写同一个表单字段。我们通过基于 Redis 的原子化认领/释放协议来防止这种情况。\n\n流程如下：\n\n1. 运营人员点击"开始共浏览"→ POST /api/sessions/:id/claim\n2. 服务器执行 Redis SET NX EX 300，键为 claim:session:<uuid>\n3. 如果键不存在，认领成功，运营人员被记录为所有者（value = user_id）\n4. 如果键已存在，运营人员看到"Alice 正在协助此会话"\n5. Admin SPA 每 60 秒发送一次心跳以刷新 TTL\n6. 运营人员点击"释放"或关闭会话时，POST /api/sessions/:id/release 执行一个 Lua 脚本，原子地检查所有权并删除键\n\n认领在三个层面强制执行：\n\n• HTTP API — 所有指令端点在接受指令前通过 Redis 检查认领所有权\n• Hub 路由 — 只有已认领运营人员的指令才会被转发给访客\n• 审计日志 — 每条指令都记录到 co_browsing_commands 表，包含 operator_id，可追溯谁做了什么\n\n这种"认领即能力"的模式意味着系统默认是安全的。未认领会话的运营人员可以观察事件，但无法发送指令。`,
        code: `-- Lua：带所有权检查的原子化释放
local key = KEYS[1]
local expectedOwner = ARGV[1]
local current = redis.call('GET', key)
if current == expectedOwner then
  return redis.call('DEL', key)
end
return 0`,
        codeLanguage: 'lua',
      },
      {
        heading: '覆盖层：在回放画布上捕获运营人员操作',
        body: `运营人员的共浏览界面是一个覆盖在 rrweb 播放器上的透明层。当运营人员启用共浏览时，播放器进入"实时交互"模式：\n\n• 一个不可见的 div（z-index: 10, cursor: crosshair）位于回放 iframe 之上\n• 鼠标移动以 30fps 的频率被捕获（rAF 限流）并作为 cursor_highlight 指令发送\n• 点击操作经过 200ms 的去抖处理以防止重复触发\n• 点击坐标通过 rrweb iframe 转换，找到正确的 DOM 节点\n\n光标高亮值得特别关注。运营人员的鼠标位置被发送到访客浏览器，访客端渲染一个固定位置的 SVG 圆形图标并显示运营人员姓名。这个圆点跟随运营人员的移动，使用平滑的 CSS 过渡（transform: translate() 50ms linear），营造出有人在访客真实屏幕上指点东西的错觉。\n\n对于滚动，我们直接发送 window.scrollTo(x, y) 指令。这有意设计得简单——运营人员端的复杂滚动动画不需要逐帧复制。访客浏览器跳转到正确位置，对于"看下面这个区域"来说已经足够。`,
      },
      {
        heading: '表单填写：共浏览最难的问题',
        body: `在别人的浏览器上填写表单听起来很简单：设置值然后派发一个事件。但现代前端框架（React、Vue、Angular）不监听 value 属性的变化——它们监听具有特定属性的合成 input 事件。\n\n如果你只做 element.value = 'text' 然后派发一个普通的 Event('input')，React 不会感知到。框架使用 HTMLInputElement.prototype 上的属性描述符来拦截原生 setter。你需要直接调用那个 setter。\n\n我们的解决方案：\n\n1. 保存 HTMLInputElement.prototype 上 value 的原生属性描述符\n2. 调用原生 setter 来更新值\n3. 派发 'input' 和 'change' 事件，附带 bubbles: true 和 cancelable: true\n4. 框架捕获到变化并更新其虚拟 DOM\n\n我们还为每个字段添加了 300ms 的去抖。当运营人员在表单字段中打字时，每次按键都作为独立的 fill_input 指令发送。300ms 去抖防止了写入重叠——如果访客的网络较慢，指令会排队并按序执行。\n\n在表单填写过程中，访客会看到活动字段上有一个微妙的蓝色边框，以及一条 toast 通知："运营人员正在填写 [字段标签]。"这种透明性对建立信任至关重要——访客始终知道运营人员何时在与他们的页面交互。\n\n如果某个字段 5 秒无操作，视觉锁定会自动释放。运营人员也可以显式释放对所有字段的控制。`,
        code: `// 调用原生 value setter 以实现框架兼容
function setNativeValue(element: HTMLInputElement, value: string) {
  const nativeSetter = Object.getOwnPropertyDescriptor(
    window.HTMLInputElement.prototype, 'value'
  )?.set;
  nativeSetter?.call(element, value);
  element.dispatchEvent(new Event('input', { bubbles: true }));
  element.dispatchEvent(new Event('change', { bubbles: true }));
}`,
        codeLanguage: 'typescript',
      },
      {
        heading: '安全设计：URL 校验、指令白名单和紧急退出',
        body: `赋予运营人员控制访客浏览器的能力是一个安全敏感的功能。我们构建了多层安全保护：\n\n指令白名单。只有 8 种指令类型被接受：cursor_highlight、click、scroll、fill_input、navigate、release_control、show_popup 和 chat_message。任何未知类型都会被静默丢弃。\n\n导航 URL 校验。当运营人员发送 navigate 指令时，服务器会校验目标 URL 是否与会话的来源同源。同源导航自由放行。跨源导航会被阻止，除非站点运营人员在管理后台显式白名单了该域名。这防止了运营人员将访客重定向到恶意站点。\n\n弹窗 URL 消毒。show_popup 指令的 action_url 经过严格的 scheme 白名单校验：只允许 https: 和 mailto:。javascript: 和 data: URL 在 API 层被拒绝。\n\n访客端紧急退出。访客可以通过以下方式立即结束共浏览会话：\n\n• 1 秒内按三次 Escape 键\n• 按 Ctrl+Shift+X\n• 点击共浏览横幅上的"退出"按钮\n\n三种触发方式都会调用 onReleased() 回调，移除运营人员光标、清除视觉锁定、关闭共浏览通道。运营人员会看到"访客已结束会话。"\n\n这些安全措施不是理论上的——它们已经通过我们的 e2e 测试套件验证（Playwright fork-5 测试，模拟完整的共浏览会话及紧急退出场景）。`,
      },
      {
        heading: 'GDPR 同意：默认透明',
        body: `共浏览涉及一个真实的人类实时查看访客的屏幕。这在 GDPR、CCPA 等法规下显然有隐私方面的考量。\n\n当运营人员发起共浏览时，访客的 SDK 会在任何运营操作生效前触发一个同意横幅。横幅显示：\n\n• 一个 SVG 眼睛图标（视觉上表明有人在看）\n• 运营人员的名字（让访客知道是谁）\n• 一个"退出"按钮用于立即终止\n\n横幅固定在视口顶部，z-index: 999999，确保始终可见。它在整个共浏览会话期间保持显示。如果访客关闭了它，共浏览继续，但关键交互（如表单填写）时可以重新显示。\n\n这不仅仅是好的 UX——在大多数司法管辖区，这是法律要求。而且因为 PinConsole 是自托管的，你可以自定义同意文本、样式和行为，以满足你的特定合规需求。\n\n对于需要更严格控制的站点，我们还支持被动监控模式。在这种模式下，运营人员可以看到访客的屏幕，但不能交互，直到访客显式点击"允许协助"。这是金融服务和医疗保健部署的默认模式。`,
      },
      {
        heading: '可观测性：端到端追踪运营人员操作',
        body: `当共浏览过程中出现问题时——一条未执行的指令、一个断开的访客——你需要追踪发生了什么。我们用 trace_id 传播来贯穿整个指令生命周期。\n\n当运营人员发送一条指令时，服务器生成一个 trace_id 并将其嵌入 MsgCommand 信封中。访客的 SDK 接收指令、执行它，并将 trace_id 缓存最多 5 秒（或 10 个事件）。由此产生的任何 rrweb 事件都携带相同的 trace_id 通过服务器返回给管理面板。\n\n这就形成了一个端到端的追踪回路：\n\n运营人员点击 → [trace_id: abc123] → 服务器 → [trace_id: abc123] → SDK 执行点击 → [trace_id: abc123] → rrweb 记录产生的 DOM 变化 → [trace_id: abc123] → 服务器存储事件 → [trace_id: abc123] → 管理面板显示带有 trace 的事件\n\n实践中，这意味着一个发送了"点击"指令但没有看到视觉反馈的运营人员可以查看追踪：指令收到了吗？执行了吗？DOM 变化传回来了吗？答案都在日志里，由同一个 trace_id 串联。\n\n每条指令也被记录到 co_browsing_commands PostgreSQL 表中，包含 operator_id、session_id、command_type、target_node_id 和 payload。这个审计轨迹对调试和合规都至关重要。`,
      },
      {
        heading: '对比：与专业共浏览工具的差异',
        body: `以下是 PinConsole 的共浏览与专业工具的对比：\n\n| 特性 | PinConsole | Cobrowse.io | Upscope |\n|---|---|---|---|\n| **价格** | 免费 (AGPL) | $149/运营/月 | $99/运营/月 |\n| **自托管** | ✓ | ✗（仅企业版） | ✗ |\n| **开源** | ✓ AGPL-3.0 | ✗ | ✗ |\n| **会话回放 + 共浏览** | ✓ 统一平台 | 分离产品 | 分离产品 |\n| **表单填写** | ✓ (300ms 防抖) | ✓ | ✓ |\n| **导航** | ✓（同源） | ✓ | ✓ |\n| **访客同意横幅** | ✓ (GDPR 就绪) | ✓ | ✓ |\n| **运营锁定** | ✓ 原子化认领 | ✓ | ✓ |\n| **紧急退出** | ✓ 三击 Escape | ✓ | ✓ |\n| **共浏览审计日志** | ✓ PostgreSQL | 有限 | 有限 |\n| **端到端追踪** | ✓ trace_id 回路 | ✗ | ✗ |\n\n关键差异在于集成。因为我们的共浏览与会话回放和实时监控运行在同一平台上，运营人员可以观看来访者的会话、定位痛点、然后无缝切换进入共浏览，无需切换工具。访客的上下文在过渡中完整保留——没有"新建会话"的手续，没有丢失的历史。\n\n以 $0/月无限运营人员（自托管）的价格，成本优势显而易见。但真正的价值在于统一的工作流：会话回放 → 实时监控 → 共浏览，全部在一个自托管的二进制中。`,
      },
      {
        heading: '共浏览的下一个里程碑',
        body: `我们的共浏览实现覆盖了核心用例——点击、滚动、表单填写、导航和聊天。但还有成长空间：\n\n• 移动端共浏览 —— rrweb 节点 ID 方法在移动浏览器上同样有效，但覆盖层的 UX 需要针对触控界面重新设计。我们正在探索一种"镜像模式"，运营人员可以看到一个手机尺寸的视口。\n\n• 多运营切换 —— 目前是 1:1 锁定。一个交接协议可以让一个运营人员将控制权转移给另一个，而无需断开访客。\n\n• 屏幕共享兜底 —— 对于我们的指令协议无法处理的复杂交互（拖拽、画布绘图），基于 WebRTC 的屏幕共享兜底可以覆盖这些边缘场景。\n\n• 共浏览录制 —— 将运营人员的交互与访客的会话一起录制，用于培训和质控。\n\n这些都列在 post-v1 待办清单上。当前的实现已经能处理 90% 的用例：一个支持人员实时帮助访客导航一个 web 应用。\n\n如果你正在构建一个需要共浏览的产品——或者你已经厌倦了为一项本应属于会话回放工具的功能每月支付 $149/运营人员——PinConsole 是免费的、自托管的、开源的。5 分钟即可尝试。`,
        code: `git clone https://github.com/iannil/pinconsole\ncd pinconsole\ndocker compose up -d`,
        codeLanguage: 'bash',
      },
    ],
  },
  relatedPosts: [
    { title: '如何构建一个自托管的 FullStory 替代品', url: '/blog/self-hosted-fullstory-alternative/', description: 'PinConsole 的完整架构——Go、rrweb、MinIO 和单二进制部署。' },
    { title: '开源会话回放的隐私设计', url: '/blog/privacy-by-design-zh/', description: 'PinConsole 如何实现 GDPR 优先的会话监控。' },
  ],
  cta: {
    title: '在 PinConsole 上体验共浏览',
    subtitle: '自托管的会话回放 + 共浏览，一个二进制搞定。免费。AGPL-3.0。',
    primary: { label: '在 GitHub 上开始', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: '联系维护者', href: '#consult' },
  },
};
