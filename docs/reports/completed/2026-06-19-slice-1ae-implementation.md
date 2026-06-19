# 切片 1ae test-health-fix — 完成报告

**报告日期**:2026-06-19
**Spec**:[`docs/progress/2026-06-19-slice-1ae-spec.md`](../progress/2026-06-19-slice-1ae-spec.md) → 移到 completed
**关联审计**:[`docs/audits/2026-06-19-test-health-audit.md`](../audits/2026-06-19-test-health-audit.md)
**Verification Depth**:🟢 touched(7 项修复 + 重跑 mutation 全部 KILLED + e2e flaky 0%)

## 范围与状态

### ✅ 全部 9 项完成

| ID | 内容 | 文件 |
|---|---|---|
| **R5** | T0-1h-ui-3 虚标修复:mount 真 LoginView + i18n + DOM 断言 | `admin/tests/session-expired.test.ts` |
| **R1** | operatorWS 行为级测试:真调 handler + 断言 401 + 无 Upgrade header | `server/internal/api/ws_auth_test.go` |
| **R2** | erasure CASCADE 隔离测试:PgxPool interface + txAsPool + ALTER FK NO ACTION | `server/internal/storage/erasure_test.go` + `postgres.go` |
| **R3a** | auth_bcrypt 行为级测试:真 bcrypt 校验 + 参数顺序反模式检测 | `server/internal/api/auth_bcrypt_test.go` |
| **R3b** | auth_cookie 行为级测试:抽 setSessionCookie helper + httptest recorder 真 Set-Cookie header 断言 | `server/internal/api/auth.go` + `auth_cookie_test.go` |
| **R3c** | migrations 行为级测试:failingPool mock + 真调 runMigrations fail-fast | `server/cmd/server/migrations_test.go` |
| **R3d** | flusher 行为级测试:真 Redis/MinIO/PG 集成,验证 MinIO 上传 + Redis XTRIM + PG 行写入 | `server/internal/recording/flusher_wiring_test.go` |
| **R3e** | observability 行为级测试:真调 Lifecycle + buffer logger + 验证日志产出 + 真 ClaimHandler 触发 | `server/internal/api/observability_wiring_test.go` |
| **R4** | e2e 1d-replay flaky 修复:移除 silent skip + polling 等 flusher(40s deadline) | `e2e/tests/1d-replay.spec.ts` |

## 验收证据

### 测试统计

| 套件 | 之前 | 之后 | 增量 |
|---|---|---|---|
| Go test(12 包) | 12 包 ALL PASS | 12 包 ALL PASS | +11 测试 |
| TS test(admin) | 77 | 78 | +1 |
| TS test(SDK) | 60 | 60 | 0 |
| e2e 1d-replay(scene 3) | 1/5(20% flaky) | **5/5(0% flaky)** | ✅ 修复 |

### Mutation cross-check 重跑(关键验收)

| Mutation | 描述 | 之前(1ae 前) | 之后(1ae 后) |
|---|---|---|---|
| **M1** | authz owner check `!=` → `==` | KILLED | ✅ KILLED |
| **M2** | release Lua owner-only `== ARGV[1]` → `if true` | KILLED | ✅ KILLED |
| **M3** | erasure 跳过 `DELETE FROM chat_messages` | ❌ **SURVIVED** | ✅ **KILLED**(R2 修复 — FK violation 触发) |
| **M4** | operatorWS auth 包进 `if false` | ❌ **SURVIVED** | ✅ **KILLED**(R1 修复 — handler panic) |
| **M5** | rate limit fail-open → fail-closed | KILLED(半,NoPanic 弱) | ✅ KILLED |
| **M6** | rrweb maxRetries 3 → 0 | KILLED | ✅ KILLED |
| **M7** | rrweb maskAllInputs true → false | KILLED | ✅ KILLED |

**Mutation Score:5/7 = 71.4% → 7/7 = 100%** 🎉

### Badge 升级

| 维度 | 1ae 前 | 1ae 后 |
|---|---|---|
| **D1** Badge 准确性 | 38.2% PASS(26/68) | ~55% PASS(估算 +10 项升 PASS + R5 虚标修复) |
| **D2** 弱断言数 | 20 | ~15(R5 移除 1 处虚标 + R3d/R3e 升级附带清理) |
| **D3** Mutation Score | 71.4% | **100%**(7/7 killed) |
| **D4** Flakiness | e2e 20% | **0%**(e2e scene 3 5/5 pass) |

**整体 verdict:🔴 → 🟡**(D2/D3/D4 全过,D1 部分提升但未到 80%)

## 关键技术决策

### D1 — PgxPool interface 重构(R2 支撑)

把 `Postgres.Pool` 从 `*pgxpool.Pool` 改为 `PgxPool interface`,5+1 个方法(Exec/Query/QueryRow/Begin/Ping/Close)。

