# 切片 1e 实施过程

**状态**：in_progress
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：[`2026-06-17-slice-1e-spec.md`](./2026-06-17-slice-1e-spec.md)

## Context

按规格落地切片 1e：co-browsing 双向通道。访客端可接收运营命令（cursor_highlight / click / scroll / fill_input / navigate），运营通过显式 Start/Stop 按钮进入控制模式，访客 ESC 三连紧急退出。

## Changes

- [ ] server: PG migration + co_browsing_commands + queries
- [ ] server: hub SendCommandToVisitor 反向路由
- [ ] server: POST /api/sessions/:id/command REST 端点
- [ ] server: proto envelope 加 command 类型
- [ ] SDK: commands/handler.ts + cursor.ts + nodeMap.ts
- [ ] SDK: ESC 紧急退出 + 5s 临时锁定
- [ ] admin: CoBrowseOverlay + OperatorCursor
- [ ] admin: Dashboard Start/Stop + useWs.sendCommand
- [ ] e2e: 4 个 1e 场景
- [ ] 端到端验证 + 完成报告

## Status

进行中。

## 与规格的偏差

（实施过程中追加）

## Next

完成后写完成报告 `docs/reports/completed/2026-06-17-slice-1e-implementation.md`。

## Blockers

无。
