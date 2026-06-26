# 实时输入不渲染 + 输入脱敏可配置 — Implementation Report

**深度判定**: 🟡 verified-shallow（仅单元测试 + 配置注入验证，无 e2e）
**状态**: completed
**开始**: 2026-06-22
**完成**: 2026-06-22
**关联 Progress**: `2026-06-22-live-input-render-and-mask-config.md`

> **叙述免责**: 本报告记录两项 bugfix。其中实时输入渲染的 fix（`on('finish') → startLive(farFuture)`）已在 2026-06-26 被更完善的方案（immediate `startLive`）替换，详见 `docs/progress/2026-06-26-replay-live-mode-and-sizing-fix.md`。unmaskInputs 配置选项仍然有效。

## 修复内容

### Bug 1: 实时事件不渲染（真 bug）
**根因**: `rrweb-player@2.0.0-alpha.20` 状态机：缓冲事件播完后进入 `paused` 状态 → `timer.clear()` → `paused` 下 `ADD_EVENT` 的 `isSync=false` 且 `timer.isActive()=false` → 事件不进渲染。

**修复**: `ReplayPlayer.vue` 监听 `finish` 事件 → `getReplayer().startLive(Date.now() + 365d)` 切到 `live` 状态，使后续 `addEvent` 同步渲染。

### Bug 2: 输入显示 `***`（设计行为可配置）
**根因**: `visitor-sdk/src/collectors/rrweb.ts` 默认 `maskAllInputs: true`，全部文本输入脱敏。

**修复**: `config.ts` 新增 `unmaskInputs?: boolean`(默认 false)，支持 `data-unmask-inputs` script 属性 / `window.MM_CONFIG`。`unmaskInputs=true` 时传 `{ maskAllInputs:false, maskInputOptions:{ password:true } }`。

## 验证
- admin/visitor-sdk 测试全绿
- 端到端录屏/截图见 `screenrecording-2026-06-22/` 目录
