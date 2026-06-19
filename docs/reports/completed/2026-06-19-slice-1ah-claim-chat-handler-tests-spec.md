# 切片 1ah claim-chat-handler-tests — Spec

**切片编号**:1ah
**类型**:测试深化(T2 backlog 续)
**创建时间**:2026-06-19
**状态**:approved
**关联**:[1ag impl](../completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-implementation.md)、[1ad T2 backlog](../completed/2026-06-19-slice-1ad-implementation.md)

## Context

为什么做这个切片?

- **触发原因**:1ag 完成后跑覆盖率,claim.go / chat.go 仍有显著缺口:
  - claim.go:`Register` 0% / `getClaim` 0% / `claim` 25% / `release` 38%
  - chat.go:`Register` 0% / `listMessages` 0%
  - privacy.go 已被 1t/1ac 充分覆盖(getConsent 40% / postConsent 46% / deleteVisitor 68%),本切片不重做
- **业务/技术价值**:claim 是 1k P0-4 race-safety 的关键路径,getClaim 完全无测试意味着前端轮询 claim 状态的回归无保护;chat listMessages 是 admin 看历史聊天的入口,UUID 校验回归无保护。
- **不做的代价**:post-v1 改 router/UUID 校验时(如把 `:id` 改成 `:session_id`),claim/chat 入口退化无人发现。

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 范围 | A: claim+chat+privacy 全做 / B: claim+chat / C: 仅 claim | **B** | privacy 已被 1t/1ac 覆盖(68% deleteVisitor),ROI 低;A 工时翻倍但增量覆盖率小;C 漏 chat listMessages |
| 2 | 测试深度 | A: 仅 UUID 拒绝路径(零依赖) / B: UUID + Redis 行为级 / C: + PG seed happy path | **B** | A 太浅(getClaim 仍 0%);C 需 PG fixture 超 budget;B 沿用 1x/1ag 既定 Redis+skip 模式 |
| 3 | getClaim 测试 | A: 只测 UUID 拒绝 / B: + claimed/not-claimed 状态 / C: + 拒绝路径(无 Redis 错误) | **B** | A 漏核心业务状态;C 需 stub store interface;B 用真 Redis seed 最自然 |
| 4 | release 测试 | A: 不测 / B: 仅 UUID / C: + owner/non-owner 行为 | **C** | release 是 1k P0-5 核心(owner-only),目前 38% 主要被 1ac Lua 原语覆盖,handler 层 non-owner 403 路径无测试 |
| 5 | postMessage 测试 | A: 不测 / B: 测 | **A** | postMessage 第一行就是 `requireClaimOwnership`(需 Redis+PG),binding 在其后,无零依赖路径;留 backlog |

## Acceptance(可验证的成功标准)

### 必须满足

- [ ] **A1**:新增 `claim_http_test.go`,含至少 6 个测试:
  - `TestClaimRegister_Routes` — POST/GET 3 路由正确注册
  - `TestClaim_InvalidUUID_Returns400` — POST /api/sessions/not-a-uuid/claim → 400
  - `TestRelease_InvalidUUID_Returns400` — 同上 release
  - `TestGetClaim_InvalidUUID_Returns400` — 同上 getClaim
  - `TestGetClaim_NotClaimed_ReturnsFalse`(Redis) — 未 seed → 200 + claimed:false
  - `TestGetClaim_Claimed_ReturnsOwner`(Redis) — seed → 200 + claimed:true + claimed_by
  - `TestRelease_NonOwner_Returns403`(Redis) — seed uid1,POST release uid2 → 403 not_claim_owner
- [ ] **A2**:新增 `chat_http_test.go`,含至少 2 个测试:
  - `TestChatRegister_Routes` — GET/POST 2 路由正确注册
  - `TestListMessages_InvalidUUID_Returns400` — GET /api/sessions/not-a-uuid/messages → 400

### 验证维度

- [ ] `go test ./internal/api/` 全绿(预计 +9 测试)
- [ ] `go test ./...` 全绿(无 regression)
- [ ] api 包覆盖率:25.5% → **≥30%**(claim + chat 路径覆盖提升)
- [ ] claim.go 覆盖率:`getClaim` 0% → ≥70%、`release` 38% → ≥60%
- [ ] chat.go 覆盖率:`listMessages` 0% → ≥30%
- [ ] Mutation 抽样 2 项 KILLED

### 不在本切片范围

- privacy.go 测试 — 已被 1t/1ac/1l 充分覆盖
- postMessage handler 测试 — 需 requireClaimOwnership happy path(留 1ai)
- claim happy path(PG session 校验) — 需 PG seed(留 backlog)
- chat listMessages happy path — 需 PG(留 backlog)

## 工时预算

| 项 | 工时 |
|---|---|
| 写 spec(本文件) | 15min |
| claim_http_test.go 6 测试 | 1.5h |
| chat_http_test.go 2 测试 | 30min |
| 跑测试 + mutation + 覆盖率 | 30min |
| 写 impl + 更新 status | 30min |
| **合计** | **~3.5h** |

## 完成后预期

- api 包覆盖率 25.5% → ≥30%(首次破 30%)
- claim.go 覆盖率:25%/38%/0%(claim/release/getClaim) → 显著提升
- chat.go 覆盖率:listMessages 0% → ≥30%
- 1k 切片深度:🟢 touched → 维持(handler 层加深)
- 1g 切片深度:🟢 touched → 维持

## Verification Depth Badge 目标

🟢 touched — claim/chat HTTP 入口有行为级覆盖,mutation 抽样 KILLED。

## 关联

- 前置:1ag(auth/replay handler 测试)
- 后续:1ai(storage 接口重构,解锁 happy path 集成测试)
