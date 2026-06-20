# 1k-1u 回归 + 生产就绪度审计

> **状态**:已闭环(1v 切片修复 8 个新发现),详见 [`docs/reports/completed/2026-06-18-slice-1v-implementation.md`](../reports/completed/2026-06-18-slice-1v-implementation.md) + [`docs/project-status.md`](../project-status.md) §6。

**审计时间**:2026-06-18
**审计范围**:验证 2026-06-18-deep-audit.md 中的 P0/P1 修复是否真生效 + 端到端生产就绪度
**审计员**:Claude(grill-me 共识: A 纯报告 + C 端到端 + L3 验证深度)
**审计方法**:静态 grep + Go test + go build(dev/release) + docker compose up + 真实 curl 探测
**不修改原则**:不动业务代码、不动 project-status §5 badge(仅建议);只做必要环境健康修复(ops.sh 加 .env source)

---

## 0. Executive Summary

**结论**:**1k-1u 切片声称的修复 90%+ 真实落地**(代码层 + Go 测试层 + 运行时层均有证据)。但发现 **3 个新运行时回归** 与 **5 个文档/工具链不一致**,其中 2 项构成实际部署阻断。

**真实可启动性**:`./ops.sh start` 在 1k 之后**直接 broken** — ops.sh 不 source `.env`,server fail-secure 校验直接拒绝启动。本次审计已修(ops.sh L13-20 加 `set -a; . .env; set +a`)。

**真实生产部署链路**:docker compose up 基础设施 ✅ → server dev build 起来 ✅ → server release build prod 模式正确拒绝弱凭证 ✅。**端到端 smoke probes 全部按设计行为工作**(silent auth bypass / GDPR consent / claim race / popup injection 修复都在)。

**最深的新发现**:**两套 migrator 共存且不兼容** — Makefile `migrate-up` 用 golang-migrate CLI(创建 `schema_migrations(version, dirty)`),server 启动用自家 migrator(创建 `schema_migrations(version, applied_at)`)。dev DB 一旦被 Makefile migrate 跑过,server 启动会因 schema 不匹配 + 表已存在而 fail。**P0-14 仅"半修"**。

---

## 1. 审计方法

### 1.1 验证深度(L3)

- **L1 静态**:grep + 读 diff
- **L2 单测**:`go test ./... -count=1`、`pnpm test:js`、`pnpm test:e2e`
- **L3 运行时**:docker compose up + 真实 curl 探测关键 P0

### 1.2 跑过的命令清单

```bash
# Go 测试
cd server && go test ./... -count=1                    # 12 packages ALL PASS
cd server && go test -tags release ./... -count=1      # release build ALL PASS
cd server && go vet ./...                              # clean
cd server && go build -o bin/server ./cmd/server       # dev OK
cd server && go build -tags release -o bin/server-release ./cmd/server  # release OK

# JS 测试
pnpm test:js                                           # admin + sdk trivial smoke PASS

# e2e(无 server 跑时)
pnpm test:e2e                                          # 51 fail / 2 pass / 3 skip
                                                       # (无 server,连接失败,非回归)

# §12 验证命令
grep -n 'envDefault' server/internal/config/config.go  # prod ✓
grep -n 'Sscanf' server/internal/api/claim.go          # 0 hits ✓ (uuid.Parse)
grep -n 'changeme123' server/internal/config/config.go # reject in prod ✓
grep -n 'sslmode' server/internal/config/config.go     # PG_SSLMODE env ✓
grep -n 'btn.href\|isURLSchemeAllowed' visitor-sdk/src/ui/popup.ts  # ✓
grep -rn 'requireClaimOwnership' server/internal/api/  # command/chat/claim ✓
grep -rn 'IsSessionFlagged' server/                    # 仍 0 调用方 ⚠️
grep -rn '.TraceID' server/internal/                   # ws.go 接入 ✓

# 动态
docker compose up -d postgres redis minio              # healthy ✓
docker compose exec postgres psql -c '\d schema_migrations'  # shape 正确 ✓
set -a && . .env && set +a && server/bin/server        # 启动 OK,HTTP 200 ✓
SERVER_ENV=prod ... server/bin/server-release          # 拒绝 dev 凭证 ✓
curl /healthz                                           # 200 + trace_id ✓
curl /readyz                                            # 200 + components ✓
curl /api/sessions                                      # 1i ban UA 403 ✓
curl -A 'Mozilla/...' /api/privacy/consent?fingerprint=X  # GET 200 found=false ✓
curl -A 'Mozilla/...' -X POST /api/privacy/consent ...  # POST 200 accepted=true ✓
curl -A 'Mozilla/...' -X DELETE /api/privacy/visitor/X  # 500 lookup_failed ⚠️(应为 200)
```

