# 切片 1c 实施过程

**状态**：in_progress
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：[`2026-06-17-slice-1c-spec.md`](./2026-06-17-slice-1c-spec.md)

## Context

按规格落地切片 1c：rrweb 全量采集 + admin 实时回放。替换 1b 的 4 类抽象事件。

## Changes

按规格 §涉及的代码改动 推进：

- [ ] server: proto/events.go 扩展 RRWeb 字段
- [ ] server: storage/redis.go 加通用 Set/Get
- [ ] server: recording/snapshot.go snapshot 缓存读写
- [ ] server: hub + ws.go subscribe 推送 snapshot
- [ ] SDK: 删除 4 个 collector
- [ ] SDK: collectors/rrweb.ts + 韧性 + visibility 检测
- [ ] SDK: collectors/screenshot.ts 选择性截图
- [ ] SDK: batch.ts 批量器
- [ ] SDK: 改 transport + index 编排
- [ ] admin: ReplayPlayer.vue 动态 import rrweb-player
- [ ] admin: 改 VisitorPanel + store 适配 RRWeb 事件
- [ ] e2e: 加 1c 4 验收场景
- [ ] 端到端验证 + 完成报告

## Status

进行中。

## 与规格的偏差

（实施过程中追加）

## Next

完成后写完成报告 `docs/reports/completed/2026-06-17-slice-1c-implementation.md`。

## Blockers

无。

## Notes

- 1c 不实现 co-browsing（1e 起）
- 1c 不实现录像回放（1d 起）
- 1c 不引入新表
- ReplayPlayer 动态 import 避免进首屏 bundle
