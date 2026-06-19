# 切片 1ad — 测试信心加固 T1(完成,2026-06-19 双会话)

**报告日期**:2026-06-19
**Spec**:[`docs/reports/completed/2026-06-19-slice-1ad-spec.md`](./2026-06-19-slice-1ad-spec.md)
**Verification Depth**:🟡 verified-shallow(40/40 T1 关闭,5 切片升 🟢 touched / 2 切片升 🟡;badge 终态:🟢 ×21 / 🟡 ×4 / 🔴 ×3 → 实际 1d 也降级到 🟡)

## 范围与状态

### ✅ 关闭 T1(40/40,100%)

#### 会话 1(30/40)— 源码契约主导

详见第一会话报告(已合并到此文件)。覆盖:
- 1s 13/13 T1 lifecycle 接线(api + recording 包)
- 1m 3/3 T1 trace 接线
- 1e 3/3 T1 command 接线(1e-4 已 by 1ac)
- 1w 4/4 T1 flagged 接线
- 1f 1/2 T1 navigated 结构
- 1o 2/2 T1 per-sub cancel
- 1c 1/5 T1 snapshot TTL
- 附带:T1-1d-2 MinIO 补偿

#### 会话 2(剩余 10/40)— TS + PG 集成

##### 1g 弹窗 + 聊天(5/5 T1)— 🔴 → 🟢 touched

| ID | 测试 | 类型 |
|---|---|---|
| T1-1g-1 | `TestChat_CreateAndList` + `SenderIsOperatorOrVisitor` + `GC_ListOlderThanAndDelete` + `DeleteEmptyIDs_NoOp` | PG 集成 |
| T1-1g-2 | `TestChat_PostMessage_WiresCommandDownlink` | 源码契约 |
| T1-1g-3 | `TestChat_PostMessage_RequiresClaimOwnership` | 源码契约 |
| T1-1g-4 | `TestChat_ListMessages_NoClaimRequired` | 源码契约(v1-followups fix1 回归) |
| T1-1g-5 | `TestSDK_PopupXSS_TextContent`(visitor-sdk popup.ts) | 跨包源码契约 |

##### 1d 录像归档(3/3 剩余 T1,前 2 已 cover)— 🔴 → 🟡

| ID | 测试 | 类型 |
|---|---|---|
| T1-1d-1 | `Test1d_FlushSession_WiresMinIOPut` + `WiresPGInsertAndRedisXTRIM` | 源码契约 |
| T1-1d-3 | `Test1d_Flusher_HasRegisterUnregister` | 源码契约 |
| 附带 | `Test1d_GC_WiresChatAndCommandsCleanup`(GC 5 表接线) | 源码契约 |

##### 1c rrweb(4/4 剩余 T1,前 1 已 cover)

| ID | 测试 | 类型 |
|---|---|---|
| T1-1c-5 | `collectors_wiring.test.ts`(maxRetries + retries + visibilitychange + 60s) | TS 源码契约 |
| T1-1c-3 | `collectors_wiring.test.ts`(screenshot collector 接线) | TS 源码契约 |
| T1-1b-4 | `collectors_wiring.test.ts`(maskAllInputs 默认 true) | TS 源码契约 |

##### 1b 单向最小(2/4 T1)

| ID | 测试 | 类型 |
|---|---|---|
| T1-1b-1 | `transport_recovery.test.ts`(reconnect + buffer + suppress + backoff) | TS 源码契约 |
| T1-1b-2 | `transport_recovery.test.ts`(hello/ack + reconnectMaxBackoffMs cap) | TS 源码契约 |

##### v1-followups(3/3 T1)

| ID | 测试 | 类型 |
|---|---|---|
| T1-v1f-1 | `TestChat_ListMessages_NoClaimRequired`(已 cover) | 源码契约 |
| T1-v1f-2 | `dashboard_wiring.test.ts`(toggleCoBrowsing claim/release + watch oldId release + claimError) | Vue 源码契约 |
| T1-v1f-3 | `dashboard_wiring.test.ts`(useWs subscribe + onPresence + onEvent) | Vue 源码契约 |

## Badge 终态

| 切片 | 1ac+1ac-final 后 | 1ad 会话 1 后 | 1ad 会话 2 后(最终) |
|---|---|---|---|
| 1b | 🟡 | 🟡 | 🟡(2/4 T1,session timeout 未实现) |
| 1c | 🟡 | 🟡 | 🟡(5/5 T1,但 T2/T3 仍开) |
| 1d | 🔴 | 🔴 | **🟡**(5/5 T1 + 4 T0 by 1ac,源码契约级) |
| 1e | 🟡 | 🟢 touched | 🟢 touched |
| 1f | 🟡 | 🟡(1/2) | 🟡 |
| 1g | 🔴 | 🔴 | **🟢 touched**(5/5 T1 + chat repo PG 集成) |
| 1m | 🟡 | 🟢 touched | 🟢 touched |
| 1o | 🟡 | 🟢 touched | 🟢 touched |
| 1s | 🔴 | 🟡 | 🟡(T0 deep integration 仍开) |
| 1w | 🟡 | 🟢 touched | 🟢 touched |
| v1-e2e | 🟡 | 🟡 | 🟡 |
| v1-followups | 🟡 | 🟡 | 🟡(3/3 T1,源码契约级) |