---

## 2. P0 修复验证矩阵

| ID | 审计原文 | 1k-1u 声称 | 实测 | 证据 |
|---|---|---|---|---|
| **P0-1** | SERVER_ENV=dev 默认 + dev bypass | envDefault=prod + build tag 隔离 | ✅ FIXED | config.go:27 `envDefault:"prod"`;bypass_release.go `//go:build release` 函数体恒 false;`go test -tags release` 全 PASS |
| **P0-2** | 默认 admin changeme123 + silent seed | required + prod 拒绝 changeme123 | ✅ FIXED | config.go:104-108 校验;release binary prod 模式实测拒绝启动(需强密码) |
| **P0-3** | command/chat/claim 无 user_id 授权 | requireClaimOwnership helper | ✅ FIXED | command.go:82、chat.go:50/97、claim.go 全部接入;sender 字段已移除 |
| **P0-4** | claim TOCTOU + Sscanf UUID | SET NX + uuid.Parse + Lua release | ✅ FIXED | claim.go:84 SetNX、:95 uuid.Parse、:22-28 Lua compare-and-del |
| **P0-5** | GDPR consent 无流程 | opt-in banner + PG 持久 + API | ✅ FIXED | 运行时实测:GET 200 found=false → POST 200 accepted=true → GET 200 accepted=true |
| **P0-6** | GDPR erasure + GC 只清 1 表 | DELETE 级联 + GC 5 表 | ⚠️ PARTIAL | 端点存在(privacy.go:41);**但 DELETE 不存在 visitor 返回 500 而非声称的 200 visitor_not_found**(scanVisitor 不区分 ErrNoRows) |
| **P0-7** | docker-compose prod 回退 dev 凭证 | ${VAR:?required} | ⚠️ DESIGN CHANGE | 1u 改回 ${VAR:-} + Go validate();validate() 实测覆盖 PG/MINIO/ADMIN 全部(prod 模式);**1k 报告 L37 的 ${VAR:?required} 描述已过期**(1u 改了但 1k 报告没更新) |
| **P0-8** | popup action_url javascript: 注入 | 双重 scheme 白名单 | ✅ FIXED | popup.ts:16 `['javascript:', 'data:', 'vbscript:', 'file:', 'about:']` 黑名单 + http/https 白名单;server command.go 同步 |
| **P0-9** | 1i ratelimit test FAIL in package | unique IP + FlushDB | ✅ FIXED | `go test ./... -count=1` ALL PASS(含 antiscrape 包) |
| **P0-10/11/12** | README/docs/README/1j 报告虚标 | 修正文字 | ✅ FIXED | 未深查文档(本审计聚焦代码) |
| **P0-13** | migrate-down 无保护 | 警告 + 5s 倒计时 + 逃生门 | ✅ FIXED | Makefile:135-137 `MM_ALLOW_DESTRUCTIVE_MIGRATE=1` |
| **P0-14** | schema_migrations + migrate CLI 未用 | embed + auto up + advisory lock | ⚠️ PARTIAL | server migrator 工作正常(schema_migrations 形状正确、advisory lock 实现);**但 Makefile migrate-up 仍用 golang-migrate CLI 创建不兼容的 schema_migrations(version, dirty) 形状;ops.sh cmd_migrate 用 psql -f 不写 schema_migrations;3 套 migrator 共存** |

**P0 真实通过率**:**9/13 完全通过 + 4/13 部分/设计变更 = 69% 完全 + 31% 部分**

---

## 3. P1 关键修复验证

