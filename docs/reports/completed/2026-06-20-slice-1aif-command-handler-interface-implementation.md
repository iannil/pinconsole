# 切片 1ai-f command-handler-interface — Implementation(精简版)

**切片编号**:1ai-f
**类型**:重构 + 测试深化(api 包 Phase 4)
**创建时间**:2026-06-20
**状态**:completed
**关联**:[spec](./2026-06-20-slice-1aif-command-handler-interface-spec.md)、[1ai-e impl](../completed/2026-06-20-slice-1aie-claim-chat-interface-implementation.md)

## 执行摘要

| 维度 | 结果 |
|---|---|
| 新增测试 | **4 个**(Register wireup + no-user 401 + not-owner 403 + 字段契约) |
| 新增接口 | 1(`commandRepo`) |
| CommandHandler 字段 | 加 sessionRepo / redis / commandRepo 接口(stores 保留供 requireClaimOwnership) |
| command.go postCommand | 0% → **16.0%**(requireClaimOwnership 失败路径) |
| api 包 | 33.7% → **33.8%** |
| Mutation | ✅ 1/1 KILLED |

## 重构内容

### `claim_chat_interfaces.go` 追加 `commandRepo`

```go
type commandRepo interface {
    CreateCoBrowsingCommand(ctx, cmd CoBrowsingCommand) (*CoBrowsingCommand, error)
}
```

### `command.go` 字段重构

```go
type CommandHandler struct {
    stores         *storage.Stores    // 留给 requireClaimOwnership(1ai-g 再删)
    sessionRepo    claimSessionRepo   // 复用 1ai-e 接口
    redis          claimRedisStore     // 复用 1ai-e 接口
    commandRepo    commandRepo
    hub            CommandHub
    logger         *slog.Logger
    allowedDomains []string
}
```

NewCommandHandler 签名不变,内部抽取。
postCommand 内 `h.stores.PG.CreateCoBrowsingCommand` → `h.commandRepo.CreateCoBrowsingCommand`。

## 新增测试

| 测试 | 验证 |
|---|---|
| `TestCommandRegister_Routes` | 1 路由 wireup(POST /api/sessions/:id/command) |
| `TestPostCommand_NoUserID_Returns401` | 无 user_id → 401 not_authenticated |
| `TestPostCommand_NotClaimOwner_Returns403` | seed Redis claim as ownerUID,以 callerUID 调 → 403 + claim 未被改 |
| `TestCommandHandler_FieldsContract` | 编译时契约:防字段误删 |

## Mutation KILLED

- authz.go `not_claim_owner` 403 → 200 → `TestPostCommand_NotClaimOwner_Returns403` 失败

## Verification Depth Badge

🟢 touched — CommandHandler 接口化 + 拒绝路径覆盖。

## Follow-up

- **1ai-g**:requireClaimOwnership 接口化 → 解锁 postMessage + postCommand happy path
- replay/session handler 接口化

## 提交

2 个 commit:
1. `refactor(1ai-f): CommandHandler 接口化 — 4 接口字段(保留 stores 给 requireClaimOwnership)`
2. `test(1ai-f): postCommand 拒绝路径 — Register/no-user 401/not-owner 403(3 测试)`
