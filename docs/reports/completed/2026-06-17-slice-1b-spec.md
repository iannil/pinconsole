# 切片 1b 规格说明（单向最小：SDK → WS → admin）

**状态**：in_progress（规格已定，实施待启动）
**开始**：2026-06-17
**完成**：（未完成）
**规格来源**：本文件
**关联**：[`PLAN.md`](../../PLAN.md) §7、[`docs/project-status.md`](../project-status.md) §5
**前置**：[切片 1a 完成](./2026-06-17-slice-1a-implementation.md)

## Context

切片 1a 落地了仓库骨架，三 app 都能 build/run/embed，但无业务逻辑。切片 1b 是 v1 第一个真正的实时管道：访客 SDK 采集事件 → WebSocket → 后端 hub 路由 → admin 实时显示 + 录像归档。

这是 PLAN.md §7 中 1b 的具体落地。验证后端 hub 架构、WS 握手协议、MessagePack envelope、Redis Stream → MinIO 归档链路。

## 范围（acceptance criteria）

实施完成的判定标准：

- [ ] 访客访问 `http://localhost:8080/` → SDK 自动连接 `/ws/visitor`，握手成功
- [ ] SDK 采集鼠标移动（rAF 节流 60fps）+ 点击 + 滚动 + 表单 onSubmit，通过 WS 上行
- [ ] admin 在 `http://localhost:8080/admin` 看到该访客出现在左侧访客列表
- [ ] 点击列表中的访客 → admin 显式 `subscribe session:<id>` → 右侧实时面板显示鼠标圈跟随、点击闪烁、滚动指示、表单提交列表
- [ ] 同时 10 个 SDK 连接，admin 能逐个订阅/取消订阅，带宽与 CPU 可控
- [ ] 后端中途重启 → SDK 自动指数退避重连 → session_id 复用 → 重连期间事件缓冲后不丢
- [ ] 会话结束后 30s 或 1000 events 触发快照 → MinIO 有 msgpack blob 文件 → PG `event_blobs` 表有对应记录 + checksum 匹配
- [ ] Go testcontainers 集成测试覆盖：握手、订阅、广播、重连、快照触发
- [ ] Playwright e2e 覆盖：访客访问 → admin 看到列表 → 订阅 → 看到鼠标轨迹
- [ ] `make lint` 与 `make test` 通过
- [ ] GitHub Actions CI 通过

## 16 项锁定决策

| # | 维度 | 选择 | 备注 |
|---|---|---|---|
| 1 | DB 库 | pgx + sqlc | pgx v5 底层 + sqlc 生成类型安全查询 |
| 2 | WS 端点 | 分离：`GET /ws/visitor` + `GET /ws/operator`；初始 hello 消息 | 携带 capabilities/SDK 版本 |
| 3 | 消息封装 | MessagePack 二进制 envelope | `{v, type, session_id, trace_id, payload, ts}` |
| 4 | 1b 事件类型 | mouse_move + click + scroll + form_submit | 1c 起加 rrweb 全量 |
| 5 | 事件 schema | Discriminated union | `{type, ts, data}`，data 按 type 分支 |
| 6 | 事件存储 | Redis Stream（hot）+ MinIO（cold msgpack）+ PG 元数据 | 与 PLAN.md 一致 |
| 7 | Hub 路由 | Visitor-only 通道 + admin 显式 `subscribe session:<id>` | 避免广播风暴 |
| 8 | Admin UI | 两栏：列表（左）+ 实时面板（右） | 不渲染访客页面，验证数据管道 |
| 9 | 访客身份 | `visitor_id` (localStorage) + `session_id` (server-issued via `POST /api/session/init`) | 30 分钟无活动开新 session |
| 10 | WS 测试 | Go testcontainers + Playwright | 单语言集成测试 + 端到端 |
| 11 | 鼠标节流 | requestAnimationFrame，最大 60fps | 带宽可控 |
| 12 | PG schema | 3 表：`visitors` + `sessions` + `event_blobs`，`tenant_id` 预留 | 与"single-tenant + schema 预留"一致 |
| 13 | Stream 快照 | Per-session Stream；1000 events 或 30s 触发；flush msgpack blob 到 MinIO + 写 event_blobs | 滑动窗口保留最近 N 事件供实时订阅 |
| 14 | Admin WS 客户端 | Pinia store + `useWs` composable | Vue 3 习惯、类型安全 |
| 15 | 访客列表 | Hub 内存索引 + REST 初始 + WS 增量广播到 `room:tenant:<id>` | 不依赖 PG 轮询 |
| 16 | SDK 重连 | 指数退避 1→2→4→…→30s；本地 buffer 上限 1000；session_id 复用 | 核心韧性 |

