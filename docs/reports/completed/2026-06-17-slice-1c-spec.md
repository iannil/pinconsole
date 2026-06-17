# 切片 1c 规格说明（rrweb 接入：全量采集 + 实时回放）

**状态**：in_progress（规格已定，实施中）
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：本文件
**关联**：[`PLAN.md`](../../PLAN.md) §7、[`docs/project-status.md`](../project-status.md) §5
**前置**：[切片 1b 完成](./2026-06-17-slice-1b-implementation.md)

## Context

切片 1b 实现了"事件 → WS → admin 抽象面板"管道，admin 看到的是坐标圈点 + 事件列表，看不到访客真实页面。切片 1c 把 SDK 采集从 4 类手写 collector 换成 rrweb v2 alpha 全量采集，admin 用 rrweb-player 动态加载做实时回放。这是 v1 第一次让运营看到访客"真实页面"。

## 范围（acceptance criteria）

- [ ] SDK 启动时初始化 rrweb record()，事件批量（100ms / 50 events）通过 WS 上行
- [ ] SDK 检测页面含 canvas/WebGL/跨域 iframe 时启动选择性截图（1fps WebP q70）
- [ ] SDK 韧性：rrweb 错误时 try/catch + 3 次重试重启，page hidden >60s 主动 takeFullSnapshot
- [ ] 输入脱敏：password/credit card 强制 mask，其他 input 默认 mask，form_submit 仍发送明文（PLAN.md §提交前按键已锁定，但 form_submit 是提交后，明文合理）
- [ ] 同源 iframe 采集，跨域 iframe 显示占位框
- [ ] 服务端缓存每个 session 最近 full snapshot（Redis `snapshot:session:{id}`，TTL 5 分钟）
- [ ] admin 订阅 session 时，服务端先发缓存 snapshot 再广播增量
- [ ] admin 动态 import rrweb-player（点击访客后才加载，不进首屏 bundle）
- [ ] ReplayPlayer 用 rrweb-player + append，新事件实时进入回放
- [ ] 端到端验证：访客访问 landing → admin 列表 → 点击 → 看到 rrweb-player 实时回放（不是抽象圈点）
- [ ] 订阅延迟 < 1s：admin 订阅后 1 秒内能看到访客当前页面状态
- [ ] 脱敏验证：访客 input 输入 "test"，admin 看到的不是明文 "test"
- [ ] Go 单元测试 + Playwright e2e 全部通过

## 16 项锁定决策

| # | 维度 | 选择 |
|---|---|---|
| 1 | rrweb 版本 | v2 alpha（已装 `rrweb@2.0.0-alpha.18`） |
| 2 | 与 1b collectors | 完全替换（删 4 个，rrweb 覆盖全部） |
| 3 | Full snapshot | 初始 1 次 + 每 30s 或 50 incremental 触发周期性 full |
| 4 | rrweb 包装 | 复用 envelope：`{type:'event', payload:{type, ts, rrweb:{type, data}}}` |
| 5 | Admin 实时回放 | rrweb-player + append，逐条 append |
| 6 | 订阅初始状态 | 服务端 Redis 缓存最近 full snapshot |
| 7 | 输入脱敏 | rrweb `maskAllInputs + maskInputOptions`（password/card 强制） |
| 8 | iframe | 同源采集，跨域占位框 |
| 9 | 移动事件 | rrweb 默认含触摸事件 |
| 10 | 选择性截图 | SDK 检测到 canvas/WebGL/跨域 iframe 才启动 |
| 11 | 事件批量 | 100ms / 50 events 阈值，msgpack array 单 envelope |
| 12 | admin 多访客 | 保持 1b 两栏布局，panel 换为 rrweb-player |
| 13 | blob 格式 | event_blobs 仍存 envelope array，schema 不变 |
| 14 | SDK 韧性 | 错误捕获 + 3 次重试 + visibility 检测 |
| 15 | admin 包优化 | 动态 import rrweb-player（点击访客后才加载） |
| 16 | 1c 验收 | 4 项：端到端实时回放 / 订阅<1s 可见 / 脱敏验证 / snapshot 传输 |

## 涉及的代码改动