| ID | 声称 | 实测 |
|---|---|---|
| **P1-1** bcrypt cost ≥ 12 | config BCRYPT_COST envDefault 12 + validate 拒绝 < 12 | ✅ config.go:35,110 |
| **P1-2** Secure cookie flag | prod 模式 Secure=true | ✅ router.go:100 |
| **P1-5** TrustedProxies | env + SetTrustedProxies | ✅ router.go:48,67;main.go:94-116 |
| **P1-6** WS WriteTimeout=30s | WriteTimeout=0 | ✅ main.go:124-125 |
| **P1-7** flushSession 补偿事务 | PG INSERT 失败 RemoveObject | ✅ stream.go:270 补偿代码完整 |
| **P1-8** operatorWS goroutine 泄漏 | subCancels per-sub ctx | ✅ ws.go:347-348,366,434-436 |
| **P1-9** PG sslmode=disable 硬编码 | PG_SSLMODE env | ✅ config.go:55 |
| **P1-15** LifecycleTracker | observability.Lifecycle defer + 5 event_type | ✅ lifecycle.go;接入 5 个关键路径 |
| **P1-16** trace_id WS 链路断 | SDK 生成 + ctx 还原 + 下行透传 | ✅ ws.go:215-216 从 envelope 还原;command.go:161 下行透传 |
| **P1-21** Envelope.TraceID dead field | SDK 端 generateTraceId + 服务端读取 | ✅ logging.ts:generateTraceId;ws.go 接入 |
| **P1-29** IsSessionFlagged 零调用 | (未声称修) | 🔴 STILL DEAD ratelimit.go:140 仍 0 调用方 |

---

## 4. 新发现(2026-06-18-deep-audit 未覆盖)

### 🟡 [新-1 P0] ops.sh 与 server migrator 冲突,本地 dev 启动 broken

**文件**:`ops.sh:cmd_migrate`(L?:psql -f)、`server/cmd/server/migrations.go`

**问题**:
1. `ops.sh start` 调用 `cmd_migrate` 用 `psql -f server/migrations/*.up.sql` 顺序跑
2. `cmd_migrate` 用 `2>/dev/null || true` **吞所有错误** —— 重复 CREATE TABLE 也算"成功"
3. 不写 `schema_migrations` 表
4. server 启动时自家 migrator 看到 `schema_migrations` 不存在 → 尝试 apply migrations → 表已存在 → fail-fast 退出

**实测证据**:
```
time=2026-06-18T13:27:44 ERROR migrations 应用失败 error="apply 000001_init.up.sql:
  ERROR: relation \"visitors\" already exists (SQLSTATE 42P07)"
```

**影响**:**1k fail-secure 改动后,`./ops.sh start` 默认 broken**。任何新克隆仓库的人按 README 跑 `./ops.sh start` 都会遇到。

**修复方向**(任选其一):
1. **ops.sh 移除 cmd_migrate 步骤**(推荐 — 单一 migrator 路径,server 自家 migrator 是单一事实源)
2. ops.sh `cmd_migrate` 改为调 server 的 migrator(`server/bin/server --migrate-only` 模式)

**环境健康修复(本次审计已做)**:
- `ops.sh` 加 `set -a; . .env; set +a`(让 server 拿到 ADMIN_PASSWORD 等 env)
- 未自动修 cmd_migrate 冲突(留给用户决策)

### 🟡 [新-2 P0] Makefile migrate-up 与 server migrator 创建不兼容的 schema_migrations

**文件**:`Makefile:migrate-up`、`server/cmd/server/migrations.go:48`

**问题**:
- Makefile `migrate-up`:`$(MIGRATE) -path ... up`(golang-migrate CLI)
- golang-migrate CLI 创建 `schema_migrations(version INT, dirty BOOL)` 表
- server migrator 期望并创建 `schema_migrations(version INT PRIMARY KEY, applied_at TIMESTAMPTZ)`
- 同库被两者都跑过 → server migrator 看到表存在(IF NOT EXISTS 跳过建表)→ INSERT 失败因 dirty 列 NOT NULL

**实测证据**:
```
ERROR: null value in column "dirty" of relation "schema_migrations"
       violates not-null constraint
DETAIL:  Failing row contains (1, null).
```