## 涉及的代码改动（按文件清单）

### server/

新增：

- `internal/hub/hub.go` — Hub 单例，管理 visitor 通道与 operator 订阅
- `internal/hub/room.go` — 房间模型（per-session channel + per-tenant room）
- `internal/hub/client.go` — 单个 WS 连接封装（visitor 与 operator 共用，role 区分）
- `internal/api/ws.go` — `/ws/visitor` 与 `/ws/operator` HTTP→WS upgrade handler
- `internal/api/session.go` — `POST /api/session/init`（签发 session_id）+ `GET /api/sessions?status=active`（初始列表）
- `internal/recording/stream.go` — Redis Stream 写入（XADD per session）
- `internal/recording/flusher.go` — 后台 worker，按 1000 events / 30s 触发，XREAD 消费 + MinIO PutObject + PG event_blobs 插入
- `internal/storage/queries.sql` — sqlc 输入 SQL
- `internal/storage/queries_gen.go` — sqlc 生成（自动）
- `internal/storage/models_gen.go` — sqlc 生成（自动）
- `internal/proto/envelope.go` — MessagePack envelope Go 类型 + 编解码
- `internal/proto/events.go` — Discriminated union 事件类型（mouse_move / click / scroll / form_submit）
- `migrations/000001_init.up.sql` + `000001_init.down.sql` — 3 表 schema

修改：

- `cmd/server/main.go` — 装配 hub、recording flusher，注入 ws handler
- `internal/api/router.go` — 注册 `/ws/*` 与 `/api/session/*` 路由
- `internal/config/config.go` — 加 `SessionTimeoutMinutes`、`StreamFlushEventThreshold`、`StreamFlushIntervalSeconds`、`ReconnectMaxBackoffSeconds` 等可调参数
- `internal/storage/redis.go` — 暴露 Stream 操作辅助方法
- `go.mod` — 加 `coder/websocket`、`github.com/vmihailenco/msgpack/v2`、`github.com/testcontainers/testcontainers-go`

### admin/

新增：

- `src/proto/envelope.ts` — MessagePack envelope TS 类型
- `src/proto/events.ts` — Discriminated union 事件类型
- `src/stores/visitors.ts` — Pinia store（访客列表、订阅状态、连接状态）
- `src/composables/useWs.ts` — WS 客户端封装（connect / reconnect / subscribe / disconnect）
- `src/views/Dashboard.vue` — 两栏布局入口
- `src/components/VisitorList.vue` — 左侧访客列表（含状态、IP、开始时间）
- `src/components/VisitorPanel.vue` — 右侧实时面板（鼠标圈 SVG + 事件列表）
- `src/router/index.ts` — Vue Router 配置（`/` → Dashboard）

修改：

- `src/main.ts` — 装配 Vue Router
- `package.json` — 加 `@msgpack/msgpack`、`vue-router`

### visitor-sdk/

新增：

- `src/proto/envelope.ts` — MessagePack envelope TS 类型（与 admin 共享结构）
- `src/proto/events.ts` — Discriminated union 事件类型（与 admin 共享）
- `src/transport/ws.ts` — WS 客户端 + 指数退避重连 + 本地 buffer
- `src/collectors/mouse.ts` — mouse_move（rAF 节流）+ click 采集
- `src/collectors/scroll.ts` — scroll 采集
- `src/collectors/form.ts` — form_submit 采集（onSubmit）
- `src/session.ts` — visitor_id 持久化 + session_id 获取（POST /api/session/init）

修改：

- `src/index.ts` — 编排所有 collectors + transport，自动启动
- `src/config.ts` — 加 `endpoint`、`visitorIdKey`、`bufferMaxEvents`、`reconnectMaxBackoffMs` 等参数
- `package.json` — 加 `@msgpack/msgpack`

### e2e/

新增：

- `tests/realtime.spec.ts` — 4 个验收场景的 Playwright 测试

### docs/

新增（实施期间追加）：

- `progress/{date}-slice-1b-implementation.md`
- `reports/completed/{date}-slice-1b-implementation.md`

### 共享协议定义

由于 Go / TS 两端需要 envelope 与 event schema 同源，建议：

- 在 `docs/standards/proto-spec.md` 写一份**协议规格文档**（人类可读）
- Go 与 TS 各自从该规格生成代码（Go 手写 + TS 手写，1b 期间不引入 codegen）
- 未来若协议稳定，引入 quicktype / buf 等 codegen 工具

## 消息流（时序）

