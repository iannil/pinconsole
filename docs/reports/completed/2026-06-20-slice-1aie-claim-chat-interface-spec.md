# 切片 1ai-e claim-chat-interface — Spec

**切片编号**:1ai-e
**类型**:重构 + 测试深化(api 包 Phase 3,1ai-c/d 续做)
**创建时间**:2026-06-20
**状态**:approved
**关联**:[1ai-c impl](../completed/2026-06-20-slice-1aic-auth-handler-interface-implementation.md)、[1ai-d impl](../completed/2026-06-20-slice-1aid-me-logout-happy-path-implementation.md)

## Context

1ai-c/d 完成 AuthHandler 接口化 + 全 4 handler happy path 覆盖。本切片同模式扩展到 ClaimHandler + ChatHandler,补 happy path:
- claim.go claim handler:25% → 目标 ≥70%
- chat.go listMessages:57% → 目标 ≥80%
- chat.go postMessage:35%(留 backlog,需 requireClaimOwnership 接口化)

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 范围 | A: claim+chat+postMessage / B: claim+chat listMessages / C: 仅 claim | **B** | A 的 postMessage 需 requireClaimOwnership 重构(影响 command.go),扩大 churn;C 太窄 |
| 2 | 接口设计 | A: 共享 chatRepo / B: 各 handler 各自最小接口 | **B** | 遵循 ISP;claim 不需要 chat 方法,chat 不需要 SetNX |
| 3 | requireClaimOwnership | A: 不动(继续接受 *storage.Stores) / B: 改接受接口 | **A** | postMessage 留 backlog;requireClaimOwnership 仅 postMessage/command.go 用,本切片不动 |
| 4 | 测试模式 | A: 复用 1ai-c mockUserRepo/mockRedisStore / B: 新 mock | **A** | mockRedisStore 已有 SetNX/Get/EvalLua/TTL;mockUserRepo 不需要;加 1 个 mockSessionRepo + 1 个 mockChatRepo |
| 5 | 边界覆盖 | A: 仅 happy path / B: happy + already-claimed 冲突 / C: + session-ended | **B** | B 覆盖核心 1k P0-4 race-safety 行为;C 留 backlog |

## Acceptance

### 必须满足

- [ ] **A1**:新增 `internal/api/claim_chat_interfaces.go`(或追加 auth_interfaces.go):
  - `claimSessionRepo`:GetSession
  - `claimRedisStore`(可与 authRedisStore 合并):Get / SetNX / EvalLua
  - `chatMessageRepo`:ListChatMessagesBySession / CreateChatMessage
- [ ] **A2**:ClaimHandler 字段改接口;ChatHandler 字段改接口(postMessage 路径暂留 stores 兼容)
- [ ] **A3**:新增至少 4 测试:
  - `TestClaim_Success_Returns200_ClaimedBy`(mock session active + Redis SetNX 成功)
  - `TestClaim_AlreadyClaimed_Returns409`(mock SetNX 失败 + Redis.Get 返现有 owner)
  - `TestListMessages_Success_ReturnsArray`(mock 返多条消息)
  - `TestListMessages_Empty_ReturnsEmptyArray`(mock 返空,防 JSON null)

### 验证维度

- [ ] `go test ./...` 全绿
- [ ] auth.go claim 25% → **≥70%**
- [ ] chat.go listMessages 57% → **≥80%**
- [ ] api 包 31.8% → **≥34%**
- [ ] Mutation 抽样 2 项 KILLED

### 不在本切片范围

- chat.go postMessage happy path(需 requireClaimOwnership 重构,留 1ai-f)
- command.go handler 接口化(留 1ai-f)
- replay.go / session.go handler 接口化(留 1ai-g)

## 工时预算

| 项 | 工时 |
|---|---|
| 写 spec | 15min |
| 定义接口 + 重构 ClaimHandler/ChatHandler | 1h |
| 更新既有测试(claim_http_test.go 3 处) | 30min |
| 写 4 happy path 测试 + 2 新 mock | 1.5h |
| 跑测试 + mutation + 报告 + commit | 30min |
| **合计** | **~3.5h** |

## 完成后预期

- claim.go claim 25% → ≥70%
- chat.go listMessages 57% → ≥80%
- api 包 31.8% → ≥34%
- 1g 切片深度维持 🟢 touched(chat listMessages 行为级覆盖加深)
- 1k 切片深度维持 🟢 touched(claim happy path 行为级覆盖加深)

## Verification Depth Badge

🟢 touched
