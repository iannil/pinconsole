# 切片 1ah claim-chat-handler-tests — Implementation

**切片编号**:1ah
**类型**:测试深化(T2 backlog 续)
**创建时间**:2026-06-19
**状态**:completed
**关联**:[spec](./2026-06-19-slice-1ah-claim-chat-handler-tests-spec.md)、[1ag impl](../completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-implementation.md)

## Context

1ag 完成后 claim.go / chat.go 仍有显著缺口(getClaim 0% / listMessages 0% / Register 0%)。本切片同模式扩展,把 api 包覆盖推到接近 30%。

## 执行摘要

| 维度 | 结果 |
|---|---|
| 新增测试 | **10 个**(claim 7 + chat 3) |
| 新增测试文件 | 2(`claim_http_test.go`、`chat_http_test.go`) |
| api 包覆盖率 | **25.5% → 29.1%**(+3.6pp,稍低于 spec 目标 30%,因 claim.go `claim` happy path 需 PG session 检查) |
| Go 全测试 | ✅ ALL PASS(12 包) |
| Mutation 抽样 | ✅ 2/2 KILLED |

## 新增测试列表

### `claim_http_test.go`(7 测试,3 skip-on-no-Redis)

| 测试 | 验证 |
|---|---|
| `TestClaimRegister_Routes` | 3 路由正确注册(POST/POST/GET) |
| `TestClaim_InvalidUUID_Returns400` | POST /api/sessions/not-a-uuid/claim → 400 |
| `TestRelease_InvalidUUID_Returns400` | 同上 release |
| `TestGetClaim_InvalidUUID_Returns400` | 同上 getClaim |
| `TestGetClaim_NotClaimed_ReturnsFalse`(Redis) | 未 seed → 200 + claimed:false |
| `TestGetClaim_Claimed_ReturnsOwner`(Redis) | seed → 200 + claimed:true + owner UID |
| `TestRelease_NonOwner_Returns403`(Redis) | seed uid1,以 uid2 调 release → 403 + claim 未被误删 |

### `chat_http_test.go`(3 测试)

| 测试 | 验证 |
|---|---|
| `TestChatRegister_Routes` | 2 路由正确注册(GET/POST) |
| `TestListMessages_InvalidUUID_Returns400` | GET /api/sessions/not-a-uuid/messages → 400 |
| `TestListMessages_ValidUUIDPassesParsing` | 合法 UUID 通过(不返 400) |

## 覆盖率前后对比(1ag → 1ah)

### `internal/api/claim.go`

| 函数 | 1ag 后 | 1ah 后 |
|---|---|---|
| `Register` | 0% | **100%** |
| `claim` | 25% | 25%(happy path 需 PG,留 backlog) |
| `release` | 38.1% | **76.2%**(+non-owner 403 路径) |
| `getClaim` | 0% | **83.3%**(+UUID 拒绝 + claimed/not-claimed 路径) |

### `internal/api/chat.go`

| 函数 | 1ag 后 | 1ah 后 |
|---|---|---|
| `Register` | 0% | **100%** |
| `listMessages` | 0% | **57.1%**(+UUID 拒绝 + valid UUID 路径) |
| `postMessage` | 35% | 35%(requireClaimOwnership 在前,留 backlog) |

### api 包总体

**25.5% → 29.1%**(+3.6pp,首次接近 30%)

距 spec 目标 30% 差 0.9pp,主因 claim.go `claim` happy path 需 PG session 校验,留 1ai(storage 接口重构)解锁。

## Mutation 验证(抽样 2 项)

| Mutation | 应失败测试 | 结果 |
|---|---|---|
| claim.go `invalid_session_id` 400 → 200 | `TestGetClaim_InvalidUUID_Returns400` | ✅ KILLED |
| chat.go `invalid_session_id` 400 → 200 | `TestListMessages_InvalidUUID_Returns400` | ✅ KILLED |

## 关键模式

### `release` non-owner 测试用 CreateTestContext 注入 user_id

```go
c, _ := gin.CreateTestContext(w)
c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
c.Set("user_id", callerUID)  // 模拟 AuthMiddleware 注入
h.release(c)
```

不走真路由,因为 user_id 是 middleware 注入的(不在 URL/body)。直调 handler + 手设 context 是最干净的隔离方式。

### Redis 测试 key 隔离

每个 Redis 测试用 `uuid.New()` 生成唯一 sessionID + `defer rdb.Del(ctx, claimKey(sessionID))`,不污染其他测试或并发跑。

### `TestListMessages_ValidUUIDPassesParsing` 用 defer recover 兜底

合法 UUID 通过 parse 阶段,但下一步 PG 调用会 nil deref(stores 未注入)。用 `defer recover` 接住 panic,关键断言:不返 400。

这种"边界通过性"测试是有价值的——捕获未来把 `:id` 改成 `:session_id` 等 router 改名的回归。

## Verification Depth Badge

**🟢 touched** — claim/chat HTTP 入口有行为级覆盖,2 项 mutation 抽样 KILLED。

切片深度维持:
- **1g 弹窗 + 聊天**:🟢 touched(chat listMessages UUID 路径补全)
- **1k 安全阻断栈**:🟢 touched(getClaim claimed 状态 + release non-owner 403 补全)

## Follow-up(留 backlog)

1. **claim.go `claim` happy path** — 需 PG session seed(留下一切片需 storage 接口)
2. **chat.go `postMessage` happy path** — 需 requireClaimOwnership happy path + PG seed
3. **api 包覆盖破 30%** — 需 storage 接口重构(1ai)解锁 happy path

## 提交

建议拆 2 个 commit:

1. `test(1ah): claim handler 行为测试 — UUID 拒绝 + getClaim 状态 + release non-owner(7 测试)`
2. `test(1ah): chat handler 行为测试 — Register wireup + listMessages UUID(3 测试)`

## 下一步

### 立即可做

- 用户审阅 + commit 1ag + 1ah(共 4 个 commit)
- 把 1ah spec + impl 移到 `docs/reports/completed/`
- 更新 `project-status.md` 加 1ah 行

### 短期 backlog

- **1ai**:storage 接口重构(把 `*Postgres`/`*Redis`/`*MinIO` 改 interface,解锁 happy path mock 注入,~5-8h)— 这一步能把 api 覆盖推到 50%+
- **1aj**:补 1ag follow-up bugs(parseSince 负数、TestCheckWSRateLimit flaky,~1.5h)
- 修 `parseSince("-1d")` 接受负数(~30min)
- 排查 `TestCheckWSRateLimit` flaky(~1h)

### 长期(post-v1)

- 自定义域名 / 页面编辑器 / Tauri
- Mutation CI 集成(R7)
