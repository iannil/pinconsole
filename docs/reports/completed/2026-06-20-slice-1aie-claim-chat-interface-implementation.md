# 切片 1ai-e claim-chat-interface — Implementation

**切片编号**:1ai-e
**类型**:重构 + 测试深化(api 包 Phase 3)
**创建时间**:2026-06-20
**状态**:completed
**关联**:[spec](./2026-06-20-slice-1aie-claim-chat-interface-spec.md)、[1ai-d impl](../completed/2026-06-20-slice-1aid-me-logout-happy-path-implementation.md)

## 执行摘要

| 维度 | 结果 |
|---|---|
| 新增测试 | **4 个**(claim success/already-claimed + listMessages success/empty) |
| 新增文件 | 2(`claim_chat_interfaces.go`、`claim_chat_happy_path_test.go`) |
| 接口 | 3 个新(claimSessionRepo / claimRedisStore / chatMessageRepo) |
| 新 mock | 3 个(mockSessionRepo + mockChatMessageRepo + mockRedisStoreForClaim) |
| api 包覆盖率 | 31.8% → **33.7%** |
| auth.go claim | 25% → **78.1%** |
| chat.go listMessages | 57% → **76.2%** |
| Mutation | ✅ 2/2 KILLED |

## 重构内容

### `claim_chat_interfaces.go`(新)

3 接口(ISP 原则):
- `claimSessionRepo`:GetSession(ClaimHandler.claim 用)
- `claimRedisStore`:Get / SetNX / EvalLua(同时满足 *storage.Redis)
- `chatMessageRepo`:ListChatMessagesBySession(ChatHandler.listMessages 用)

### `claim.go`

字段重构:
```go
type ClaimHandler struct {
    sessionRepo claimSessionRepo
    redis       claimRedisStore
    logger      *slog.Logger
}
```
NewClaimHandler 签名不变,内部抽取。

### `chat.go`(混合模式)

ChatHandler 同时保留 `stores`(postMessage 用)+ 新增 `messageRepo`(listMessages 用)。
postMessage 接口化留 1ai-f(需 requireClaimOwnership 重构)。

### 既有测试同步

- `claim_http_test.go` 3 处:`stores: &storage.Stores{Redis: ...}` → `redis: &storage.Redis{...}`
- `observability_wiring_test.go`:误改的 CommandHandler 引用回滚

## 新增测试

| 测试 | 验证 |
|---|---|
| `TestClaim_Success_Returns200_ClaimedBy` | mock session active + SetNX 成功 → 200 + claimed_by |
| `TestClaim_AlreadyClaimed_Returns409` | SetNX 失败 + 现有 owner → 409 already_claimed + owner 保留(1k P0-4 race-safety) |
| `TestListMessages_Success_ReturnsArray` | mock 多条消息 → 200 + JSON array |
| `TestListMessages_Empty_ReturnsEmptyArray` | mock 空 → 200 + `messages:[]`(防 JSON null 让 admin 前端崩) |

## 覆盖率前后

| 函数 | 1ai-d 后 | 1ai-e 后 |
|---|---|---|
| `claim` | 25% | **78.1%** |
| `release` | 38% | 76.2%(non-owner 测试带动) |
| `getClaim` | 0% | 83.3%(1ah 已覆盖) |
| `listMessages` | 57% | **76.2%** |

api 包:**31.8% → 33.7%**

## Mutation KILLED

1. claim.go `"already_claimed"` → `"claim_acquired"` → TestClaim_AlreadyClaimed 失败
2. chat.go `Sender: m.Sender` → `Sender: "system"` → TestListMessages_Success 失败(返 "system" 而非真实 sender)

## Verification Depth Badge

🟢 touched — ClaimHandler + ChatHandler.listMessages happy path 全覆盖,接口化模式 PoC 在 auth/claim/chat 三 handler 验证。

切片深度:
- **1g 弹窗 + 聊天**:🟢 touched(chat listMessages happy path)
- **1k 安全阻断栈**:🟢 touched(claim race-safety happy path)

## Follow-up

- chat.go postMessage happy path — 需 requireClaimOwnership 接口化(1ai-f)
- command.go handler 接口化(1ai-f)
- replay.go / session.go handler 接口化(1ai-g)

## 提交

3 个 commit:
1. `refactor(1ai-e): claim_chat_interfaces.go + ClaimHandler/ChatHandler 字段重构`
2. `test(1ai-e): claim+chat happy path — claim 78%、listMessages 76%(4 测试 + 3 mock)`
3. `docs: 同步 1ai-e 完成`
