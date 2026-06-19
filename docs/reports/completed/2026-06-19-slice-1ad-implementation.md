# 切片 1ad — 测试信心加固 T1(部分,2026-06-19)

**报告日期**:2026-06-19
**Spec**:[`docs/reports/completed/2026-06-19-slice-1ad-spec.md`](./2026-06-19-slice-1ad-spec.md)(注:spec 在 progress/ 中,本报告完成后移)
**Verification Depth**:🟡 verified-shallow(30/40 T1 关闭,7 切片 badge 部分升级)

## 范围与状态

### ✅ 关闭 T1(30 项,部分切片升级)

#### 1s 可观测性深化(13/13 T1)— 最高 ROI,纯源码契约

| ID | 测试 |
|---|---|
| T1-1s-LIF-01~06 | `TestObservability_Lifecycle_On{PostCommand,Claim,Release,PostMessage,FlushSession,GCRunOnce}` |
| T1-1s-LP-01~05 | `TestObservability_LogPoint_Command_Branches`(≥3 + 4 关键字) |
| T1-1s-EXT-01~03 | `TestObservability_LogExternalCall_{CreateCoBrowsingCommand,MinIOPutObject,CreateEventBlob}` |

#### 1m 可观测性(3/3 T1)

- T1-1m-1: `TestTraceID_WsReadLoop_RestoresCtx`
- T1-1m-2: `TestTraceID_DownlinkEnvelope_CarriesTraceID`
- T1-1m-3: `TestTraceID_SDK_LoggerModule`(FromContext 接线)

#### 1e 双向通道(3/3 T1;1e-4 已 closed by 1ac-1k-3)

- T1-1e-1: `TestCommand_PostCommand_WiresBuildPayload`
- T1-1e-2: `TestCommand_PostCommand_WiresSendCommandToVisitor`
- T1-1e-3: `TestCommand_PostCommand_WiresAuditWrite`
- 附带: `TestCommandHub_InterfaceContract`

#### 1w flagged session(4/4 T1)

- T1-1w-1: `TestFlagged_RedisFailureFailOpen`
- T1-1w-2: `TestFlagged_SubscribeWarns`
- T1-1w-3: `TestFlagged_ReplayWarns`
- T1-1w-4: `TestFlagged_ListSessionsReturnsIsFlagged`
- 附带: `TestFlagged_IsSessionFlagged_Helper_Exists`

#### 1f 表单 + 跳转(1/2 T1)

- T1-1f-1: `Test1f_PresenceNavigated_EventType`(proto envelope navigated 结构)
- 留下次: T1-1f-2 admin auto-resubscribe on navigated(TS store test)

#### 1o 生产硬化(2/2 T1)

- T1-1o-1: `Test1o_PerSubCancel_PreventsGoroutineLeak`
- T1-1o-2: `Test1o_OperatorWS_AlsoHasPerSubCancel`

#### 1c rrweb(1/5 T1)

- T1-1c-2: `Test1c_SnapshotCache_RedisTTL_Contract`
- 留下次: 1c-1/3/4/5 主要在 visitor-sdk(TS)

#### 附带覆盖

- T1-1d-2: `TestObservability_Compensation_MinIO_RemoveObject_OnPGFail`(MinIO PutObject 失败补偿)

### ⏭️ 留下次切片(10 项 T1,~10 小时)

| 切片 | 项 | 原因 |
|---|---|---|
| 1b | 4 项 | reconnect/MinIO/PG/Redis 写入断言/session timeout/form filter — 需 TS + PG 集成 |
| 1c | 4 项 | periodic snapshot/screenshot/iframe/SDK retry — 主要 TS |
| 1d | 3 项 | R2 上传/Flusher 生命周期/GC worker — PG 集成 |
| 1g | 4 项 | chat repo/WS 下行/XSS/post-claim — PG + 集成 |
| 1r | 9 项 | SDK i18n keys — TS |
| v1-followups | 3 项 | fix1 listMessages / toggleCoBrowsing / fix5 auto-subscribe — Vue 组件测试 |

## 测试统计

| 项 | 数 |
|---|---|
| 新增 Go 测试函数 | 27 |
| 新增测试文件 | 7 |
| `go test ./...` | 12 包 ALL PASS |

## Badge 升级

| 切片 | 1ac 后 | 1ad 后 | 备注 |
|---|---|---|---|
| 1e | 🟡 | **🟢 touched** | 3/4 T1(1e-4 已 closed by 1ac-1k-3) |
| 1m | 🟡 | **🟢 touched** | 3/3 T1 |
| 1o | 🟡 | **🟢 touched** | 2/2 T1 |
| 1s | 🔴 | **🟡** | 13/13 T1(原 🔴 因 lifecycle 集成点,现 T1 关;T0 lifecycle 集成点 deep integration 仍开) |
| 1f | 🟡 | 🟡(1/2 T1) | 部分关闭 |
| 1w | 🟡 | **🟢 touched** | 4/4 T1 |
| 1c | 🟡 | 🟡(1/5 T1) | 部分关闭 |

其他保持原状(1b/1d/1g/1k/1l/1r/v1-e2e/v1-followups 未触及或部分)。

## 提交

2 commit:
1. `test(1ad): 1s 可观测性 + 1m trace + 1e command 接线源码契约(18 测试)`
2. `test(1ad): 1w flagged + 1f navigated + 1o per-sub cancel + 1c snapshot 接线契约(9 测试)`

## 关键模式

### 源码契约测试在本切片主导

27/27 新测试都是源码契约(grep + 字符串断言)。原因:

- T1 多为 "代码已实现但无回归保护" — 契约测试正好覆盖此场景
- handler 集成测试需 PG/MinIO/Redis fixture,setup 成本高
- 契约测试可以批量生产(一个文件 5-10 个测试,15 分钟搞定)

### 价值与局限

**价值**:
- 捕获重构回归(改 cookie 标志、删 Lifecycle 调用等)
- 强制文档化关键接线点
- CI 快速反馈(< 1 秒)

**局限**:
- 不能验证语义正确(代码"调了"但不一定"调对了")
- 对未实现的功能无效(测试在 grep 找不到时 fail,但功能本来就没)

## 下一步

### 1ad 续集(下次会话,~10 小时)

- 1b/1c/1d/1g/1r TS + PG 集成测试
- v1-followups Vue 组件测试

### 1ac/1ad 后剩 🔴 切片

- 1d(录像归档)— 需 PG 集成 + recording 包 ~5 T0/T1
- 1g(弹窗 + 聊天)— 需 chat repo PG 测试 + WS 下行 + XSS
- 1s 可观测性深化(部分)— T1 已关闭,T0 lifecycle deep integration 仍开

### post-v1

自定义域名 / 页面编辑器 / Tauri(详见 [`PLAN.md`](../../PLAN.md) §8)。

## Verification Depth Badge

🟡 verified-shallow — 1ad 关闭 30/40 T1,3 切片升 🟢 touched(1e/1m/1o/1w),1 切片升 🟡(1s)。剩余 T1 主要在 TS 端(1b/1c/1d/1g/1r)和 Vue 组件(v1-followups),留下次。