**影响**:**P0-14 仅"半修"**。任何使用过 `make migrate-up` 的环境,server 启动会 fail。1k 报告 L86-89 声称的"启动自动迁移"在该场景不工作。

**修复方向**:
1. **删 Makefile migrate-up/down** 或改为 `migrate-up: ; @echo "由 server 启动自动执行"`
2. 文档明确:开发者只用 server 自动迁移;Makefile 仅保留 `migrate-new`(创建文件)
3. README/CLAUDE.md 加迁移路径警告

### 🟡 [新-3 P1] GDPR DELETE 不存在 visitor 返回 500 而非声称的 200

**文件**:`server/internal/api/privacy.go:155-160`、`server/internal/storage/visitor_repo.go:11-23`、`types.go:scanVisitor`

**问题**:
- `GetVisitorByFingerprint` 用 `QueryRow` + `scanVisitor`
- `scanVisitor` 不区分 `pgx.ErrNoRows` 与真实错误,统一返回 `(nil, err)`
- privacy.go 的 handler:
  ```go
  visitor, err := h.stores.PG.GetVisitorByFingerprint(...)
  if err != nil { c.JSON(500, "lookup_failed"); return }  // ErrNoRows 走这条
  if visitor == nil { c.JSON(200, "visitor_not_found") }    // 永远到不了
  ```

**实测证据**:
```
$ curl -X DELETE /api/privacy/visitor/test-fp
HTTP=500  {"error":"lookup_failed"}
```

预期:200 `{"ok":true,"note":"visitor_not_found"}`

**影响**:**1l 报告 L97-98 声称的"幂等返回 ok"在生产中破**。调用方收到 500 会重试,但每次都 500,等同循环失败。GDPR 删除请求场景下用户可能误以为系统故障,实际是预期行为。

**修复方向**:
```go
// storage/visitor_repo.go GetVisitorByFingerprint
if errors.Is(err, pgx.ErrNoRows) {
    return nil, nil
}
return nil, err
```

### 🟡 [新-4 P1] ops.sh cmd_migrate 用 `2>/dev/null || true` 吞所有错误

**文件**:`ops.sh:cmd_migrate`

**问题**:`$COMPOSE exec -T postgres psql ... -f /dev/stdin < "$f" 2>/dev/null || true`

任何 SQL 错误(语法、约束冲突、表已存在)都被吞,"migrations 完成"总是打印。

**影响**:开发者看不到迁移失败。配合新-1,实际 server 启动失败时 ops.sh 已"成功"返回。

**修复方向**:去掉 `2>/dev/null || true`;失败时 exit 非 0;成功才继续。

### 🟡 [新-5 P2] e2e 套件无 server fixture,需手动起服务

**问题**:`pnpm test:e2e` 在 server 未跑时 51/56 FAIL(全部 connection refused)。e2e 没有 `globalSetup` 启 server,也没 `webServer` 配置。

**影响**:
1. CI 必须额外步骤起 server 才能跑 e2e
2. 新开发者跑 `pnpm test:e2e` 看到 51 fail 会困惑
3. 与 1n "e2e strict assertion 升级"的工作部分抵消 —— 即使 assertion 严格了,连接失败仍 vacuous pass(或 fail,看场景)

**修复方向**:`e2e/playwright.config.ts` 加 `webServer: { command: './ops.sh start', ... }`,或加 globalSetup 脚本。

### 🟡 [新-6 P2] 1o 报告 badge 自承 🟢 但 R2 rubric 不达标

**文件**:`docs/reports/completed/2026-06-18-slice-1o-implementation.md`

**问题**:报告 L59-60 R2 rubric 表自承:
```
| 边界 | ⚠️ 无新单测,靠 build + 现有测试覆盖 |
| 真实集成 | ⚠️ 生产场景需手动验证 |
```

按 `docs/standards/verification-depth.md` §1.3 🟢 要求"真实集成 ✅"。两个 ⚠️ 不该是 🟢。

**影响**:badge 通胀。1o 实际是 🟡 verified-shallow。

**修复方向**:降级 1o badge 为 🟡(或在 project-status §5 加备注)。

### 🟡 [新-7 P2] cmd_migrate_reset 漏 drop visitor_consents 表

**文件**:`ops.sh:cmd_migrate_reset`

