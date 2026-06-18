# 切片 1z-prod-readiness-gaps

**状态**:completed（v1 e2e acceptance 已验证，详见 [v1-e2e-acceptance](./2026-06-18-v1-e2e-acceptance.md)）
**开始**:2026-06-18
**完成**:2026-06-18
**关联**:第三轮 grill-me 审计(2026-06-18 生产就绪度专项)

## Context

第三轮 grill-me 审计(用户选 "生产就绪度" 为审计目标 + "OSS 项目不管生产拓扑" 为部署边界)发现 v1 主干在生产就绪度上有 4 处具体缺口。本切片合并修复其中已与用户达成共识的 4 项;其余 P2 项(e2e fixture / docker-compose 资源限制 / MinIO image pin)留待 1aa+ 或单独处理。

**用户共识**:
- ✅ trace_id 端到端断点 → "v1 补全端到端"
- ✅ 连接池未调优 → 由 P1 严重度推论(默认 pgx 4 / go-redis 40 在多访客场景明显不够)
- ✅ fail-secure 缝隙 → 由 P1 严重度推论(SSL/useSSL/Env 白名单补全)
- ✅ i18n `@` SyntaxError → 浏览器实测命中,P0 直接修

不在本切片范围:
- 500 WS SLO 压测(用户选 "v1 不验证") → 仅补 README disclaimer
- 单实例 hub 限制(用户选 "明文文档化") → 仅补 README/PLAN
- 反爬 / co-browse / SDK SPA 重载 / cookie 加固 / 备份恢复 → 留待下一轮 grill 或后续切片

## Changes

### P0 已修:i18n `@` SyntaxError(本切片已完成)

vue-i18n v10 把 `@` 解析为 linked-message 引用,`admin@example.com` / `admin@marketing-monitor.local` 触发 `INVALID_LINKED_FORMAT`(error code 10)。浏览器实测命中。

- ✅ `admin/src/i18n/{en-US,zh-CN}.ts`:
  - 删除 `login.default_hint`(含 `@`)
  - 改 `login.email_placeholder` 为不含 `@` 的中性文案
  - 新增 `login.default_email_hint` 用 `{email}` 参数占位
- ✅ `admin/src/views/LoginView.vue`:
  - 抽 `DEFAULT_ADMIN_EMAIL` 常量(取代 i18n 内嵌字面量)
  - placeholder 直接用常量
  - hint 用 `t('login.default_email_hint', { email: DEFAULT_ADMIN_EMAIL })` 传参
- ✅ `pnpm --filter @marketing-monitor/admin build` 通过
- ✅ `pnpm --filter @marketing-monitor/admin test` 通过

### P1-1 已完成:trace_id 端到端补全(1m badge 修正)

1m badge "全链路接入" 实测两处断裂:
1. admin SPA 不发 `X-Trace-Id` 头 → server 每次新生成 trace_id
2. SDK 收到 command 后丢弃其 trace_id,后续事件 envelope 用新 ID

修复:
- ✅ `admin/src/api/client.ts`(新建):统一 `apiFetch` + `apiJson`,自动注入 X-Trace-Id
- ✅ `admin/src/api/{auth,sessions,privacy}.ts`:全部改用 `apiJson`/`apiFetch`
- ✅ `visitor-sdk/src/transport/ws.ts`:
  - 收到 `type:'command'` envelope 时缓存 trace_id + 时间戳
  - 后续 N=10 个事件或 M=5 秒内的事件 envelope 用缓存的 trace_id(常量 `TRACE_INHERIT_MAX_EVENTS` / `TRACE_INHERIT_TTL_MS`)
  - 过期或耗尽后回到 `generateTraceId()`
- ✅ admin/SDK 各加 vitest 覆盖:
  - `admin/tests/api-client.test.ts`(9 测试:trace_id 格式 + 注入 + skipTraceId + headers merge + credentials default + 响应头读取 + JSON 解析 + 错误携带 trace_id)
  - `visitor-sdk/tests/ws-trace-inherit.test.ts`(6 测试:窗口内继承 + 多事件共享 + TTL 失效 + N 事件耗尽 + 无 command fresh + command 无 trace_id 优雅降级)

### P1-2 已完成:连接池调优

当前(修复前):
- `server/internal/storage/postgres.go:20` `pgxpool.New(ctx, cfg.DSN())` → 默认 `MaxConns = max(4, runtime.NumCPU)`
- `server/internal/storage/redis.go:21` `redis.NewClient(&redis.Options{Addr, Password})` → 默认 `PoolSize = 10 * runtime.NumCPU`

4 核机器 → 4 PG conns / 40 Redis conns。多并发访客(每个会话占 1 PG 写连接 + 1 Redis stream append + 1 Redis presence)快速耗尽。

修复:
- ✅ `server/internal/config/config.go`:
  - 新增 `Postgres.MaxConns int env:"PG_MAX_CONNS" envDefault:"25"`
  - 新增 `Redis.PoolSize int env:"REDIS_POOL_SIZE" envDefault:"50"`
