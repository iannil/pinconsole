# 切片 1ai-f command-handler-interface — Spec(精简版)

**切片编号**:1ai-f
**类型**:重构 + 测试深化(api 包 Phase 4,精简)
**创建时间**:2026-06-20
**状态**:approved
**关联**:[1ai-e impl](../completed/2026-06-20-slice-1aie-claim-chat-interface-implementation.md)

## Context

1ai-e 完成 ClaimHandler + ChatHandler.listMessages 接口化。本切片扩展到 CommandHandler(postCommand),延续同模式。
postMessage/requireClaimOwnership 大重构影响面广,拆到 1ai-g。

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 范围 | A: 含 requireClaimOwnership + postMessage / B: 仅 CommandHandler 接口化 + postCommand 测试 / C: 仅 postCommand 拒绝测试 | **B** | A 大重构 ~5h;C 太浅;B 沿用 1ai-e claim.go 模式,~1.5h |
| 2 | postCommand 测试深度 | A: happy path(需 requireClaimOwnership mock) / B: 拒绝路径(not-claim-owner) | **B** | A 需 requireClaimOwnership 接口化(C 范围);B 用现有 1ah claim_http_test 的 CreateTestContext 模式 |
| 3 | CommandHandler 字段 | A: 全接口化 / B: 保留 stores(requireClaimOwnership 用) | **A** | 沿用 1ai-e claim.go 模式,字段全改接口;requireClaimOwnership 接受 stores 仍可工作 |

## Acceptance

### 必须满足

- [ ] **A1**:新增接口 `commandRepo`:CreateCoBrowsingCommand
- [ ] **A2**:CommandHandler 字段改接口(sessionRepo / redis / commandRepo / hub);NewCommandHandler 签名不变
- [ ] **A3**:postCommand 内 h.stores.X → h.sessionRepo.X / h.redis.X / h.commandRepo.X
- [ ] **A4**:既有 command_test.go / observability_wiring_test.go 同步(Stores 引用调整)

### 验证维度

- [ ] `go test ./...` 全绿(无 regression)
- [ ] command.go postCommand 覆盖率提升(原 0%)
- [ ] api 包 33.7% → ≥34%

### 不在本切片范围

- postMessage happy path(留 1ai-g,需 requireClaimOwnership 接口化)
- replay/session handler 接口化(留 1ai-g+)

## 工时预算

| 项 | 工时 |
|---|---|
| 写 spec | 10min |
| 接口化 CommandHandler + 同步既有测试 | 1h |
| 跑测试 + 报告 + commit | 20min |
| **合计** | **~1.5h** |

## 完成后预期

- CommandHandler 全字段接口化(模式 PoC 第 4 个 handler)
- 1e 切片深度维持 🟢 touched
- 为 1ai-g(requireClaimOwnership 重构)铺底

## Verification Depth Badge

🟢 touched