```
访客侧                                  后端                                  运营侧
─────                                  ────                                  ────
[SDK load]
   │
   ├──POST /api/session/init──────────▶
   │                                     │
   │                                     ├──(check visitor_id in PG?)
   │                                     ├──(create/lookup visitor)
   │                                     ├──(create session)
   │                                     ├──(register in hub memory)
   │◀──────────{session_id, visitor_id}──┤
   │                                     │
   ├──WS connect /ws/visitor────────────▶
   │                                     │
   │──hello {visitor_id, session_id,────▶│
   │        sdk_version, capabilities}   │
   │                                     │──broadcast to room:tenant:<tid>
   │                                     │   {type: visitor_online, session_id}
   │                                     │                                     │
   │                                     │                                     │◀──operator 已订阅 room:tenant
   │                                     │                                     │   (列表新增 visitor)
   │◀───────────────{ack: hello}─────────┤                                     │
   │                                     │                                     │
   ├──event mouse_move──────────────────▶│                                     │
   │   (msgpack envelope)                │                                     │
   │                                     │──XADD stream:session:<sid>          │
   │                                     │──broadcast to session:<sid>         │
   │                                     │                                     │   (仅当 operator subscribe)
   │                                     │                                     │◀──{event: mouse_move}
   │                                     │                                     │   (实时面板更新)
   │   ...更多事件...                     │                                     │
   │                                     │                                     │
   │                                  [1000 events 或 30s]                   │
   │                                     │──flush msgpack blob to MinIO       │
   │                                     │──INSERT event_blobs in PG          │
   │                                     │──XTRIM stream:session:<sid>        │
   │                                     │                                     │
[页面关闭]                                                                       │
   │──WS close                           │                                     │
   │                                     │──broadcast visitor_offline          │
   │                                     │   to room:tenant:<tid>              │
   │                                     │                                     │   (列表移除 visitor)
```

## DB Schema（详细）

### visitors

| 字段 | 类型 | 说明 |
|---|---|---|
| id | UUID PK | server 生成 |
| tenant_id | UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000' | 预留，v1 不启用 |
| fingerprint | TEXT NOT NULL | SDK 生成的 visitor_id（localStorage） |
| ua | TEXT | User-Agent |
| ip_first_seen | INET | 首次见到的 IP |
| first_seen_at | TIMESTAMPTZ NOT NULL DEFAULT NOW() | |
| last_seen_at | TIMESTAMPTZ NOT NULL DEFAULT NOW() | |
| meta | JSONB DEFAULT '{}' | 灵活附加字段 |

UNIQUE(tenant_id, fingerprint) — 同租户下 fingerprint 唯一

### sessions

| 字段 | 类型 | 说明 |
|---|---|---|
| id | UUID PK | server 生成（即 SDK 持有的 session_id） |
| tenant_id | UUID NOT NULL DEFAULT '00000000-...' | 预留 |
| visitor_id | UUID NOT NULL REFERENCES visitors(id) | |
| started_at | TIMESTAMPTZ NOT NULL DEFAULT NOW() | |
| last_event_at | TIMESTAMPTZ | 最后事件时间（用于活跃判定） |
| ended_at | TIMESTAMPTZ | NULL = 进行中 |
| status | TEXT NOT NULL DEFAULT 'active' | active / ended / timed_out |
| event_count | INTEGER NOT NULL DEFAULT 0 | 累计事件数 |
| ua | TEXT | |
| ip | INET | |

INDEX(tenant_id, status, last_event_at DESC) — 列表查询用

### event_blobs

| 字段 | 类型 | 说明 |
|---|---|---|
| id | UUID PK | |
| session_id | UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE | |
| tenant_id | UUID NOT NULL DEFAULT '00000000-...' | 预留，便于按租户清理 |
| blob_index | INTEGER NOT NULL | 该 session 内的第几个 blob（0-based） |
| started_at | TIMESTAMPTZ NOT NULL | blob 含事件的最早时间 |
| ended_at | TIMESTAMPTZ NOT NULL | blob 含事件的最晚时间 |
| event_count | INTEGER NOT NULL | |
| minio_object_key | TEXT NOT NULL | 如 `sessions/<sid>/<index>.msgpack.zst` |
| size_bytes | BIGINT NOT NULL | |
| checksum_sha256 | TEXT NOT NULL | |
| created_at | TIMESTAMPTZ NOT NULL DEFAULT NOW() | |

UNIQUE(session_id, blob_index)
INDEX(session_id, blob_index)

## MessagePack Envelope（伪 schema）

```typescript
interface Envelope {
  v: 1;                  // 协议版本
  type: MessageType;     // 'hello' | 'ack' | 'event' | 'command' | 'subscribe' | 'unsubscribe' | 'presence' | 'error'
  session_id?: string;   // UUID，按消息类型可选
  trace_id?: string;     // 用于日志关联
  ts: number;            // 毫秒时间戳
  payload?: unknown;     // 按 type 分支
}

type EventPayload =
  | { type: 'mouse_move'; x: number; y: number }
  | { type: 'click'; x: number; y: number; button: number; target_selector?: string }
  | { type: 'scroll'; x: number; y: number }
  | { type: 'form_submit'; form_id?: string; fields: Record<string, string> };
```

