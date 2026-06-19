# 切片 1ag api-handler-behavioral-tests — Implementation

**切片编号**:1ag
**类型**:测试深化(T2 backlog 子集)
**创建时间**:2026-06-19
**状态**:completed
**关联**:[spec](./2026-06-19-slice-1ag-api-handler-behavioral-tests-spec.md)、[1ad T2 backlog](../completed/2026-06-19-slice-1ad-implementation.md)、[1af impl](../completed/2026-06-19-slice-1af-implementation.md)

## Context

1af 完成后,api 包仅 20% 覆盖率。本切片给 auth.go + replay.go 的 HTTP 入口补 handler 级行为测试,把 api 覆盖拉到 ≥25%,捕获 binding/cookie/UUID/since 解析等回归。

## 执行摘要

| 维度 | 结果 |
|---|---|
| 新增测试 | **12 个**(auth 5 + replay 7) |
| 新增测试文件 | 2(`auth_http_test.go`、`replay_http_test.go`) |
| api 包覆盖率 | **20.0% → 25.5%**(+5.5pp,稍低于 spec 目标 30%,因 happy path 需 PG+MinIO seed 留 backlog) |
| Go 全测试 | ✅ ALL PASS(12 包) |
| Mutation 抽样 | ✅ 2/2 KILLED |
| 模式 | gin engine + httptest.Recorder(走真路由,捕获 binding) |

## 新增测试列表

### `auth_http_test.go`(5 测试,1 skip-on-no-Redis)

| 测试 | 验证 |
|---|---|
| `TestLogin_InvalidJSON_Returns400` | POST `{not-json` → 400 + `invalid_json` |
| `TestLogin_MissingFields_Returns400`(4 子用例) | 缺 password / 缺 email / 空值 → 400 |
| `TestLogin_Locked_Returns429_WithRetryAfter`(Redis) | seed throttle 计数到阈值 → 429 + Retry-After header + too_many_attempts |
| `TestMe_NoUserIDContext_Returns401` | 未注入 user_id → 401 + not_authenticated |
| `TestLogout_ClearsCookie_AndReturns200` | Set-Cookie value 清空 + status 200 |

### `replay_http_test.go`(7 测试)

| 测试 | 验证 |
|---|---|
| `TestParseSince_DefaultsTo24h` | 空字符串 → 24h |
| `TestParseSince_HoursAndDays`(5 子用例) | 1h/12h/1d/7d/30d |
| `TestParseSince_InvalidUnit_ReturnsError`(5 子用例) | 7w/30m/60s/1y/1x → error |
| `TestParseSince_TooShort_ReturnsError`(2 子用例) | "x"/"1" → error |
| `TestParseSince_NonNumericPrefix_ReturnsError`(2 子用例) | abd/7.5h → error |
| `TestGetSessionReplay_InvalidUUID_Returns400` | 非 UUID → 400 + invalid_session_id |
| `TestGetSessionReplay_ValidUUIDPassesParsing` | 合法 UUID 通过(不返 400) |
| `TestListEndedSessions_InvalidSince_Returns400` | since=invalid → 400 + invalid_since |
| `TestListEndedSessions_ValidSincePassesParsing` | since=7d 通过(不返 400) |

## 覆盖率前后对比

### `internal/api/auth.go`

| 函数 | Before | After |
|---|---|---|
| `Register` | 0% | **100%** |
| `login` | 0% | **37.8%**(invalid JSON + missing fields + locked 路径) |
| `logout` | 0% | **85.7%** |
| `me` | 0% | **40%**(no user_id 路径) |
| `setSessionCookie` | 100% | 100%(1ae 已覆盖) |

### `internal/api/replay.go`

| 函数 | Before | After |
|---|---|---|
| `Register` | 0% | **100%** |
| `listEndedSessions` | 0% | **35.5%**(invalid since + valid since 路径) |
| `getSessionReplay` | 0% | **17.8%**(invalid UUID + valid UUID 路径) |
| `parseSince` | 0% | **100%** |

### api 包总体

**20.0% → 25.5%**(+5.5pp)