### server/

新建：
- `internal/recording/snapshot.go`：Redis snapshot 缓存读写（key、TTL、序列化）

修改：
- `internal/proto/events.go`：扩展 EventPayload 加 `RRWeb *RRWebEvent` 字段
- `internal/hub/hub.go`：subscribe 时由 ws.go 显式调用 snapshot 推送（hub 不直接耦合 Redis）
- `internal/api/ws.go`：visitor 收到 full snapshot 类型时写 Redis；operator subscribe 时先发 snapshot
- `internal/storage/redis.go`：暴露通用 `Set/Get(ctx, key, value, ttl)`

### visitor-sdk/

删除：
- `src/collectors/{mouse,scroll,form}.ts`（4 类抽象 collector）

新建：
- `src/collectors/rrweb.ts`：rrweb record() 包装 + 韧性（错误重试、visibility 触发 full）
- `src/collectors/screenshot.ts`：canvas/WebGL/iframe 检测 + 选择性截图
- `src/batch.ts`：100ms / 50 events 批量器

修改：
- `src/proto/events.ts`：加 RRWeb 事件类型
- `src/index.ts`：编排 rrweb + screenshot + batch
- `src/transport/ws.ts`：暴露 `sendBatch(events)` API

### admin/

新建：
- `src/components/ReplayPlayer.vue`：rrweb-player 封装（动态 import + append）

修改：
- `src/proto/events.ts`：加 RRWeb 事件类型（与 SDK 同步）
- `src/components/VisitorPanel.vue`：替换 SVG 鼠标圈为 ReplayPlayer
- `src/stores/visitors.ts`：事件按 rrweb 类型分发；保留最近 full snapshot 用以 ReplayPlayer 初始化

### e2e/

修改：
- `tests/realtime.spec.ts`：加 1c 验收场景（端到端 rrweb 回放、订阅延迟、脱敏、snapshot 传输）

## 协议扩展（伪 schema）

```typescript
// 1b 的 EventPayload + 1c 的 RRWeb
interface EventPayload {
  type: EventType;          // 'rrweb' (新) | 1b 残留枚举值
  ts: number;
  rrweb?: RRWebEvent;       // 新增
  // 1b 字段保留以便兼容
  mouse_move?: MouseMoveData;
  click?: ClickData;
  scroll?: ScrollData;
  form_submit?: FormSubmitData;
}

interface RRWebEvent {
  type: number;             // rrweb 事件类型枚举（FullSnapshot=2, IncrementalSnapshot=3, Meta=4）
  data: unknown;            // rrweb 原生 data（不类型化，rrweb-snapshot 已定义）
  timestamp: number;
}
```

## 估时

- **Solo 全职**：约 8-10 天
  - Day 1：proto 扩展 + Redis KV 包装 + server snapshot 缓存
  - Day 2：hub subscribe 时推送 snapshot
  - Day 3-4：SDK rrweb collector + 韧性 + 批量
  - Day 5：SDK 选择性截图
  - Day 6-7：admin ReplayPlayer + 动态 import + 接入 store
  - Day 8：e2e 4 场景
  - Day 9-10：buffer（性能调优、脱敏边界、文档）
- **Solo 业余**：约 3-4 周

## Next

实施完切片 1c 后：

1. 写 `docs/reports/completed/{date}-slice-1c-implementation.md`
2. 更新 `docs/project-status.md` §5 切片状态表（1c → completed）
3. 启动切片 1d（录像归档：基于 1c 的 rrweb 事件做完整会话回放）

## Blockers

无。1b 端到端实时管道已就绪，1c 是采集层 + admin UI 层的替换。

## Notes

- 1c 不实现 co-browsing（1e 起），admin 仍只能"看"不能"控制"
- 1c 不实现录像回放（1d 起），admin 实时回放但会话结束无法回看
- 1c 不引入新表，所有 rrweb 事件仍写入 1b 的 Redis Stream + event_blobs
- ReplayPlayer 用动态 import，admin 首屏 bundle 不增加
- 选择性截图仅在 SDK 检测到 canvas/WebGL/iframe 时启动，默认 landing 页无此内容时不发截图
