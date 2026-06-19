# 切片 1ad — 测试信心加固 T1(40 项 important 路径回归测试)

**Spec 日期**:2026-06-19
**来源**:[`docs/audits/2026-06-19-test-confidence-audit.md`](../audits/2026-06-19-test-confidence-audit.md) §5.3
**目标**:关闭审计发现的 40 个 T1 gap,把 11 个 🟡 切片升回 🟢 touched

## 范围

### 1s 可观测性深化(13 项 T1)— 最高 ROI,纯源码契约

| ID | 描述 | 文件 |
|---|---|---|
| T1-1s-LIF-01~06 | 5 handler 的 Lifecycle Start/End 埋点(PostCommand/Claim/Release/PostMessage/FlushSession/GC.runOnce) | `observability_lifecycle_test.go`(新建) |
| T1-1s-LP-01~05 | 5 LogPoint 分支(claim_check ok/failed, command_type, navigate_check, popup_url_check) | 同上 |
| T1-1s-EXT-01~03 | 3 LogExternalCall(pg.CreateCoBrowsingCommand, minio.PutObject, pg.CreateEventBlob) | 同上 |

### 1d 录像归档(5 项 T1)— PG/MinIO 集成

| ID | 描述 | 文件 |
|---|---|---|
| T1-1d-1 | R2 上传(MinIO PutObject) | `recording/stream_test.go`(新建) |
| T1-1d-2 | PG INSERT 失败 → MinIO RemoveObject 补偿 | 同上 |
| T1-1d-3 | Flusher Register/Unregister/EndSession 生命周期 | 同上 |
| T1-1d-4 | GC runOnce(已部分覆盖于 1ac-1l-4) | reference |
| T1-1d-5 | GDPR erasure MinIO cleanup(已覆盖于 1ac-1l-2) | reference |

### 1g 弹窗 + 聊天(5 项 T1)

| ID | 描述 | 文件 |
|---|---|---|
| T1-1g-1 | chat repo CreateChatMessage / ListChatMessagesBySession | `storage/chat_repo_test.go`(新建) |
| T1-1g-2 | chat WS 下行(admin → visitor via command) | source contract |
| T1-1g-3 | chat post-claim ownership required | source contract |
| T1-1g-4 | chat listMessages 不要求 claim | source contract |
| T1-1g-5 | popup body textContent XSS 安全 | source contract |

### 1e 双向通道(4 项 T1)— 源码契约

| ID | 描述 |
|---|---|
| T1-1e-1 | buildCommandPayload 8 种命令类型 |
| T1-1e-2 | SendCommandToVisitor 路由 |
| T1-1e-3 | co_browsing_commands 审计行 PG INSERT 断言 |
| T1-1e-4 | OperatorID 用 callerUID(已 closed by 1ac-1k-3)|

### 1b 单向最小(4 项 T1)

| ID | 描述 |
|---|---|
| T1-1b-1 | reconnect event recovery(buffer 恢复) |
| T1-1b-2 | MinIO PutObject / PG INSERT / Redis XADD/XTRIM 写入断言 |
| T1-1b-3 | session-timeout-30min |
| T1-1b-4 | form sensitive field filter |

### 1c rrweb(5 项 T1)

| ID | 描述 |
|---|---|
| T1-1c-1 | periodic full snapshot(30s/50 events) |
| T1-1c-2 | snapshot Redis cache |
| T1-1c-3 | selective screenshot trigger |
| T1-1c-4 | iframe policy |
| T1-1c-5 | SDK retry 3 次 + visibility > 60s |

### 1m 可观测性(3 项 T1)

| ID | 描述 |
|---|---|
| T1-1m-1 | 服务端从 envelope.TraceID 还原 ctx |
| T1-1m-2 | 下行 envelope 携带 ctx TraceID |
| T1-1m-3 | SDK logger 模块 |

### 1w flagged session(4 项 T1)

| ID | 描述 |
|---|---|
| T1-1w-1 | Redis 失败 warn 不阻断 |
| T1-1w-2 | 订阅 flagged session warn |
| T1-1w-3 | replay flagged session warn |
| T1-1w-4 | admin store 映射 is_flagged 字段 |

### v1-followups(3 项 T1)— 回归测试已知 fix

| ID | 描述 |
|---|---|
| T1-v1f-1 | listMessages 不要求 claim(fix1)回归测试 |
| T1-v1f-2 | toggleCoBrowsing 自动 claim(fix1)回归测试 |
| T1-v1f-3 | 选择 visitor 自动 subscribe(fix5)回归测试 |

### 1f/1o/1r/v1-e2e 剩余

- 1f: presence.navigated 广播 + admin auto-resubscribe(2 项)
- 1o: per-sub cancel ctx + goroutine leak count(2 项)
- 1r: SDK i18n keys(9 项,部分源码契约)
- v1-e2e: 13 处 indirect coverage(部分无法转为 direct)

## 验收

每个 T1 测试必须:
1. 测试存在 + 通过
2. 对应源码 mutation 后 FAIL(关键路径)或源码契约(grep 关键字符串)
3. 报告带 file:line 引用

完成后:
- 11 个 🟡 切片 → 🟢 touched(或保持 🟡 但 T1 数大幅下降)
- `project-status.md` §5 更新

## 工作量

预计 30 小时(solo 全职 ~4 天)。本会话优先做 1s(纯契约,快速)+ 1d/1g PG 集成。

## 风险

- 部分 T1(如 1c SDK retry / 1b reconnect)需 SDK 端 TS 测试,与 Go 测试不同栈
- 1s lifecycle 集成点测试需 stub Stores,可能要小重构(handler 提取 lifecycle 调用为可测 helper)
- v1-followups 3 个 fix 涉及前端 store,需 vitest

## Out of scope

- T2/T3(留 backlog)
- 任何业务逻辑变更(只加测试)
- 1d/1g/1s 这三个 🔴 切片不在本切片范围(1ad 只升 🟡 → 🟢,不动 🔴)

注:1s 是 🔴 不是 🟡,但 1ad 范围内的 1s 工作是 T1(13 项 lifecycle 集成点)。完成后 1s 仍可能保持 🔴(因 T0 lifecycle 集成点的强集成测试缺),但 T1 部分关闭。