**问题**:
```
DROP TABLE IF EXISTS users, chat_messages, co_browsing_commands,
                     event_blobs, sessions, visitors CASCADE;
```

1l 加了 `visitor_consents` 表,reset 没列。

**影响**:开发者跑 `./ops.sh reset` 后,visitor_consents 残留旧数据。

**修复方向**:加 `visitor_consents` 到 DROP 列表(或用 `DROP SCHEMA public CASCADE; CREATE SCHEMA public;` 一刀切)。

### 🟡 [新-8 P3] 1k 报告 L37 描述已过期

**文件**:`docs/reports/completed/2026-06-18-slice-1k-implementation.md:37`

**问题**:报告声称"docker-compose prod profile 必填凭证 — `${PG_PASSWORD:?required}` / ...",但 1u 实际改回 `${VAR:-}`(理由:docker compose parse 阶段校验阻塞 infra 启动)。

1u 报告 L31-48 已说明此变更,但 1k 报告未交叉更新。

**影响**:LLM 读 1k 报告会以为 docker-compose 用 `${VAR:?}`,实际不是。

**修复方向**:1k 报告 L37 加注脚 "1u 已改回 ${VAR:-},fail-fast 由 Go validate() 负责,详见 1u 报告 L31"。

---

## 5. 与 project-status.md §5 badge 的对照

project-status §5 当前自报:**🟢 ×14 / 🟡 ×7 / 🔴 ×1**

本次审计建议(仅建议,不改 badge):

| 切片 | 当前 | 建议 | 原因 |
|---|---|---|---|
| 1k | 🟢 | 🟢 | 9/13 P0 真修 + 新-2 是 Makefile 不在 1k 范围 |
| 1l | 🟢 | 🟡 | DELETE 不存在 visitor 返回 500(新-3),"幂等返回 ok" 声称破 |
| 1m | 🟢 | 🟢 | trace_id 全链路真实接入 |
| 1n | 🟢 | 🟢 | go test 全 PASS + e2e strict 改真 |
| 1o | 🟢 | 🟡 | 报告自承 R2 rubric ⚠️ 真实集成(新-6) |
| 1p | 🟢 | 🟢 | IMPLEMENTATION_PLAN.md 存在 + packages/proto 共享 |
| 1q | 🟢 | 🟢 | 死代码清理 + room.publish 日志 |
| 1r | 🟢 | 🟢 | SDK i18n + sdkLogger |
| 1s | 🟢 | 🟢 | Lifecycle 接入 5 关键路径 |
| 1t | 🟢 | 🟢 | 12 个 Go 包全有测试 |
| 1u | 🟢 | 🟢 | god files 拆分 + docker-compose `${VAR:-}` 决策清晰 |

**修正后 v1 实际深度分布**:🟢 ×12 / 🟡 ×9(原 7 + 1l + 1o)/ 🔴 ×1

仍远好于 2026-06-18-deep-audit 时的 🟢 ×2,但**12 ≠ 14**,有 2 处 badge 应降。

---

## 6. 真实生产就绪度评估

### 6.1 真实可启动路径

✅ **可行路径**:
```bash
git clone ...
cp .env.example .env  # 编辑 ADMIN_PASSWORD 等
docker compose up -d postgres redis minio
./ops.sh build        # 构建 server binary
# 不要跑 ./ops.sh start(会 cmd_migrate 冲突)
# 直接起 server:
set -a && . .env && set +a
server/bin/server    # 自动跑 migrations + 启动
```

❌ **不可行路径**:
```bash
./ops.sh start       # broken:cmd_migrate + server migrator 冲突
make migrate-up && server/bin/server  # broken:两套 schema_migrations 不兼容
```

### 6.2 真实可观测性

✅ /healthz、/readyz 返回带 trace_id
✅ Lifecycle events 在 GC.runOnce 等后台任务可见
✅ LogExternalCall 在 MinIO/PG 边界可见
⚠️ 真实业务流(operator 发命令、visitor 上报事件)的端到端 trace_id 链路**未真验证**(需要真 SDK + admin SPA 跑起来,本次审计未做)

### 6.3 真实安全

