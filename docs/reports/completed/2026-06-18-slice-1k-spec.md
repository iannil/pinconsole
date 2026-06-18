# 切片 1k-security-blockers 规格

**状态**:in-progress
**开始**:2026-06-18
**对应审计**:[2026-06-18 全栈深度审计](../audits/2026-06-18-deep-audit.md) §2 P0-1/P0-2/P0-3/P0-4/P0-7/P0-8/P0-13/P0-14

## Context

全栈深度审计发现当前 v1 代码不可直接上生产,主要由 3 个 silent defaults 叠加 + 端点缺授权 + claim race + migrations 不可重放构成完整远程接管链。本切片锁定审计 T0 安全栈,**不含** GDPR 合规(P0-5/P0-6,留给独立切片 `1l-privacy-gdpr`,因涉及访客同意 UI + DB schema 变更)。

## 范围

### In scope(8 个 P0)

| P0 | 文件 | 修复方向 |
|---|---|---|
| P0-1 CWE-306 | `config/config.go:19`、`api/middleware.go:17-23`、`api/router.go:103` | SERVER_ENV 默认改 `prod`;AuthMiddleware dev bypass 加 `//go:build !release` 编译 tag,release 二进制结构上无法走 dev 分支 |
| P0-2 CWE-522 | `config/config.go:24-25`、`cmd/server/main.go:135-153` | AdminPassword 去默认值,缺失时启动 panic;`Env==prod` 时若 AdminPassword==`changeme123` 也 panic |
| P0-3 CWE-639 | `api/command.go:71-147`、`api/chat.go:85-141`、`api/claim.go:37-67` | 三个 handler 开头读 `user_id`、查 Redis `claim:session:{id}` 校验 == caller UID,不匹配 403。`command.go:117` OperatorID 改用 `user_id.UUID()`(非 `c.ClientIP()`)。`chat.go` 去掉客户端可控的 `sender` 字段 |
| P0-4 CWE-362 | `api/claim.go:53-66` | claim:1) 先 `GetSession` 查 PG 存在且未结束(否则 404/409);2) Redis `SET claim:session:{id} {uid} NX EX 300` 原子;3) UUID 用 `uuid.Parse`(非 `fmt.Sscanf`)。release:Lua 脚本对比 owner 再 DEL |
| P0-7 CWE-1004 | `docker-compose.yml:23,24,56,57,83-89` | prod profile 用 `${PG_PASSWORD:?PG_PASSWORD required}` / `${MINIO_SECRET_KEY:?MINIO_SECRET_KEY required}` 必填语法,缺失变量拒绝启动 |
| P0-8 CWE-1321 | `visitor-sdk/src/ui/popup.ts:39-45`、`api/command.go`(show_popup) | 后端 `buildCommandPayload` 对 show_popup 加 scheme 白名单(只允许 http/https,拒 javascript:/data:/vbscript:/file:)。前端 popup.ts 接收时同样校验,双重防御 |
| P0-13 CWE-1188 | `server/migrations/*.down.sql`、`Makefile:132-136` | Makefile migrate-down 改为:1) 警告 + 5s 倒计时 prompt;2) `MM_ALLOW_DESTRUCTIVE_MIGRATE=1` env 绕过 |
| P0-14 | `Makefile:132-136`、`.github/workflows/ci.yml:138-142`、`cmd/server/main.go`、`server/cmd/server/migrations.go`(新建) | 1) `//go:embed migrations/*.sql` 嵌入二进制;2) golang-migrate iofs source driver;3) 启动时 migrate up 全部应用;4) `pg_advisory_lock` 防多实例并发;5) 启动失败 panic 退出(fail-fast);6) CI compose-smoke 改用 `migrate` CLI 验证 down/up 循环 |

### Out of scope(留给后续切片)

- P0-5 GDPR consent(1l-privacy-gdpr)
- P0-6 GDPR erasure(1l-privacy-gdpr)
- P0-9 1i ratelimit test fix(单独 commit,不属 1k)
- P0-10/11/12 文档虚标修正(单独 commit,不属 1k)
- P1-* 全部

## 锁定决策表

