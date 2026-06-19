# 切片 1ac — 测试信心加固 T0(28 项 critical 路径回归测试)

**Spec 日期**:2026-06-19
**来源**:[`docs/audits/2026-06-19-test-confidence-audit.md`](../audits/2026-06-19-test-confidence-audit.md) §5.2
**目标**:关闭审计发现的 28 个 T0 gap,把 7 个 🔴 切片升回 🟡/🟢

## 范围

### Priority A — deep-audit P0 回归测试(13 项)

代码已修(deep-audit 闭环),但无回归测试。**这是最高 ROI 部分**——bug 复发概率最高。

| ID | 描述 | 文件 |
|---|---|---|
| T0-1k-1 | 非 owner postCommand → 403 单测 | `authz_test.go`(新建) |
| T0-1k-2 | 非 owner postChat → 403 单测 | `authz_test.go` |
| T0-1k-3 | OperatorID 用 UUID 不用 ClientIP | `command_test.go` 扩展 |
| T0-1k-4 | claim 并发 SET NX race | `claim_test.go`(新建) |
| T0-1k-5 | release Lua compare-and-del owner-only | `claim_test.go` |
| T0-1k-6 | prod cookie Secure=true | `auth_test.go` 扩展 |
| T0-1k-7 | pg_advisory_lock 并发 migration | `migrations_test.go` 扩展 |
| T0-1k-8 | migration 失败 panic fail-fast | `migrations_test.go` |
| T0-1k-9 | compose prod 缺凭据阻断 | CI workflow + e2e(范围外,doc 标注) |
| T0-1l-1/2/3 | GDPR erasure PG/MinIO/Redis 级联 | `erasure_test.go`(新建) |
| T0-1l-4 | GC 5 表(chat/co_browsing/sessions/visitors/event_blobs) | `gc_test.go`(新建) |
| T0-1l-5 | erasure 非 admin 403 | `erasure_test.go` |
| T0-1x-1 | Lua 原子 INCR+EXPIRE 并发 | `auth_test.go` 扩展 |
| T0-1y-1/2 | WS exceed → Close + FlagSession | `ws_ratelimit_test.go` 扩展 |

### Priority B — 其他 critical 路径补缺(15 项)

| ID | 描述 | 文件 |
|---|---|---|
| T0-1h-1 | HttpOnly cookie 属性 | `auth_test.go` |
| T0-1h-2 | WS `/ws/operator` 必须 AuthMiddleware | `router_test.go`(新建) |
| T0-1h-5 | bcrypt 实际密码验证路径 | `auth_test.go` |
| T0-1h-6 | WebSocket 同源 cookie 依赖 | `router_test.go` |
| T0-1i-1 | Redis 不可用 fail-open | `ratelimit_test.go` 扩展 |
| T0-1h-ui-1 | fetchJson 401 handler 清 user + SESSION_EXPIRED | `api-client.test.ts` 扩展 |
| T0-1h-ui-2 | App.vue mount fetchMe 校验 cookie | `App.test.ts`(新建) |
| T0-1h-ui-3 | SESSION_EXPIRED UI 流 | `LoginView.test.ts`(新建) |
| T0-1l-6 | consent PG upsert + GetLatestConsent | `postgres_test.go` 扩展 |
| T0-1h-3 | claim race-safe(同 1k-4,1h-backend 视角) | `claim_test.go` |
| T0-1h-4 | owner-only release(同 1k-5) | `claim_test.go` |

## 验收

每个 T0 测试必须:
1. 测试存在,跑 `go test` / `pnpm test` 通过
2. 对应源码做 mutation(改 1 行)后,测试必须 FAIL
3. 报告带 file:line 引用 + 断言摘录

完成后:
- 7 个 🔴 切片 → 🟡 或 🟢
- `project-status.md` §5 更新
- T0 计数 28 → 0

## 工作量

预计 28 小时(solo 全职 ~3.5 天)。本会话内尽可能完成 Priority A(13 项,~13 小时),Priority B 留下次。

## 风险

- 部分 T0(如 1h-2 WS 同源)涉及多文件集成,可能需要 mock WS server
- 1l GDPR 级联需要 docker compose up 全栈(PG+MinIO+Redis)
- 1k-9 compose prod 凭据阻断属 e2e/CI 范围,1ac 仅在 doc 标注

## Out of scope

- T1/T2/T3(留 1ad)
- 任何业务逻辑变更(只加测试)
- 任何 CI workflow 改动(T0-1k-9 例外,但仅 doc 标注)
