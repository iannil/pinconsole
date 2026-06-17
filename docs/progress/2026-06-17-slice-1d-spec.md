# 切片 1d 规格说明（录像归档 + 历史回放）

**状态**：in_progress（规格已定，实施中）
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：本文件
**关联**：[`PLAN.md`](../../PLAN.md) §7、[`docs/project-status.md`](../project-status.md) §5
**前置**：[切片 1c 完成](./2026-06-17-slice-1c-implementation.md)

## Context

切片 1c 让 admin 实时回放活跃访客的页面（live）。切片 1d 让 admin 从历史角度回看已结束的会话（historical）。访客关闭页面 → session end → 服务端同步 flush → MinIO 永久归档 → admin 在 `/replay` 列表点开会话 → 完整 rrweb 回放。

## 范围（acceptance criteria）

- [ ] 服务端 GC worker 每小时清理超过 30 天的 event_blobs（删 MinIO + 删 PG）
- [ ] visitor WS 断开时同步 flush 最后一批事件
- [ ] `GET /api/sessions?status=ended&since=24h` 返回已结束会话列表
- [ ] `GET /api/sessions/:id/replay?offset=0&limit=10000` 返回完整事件流
- [ ] admin 路由：`/dashboard`（1c live）+ `/replay`（1d 列表）+ `/replay/:session_id`（1d 回放）
- [ ] 历史会话列表含时间范围筛选（24h / 7d / 30d）
- [ ] 回放页用 rrweb-player 原生控制器（播放/暂停/进度/倍速）
- [ ] admin 用 Web Worker 解码 msgpack（避免 100K+ 事件阻塞 UI）
- [ ] 端到端：live 转 historical 流程
- [ ] 短 session 即时 replay（< 30s 也立即 replayable）
- [ ] 长 session 分页 replay（1000+ 事件分页拉取正确）
- [ ] 回放控制器交互（暂停、播放、倍速、进度拖拽）
- [ ] Go 单元测试 + Playwright e2e 全部通过

## 13 项锁定决策

| # | 维度 | 选择 |
|---|---|---|
| 1 | Replay 时序 | 活跃=live（1c），结束=historical（1d） |
| 2 | Replay 数据源 | 服务端拼装，单响应 |
| 3 | 长 session | 分页拉取（offset + limit，默认 10000） |
| 4 | Admin 路由 | `/dashboard` + `/replay` + `/replay/:session_id` |
| 5 | 列表过滤 | 时间范围预设（24h / 7d / 30d） |
| 6 | 回放控制器 | rrweb-player 原生 |
| 7 | Session 结束 | WS 断开立即 end + flusher 下个 tick |
| 8 | Replay 鉴权 | 1d 不做认证（PLAN.md 1h 起） |
| 9 | Blob GC | 服务端 GC worker（每小时） |
| 10 | Admin 性能 | Web Worker 解码 |
| 11 | End flush 时机 | WS 断开同步 flush |
| 12 | 列表状态 | 仅 ended / timed_out |
| 13 | 1d 验收 | 4 项 |

## 涉及的代码改动

### server/

新建：
- `internal/api/replay.go`：`GET /api/sessions/:id/replay?offset=&limit=` 与 `GET /api/sessions?status=ended&since=`
- `internal/recording/gc.go`：GC worker（每小时扫描 + 删 MinIO + 删 PG）

修改：
- `internal/api/ws.go`：visitorWS 关闭时调用 flusher.Unregister（同步 flush）
- `internal/api/router.go`：注册 replay 路由
- `internal/storage/queries.go`：加 `ListEndedSessionsByTenant` + `ListEventBlobsOlderThan` + `DeleteEventBlobByID`
- `internal/recording/flusher.go`：暴露 `FlushSessionNow(sessionID)` 同步 flush
- `cmd/server/main.go`：装配 GC worker
- `internal/config/config.go`：加 `RetentionDays`、`GCIntervalHours`

### admin/

新建：
- `src/views/ReplayList.vue`：历史会话列表
- `src/views/ReplayViewer.vue`：单会话回放页
- `src/workers/replay-decoder.worker.ts`：Web Worker 解码 msgpack
- `src/api/sessions.ts`：REST API 客户端

修改：
- `src/router/index.ts`：加 `/replay` + `/replay/:session_id`
- `src/views/Dashboard.vue`：加导航链接
- `src/components/VisitorPanel.vue`：session 结束时提示跳转

### e2e/

修改：
- `tests/realtime.spec.ts`：加 1d 4 验收场景

## 协议扩展

```typescript
// GET /api/sessions?status=ended&since=24h
interface ListEndedSessionsResponse {
  sessions: Array<{
    session_id: string;
    visitor_id: string;
    fingerprint: string;
    started_at: number;
    ended_at: number;
    duration_ms: number;
    event_count: number;
    ua: string;
  }>;
  total: number;
}

// GET /api/sessions/:id/replay?offset=0&limit=10000
interface ReplayEventsResponse {
  session_id: string;
  events: Array<{ type: number; timestamp: number; data: unknown }>; // rrweb 原生格式
  total: number;
  offset: number;
  limit: number;
  has_more: boolean;
}
```

## 估时

- **Solo 全职**：约 7-9 天
- **Solo 业余**：约 2-3 周

## Next

实施完切片 1d 后：

1. 写 `docs/reports/completed/{date}-slice-1d-implementation.md`
2. 更新 `docs/project-status.md` §5 切片状态表（1d → completed）
3. 启动切片 1e（双向通道：admin overlay → 命令 → SDK 执行）

## Blockers

无。1c 实时回放已就绪，1d 是数据持久化层 + admin 历史视图。

## Notes

- 1d 不实现 co-browsing（1e 起），historical replay 是只读
- 1d 不做认证（PLAN.md 1h 起）
- 1d 不引入新表，复用 1b 的 visitors + sessions + event_blobs
- GC worker 默认每小时扫描，retention_days=30 可配
- Web Worker 解码仅用于历史回放页（事件多）；live dashboard 继续用主线程