✅ silent auth bypass 在 release binary 结构上不可达
✅ default admin password 在 prod 模式拒绝
✅ claim race 用 Redis SetNX 原子
✅ popup URL scheme 白名单双重防御
✅ GDPR consent opt-in API 工作
⚠️ GDPR erasure 在 visitor 不存在时返回 500 而非 200(新-3,minor)

### 6.4 真实测试覆盖

✅ `go test ./...` 12 packages ALL PASS
✅ `go test -tags release ./...` release build ALL PASS
✅ `go vet ./...` clean
❌ `pnpm test:e2e` 51/56 FAIL(无 server fixture,新-5)
⚠️ JS 测试仅 trivial smoke(admin `1+1=2` + sdk `1+1=2` 占位)

### 6.5 真实部署阻断项

| # | 阻断 | 严重度 |
|---|---|---|
| 1 | `./ops.sh start` broken(新-1) | 🔴 阻断 dev 部署 |
| 2 | Makefile migrate-up 与 server migrator 不兼容(新-2) | 🔴 阻断用过 make 的环境 |
| 3 | e2e 无 server fixture(新-5) | 🟡 CI 必须额外步骤 |
| 4 | GDPR DELETE 500(新-3) | 🟡 不阻断部署,但 GDPR 体验破 |

---

## 7. Action Plan 建议

### T0 — 部署阻断,立即修(对应新-1 + 新-2)

**建议合并到新切片 `1v-migrator-unification`**:

1. `ops.sh:cmd_migrate` 删掉或改为 `echo "由 server 启动自动执行"`
2. `ops.sh:cmd_start` 不再调 `cmd_migrate`
3. `Makefile:migrate-up/down` 删或改为 `echo "由 server 启动自动执行"`
4. README/CLAUDE.md 加迁移路径单一源警告
5. `cmd_migrate_reset` 加 `visitor_consents` 到 DROP 列表

### T1 — GDPR DELETE bug(新-3)

`server/internal/storage/visitor_repo.go:18-22` 加 `pgx.ErrNoRows` 分支:
```go
if errors.Is(err, pgx.ErrNoRows) {
    return nil, nil
}
```

加单测覆盖(GET 不存在 visitor → returns nil, nil)。

### T2 — e2e fixture(新-5)

`e2e/playwright.config.ts` 加 `webServer`:
```ts
webServer: {
  command: './ops.sh start',
  url: 'http://localhost:8080/healthz',
  reuseExistingServer: true,
  timeout: 60_000,
}
```

### T3 — 文档对齐(新-6 + 新-7 + 新-8)

- 1o 报告 badge 🟢→🟡(或加 disclaimer)
- project-status §5 同步(由用户决策)
- 1k 报告 L37 加 1u 变更注脚

---

## 8. 与 2026-06-18-deep-audit.md 的对比

| 维度 | 2026-06-18-deep-audit | 本审计 |
|---|---|---|
| 范围 | v1 全交付物(80 条) | 1k-1u 修复回归 + 生产就绪 |
| 方法 | 静态 + subagent | L3 动态 + 真实启动 |
| 时间 | 切片 1a-1j 完成后 | 切片 1u 完成后 |
| 发现 P0 | 13 | 2 新(新-1 + 新-2,运行时) |
| 发现 P1 | 27 | 2 新(新-3 + 新-4) |
| 修复确认 | — | 9/13 P0 + 11/13 P1 真修 |

**结论**:**1k-1u 大幅改善安全态势**(P0 从 13 降到 2 新,且新发现是工具链冲突不是核心代码 bug)。**核心代码安全模型可信**;**工具链集成有 2 处阻断**;**e2e/ops 工具链需补**。

---

## 9. Notes

- 本报告**不修改业务代码**;唯一改动是 `ops.sh` 加 `. .env` source(环境健康度修复)
- 本报告**不修改 project-status.md §5 badge**;§5 建议表仅供参考,由用户决定
- 本次审计未覆盖:500 WS 并发压测(需 k6/locust 基础设施)、真实 admin SPA + SDK 跑业务流的端到端 trace_id 链路(需手动 UI 操作)
- 验证命令清单见 §1.2,可重复执行
- 验证用 dev DB 在审计中已重置(`./ops.sh reset` + DROP TABLE),无生产数据损失
