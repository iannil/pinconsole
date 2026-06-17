# 切片 1d 录像归档 + 历史回放完成报告

**状态**：completed
**完成时间**：2026-06-17
**对应 progress**：[`docs/progress/2026-06-17-slice-1d-implementation.md`](../../progress/2026-06-17-slice-1d-implementation.md)
**规格来源**：[`docs/progress/2026-06-17-slice-1d-spec.md`](../../progress/2026-06-17-slice-1d-spec.md)

## Summary

按 13 项锁定决策落地 v1 历史回放：访客关闭页面 → 服务端同步 flush → MinIO 永久归档 → admin 在 `/replay` 列表点开会话 → rrweb-player 原生控制器完整回放。同时实现 GC worker 每小时清理 30 天以上 blob。所有 4 个 1d 验收场景通过，18 个 e2e + 9 个 Go 单元测试全部通过。

## Changes Delivered

### server/（2 新建 / 5 修改）

新建：
- `internal/api/replay.go`：`GET /api/sessions/ended` + `GET /api/sessions/:id/replay?offset=&limit=`
- `internal/recording/gc.go`：GC worker（每小时扫描 + 删 MinIO + 删 PG）

修改：
- `internal/storage/queries.go`：加 `ListEndedSessionsByTenant` + `ListEventBlobsOlderThan` + `DeleteEventBlobByID` + `CountEventBlobsBySession`
- `internal/recording/stream.go`：加 `Flusher.FlushSessionNow` 同步 flush
- `internal/api/ws.go`：
  - visitorWS hello 时 `flusher.Register`（修复 1b 遗留：事件未归档）
  - visitorWS 关闭时 `flusher.Unregister`（同步 flush 最后一批 + 清 snapshot 缓存）
  - WSHandler 加 `flusher *recording.Flusher` 字段
- `internal/api/router.go`：注册 replay 路由 + Options 加 Flusher
- `cmd/server/main.go`：装配 GC worker

### admin/（3 新建 / 2 修改）

新建：
- `src/views/ReplayList.vue`：历史会话列表（24h/7d/30d 时间过滤）
- `src/views/ReplayViewer.vue`：单会话回放页（rrweb-player + 分页拉取）
- `src/api/sessions.ts`：REST API 客户端

修改：
- `src/router/index.ts`：加 `/replay` + `/replay/:session_id`
- `src/views/Dashboard.vue`：加"历史会话"导航链接

### e2e/

修改：
- `tests/realtime.spec.ts`：加 4 个 1d 验收场景
- `playwright.config.ts`：改为串行（`workers: 1, fullyParallel: false`），避免并发干扰

## Verification

```bash
# 1. Go 单元测试
cd server && go test ./internal/proto/... ./internal/hub/... ./internal/recording/...
# → all ok

# 2. 前端构建
pnpm --filter @marketing-monitor/admin build
# → ReplayList/ReplayViewer/sessions 独立 chunk

# 3. release 二进制
cd server && CGO_ENABLED=0 go build -tags release -o bin/server ./cmd/server
# → 31 MB

# 4. e2e（18 测试，全过）
docker compose up -d
docker compose exec postgres psql -U mm -d marketing_monitor < server/migrations/000001_init.up.sql
./server/bin/server &
pnpm --filter @marketing-monitor/e2e test --reporter=list
# → 18 passed
```

**1d 验收 4 场景**：
- ✅ live 转 historical（访客关闭 → admin 历史回放）
- ✅ 短 session 即时 replay（< 30s 也立即 replayable）
- ✅ 长 session 分页 replay（1000+ 事件）
- ✅ replay 控制器交互（rrweb-player 原生）

## 与规格的偏差

| 偏差 | 规格 | 实际 | 理由 |
|---|---|---|---|
| Web Worker 解码 | 主线程阻塞避免 | 主线程同步解码 | 1d 实测事件量 < 10K，主线程足够；Worker 推迟到 1e+ 真有性能问题时再加 |
| `GET /api/sessions/ended` 独立路径 | 复用 `GET /api/sessions?status=ended` | 独立路径 `/api/sessions/ended` | 避免与 `/api/sessions/:id/replay` 路径冲突； Gin 路由树更清晰 |
| flusher.Register 时机 | 假定已实现 | 实际未在 visitorWS 调用 | 1b 遗留 bug：flusher 注册从未触发；1d 修复，visitorWS hello 时 Register + 关闭时 Unregister |
| e2e 并发 | 默认并行 | 改为串行（`workers: 1`） | 多测试并发访问同一后端导致访客列表相互干扰 |

## Follow-ups

- 切片 1e：双向通道（admin overlay → 命令 → SDK 执行）—— co-browsing 起步
- 1e+：补 Go testcontainers 完整集成测试（含 flusher 真实触发、GC worker 真实清理）
- 1e+：长 session 流式 replay（替代当前一次性返回）
- 性能调优：1d 当前 blob 全部加载到内存再分页，1e+ 可加 stream 处理

## Notes

- **flusher.Register 修复**：1b 实现了 flusher 但 visitorWS 从未调用 Register，事件一直只写 Redis Stream 不归档 MinIO。1d 修复后所有 session 关闭时同步归档。
- **GC worker**：默认每小时扫描 + 30 天保留。可在 `.env` 中调 `RETENTION_DAYS` / `GC_INTERVAL_HOURS`（待 1d polish 中加 config 暴露）。
- **replay API 分页**：默认 limit=10000，长 session 通过 `?offset=&limit=` 分页。admin ReplayViewer 自动连续拉取直到 has_more=false。
- **admin 路由**：`/dashboard`（1c live）+ `/replay`（1d 列表）+ `/replay/:session_id`（1d 回放）。三者独立 chunk，按需加载。
- **DOM 安全**：ReplayViewer 用 `replaceChildren()` 清空容器（与 1c ReplayPlayer 一致），避免 XSS 风险。
- **串行 e2e**：`workers: 1 + fullyParallel: false`，多测试不并发访问后端，避免访客列表相互干扰。代价是 e2e 总时间从 9s → 70s，但稳定性显著提升。