- ✅ `server/internal/storage/postgres.go`:改用 `pgxpool.ParseConfig` + `NewWithConfig`,显式设 `pgCfg.MaxConns`
- ✅ `server/internal/storage/redis.go`:`redis.Options{..., PoolSize: cfg.PoolSize}`
- ✅ `.env.example`:补 `PG_MAX_CONNS=25` / `REDIS_POOL_SIZE=50` 注释 + 估算依据
- ✅ Go 单测(`server/internal/storage/{postgres,redis}_test.go`):验证 cfg.MaxConns/PoolSize 正确传到 pgxpool.Config / redis.Options(4 + 3 个测试)

### P1-3 已完成:fail-secure 缝隙补全

修复前 `server/internal/config/config.go validate()` 只校验密码强度,漏:
1. `PG_SSLMODE=disable` 在 prod 模式允许(明文 PG 凭证)
2. `MINIO_USE_SSL=false` 在 prod 模式允许(明文录像 blob)
3. `SERVER_ENV=production`(typo)→ `cfg.Env != "prod"` 为 true → dev 模式 bypass(release 二进制安全,但 dev 二进制暴露即裸奔)

修复:
- ✅ `server/internal/config/config.go validate()`:
  - Env 白名单 `{prod, dev, test}`,其他值拒绝(typo 防御)
  - prod 模式 + 远程 PG → 拒绝 `PG_SSLMODE=disable`(本机/docker 容器名允许)
  - prod 模式 + 远程 MinIO → 拒绝 `MINIO_USE_SSL=false`(本机/docker 容器名允许)
  - 新增 `isLocalEndpoint()` helper:识别 `localhost` / `127.0.0.1` / `::1` / `postgres` / `minio` / `redis`
- ✅ Go 单测(8 个新测试):Env 白名单拒绝 typo / 允许 canonical / 大小写归一化 / prod 拒绝远程 PG sslmode=disable / prod 允许本地 PG sslmode=disable / prod 拒绝远程 MinIO useSSL=false / prod 允许本地 / dev 模式跳过 SSL 校验

### 文档同步

- ✅ `README.md`:加 "v1 已知限制" 章节(单实例 hub / 500 WS 未压测 / OSS 不管生产拓扑 / trace_id 端到端)
- ✅ `.env.example`:补 `TRUSTED_PROXIES` + `PG_MAX_CONNS` + `REDIS_POOL_SIZE` + 1z fail-secure 注释
- ✅ `docs/project-status.md`:加 1z 行(🟡 in_progress)+ 更新最后更新日期

## Status

全部 P0 + 3 个 P1 已实施完成。

剩余工作:
- [x] P0 i18n SyntaxError(已完成)
- [x] P1-1 trace_id 端到端(已完成)
- [x] P1-2 连接池调优(已完成)
- [x] P1-3 fail-secure 缝隙(已完成)
- [x] 文档同步(README + .env.example + project-status 已更新)
- [ ] 最终验收(go test 全绿 + build 全绿 + JS test 全绿 + 真实启动一次)→ 移到 docs/reports/completed/

## Next

实施完毕,等最终验收后移到 `docs/reports/completed/`。

验收命令:
```bash
cd server && go test ./... -count=1     # 应 12 packages ALL PASS
cd server && go test -tags release ./... -count=1  # release build 也应全 PASS
cd server && go vet ./...                # clean
pnpm --filter @marketing-monitor/admin test    # 10 测试
pnpm --filter @marketing-monitor/visitor-sdk test  # 7 测试
pnpm --filter @marketing-monitor/admin build   # 编译
pnpm --filter @marketing-monitor/visitor-sdk build  # 编译
```

## Blockers

无。

## Notes

- 1m badge 虚标被发现时,前两轮审计未识别此 gap(可能因为前两轮聚焦 server 侧,trace_id 在 server 内部确实闭环)。第三轮 grill-me 把"端到端"理解为"operator browser → visitor browser",发现 admin SPA + SDK 两处断裂。
- 连接池默认值是 pgx/v5 + go-redis/v9 的库默认值,与本项目的 500 WS 目标无关,但 4 PG conns 在小规模(5+ 并发访客)就明显不够。
- `MINIO_USE_SSL=false` 在 prod + endpoint 是 localhost 时允许的特例:很多 docker-compose 部署内部网络 MinIO 不开 SSL(容器间通信),强制 SSL 会破坏这种合理部署。需要按 endpoint 判断,不要一刀切。
- 浏览器实测命中的 i18n SyntaxError 是本轮审计最有价值的发现:静态审计看不到运行时编译错误,只有真实跑起来才能发现。
- SDK trace_id 继承的 N=10 / M=5s 是经验值,覆盖一次代填/导航触发的典型 burst;真攻击场景下若 operator 发命令后长时间不发事件,会自然过期 fall back,不影响正确性。
- admin SPA 的 trace_id 是 per-call 新生成,而非 per-session 复用。这保留了每个 REST 调用的独立可观测性,代价是 operator 一次"订阅+发命令"动作的两次调用 trace_id 不共享。如需共享,可改 client.ts 用 module-level singleton trace_id。
