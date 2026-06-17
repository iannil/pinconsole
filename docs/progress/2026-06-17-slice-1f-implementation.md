# 切片 1f 实施过程

**状态**：in_progress
**开始**：2026-06-17
**完成**：（进行中）
**规格来源**：[`2026-06-17-slice-1f-spec.md`](./2026-06-17-slice-1f-spec.md)

## Context

按规格把 1e 坐标 fallback 升级为精确 nodeID + 浮动输入框代填 + Toast 提示 + navigate 跨页面自动重订阅 + 白名单。

## Changes

- [ ] proto: PresencePayload 加 Navigated 字段
- [ ] SDK: 页面 unload 检测 navigate 发 presence.navigated
- [ ] SDK: toast.ts 浮动提示
- [ ] SDK: handler 集成 toast
- [ ] SDK: data-allowed-domains 读取
- [ ] admin: CoBrowseOverlay postMessage 取 nodeID
- [ ] admin: FloatingInput.vue 浮动输入框
- [ ] admin: useWs presence.navigated 自动重订阅
- [ ] server: config NavigateAllowedDomains + command.go 读取
- [ ] e2e: 4 个 1f 场景
- [ ] 端到端验证 + 完成报告

## Status

进行中。

## 与规格的偏差

（实施过程中追加）

## Next

完成后写完成报告 `docs/reports/completed/2026-06-17-slice-1f-implementation.md`。

## Blockers

无。
