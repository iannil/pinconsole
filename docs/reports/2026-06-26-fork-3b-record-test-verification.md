# Fork-3b + record 测试验收报告

**验收日期**: 2026-06-26
**验收人**: Reasonix
**验收范围**: fork-3b 上游测试转译 + record 模块测试深化 + 预存测试修复
**关联切片**: vendor-rrweb fork-3b（总纲见 `reports/completed/2026-06-25-vendor-rrweb-spec.md`）
**之前深度**: 🔴 implemented-unverified（fork-3b 未执行）
**之后深度**: 🟡 verified-shallow（核心 replay 时间线 + 状态机 + record 状态方法覆盖）

## 验收摘要

| 模块 | 之前 | 之后 | 新增测试 |
|---|---|---|---|
| fork-3b 整体 | 🔴 未执行 | 🟡 核心路径覆盖 | 3 文件 / 59 测试 |
| record/ | 🔴 0% | 🟡 3 子模块 + MutationBuffer 状态方法 | 2 文件 / 29 测试 |

## 详细交付

### 1. replay/timer.ts — Timer 类全量测试
- **文件**: `tests/timer.test.ts`（20 测试）
- **覆盖**: constructor(2)、addAction(3)、start+rafCheck(3)、clear(3)、setSpeed(1)、isActive(3)、addDelay 纯函数(5)
- **关键发现**: Timer 耗尽 actions 后设 `raf=true`（was-active 状态）而非 `raf=null`，这是设计使然——`addAction` 可从 was-active 恢复 RAF 循环

### 2. replay/machine.ts — 状态机测试
- **文件**: `tests/machine.test.ts`（21 测试）
- **覆盖**: discardPriorSnapshots 纯函数(5)、createPlayerService 状态转换(9)、createSpeedService 状态转换(7)
- **关键发现**: xstate/fsm 的 `interpret()` 需先调用 `service.start()` 才能用 `send()` 处理事件

### 3. replay/index.ts — Replayer 核心 API
- **文件**: `tests/replayer.test.ts`（18 测试）
- **覆盖**: constructor(5)、setConfig(2)、getMetaData(1)、getCurrentTime(1)、getTimeOffset(1)、getMirror(1)、enableInteract/disableInteract(3)、resetCache(1)、on/off(2)、destroy(1)
- **注意**: 所有测试使用 `liveMode: true` + 不含 FullSnapshot 的事件，避免构造时自动调度 iframe 重建（需真实浏览器）

### 4. record/ 子模块测试
- **文件**: `tests/record-helpers.test.ts`（18 测试）
- **覆盖**: error-handler(7): registerErrorHandler/callbackWrapper 核心路径 + ProcessedNodeManager(4): inOtherBuffer 边界 + StylesheetManager(7): constructor/reset/attachLinkElement

### 5. record/mutation.ts — MutationBuffer 状态方法
- **文件**: `tests/mutation-buffer.test.ts`（11 测试）
- **覆盖**: init/isFrozen/freeze/unfreeze/lock/unlock/reset/processMutations/emit
- **注意**: freeze/unfreeze/lock/unlock/reset 通过 `makeInitOptions` 提供 mock canvasManager + shadowDomManager（源码未使用 `?.` 操作符）

### 6. 预存失败修复
- **文件**: `tests/replay-parity.test.ts`（**已删除**）
- **问题**: jsdom 下 `requestAnimationFrame` 不触发 → Timer 永不推进 → finish 5s 超时
- **方案**: 删除冗余测试——已被 `e2e/tests/fork-0-replay-parity.spec.ts` (Playwright 真实浏览器) 覆盖

## 测试结果

| 包 | 文件 | 测试 | 状态 |
|---|---|---|---|
| replay-core | 8 | 145/146(1 skip: benchmark) | 🟢 **全绿** |
| admin | 24 | 203 | 🟢 无回归 |
| visitor-sdk | 14 | 219 | 🟢 无回归 |

## 代码体量变化

| 指标 | 值 |
|---|---|
| 新增测试文件 | 5 |
| 删除测试文件 | 1（冗余） |
| 新增测试用例 | +88 |
| replay-core 总测试 | 145（含 1 skip） |
| replay-core 总 LOC | ~9,700（源码）+ ~3,600（测试） |

## 未覆盖部分（留 backlog）

| 模块 | 原因 | 建议方式 |
|---|---|---|
| `record/observer.ts` (1,390 行) | 需真实 MutationObserver/DOM 事件 | Playwright e2e |
| `record/index.ts` `record()` 函数 (654 行) | 需真实 DOM 环境 + 事件循环 | Playwright e2e |
| `replay/index.ts` play/addEvent/startLive | 需 iframe 渲染 | Playwright e2e 或 jsdom + fixture |
| `snapshot/rebuild.ts` `rebuild()` 默认导出 | 需完整 snapshot 数据 | 后续切片 |