| # | 决策 | 理由 |
|---|---|---|
| 1 | silent defaults:全套 fail-secure(SERVER_ENV=prod default + 编译 tag + AdminPassword required + bcrypt 12 + `${VAR:?required}`) | 双重防御(env + 编译 tag),dev workflow 仅需显式 opt-in,部属误配最常见路径(忘 SERVER_ENV=prod)结构上不可能 |
| 2 | 授权检查:handler 层重复检查(非 middleware) | 逻辑明显易测;v1 端点数有限(3 个 handler × 4 个动作),重复可接受;未来提取 middleware 留给后续切片 |
| 3 | claim 查 PG:先验证 session 存在且未结束,再 SET NX | 防止 claim 不存在的 session ID 写入 Redis 垃圾;提供清晰错误码(404/409) |
| 4 | popup URL scheme 白名单(非域名白名单) | 与 navigate 不同,popup CTA 常跳外部落地页;只防 `javascript:` 类 RCE,不限制业务灵活性 |
| 5 | migrations:`//go:embed` + golang-migrate iofs + advisory lock + 启动 up + fail-fast | 单二进制部署一致;advisory lock 防多实例并发跑迁移;fail-fast 防止服务在 schema 不一致状态启动 |
| 6 | Down 保护:Makefile prompt + 5s 倒计时 + `MM_ALLOW_DESTRUCTIVE_MIGRATE=1` 逃生门 | 保留可逆性 + 增加摩擦;逃生门便于 CI/自动化 |
| 7 | 测试:8-10 Go 单测 + 1 e2e(SERVER_ENV=prod 端到端),目标 🟢 | R2 rubric 全覆盖;prod 模式 e2e 填补 1h 留下的"prod 行为零覆盖"空白 |
| 8 | bcrypt cost 10→12(顺手修,审计 P1-1) | CLAUDE.md 明确 ≥ 12;改动 1 行,纳入本切片成本极低 |

## 涉及代码改动

### 后端 Go(预计 ~15 个文件)

**新建**:
- `server/cmd/server/migrations.go` — `//go:embed migrations/*.sql` + `runMigrations(ctx, pool)` 函数 + advisory lock

**修改**:
- `server/internal/config/config.go` — `Env` default 改 `prod`;`AdminPassword` 去 `envDefault`;新增 `BCryptCost int env:"BCRYPT_COST" envDefault:"12"`;`PG_SSLMODE` env(顺手修 P1-9,因审计判定 P1 但与本切片 fail-secure 主题相关);DSN 改用 `PG_SSLMODE`
- `server/internal/api/middleware.go` — `AuthMiddleware` dev bypass 拆出独立函数 + `//go:build !release` tag;prod build 时返回 nil middleware 或 fail-secure
- `server/internal/api/router.go` — 调用点更新;新增 `RequireClaim` 不做(决策 2:handler 层)
- `server/internal/api/command.go` — 4 个 handler 加 claim 校验;OperatorID 改 `user_id.UUID()`;`buildCommandPayload` 对 `show_popup` 加 `isURLSchemeAllowed`
- `server/internal/api/chat.go` — 3 个 handler 加 claim 校验;`postMessageRequest.Sender` 字段移除(服务端按认证上下文决定)
- `server/internal/api/claim.go` — 加 `GetSession` 查 PG;claim 改 `SET NX EX 300`;UUID 用 `uuid.Parse`;release 改 Lua 脚本对比 owner
- `server/internal/api/auth.go` — bcrypt cost 用 cfg.BCryptCost(非 hardcoded DefaultCost)
- `server/cmd/server/main.go` — 启动时 `runMigrations(ctx, pool)`(失败 panic);AdminPassword 缺失/fail-secure panic;seedAdminUser 用 cfg.BCryptCost
- `server/internal/storage/queries.go` — 复用 `GetSession`(已有?若无需加 `GetSessionByID`)
- `server/migrations/*.sql` — 不改 SQL 本身

**测试新增**:
- `server/internal/api/middleware_test.go` — AuthMiddleware prod 三路径(无 cookie / 无效 session / 有效 session);dev bypass 在 release build 应不存在(用 `//go:build !release` 编译 tag)
- `server/internal/api/claim_test.go` — claim SET NX race(2 goroutine 并发,只 1 成功);UUID parse;session 不存在 404;session 已结束 409;release Lua owner 对比
- `server/internal/api/command_test.go` — 未 claim 时 postCommand 返回 403;非 owner claim 时 postCommand 返回 403;owner claim 时 200
- `server/internal/api/chat_test.go` — 同上结构,3 个动作
- `server/internal/api/popup_test.go`(或合并到 command_test.go) — show_popup URL scheme 黑名单(javascript:/data:/vbscript:/file: 拒;http/https 允许)
- `server/internal/config/config_test.go` — SERVER_ENV default == "prod";AdminPassword 缺失 panic;bcrypt cost default 12
- `server/cmd/server/migrations_test.go` — `//go:embed` 包含 4 个 up.sql;advisory lock 多次调用阻塞;migrate up on fresh DB 后表存在

### 前端 TS(预计 ~2 个文件)

**修改**:
- `visitor-sdk/src/ui/popup.ts` — `show_popup` 接收时校验 `^https?:` 开头,否则拒绝并 log warn

