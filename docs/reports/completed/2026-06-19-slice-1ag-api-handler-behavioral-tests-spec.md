# 切片 1ag api-handler-behavioral-tests — Spec

**切片编号**:1ag
**类型**:测试深化(T2 backlog 子集)
**创建时间**:2026-06-19
**状态**:approved
**关联**:[覆盖率评估](../../docs/audits/)、[1ad T2 backlog](../completed/2026-06-19-slice-1ad-implementation.md)、[1af impl](../completed/2026-06-19-slice-1af-implementation.md)

## Context

为什么做这个切片?

- **触发原因**:2026-06-19 跑覆盖率评估,`internal/api` 包仅 20.0% statements;login/logout/me/getSessionReplay/listEndedSessions/parseSince 等 HTTP handler 几乎全 0%(仅靠 e2e 兜底)。
- **业务/技术价值**:auth/replay 是 1h/1d 切片核心路径。补 handler 级行为测试,把 api 包覆盖 20% → 35%+,捕获 binding/cookie/UUID/since 解析等回归;e2e 慢、跑得少,单测提供快速反馈。
- **不做的代价**:post-v1 重构(自定义域名、Tauri)改 router/handler 时,如果 e2e 没跑(默认 CI 只跑单测),auth/replay 行为退化无人发现。

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 范围 | A: 5 个 handler 全做 / B: 只做 auth+replay / C: 只做 auth | **B** | auth+replay 是 1h+1d 核心安全/回放路径,ROI 最高;claim/privacy/chat 留 1ah;A 工时 ~6h 超 slice budget;C 太小 |
| 2 | Mock 策略 | A: 引入 pgxmock+miniredis / B: 真 Redis + skip(沿用 1x 模式) / C: 只测不需 stores 的路径 | **C+B** | C 路径(invalid JSON/UUID、cookie、parseSince)覆盖最多、零依赖、最稳定;B 路径(login 锁定)沿用 1x 已验证的真 Redis + skip 模式;A 引入新依赖违反 LLM friendly 与"不过度工程" |
| 3 | 测试文件 | A: 拼到 auth_test.go / B: 新文件 auth_http_test.go | **B** | auth_test.go 已是 1x throttle 主题,新文件清晰隔离 HTTP 层测试;便于 1ah 复用模式 |
| 4 | 是否升级 storage mock | A: 把 *Postgres 改接口 / B: 不动 | **B** | 接口重构是大改动,影响所有 repo 调用方;本切片只补 handler 测试,不动 storage 抽象;留 1ah+ backlog |
| 5 | 覆盖率目标 | A: ≥50% / B: 35-40% / C: 不设硬指标 | **B** | A 需大量 stores 集成测试,超 budget;B 是 reachable 通过 11 个新测试;C 缺乏可验证判据 |

## Acceptance(可验证的成功标准)

切片"完成"的可验证判据。每条都用客观断言。

### 必须满足

- [ ] **A1**:新增 `server/internal/api/auth_http_test.go`,含至少 5 个测试:
  - `TestLogin_InvalidJSON_Returns400` — POST 非法 JSON → status 400
  - `TestLogin_MissingFields_Returns400` — POST `{"email":""}` → status 400(binding required)
  - `TestMe_NoUserIDContext_Returns401` — 不注入 user_id → status 401 + `error: not_authenticated`
  - `TestLogout_ClearsCookie` — 任意调用 → Set-Cookie MaxAge<0 + status 200
  - `TestSetSessionCookie_Attributes` — 真 recorder 验证 `SameSite=Lax`/`HttpOnly`/`Secure` flagging(secureCookie true/false 两路)
- [ ] **A2**:新增 `server/internal/api/replay_http_test.go`,含至少 6 个测试:
  - `TestGetSessionReplay_InvalidUUID_Returns400` — `/api/sessions/not-a-uuid/replay` → 400
  - `TestListEndedSessions_InvalidSince_Returns400` — `?since=xyz` → 400
  - `TestParseSince_DefaultsTo24h` — 空字符串 → 24h
  - `TestParseSince_HoursAndDays` — `7d`/`12h` 正确
  - `TestParseSince_InvalidUnit_ReturnsError` — `7w` → error
  - `TestParseSince_TooShort_ReturnsError` — `x` → error
- [ ] **A3**(可 skip):`TestLogin_Locked_Returns429_WithRetryAfter` — 真 Redis seed throttle 计数到 loginMaxAttempts,POST login → 429 + `Retry-After` header + body `too_many_attempts`

### 验证维度

- [ ] `cd server && go test ./internal/api/...` 全绿(预计 +11 测试,82 → 93)
- [ ] `go test ./...` 全绿(无 regression)
- [ ] api 包覆盖率:20.0% → **≥30%**(纯路径 + cookie + parseSince)
- [ ] mutation 抽样验证:把 `c.JSON(http.StatusBadRequest, ...)` 改成 `StatusOK` → 对应测试必失败(2 项抽样)
- [ ] `gofmt -l` + `go vet` 干净

### 不在本切片范围

- claim.go / privacy.go / chat.go handler 测试 → 1ah
- login happy path(需 PG seed + bcrypt hash)→ 留 backlog,需先决定 pgxmock vs testcontainers
- getSessionReplay happy path(需 MinIO + PG seed)→ 同上
- storage `*Postgres` 接口重构 → 1ai+ backlog

## 工时预算

| 项 | 工时 |
|---|---|
| 写 spec(本文件)+ 任务拆分 | 20min |
| auth_http_test.go 5 测试 | 1h |
| replay_http_test.go 6 测试 + A3 Redis 锁定 | 1h |
| 跑测试 + 修突变 + 覆盖率对比 | 30min |
| 写 implementation + 更新 project-status | 30min |
| **合计** | **~3.5h** |

## 完成后预期

- api 包覆盖率 20.0% → ≥30%
- 11 个新测试,无 flaky(9 个零依赖 + 1 个 Redis 可 skip + 1 个抽样 mutation)
- T2 backlog 关闭 2 项(auth HTTP + replay HTTP)
- 1h 切片深度:🟡 verified-shallow → 🟢 touched(若 mutation 抽样通过)
- 1d 切片深度:🟡 verified-shallow → 🟢 touched(同上)

## Verification Depth Badge 目标

🟢 touched — auth.go + replay.go HTTP 入口有行为级测试,mutation 抽样 KILLED。

升 🟢 strict 需 happy-path 集成测试 + mutation CI(R7),留 post-v1。

## 关联

- 前置:1af(test-health-deepening 已升整体 verdict 🟢)
- 触发:本会话覆盖率评估
- 后续:1ah claim/privacy/chat handler 测试;1ai storage 接口重构(让 happy-path 测试可行)
