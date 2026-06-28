import type { BlogContent } from './types';

export const websocketHubZh: BlogContent = {
  locale: 'zh',
  htmlLang: 'zh-CN',
  meta: {
    title: '构建生产级 WebSocket Hub：支撑 500 条并发会话回放连接',
    description: '深度解析 PinConsole 的 WebSocket Hub 架构——每连接独立 goroutine、非阻塞扇出、Redis Stream 批处理、MinIO 刷盘模式，支撑 500 个并发访客实时监控。',
    ogTitle: '构建生产级 WebSocket Hub：500 条并发连接架构',
    ogDescription: '每连接独立 goroutine、非阻塞扇出、Redis Stream 批处理、MinIO 刷盘——PinConsole 实时 Hub 的架构实现。',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-28',
    readingTime: '12 分钟',
    tags: ['技术架构', 'WebSocket', 'Go', '实时监控'],
  },
  hero: {
    h1: '构建生产级 WebSocket Hub：支撑 500 条并发会话回放连接',
    subtitle: '每连接独立 goroutine、非阻塞扇出、Redis Stream 批处理、MinIO 刷盘模式——PinConsole 实时架构的实现细节。',
  },
  content: {
    sections: [
      {
        heading: '实时挑战',
        body: `会话回放本质上是一个实时系统。虽然回放已录制的会话是核心功能，但对运营人员来说真正的价值在于实时观看访客——看到他们的光标移动、表单填写、页面滚动——并在发现问题的瞬间切入共浏览。\n\n这意味着每个访客事件都必须以最小的延迟被捕获、传输、存储并广播到管理面板。对于一个访客来说，这很容易。对于 500 个并发访客，每个人都以 60fps 生成 rrweb 事件，数学计算变得严峻：500 × 60 = 每秒 30,000 个事件流经 hub。\n\n我们将每个房间 500 个并发 WebSocket 连接设为基础目标。这是一个运营人员实时服务中等规模电商网站流量所需的容量。架构需要在不断开事件、不内存爆炸、不阻塞事件管道的情况下处理这个负载。\n\n以下是我们构建的方式。`,
      },
      {
        heading: 'Hub：进程内注册中心，而非消息代理',
        body: `第一个架构决策是：我们使用外部消息代理（Redis Pub/Sub、RabbitMQ、NATS）还是进程内 hub？\n\n外部代理可以让 PinConsole 水平扩展——启动更多服务器实例来处理更多流量。但它增加了延迟、运维复杂度和部署摩擦。对于一个以"docker compose up -d"为简洁性目标的自托管工具，每个外部依赖都是有成本的。\n\n我们选择了进程内 hub：一个带有四个由互斥锁保护的映射的 Go 结构体：\n\n1. clients —— 将连接 ID 映射到 Client（WebSocket 包装器）\n2. sessions —— 将会话 ID 映射到 SessionChan（扇出通道）\n3. tenantRooms —— 将租户 ID 映射到 TenantRoom（存在广播）\n4. visitorClients —— 将会话 ID 映射到访客客户端 ID（反向指令路径）\n\nHub 是进程内的单例。所有 goroutine——访客 WebSocket 读取器、运营 WebSocket 读取器、Redis Stream 刷盘器——共享同一个 hub 实例。没有网络跳转、没有序列化开销、没有需要配置的消息代理。\n\n代价是：PinConsole 在 hub 层面不可水平扩展。所有 WebSocket 连接必须落在同一台服务器实例上。但对于每个房间 500 个并发连接，单个 Go 进程可以轻松处理。我们为 95% 分位的部署进行优化，而非 99.9%。`,
        code: `// Hub：进程内注册中心，无需外部代理
type Hub struct {
  mu             sync.RWMutex
  clients        map[uuid.UUID]*Client
  sessions       map[uuid.UUID]*SessionChan
  tenants        map[uuid.UUID]*TenantRoom
  visitorClients map[uuid.UUID]uuid.UUID
}`,
        codeLanguage: 'go',
      },
      {
        heading: 'Client：每连接独立 Goroutine 带有限写入缓冲',
        body: `每个 WebSocket 连接被包装在一个 Client 结构体中。关键的设计元素是写入循环——一个专用的 goroutine，用于序列化对 WebSocket 连接的写入。\n\n当 hub 需要向客户端发送消息时（例如，从访客会话转发的事件），它将消息推送到客户端的 writeCh 通道上。writeLoop goroutine 拾取它并调用 conn.Write()。这确保了对同一 WebSocket 连接的写入被序列化——对 WebSocket 连接的并发写入是不安全的。\n\nwriteCh 的容量为 256 条消息。如果缓冲已满，Send() 方法使用带有 default case 的非阻塞 select：不是阻塞发布者，而是记录一条警告并丢弃消息。这是一个刻意的选择——慢速读取器（网络拥塞、暂停的标签页）永远不应阻塞整个 hub。\n\n每个 Client 恰好运行一个 writeLoop goroutine。对于 500 个并发连接，就是 500 个用于写入的 goroutine。Go 调度器优雅地处理了这一点——每个 goroutine 阻塞在通道接收上，只在有工作时唤醒。内存开销极小（每个 goroutine 栈约 8KB）。\n\n当客户端断开连接时，UnregisterClient() 原子性地清理所有订阅和已加入的租户房间。closeOnce sync.Once 防止通道的双重关闭。`,
        code: `// Client 写入循环：序列化写入，不阻塞调用者
func (c *Client) writeLoop(ctx context.Context) {
  for {
    select {
    case msg := <-c.writeCh:
      ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
      err := c.Conn.Write(ctx, websocket.MessageBinary, msg)
      cancel()
      if err != nil {
        return  // 连接关闭
      }
    case <-c.closeCh:
      return
    }
  }
}

// Send：非阻塞推送，附带背压信号
func (c *Client) Send(msg []byte) bool {
  select {
  case c.writeCh <- msg:
    return true
  case <-c.closeCh:
    return false
  default:
    slog.Warn("client write queue full, dropping message",
      "client_id", c.ID)
    return false
  }
}`,
        codeLanguage: 'go',
      },
      {
        heading: 'SessionChan：非阻塞扇出到多个运营人员',
        body: `单个访客会话可以被多个运营人员同时观看。当访客生成一个事件时，它需要被广播到每个订阅了该会话的运营人员。\n\nSessionChan 是扇出机制。它维护一个订阅者通道的映射——每个运营人员一个，每个通道有 64 条消息的缓冲。当 publish() 被调用时，它遍历所有订阅者并将消息推送到每个通道的缓冲中：\n\nfor each subscriber:\n  select { case ch <- msg: default: drop }\n\n非阻塞的 default 意味着一个慢速的运营人员（例如，在一个拥塞的网络中）不会降低观看同一会话的其他运营人员的体验。慢速运营人员的通道会填满并开始丢弃，但其他每个人继续接收每个事件。\n\n相同的模式用于 TenantRoom，它将存在事件（访客上线/离线）广播给租户中的所有运营人员。存在事件是低频的，因此丢弃很少见——但非阻塞模式在整个代码库中是一致的。\n\n当运营人员取消订阅一个会话时，订阅者通道从映射中移除并关闭。关闭信号传播到运营人员的客户端写入循环，停止从该会话转发事件。`,
      },
      {
        heading: '事件管道：从浏览器到 MinIO 的三个阶段',
        body: `事件不仅需要广播——它们还需要被存储以供回放。管道有三个阶段：\n\n阶段 1：WebSocket 接收。访客 SDK 以批处理方式发送事件（100ms 或 50 个事件，以先到者为准）。每个批处理是一个 msgpack 编码的 rrweb 事件数组。服务器的 visitorWS goroutine 读取消息并：\n\n1. 追加到 Redis Stream（stream:session:{uuid}）以获得持久性\n2. 提取全快照和元事件以进行缓存\n3. 发布到 hub 以实时传递给运营人员\n4. 更新 PostgreSQL 中会话的 last_event_at 时间戳\n5. 传递给行为分析跟踪器\n\n阶段 2：Redis Stream 刷盘。一个后台 goroutine（每个会话一个）每 30 秒或每 1000 个事件运行一次，以先到者为准。它：\n\n1. 通过 XRANGE 从 Redis Stream 读取未刷盘的事件\n2. 将它们聚合成单个 msgpack blob\n3. 用 zstd 压缩\n4. 写入 MinIO PutObject\n5. 在 PostgreSQL event_blobs 中插入一条记录（session_id、minio_object_key、offset、size）\n6. 裁剪 Redis Stream 仅保留最后 200 条记录\n\n阶段 3：垃圾回收。一个每小时的 GC worker 删除已结束且超出保留期（默认 30 天）的会话的事件数据和元数据。\n\nRedis Stream 充当热路径（实时 WebSocket 接收）和冷路径（对象存储刷盘）之间的缓冲。如果 MinIO 暂时不可用，事件会在 Redis 中累积（每个会话最多 200 条未刷盘条目）。当 MinIO 恢复时，刷盘器会追赶上来。`,
        code: `// 刷盘器：Redis Stream → MinIO 带补偿事务
func flushSession(ctx context.Context, s *Session) error {
  // 1. 从 Redis Stream 读取未刷盘条目
  entries, err := s.stream.XRange(ctx, "-", "+")
  if err != nil || len(entries) == 0 {
    return err
  }

  // 2. 聚合和压缩
  blob := encodeBlob(entries)
  compressed := zstdCompress(blob)

  // 3. 先写入 MinIO
  objectKey := fmt.Sprintf("sessions/%s/events-%d.zst",
    s.sessionID, s.nextOffset)
  err = s.minio.PutObject(ctx, bucket, objectKey, compressed)
  if err != nil {
    return err  // MinIO 写入失败，下个 tick 重试
  }

  // 4. 记录到 PostgreSQL（补偿：如果 PG 失败则删除 MinIO 对象）
  err = s.db.Exec(ctx,
    "INSERT INTO event_blobs (session_id, minio_object_key, ...) VALUES ($1, $2, ...)",
    s.sessionID, objectKey)
  if err != nil {
    s.minio.RemoveObject(ctx, bucket, objectKey)  // 清理孤儿
    return err
  }

  // 5. 裁剪 Redis Stream
  s.stream.XTrim(ctx, 200)
  return nil
}`,
        codeLanguage: 'go',
      },
      {
        heading: '快照缓存：让管理面板感觉瞬间加载',
        body: `当运营人员打开一个会话进行实时监控时，他们不应该在等待下一个 rrweb 增量事件时看到空白画布。他们应该立即看到页面——就像访客加载它时那样。\n\n为了实现这一点，我们在 Redis 中缓存了两个关键事件：\n\n• 元事件（rrweb type 4）：包含视口尺寸。没有这个，rrweb 播放器不知道 iframe 应该有多大。\n• 全快照（rrweb type 2）：DOM 树的完整序列化。这是播放器渲染以展示初始页面状态的内容。\n\n两者都有 30 分钟的 TTL 缓存。元事件在每个会话中设置一次（在第一个元事件时）。全快照在 rrweb 每次拍摄新的全快照时更新（每 10 秒或页面可见性变化时）。\n\n当运营人员订阅一个会话时，服务器先发送元事件，然后发送全快照，然后流式传输增量事件。rrweb 播放器按顺序接收这些，在任何运营人员看到之前渲染初始页面。体验与观看实时页面无法区分——没有加载动画，没有"缓冲中"指示符。\n\n30 分钟的缓存 TTL 是基于典型会话时长分析选择的。大多数会话在 30 分钟以下。对于更长的会话（例如运营人员观看一个支持交互），下一个定期的全快照会刷新缓存。`,
      },
      {
        heading: 'Presence：谁在线，谁在观看',
        body: `运营人员需要知道哪些访客当前在网站上，以及哪些运营人员在观看着他们。这就是存在（presence）系统。\n\n当访客连接时，访客 WebSocket 处理器调用 Hub.VisitorOnline()，它：\n\n1. 为该会话创建一个 SessionChan\n2. 在 visitorClients 中记录访客客户端 ID\n3. 向 TenantRoom 广播 presence.online 消息\n\nTenantRoom 接收该消息并扇出到该租户中的所有已连接运营人员。每个运营人员的管理面板更新其 UI，在实时流中显示新的访客。\n\n当访客断开连接（WebSocket 关闭或会话超时）时，VisitorOffline() 广播 presence.offline 并清理会话映射。\n\n运营人员也在连接/断开时广播自己的存在——这样访客的共浏览横幅可以显示运营人员的名字，其他运营人员也知道谁在协助哪个会话。`,
      },
      {
        heading: '每一层的背压处理',
        body: `在一个有 500 条并发数据流的系统中，唯一的保证是某些消费者会比其他的慢。网络拥塞、后台标签节流和浏览器渲染延迟都会导致运营人员落后。\n\nPinConsole 在三个层面处理背压：\n\n1. Client writeCh（256 缓冲）：当运营人员的 WebSocket 连接很慢时，其 Client 的写入缓冲会填满。多余的消息被丢弃并记录警告。运营人员会错过一些事件，但会话对其他人继续。\n\n2. SessionChan 订阅者通道（64 缓冲）：当运营人员的订阅者通道填满时，路由器在消息到达 Client 之前丢弃它们。这是第二层背压——第一个丢弃消息的警告在 SessionChan 层面，而非 Client 层面。\n\n3. Redis Stream 裁剪（200 条目）：每次刷盘后 Redis Stream 被裁剪以仅保留 200 条目。如果刷盘器落后（MinIO 慢），流不会无限增长。最旧的条目被丢弃。\n\n所有三个层面使用相同的模式：当缓冲满时丢弃最旧的数据。对于会话回放，这是可以接受的——丢失一些增量事件意味着运营人员在回放中看到一个小跳跃，而不是一个大空洞。全快照确保页面状态始终正确，即使一些增量丢失了。\n\n我们用一个并发压力测试验证了这一点（50 个 goroutine 向同一会话发送，启用 -race）。Hub 不会死锁、不会崩溃、不会在压力下泄露 goroutine。`,
      },
      {
        heading: '为什么不是 WebRTC 或 P2P？',
        body: `一个常见的问题是：为什么不使用 WebRTC 或 P2P 作为访客到运营人员的数据路径，而是通过中心服务器路由所有流量？\n\nWebRTC 可以减少服务器带宽——访客直接将事件发送到运营人员的浏览器。但它引入了复杂性：STUN/TURN 服务器设置、NAT 穿越处理、浏览器兼容性问题，以及一个独立的信令通道。\n\n更重要的是，WebRTC 对会话录制没有帮助。事件仍然需要到达服务器进行存储。使用 hub-and-spoke 模型，服务器一次接收事件并同时处理存储和转发。使用 P2P，服务器需要接收每个事件的副本（违背了 P2P 的目的）或会话结束后从运营人员处请求事件（增加了复杂性）。\n\n对于一个自托管工具来说，简洁性是一种特性。Hub-and-spoke 模型在推理、调试和部署方面极其简单。瓶颈不是服务器带宽——而是存储 I/O。这由异步刷盘管道处理，而非 WebSocket hub。`,
      },
      {
        heading: '从 500 到 5000：扩展路径',
        body: `PinConsole 以每个房间 500 个并发连接作为基线目标。但架构可以通过直接的变更进一步扩展：\n\n• 增加 Redis Stream 裁剪限制（TrimKeep 200 → 500）：允许更大的背压缓冲\n• 增加 Client writeCh 缓冲（256 → 1024）：用内存换取更少的消息丢弃\n• 增加 Redis 连接池（PoolSize 50 → 200）：处理更多的并发 XADD 操作\n• 增加 PostgreSQL 连接池（MaxConns 25 → 100）：处理更多的会话元数据查询\n\n所有这些都是配置变更，而非代码变更。Hub 本身没有可调的限制——它只受可用内存和 goroutine 调度器吞吐量的限制。\n\n对于需要超过 5000 个并发连接的部署，进程内 hub 会成为瓶颈。在那个规模下，你需要将流量拆分到多个服务器实例上，前面放置一个支持 WebSocket 感知的负载均衡器，并用 Redis Pub/Sub 桥接各个 hub。那是 v2 的问题。对于一个服务单个租户的自托管工具，每个房间 500 个连接几乎覆盖所有真实世界的用例。`,
      },
    ],
  },
  relatedPosts: [
    { title: '如何构建一个自托管的 FullStory 替代品', url: '/blog/self-hosted-fullstory-alternative/', description: 'PinConsole 完整架构。' },
    { title: '开源共浏览实现', url: '/blog/building-co-browsing-zh/', description: '双向共浏览的实现细节。' },
  ],
  cta: {
    title: '体验实时会话监控',
    subtitle: '每个房间 500 条并发连接。自托管。单一二进制。',
    primary: { label: '在 GitHub 上开始', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: '联系维护者', href: '#consult' },
  },
};