## 配置参数（新增到 .env.example）

```bash
# Session
SESSION_TIMEOUT_MINUTES=30           # 无活动超时
SESSION_GC_INTERVAL_SECONDS=60        # GC 扫描间隔

# Redis Stream flush
STREAM_FLUSH_EVENT_THRESHOLD=1000    # 触发快照的事件数
STREAM_FLUSH_INTERVAL_SECONDS=30      # 触发快照的间隔
STREAM_TRIM_KEEP_EVENTS=200           # flush 后保留多少事件供实时订阅

# WebSocket
WS_MAX_MESSAGE_BYTES=1048576          # 1 MiB
WS_PING_INTERVAL_SECONDS=15
WS_WRITE_TIMEOUT_SECONDS=5

# Admin room
ADMIN_ROOM_NAME=tenant                # room:tenant:<id> 前缀
```

## SDK 协议能力协商（hello 消息）

```typescript
interface HelloPayload {
  visitor_id: string;        // 来自 localStorage
  session_id: string;        // 来自 POST /api/session/init
  sdk_version: string;
  capabilities: {
    events: ('mouse_move' | 'click' | 'scroll' | 'form_submit')[];
    co_browsing: false;     // 1b 不支持，1e+ 起
    recording: true;
  };
}
```

后端回复 ack 后才正式加入房间。

## 验收 4 场景的具体步骤

### 1. 端到端实时跟随

```bash
# 终端 A
docker compose up -d
make dev

# 终端 B（启 release server 用于 e2e）
make build
./server/bin/server &

# 浏览器 A：访问 http://localhost:8080/  → SDK 加载
# 浏览器 B：访问 http://localhost:8080/admin → 看到访客
#           点击访客 → 右侧面板出现 → 鼠标在 A 移动 → B 看到圆点跟随
```

### 2. 10 访客并发

通过 Playwright 起多个 context 访问 landing，admin 应能稳定显示 10 个访客，逐个订阅/取消订阅，admin 内存与 CPU 增长可控（无暴涨）。

### 3. SDK 重连

后端 SIGTERM → SDK 立即检测断开 → 指数退避重连 → 后端重启完成 → SDK 重连成功 → session_id 复用 → buffer 中事件被发出 → admin 看到事件流恢复。

### 4. MinIO 录像快照

访客持续触发事件 → 后端达到 1000 events 或 30s → 触发 flush → 检查：
- `mc ls local/marketing-monitor/sessions/<sid>/` 有 blob 文件
- `SELECT * FROM event_blobs WHERE session_id = '...'` 有记录
- `sha256sum <blob>` 与 `event_blobs.checksum_sha256` 一致

## 估时

- **Solo 全职**：约 14-17 天（3 周）
  - Day 1-2：PG schema + sqlc 配置 + 3 表 migration
  - Day 3-5：hub + WS handler + MessagePack envelope（Go 侧）
  - Day 6-7：SDK 采集器 + transport + 重连
  - Day 8-9：admin Pinia + useWs + 两栏 UI
  - Day 10-11：Redis Stream + MinIO flusher + event_blobs 写入
  - Day 12-13：testcontainers 集成测试
  - Day 14：Playwright e2e + 验收 4 场景
  - Day 15-17：buffer（性能调优、文档更新、CI 调通）
- **Solo 业余**（10-15h/week）：约 5-7 周

## Next

实施完切片 1b 后：

1. 写 `docs/progress/{date}-slice-1b-implementation.md`（按 [progress 模板](../templates/progress.md)）
2. 写 `docs/reports/completed/{date}-slice-1b-implementation.md`（按 [report 模板](../templates/report.md)）
3. 更新 `docs/project-status.md` §5 切片状态表（1b → completed，1c → in_progress）
4. 更新 `memory/daily/{date}.md`
5. 启动切片 1c（rrweb 接入）

## Blockers

无。所有架构决策已锁定，1a 骨架已就绪。

## Notes

- 1b 不集成 rrweb（1c 起加），所以访客"看到"的是抽象事件（坐标、点击位置），不是真实页面回放。这是有意的——1b 验证管道，1c 验证可视化。
- 1b 不实现 co-browsing 双向通道（1e 起）；admin 只能"看"，不能"控制"。
- 1b 不做认证（1h 起）；admin Web 端任何人可访问，仅本机/dev 环境。
- 与 1a 一样，所有偏差在此文件"与规格的偏差"小节记录（实施时追加）。
