# 切片 1ai-d me-logout-happy-path — Spec

**切片编号**:1ai-d
**类型**:测试深化(api 包 Phase 2,1ai-c 续做)
**创建时间**:2026-06-20
**状态**:approved
**关联**:[1ai-c impl](../completed/2026-06-20-slice-1aic-auth-handler-interface-implementation.md)

## Context

1ai-c Phase 1 给了 login happy path,但 me 与 logout 仍是浅覆盖:
- me handler 当前 40%(no user_id → 401 路径已测;happy path 未测)
- logout handler 当前 85.7%(set-cookie 已测,但 Redis.Del 调用未断言)

本切片同模式扩展,补 me/logout happy path,完成 AuthHandler 全覆盖。

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 范围 | A: 仅 me / B: me + logout / C: + NewAuthHandler | **B** | A 太窄;C 的 NewAuthHandler 需 *storage.Stores 构造,简单但偏离 happy path 主题 |
| 2 | 接口扩展 | A: 不动 / B: 加 userRepo mock 已足够 | **A** | 1ai-c 已加 authUserRepo/authRedisStore,本切片不需新接口 |
| 3 | logout 测试深度 | A: 仅测 200 + cookie 清 / B: + Redis.Del 被调断言 | **B** | 1ag 已测 200+cookie;B 增量验证 session 失效副作用(防 logout 不删 Redis 的回归) |
| 4 | me 边界用例 | A: 仅 happy / B: happy + user-not-found | **B** | me 返 401 user_not_found 路径(用户被删但 cookie 仍有效)未覆盖 |

## Acceptance

### 必须满足

- [ ] **A1**:新增至少 3 测试:
  - `TestMe_Success_Returns200_Body` — 注入 user_id + mock userRepo 返 user → 200 + meResponse
  - `TestMe_UserNotFound_Returns401` — mock userRepo 返 error → 401 user_not_found
  - `TestLogout_DeletesRedisSession` — 带 cookie 的请求 → Redis.Del×1 + 200 + cookie 清

### 验证维度

- [ ] `go test ./...` 全绿
- [ ] auth.go `me` 覆盖率:40% → **≥80%**
- [ ] auth.go `logout` 覆盖率:85.7% → **100%**
- [ ] Mutation 抽样 1 项 KILLED

### 不在本切片范围

- NewAuthHandler 覆盖(简单但偏离 happy path 主题)
- claim/chat/replay handler 接口化(留 1ai-e/f)

## 工时预算

| 项 | 工时 |
|---|---|
| 写 spec | 10min |
| 3 测试 + 复用 1ai-c mock | 30min |
| 跑 + 报告 + commit | 20min |
| **合计** | **~1h** |

## 完成后预期

- auth.go me 40% → ≥80%
- auth.go logout 85.7% → 100%
- api 包覆盖 31.2% → ~32%
- AuthHandler 全部函数 happy path 覆盖完成

## Verification Depth Badge

🟢 touched