**理由**:让测试可注入 `txAsPool{tx: tx}` 包装器,在事务内 ALTER FK NO ACTION,强制 erasure 函数自己执行显式 DELETE(不能依赖 PG CASCADE)。

**影响**:
- 改 1 个 production file:`storage/postgres.go`
- 改 1 个 caller:`cmd/server/migrations.go`(signature 从 `*pgxpool.Pool` → `storage.PgxPool`)
- 12 个 `&Postgres{Pool: pool}` 调用点自动兼容(*pgxpool.Pool 满足 interface)

### D2 — setSessionCookie helper 抽取(R3b 支撑)

从 `auth.go` login/logout 抽出 `setSessionCookie(c, sessionID, maxAge)` 方法。

**理由**:让测试可真调此方法 + 断言 Set-Cookie header 属性(Secure/HttpOnly/SameSite/MaxAge),不依赖 grep 字符串。

**影响**:
- 改 1 个 production file:`api/auth.go`(login + logout 都用 helper)
- 加 2 个测试:prod mode + dev mode 行为

### D3 — e2e polling 替代固定 wait(R4 修复)

把 `await new Promise(r => setTimeout(r, 2000))` 改为 polling loop(每 2s 查一次,40s deadline)。

**理由**:flusher 默认 30s 间隔,固定 2s wait 不足以保证 events 已 flush 到 MinIO。polling 让测试在 events 真就绪时通过,稳定且快(平均 5s 通过)。

**附带修复**:移除 `if (sessionsWithEvents.length === 0) return;` silent skip(audit D2 反模式 3)。现在测试 fail loudly 如果没有 session with events。

## 测试类型分布(1ae 新增 11 测试)

- **PG + Redis + MinIO 全栈集成**:1 测试(flusher behavioral)
- **PG 事务隔离测试**:1 测试(erasure CASCADE isolation)
- **真 Redis 集成**:3 测试(operatorWS behavioral ×2 + auth_bcrypt HandlerConstructible)
- **真 bcrypt 行为**:1 测试(auth_bcrypt WrongPasswordBehavioral)
- **gin httptest recorder 行为**:2 测试(auth_cookie SetSessionCookie ×2)
- **failingPool mock 行为**:2 测试(migrations FailFast ×2)
- **observability buffer logger 行为**:3 测试(Lifecycle ×3 含 panic 路径)
- **Vue mount + i18n 行为**:2 测试(LoginView SESSION_EXPIRED + invalid_credentials)

**模式**:60% 行为级 + 40% 源码契约(保留)。源码契约仍作重构回归保护,行为级提供语义正确性保证。

## 关键模式

### 测试驱动 bug 发现

R4 调试中发现 e2e 1d-replay scene 3 的两个问题:
1. **silent skip** — `if (sessionsWithEvents.length === 0) return;` 让测试在空状态下静默 pass
2. **timing 假设** — 2s wait 不足以等 flusher 30s 间隔

两个都是 audit D2 反模式,在 1ae 修复后变 fail-loudly。

### Interface 重构的最小化原则

R2 引入 PgxPool interface 时:
- 只暴露 storage 包**实际使用**的 6 个方法(不是 pgxpool.Pool 全部)
- 改 1 个 caller(migrations.go)
- 不改任何 `&Postgres{Pool: pool}` 调用点

避免过度设计。

## 提交

本切片建议拆 3 个 commit:

1. `refactor(1ae): storage.PgxPool interface + auth.setSessionCookie helper(R2+R3b 支撑)`
2. `test(1ae): 11 行为级测试 — operatorWS/erasure/bcrypt/cookie/migrations/flusher/observability/LoginView(R1-R3+R5)`
3. `fix(1ae): e2e 1d-replay polling 替代 fixed wait + 移除 silent skip(R4)`

(未自动 commit,留给用户审阅后决定)

## Verification Depth Badge

🟢 touched — 9 项修复全部完成,mutation cross-check 7/7 KILLED,e2e flakiness 0%。剩余 D1 PASS 率未达 80%(因 41 项 PARTIAL 中只升级了 11 项),整体 verdict 🟡。

升级 🟢 strict 需要:
- R3 剩余 30 项源码契约测试全部升级(预计 ~15h)
- Mutation CI 集成(stryker/mutesting)
- CI 90 天历史分析

留 post-v1 backlog。

## 下一步

### 立即可做

- 用户审阅 1ae 改动 + commit(3 个建议拆分)
- 更新 `docs/project-status.md` §5 加 1ae 行
- 更新 `IMPLEMENTATION_PLAN.md` 加 1ae 到 v1 已交付切片

### Backlog(post-v1)

- R3 剩余源码契约测试升级(~30 项,~15h)
- R6 ws-trace-inherit 模式推广到所有 SDK 测试
- R7 mutation testing CI 集成
- R8 CI 90 天历史分析(需 `gh auth login`)
- 1ad 报告中剩余 T2/T3 测试 gap(40 项,~15h)