未达 spec 目标 30% — 因 login/logout/me happy path 与 listEndedSessions/getSessionReplay happy path 需要 PG + MinIO seed,留 backlog(需先决定 pgxmock vs testcontainers,见 §Follow-up)。

## Mutation 验证(抽样 2 项)

| Mutation | 应失败测试 | 结果 |
|---|---|---|
| auth.go `invalid_json` 400 → 200 | `TestLogin_InvalidJSON_Returns400` | ✅ KILLED |
| replay.go `invalid_session_id` 400 → 200 | `TestGetSessionReplay_InvalidUUID_Returns400` | ✅ KILLED |

## 模式说明

### 为什么用 `r.ServeHTTP` 而非 `gin.CreateTestContext` 直调 handler

- 走真路由:`r.POST("/api/auth/login", h.login)` + `r.ServeHTTP(w, req)`
- 触发 gin 真 binding:`ShouldBindJSON` 在路由层执行
- 更接近生产:捕获未来重构把 `Register` 改错路由、把 binding tag 改坏等回归

### 为什么 Redis 测试用真 Redis + skip 而非 miniredis

- 沿用项目 1x 既定模式(`helperRedisIfAvailable`)
- 不引入新依赖(alicebob/miniredis)— LLM friendly 原则
- Redis 测试 key 用唯一 email+IP(`1ag-locked@example.com` + `10.99.99.42`)+ `defer rdb.Del`,不污染其他测试

### 为什么 happy path 留 backlog

- `*Postgres` 和 `*Redis` 是具体类型不是接口,无法直接 mock
- 引入 pgxmock 是 1ai 范围(需重构 storage 接口)
- 引入 testcontainers 启动 PG 太慢(单测应 <10s)
- happy path 已被 e2e 覆盖(`1h-auth.spec.ts`、`1d-replay.spec.ts`),ROI 较低

## Verification Depth Badge

**🟢 touched** — auth.go + replay.go HTTP 入口有行为级测试,2 项 mutation 抽样 KILLED。

升 🟢 strict 需:
- happy path 集成测试(需 storage 接口重构或 testcontainers)
- Mutation CI 集成(R7)
- 这两项留 post-v1 backlog

切片深度更新:
- **1h 认证 + 多运营 后端**:🟡 → **🟢 touched**(login/logout/me 路径行为级覆盖)
- **1d 录像归档**:🟡 → **🟢 touched**(parseSince + UUID 拒绝路径覆盖)

## Follow-up(留 backlog)

1. **`parseSince("-1d")` 接受负数** — 应拒绝(Atoi("-1") 不报错,产生 -24h 负 duration)。本切片不修(避免改 handler 代码),记入 backlog。
2. **api 包覆盖率 30%+** — 需 happy path 集成测试。
3. **storage `*Postgres` 接口重构** — 让 mock 注入可行(参考 1ae R2 引入 `PgxPool` interface 模式)。
4. **claim/privacy/chat handler 测试** — 下一切片 1ah。
5. **`TestCheckWSRateLimit_OverMsgCount` flaky**(pre-existing,本切片未引入) — 跑 `go test ./...` 偶发失败,单独跑 PASS,跟 Redis 共享 key 污染相关。需独立排查。

## 提交

建议拆 2 个 commit(未自动 commit,留给用户审阅):

1. `test(1ag): auth.go HTTP 行为测试 — invalid JSON/missing fields/locked/logout/me(5 测试)`
2. `test(1ag): replay.go HTTP 行为测试 — parseSince 全覆盖 + UUID/since 拒绝路径(7 测试)`

## 下一步

### 立即可做

- 用户审阅 + commit(2 个建议拆分)
- 把 spec + impl 移到 `docs/reports/completed/`
- 更新 `project-status.md` §5 加 1ag 行 + 把 1d/1h 升 🟢 touched

### 短期 backlog

- 1ah claim/privacy/chat handler 测试(同模式,~3h)
- 1ai storage 接口重构(让 happy path 测试可行,~5-8h)
- 修复 `parseSince` 负数 bug(~30min)
- 排查 `TestCheckWSRateLimit` flaky(~1h)

### 长期(post-v1)

- 自定义域名 / 页面编辑器 / Tauri
- Mutation CI 集成(R7)
