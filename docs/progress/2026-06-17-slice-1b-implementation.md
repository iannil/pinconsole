# 切片 1b 实施过程

**状态**：in_progress
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：[`2026-06-17-slice-1b-spec.md`](./2026-06-17-slice-1b-spec.md)
**关联**：[`PLAN.md`](../../PLAN.md) §7

## Context

按已敲定的切片 1b 规格（16 项决策 + 4 个验收场景）落地"SDK 全量采集（鼠标 + 点击 + 滚动 + 表单）→ WebSocket → hub → admin 实时面板 + 录像归档"端到端实时管道。

## Changes

按规格 §涉及的代码改动 的文件清单逐项落地：

- [ ] server: PG 3 表 migration + sqlc 配置 + queries
- [ ] server: internal/proto/{envelope,events}.go（MessagePack + Discriminated union）
- [ ] server: internal/hub/{hub,room,client}.go（路由 + 订阅）
- [ ] server: internal/api/{ws,session}.go（端点 + REST）
- [ ] server: internal/recording/{stream,flusher}.go（Redis Stream + MinIO）
- [ ] visitor-sdk: src/proto/{envelope,events}.ts
- [ ] visitor-sdk: src/transport/ws.ts（含指数退避重连）
- [ ] visitor-sdk: src/collectors/{mouse,scroll,form}.ts
- [ ] visitor-sdk: src/session.ts（visitor_id + session_id）
- [ ] admin: src/proto/{envelope,events}.ts
- [ ] admin: src/stores/visitors.ts（Pinia）
- [ ] admin: src/composables/useWs.ts
- [ ] admin: src/views/Dashboard.vue + components/{VisitorList,VisitorPanel}.vue
- [ ] admin: src/router/index.ts
- [ ] e2e: tests/realtime.spec.ts（4 验收场景）
- [ ] Go testcontainers 集成测试
- [ ] 端到端验证 + 完成报告

## Status

进行中。

## 与规格的偏差

实施过程中如出现与规格偏离的决策，记录于此（暂无）。

## Next

按任务清单推进。完成后写完成报告 [`docs/reports/completed/2026-06-17-slice-1b-implementation.md`](../reports/completed/2026-06-17-slice-1b-implementation.md)。

## Blockers

无。

## Notes

- 1b 不集成 rrweb（1c 起加）
- 1b 不实现 co-browsing 双向通道（1e 起）
- 1b 不做认证（1h 起）
- 实施期间如发现规格不可行，停下来与用户确认是否更新规格，不擅自改方案
