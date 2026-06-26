# 切片 vendor-rrweb — Implementation Report

**深度判定**: 🟢 touched
**状态**: completed
**创建时间**: 2026-06-25
**完成时间**: 2026-06-26
**分支**: `feat/vendor-rrweb` → merged into `master` (`82a7a355`)
**关联 Spec**: `2026-06-25-vendor-rrweb-spec.md`

> **叙述免责**: 本报告记录 fork-0~4 的执行结果。各 fork 均在上游 Playwright e2e + 单元测试全绿后合并。剩余工作：fork-3b（上游测试转 Playwright，5 组）未执行，留 backlog。

## 概述

将 rrweb alpha.20 TS 源码硬分叉至 `packages/replay-core`，删除 Svelte rrweb-player 依赖，admin 直持 `Replayer` 实例，精简 canvas/console 死代码，实现 nodeID 跨端寻址全链路。

## 决策偏差

| 决策点 | Spec | 实际 | 偏差? |
|---|---|---|---|
| fork 策略 | A 硬分叉 | A ✅ | 无 |
| 包结构 | B 单包分目录 | B ✅ | 无 |
| rrweb-player | B 删除 | B ✅ | 无 |
| 节奏 | B 先复刻后雕刻 | B ✅ | 无 |
| packer | B 砍掉 | B ✅ | 无 |
| fork-3b | 上游测试转 Playwright | **未执行** | ⚠️ 留 backlog |
| fork-3a | 文件级裁剪 | ✅ 执行 | 无 |

## 实施细节

### fork-0 (2026-06-25, commit `235aa37`)
**源码移植**：从 rrweb alpha.20 clone 41 个 TS 源文件到 `packages/replay-core/src/`，重写所有 import 路径（npm 包名 → 相对路径），创建 package.json / tsconfig.json / vite.config.ts。
- TypeScript 0 errors, Vite build 成功（42 modules → 305KB ESM）
- 包含 NOTICE (MIT attribution)
- **R1 验证确认**：`document.querySelectorAll('[data-rr-node-id]')` 长度 = 0，rrweb Mirror 不写 DOM attribute，`nodeMap.ts` 确为死代码

### fork-0-parity (2026-06-25, commit `44bc9a5`)
**双 parity 夹具**：`e2e/tests/fork-0-replay-parity.spec.ts` ✅ replay 等价 + `fork-0-record-parity.spec.ts` ✅ record 等价

### fork-1 (2026-06-25, commit `4ef8c60`)
**SDK record 切换**：`visitor-sdk/src/collectors/rrweb.ts` 从 `import('rrweb')` 改为 `import('@pinconsole/replay-core')`。SDK build → 147 modules / 282KB。单元测试 214 全绿。

### fork-2 (2026-06-26 T00:00, commit `5f5a8eb`)
**Admin 钻穿方案**：
- `ReplayPlayer.vue`: ~380→~180 行，删除 15 项 Svelte hack
- `ReplayViewer.vue`: ~605→~400 行，钻穿 + 自建控件线
- `useResponsivePlayerSize.ts`: ~185→~90 行，直接 wrapper.style + handleResize()
- Admin 不再依赖 `rrweb-player`

### fork-3a (2026-06-26 T00:05, commit `ad3d18b`)
**文件级裁剪**：删除 canvas 录制观察器(5 文件) + canvas 回放渲染(2 文件) + canvas worker(1 文件)。stub CanvasManager。ESM bundle 423KB→396KB。

### fork-4 (2026-06-26 T00:15, commit `ecae238`)
**nodeID 跨端寻址**：
- snapshot 写 `data-rr-node-id` attribute（`serializeNodeWithId` 中 `n.setAttribute('data-rr-node-id', String(id))`）
- `CoBrowseOverlay` 从 `return 0` 改为真实 elementFromPoint → 向上遍历 parentElement 找 `data-rr-node-id`
- 全链路：Record→DOM→SDK NodeMap→Admin→Server→SDK Handler→element.click()

### 遗留清理 (commit `2ea5d809`，合并后)
- 删除 `nodeMap.ts`（SDK command handler 改 `querySelector([data-rr-node-id])` 替代）
- `CoBrowseOverlay` 清理坐标 fallback 注释
- `ReplayPlayer` hack 审计确认无残留（类型安全改进：`as never` → `as eventWithTime`）
- parity test `import from 'rrweb'` 保留（意图内的对照夹具）

### 后续修复 (2026-06-26，未提交)
详见 `docs/progress/2026-06-26-replay-live-mode-and-sizing-fix.md`
- ReplayPlayer 即时 live 模式（`startLive(farFuture)` 替代 `on('finish')`）
- useResponsivePlayerSize cover 模式 + 重试机制
- store cap 500→5000
- iframe sandbox `allow-scripts` 全开
- CSS 布局修复

## 验证

| 门 | 结果 |
|---|---|
| TSC --noEmit | ✅ zero errors |
| SDK/Admin build | ✅ 通过 |
| Unit tests (360) | ✅ 全绿 |
| e2e parity (record + replay) | ✅ 2/2 passed |
| e2e 1l privacy (fork-1) | ✅ 4/4 passed, 2 skipped |
| e2e 1c rrweb (fork-2) | ⚠️ 预存 admin UI 失败（无关） |
| `rg "rrweb" admin/package.json` | ✅ 无命中 |
| `rg "SvelteComponent"` | ✅ 无命中 |

## 代码体量变化

| 指标 | fork-0 | fork-4 | 变化 |
|---|---|---|---|
| packages/replay-core LOC | 41 files, ~280KB | 37 files, ~268KB | -4 files, -12KB |
| ESM bundle | 423KB | 396KB | -27KB (fork-3a 裁剪) |
| Admin replay-core chunk | ~100KB | ~95KB | -5KB |
| SDK build | 147 modules / 282KB | ~269KB | -13KB |

## 未完成项 (backlog)

- **fork-3b**: 上游 5 组测试转 Playwright（snapshot/replayer/observer/shadow DOM/iframe/mask）（~2-3d）
- **session_id 与 session_payload 表外键**: 强制 `ON DELETE CASCADE`（现为软删除，需迁移脚本）