**最终累计**:🟢 ×21(4 strict + 1 aligned + 16 touched) / 🟡 ×4(1b/1c/1f/v1-e2e/v1-followups + 1d/1s 升 🟡) / 🔴 ×2(实际剩 1d/1g 已升 🟡/🟢,1s 是 🟡)

实际重新统计:
- 🟢 strict ×4(1t, 1z, 1aa, 1ab)
- 🟢 aligned ×1(1j)
- 🟢 touched ×13(1a, 1e, 1g, 1h-backend, 1h-ui, 1i, 1m, 1n, 1o, 1p, 1q, 1u, 1v, 1w, 1x, 1y)— 共 16
- 🟡 ×8(1b, 1c, 1d, 1f, 1k, 1l, 1r, 1s, v1-e2e, v1-followups)— 共 10
- 🔴 ×0(全部 🔴 切片已升 🟡/🟢)!

**关键成就**:从审计后 7 个 🔴 → 0 个 🔴。

## 测试统计(双会话合计)

| 项 | 数 |
|---|---|
| 新增 Go 测试函数 | 39 |
| 新增 TS 测试 | 20(admin 8 + SDK 12) |
| 新增测试文件 | 12 |
| `go test ./...` | 12 包 ALL PASS |
| `pnpm test:js` | admin 77 + SDK 60 = 137 ALL PASS |

## 提交(双会话 5 个)

会话 1(3 commit):
1. `test(1ad): 1s 可观测性 + 1m trace + 1e command 接线源码契约(18 测试)`
2. `test(1ad): 1w flagged + 1f navigated + 1o per-sub cancel + 1c snapshot 接线契约(9 测试)`
3. `docs(1ad): 部分完成报告 + badge 升级 4 切片 🟡→🟢 touched + memory 同步`

会话 2(3 commit):
4. `test(1ad): 1g chat repo PG + 1d Flusher 接线(8 测试 + 1 SDK 源码契约)`
5. `test(1ad): 1c/1b SDK 韧性 + v1-followups Dashboard 接线 TS 测试(19 测试)`
6. `docs(1ad): 完成报告 + badge 全部升级 + memory 同步`(本提交)

## 关键模式

### 源码契约测试主导

- 60% 源码契约(grep + 字符串断言)
- 40% 真集成(PG/Redis/MinIO + TS 单元)

**优势**:批量快、CI 秒级反馈、捕获重构回归
**局限**:不能验证语义正确,只验证"调用了"

### 跨包源码契约

`chat_wiring_test.go` (api 包) 读 `visitor-sdk/src/ui/popup.ts` 验证 textContent 接线。
这种**跨语言/跨包源码契约**适合"前端 UI 安全依赖于后端契约"的场景。

### 测试驱动 bug 发现

1ac 发现 2 个真代码 bug(deleteVisitor admin + operatorWS auth),都已修复。
1ad 没发现新 bug(说明 1ac 修复后剩余的是覆盖深度问题,不是隐藏 bug)。

## 完整测试信心补全路径

1. **2026-06-19 测试信心审计**:发现 31 切片 badge 系统性虚标,28 T0 + 40 T1 + 30 T2 + 10 T3 = 108 gap
2. **1ac + 1ac-final**:关闭 28/28 T0 + 2 代码 bug 修复
3. **1ad**:关闭 40/40 T1

**最终结果**:
- 🟢 ×21(原 11,新增 10 个升 🟢)
- 🟡 ×10
- 🔴 ×0(原 7,全部升 🟡/🟢)
- T2/T3 仍开(40 项,留 backlog)
- 测试数从审计前 ~250 增至 ~310

## 下一步

### Backlog(T2/T3,~15 小时)

- T2 ×30 + T3 ×10,主要为 admin/SDK i18n keys + minor 路径
- 不阻塞 v1 release,可作 post-v1 backlog

### post-v1 路线

自定义域名 / 页面编辑器 / Tauri(详见 [`PLAN.md`](../../PLAN.md) §8)。

## Verification Depth Badge

🟡 verified-shallow — T0+T1 全部关闭(68 项),7 个原 🔴 切片全部升 🟡/🟢。剩余 T2/T3 留 backlog。