**测试新增**:
- `visitor-sdk/src/ui/popup.test.ts`(若无测试基建,跳过单测,e2e 覆盖)

### 配置 / 部署(预计 ~4 个文件)

**修改**:
- `docker-compose.yml` — prod profile:`PG_PASSWORD:?`、`MINIO_SECRET_KEY:?`、`SERVER_ENV: prod`(已有)、`ADMIN_EMAIL:?`、`ADMIN_PASSWORD:?`
- `.env.example` — 加 `SERVER_ENV=prod`、`ADMIN_EMAIL=...`、`ADMIN_PASSWORD=...`、`BCRYPT_COST=12`、`PG_SSLMODE=prefer` 标注生产必改
- `Makefile` — `migrate-down` 改为 prompt + 5s 倒计时 + `MM_ALLOW_DESTRUCTIVE_MIGRATE=1` 逃生门;`migrate-up` 保留
- `.github/workflows/ci.yml` — `compose-smoke` 改用 `migrate` CLI 验证 down(加逃生门 env)+ up 循环;CI 加 SERVER_ENV=prod 模式 e2e job

### E2E(预计 ~1 个新文件)

**新建**:
- `e2e/tests/1k-security.spec.ts` — SERVER_ENV=prod 端到端:
  - 场景1:匿名访问 `/api/sessions/ended` 返回 401
  - 场景2:登录 → cookie 设置 → 访问 `/api/sessions/ended` 返回 200
  - 场景3:未 claim 时 `POST /api/sessions/:id/command` 返回 403
  - 场景4:claim 后非 owner 调用返回 403
  - 场景5:popup action_url 含 `javascript:` 时后端返回 400
  - 场景6:fresh DB 启动 → migrations 自动应用 → 表存在

## Verification

```bash
# Go 单测
cd server && go test ./... -count=1 -v

# 关键测试单独跑(验证 race 不再 flaky)
cd server && go test ./internal/api/ -run TestClaim -count=10 -v

# Release build 验证 dev bypass 已移除
cd server && go build -tags release -o /tmp/mm-release ./cmd/server
/tmp/mm-release --help  # 应无 dev 提示

# Prod 模式 e2e
cd e2e && SERVER_ENV=prod pnpm test 1k-security

# Down 保护验证
make migrate-down  # 应看到 prompt
MM_ALLOW_DESTRUCTIVE_MIGRATE=1 make migrate-down  # 应直接执行

# Fresh DB 验证
docker compose down -v
docker compose --profile prod up -d
# 容器启动后应自动跑 migrations,无需手动 psql -f
```

**预期结果**:
- 所有 Go 单测 PASS
- e2e 6 个场景全 PASS
- release binary 不含 dev bypass 代码
- compose 启动后 schema_migrations 表存在,4 个表存在
- make migrate-down 默认会 prompt,逃生门可绕过

## 深度目标

🟢 verified-deep(R2 rubric 全覆盖):
- ✅ Happy path:登录 + claim + postCommand + popup 全链路
- ✅ Negative case:401 / 403 / 404 / 409 / 400
- ✅ 边界:claim race / session 不存在 / session 已结束 / popup 各 scheme
- ✅ 真实集成:真实 PG + Redis + cookie,非 mock
- ✅ 可重复运行:`-count=1` 无 flaky;advisory lock 防并发

## 估时

solo 全职:1-1.5 天

## 风险

- **Dev workflow 破坏**:SERVER_ENV=prod default + AdminPassword required 后,本地 `make dev` 启动会 panic。**缓解**:Makefile dev target 显式 export SERVER_ENV=dev + ADMIN_PASSWORD=changeme123;`.env.example` 标注
- **e2e SERVER_ENV=prod 复杂度**:需要单独 docker-compose profile 或环境变量,可能与现有 dev e2e 冲突。**缓解**:用 Playwright projects 区分 dev/prod
- **Advisory lock 阻塞**:若 server 异常退出未释放 lock,下次启动会阻塞。**缓解**:lock 加超时 + 启动 timeout fail-fast
- **bcrypt 12 性能**:12 vs 10 慢约 4 倍,登录 latency 增加。**缓解**:可配置,v1 接受

## Follow-ups

- `1l-privacy-gdpr`:P0-5(consent)+ P0-6(erasure)+ GDPR 合规整体
- `1m-observability`:LifecycleTracker + event_type + WS trace_id(P1-15/16)
- `1n-test-depth`:7 切片 badge 升级回 🟢(P1-11/12/13/14 + 1i 测试 fix P0-9)
- `1h-ui`:admin LoginView + Vue Router 守卫(已规划,与 1k 配合形成完整 prod 模式)
