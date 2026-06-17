# 切片 1d 实施过程

**状态**：in_progress
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：[`2026-06-17-slice-1d-spec.md`](./2026-06-17-slice-1d-spec.md)

## Context

按规格落地切片 1d：录像归档 + 历史回放。基于 1b/1c 的 Redis Stream + MinIO flusher + rrweb 事件，加 GC worker、replay API、admin 历史列表与回放页。

## Changes

- [ ] server: PG queries 加 ended 列表 + 旧 blob 查询/删除
- [ ] server: replay API（GET /api/sessions/:id/replay 分页）
- [ ] server: GC worker（每小时）
- [ ] server: flusher.FlushSessionNow + visitorWS 关闭同步 flush
- [ ] admin: api/sessions.ts REST 客户端
- [ ] admin: ReplayList.vue + ReplayViewer.vue
- [ ] admin: Web Worker msgpack 解码
- [ ] admin: router + Dashboard 导航
- [ ] e2e: 4 个 1d 场景
- [ ] 端到端验证 + 完成报告

## Status

进行中。

## 与规格的偏差

（实施过程中追加）

## Next

完成后写完成报告 `docs/reports/completed/2026-06-17-slice-1d-implementation.md`。

## Blockers

无。
