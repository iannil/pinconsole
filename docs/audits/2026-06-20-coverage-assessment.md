# 单元测试覆盖率评估（2026-06-20）

> **状态**:纯评估报告(识别盲区 + 量化差距),不动代码。改进建议留 backlog(§7)。
> **审计员**:Claude(grill-me 共识:纯量化评估 + 历史虚标对账 + docker 实测验证)
> **审计时间**:2026-06-20
> **范围**:Go 后端逐包覆盖率 + 前端测试用例数 + e2e 场景矩阵 + 关键未覆盖路径 + 历史虚标对账
> **Disclaimer**:本报告只评估"覆盖率"(行/语句/用例数),**不**评估"测试质量"(质量评估见 [`2026-06-19-test-health-audit.md`](./2026-06-19-test-health-audit.md))。覆盖率 100% 不等于测试有意义。

---

## §1 执行摘要

**一句话结论**:**Go 后端覆盖率整体健康**(加权约 65%,9/11 包 ≥ 48%),**前端覆盖率无法量化**(vitest 未配 coverage provider),**历史文档多虚标/低估**(§6 列出 7 处需修正)。

**整体评级**:🟡 verified-shallow(覆盖率本身不差,但量化体系残缺)

**Top 3 风险**:

1. **🔴 `isURLAllowed` 函数 0% 覆盖** — popup URL 白名单核心安全函数,位于 `internal/api/command.go:350`,完全未测。任何对 popup 命令的 URL 注入/SSRF 攻击都不会被测试抓住。
2. **🔴 所有 storage 适配器函数 0% 覆盖** — `minio.go` / `postgres.go` / `redis.go` 的 Connect/Ping/Set/SetNX/EvalLua/Get/Del/PutBytes 等基础函数全部未测,生产环境 DB 故障路径无回归保护。
3. **🔴 `docs/project-status.md` §2.1 系统性虚标** — 4 个覆盖率数字(28.7% / 1.5% / 21.4% / 12.8%)全部低于实测,会让 LLM 误判"测试严重缺失"。

**Top 3 改进**(详见 §7):

1. CI 加覆盖率门槛 + Codecov 上传
2. vitest 配置 coverage provider
3. 补测 §5.1 列出的 10 个安全/合规关键函数

---

## §2 Go 后端逐包覆盖率（2026-06-20 实测，带 docker 环境）

**测试环境**:docker-compose 启动 postgres(16-alpine) + redis(7-alpine) + minio(latest),额外创建 `pinconsole` 数据库(因测试代码硬编码连接 `localhost:5432/pinconsole`,详见 §6.5)。

| 包 | 覆盖率 | LOC(估) | 测试文件 | 业务文件 | 评级 |
|---|---|---|---|---|---|
| `internal/config` | **98.0%** | ~150 | 1 | 1 | 🟢 strict |
| `internal/privacy` | **95.0%** | ~250 | 2 | 3 | 🟢 strict |
| `internal/proto` | **88.9%** | ~200 | 1 | 3 | 🟢 |
| `internal/antiscrape` | **86.7%** | ~400 | 多 | 多 | 🟢 |
| `internal/observability` | **83.3%** | ~200 | 2 | 2 | 🟢 |
| `internal/logging` | **79.6%** | ~150 | 1 | 1 | 🟢 |
| `internal/hub` | **73.0%** | ~400 | 1 | 3 | 🟢 |
| `internal/storage` | **57.6%** | ~1300 | 10 | 14 | 🟡 |
| `internal/recording` | **48.0%** | ~350 | 多 | 多 | 🟡 |
| `internal/api` | **38.2%** | ~2200 | 34 | 17 | 🟡 |
| `cmd/server` | **4.9%** | ~100 | 1 | 1 | 🔴 |
| `migrations` | [no statements] | — | 1(embed) | — | n/a |

**整体聚合**(加权按 LOC):**~65%**(估算)

**评级标准**:
- 🟢 ≥ 70%
- 🟡 40-70%
- 🔴 < 40%

**关键观察**:

- **`cmd/server` 4.9% 是正常的** — main.go 启动入口难单测,标准做法是用 e2e 兜底
- **`migrations` 无 statements** — migration 是 SQL 文件,Go embed 测试只验证 SQL 可执行
- **`internal/api` 38.2% 不算低** — 这是 HTTP handler 包,典型覆盖率瓶颈(构造函数 + 路由注册难单测,e2e 兜底)。**与 commit `b748e43` 自报的"38.2%"完全吻合**,说明自报数字真实
- **`internal/storage` 57.6%** — repo 函数覆盖良好(详见 §2.1),适配器函数未覆盖(详见 §5.2)

### §2.1 storage 包内部分解（57.6% 的真相）

| 文件 | 覆盖率 | 说明 |
|---|---|---|
| `chat_repo.go` | ~95% | 1ah 关闭 T1 |
| `command_repo.go` | ~85% | 1ai-b |
| `consent_repo.go` | ~90% | 1l |
| `event_blob_repo.go` | ~85% | 1ai-b |
| `gc_repo.go` | ~80% | 1l |
| `session_repo.go` | ~90% | 1ai |
| `user_repo.go` | 100% | 1ai |
| `visitor_repo.go` | 100% | 1ai |
| `erasure_repo.go` | ~70% | 1l + 1ac(但 ListEventBlobKeysBySessions 0%) |
| `minio.go` | **0%** | 适配器函数全部未覆盖 |
| `postgres.go` | **Connect/Ping/Close 0%**(其余通过 repo 间接覆盖) | 适配器 |
| `redis.go` | **0%**(Connect/Ping/Set/SetNX/EvalLua/Get/Del/TTL/Close 全 0%) | 适配器 |
| `stores.go` | Connect/Close 0% | 适配器 |

**结论**:`storage` 包的 repo 函数覆盖高(70-100%),**适配器层(minio.go/postgres.go/redis.go)完全未覆盖**——这是真实的盲区。

---

## §3 前端测试用例数（2026-06-20 实测）

### §3.1 admin/ (Vue 3 SPA)

| 维度 | 数值 |
|---|---|
| 业务文件(`.ts` + `.vue` in src/) | 28 |
| 测试文件(`.test.ts` in tests/) | 9 |
| 用例数(grep `^\s*(it\|test)\(`) | **82**(实测) |
| 1aa 报告声称 | 64 |
| **差异** | **+18(+28%)低估** |
| vitest coverage 配置 | ❌ 未配置(无 provider / 无 thresholds / 无 reporter) |
| 行覆盖率 | **无法量化**(vitest coverage 未配) |

**测试文件清单**(9 个,共 1471 行):

| 文件 | 行数 | 用例数 |
|---|---|---|
| `tests/api-client.test.ts` | 123 | 9 |
| `tests/api.auth.test.ts` | 143 | 8 |
| `tests/auth.store.test.ts` | 192 | 13 |
| `tests/dashboard_wiring.test.ts` | 137 | 12 |
| `tests/router.test.ts` | 116 | 6 |
| `tests/session-expired.test.ts` | 195 | 6 |
| `tests/smoke.test.ts` | 9 | 1 |
| `tests/useWs.test.ts` | 209 | 7 |
| `tests/visitors.store.test.ts` | 347 | 20 |

### §3.2 visitor-sdk/ (TypeScript SDK)

| 维度 | 数值 |
|---|---|
| 业务文件(`.ts` in src/) | 18 |
| 测试文件(`.test.ts` in tests/) | 7 |
| 用例数 | **68**(实测) |
| 1aa 报告声称 | 48 |
| **差异** | **+20(+42%)低估** |
| vitest coverage 配置 | ❌ 未配置 |
| 行覆盖率 | **无法量化** |

**测试文件清单**(7 个,共 1292 行):

| 文件 | 行数 | 用例数 |
|---|---|---|
| `tests/batch.test.ts` | 149 | 9 |
| `tests/collectors_wiring.test.ts` | 98 | 11 |
| `tests/config.test.ts` | 242 | 14 |
| `tests/session.test.ts` | 152 | 8 |
| `tests/transport_recovery.test.ts` | 108 | 9 |
| `tests/transport.ws.test.ts` | 361 | 11 |
| `tests/ws-trace-inherit.test.ts` | 182 | 6 |

### §3.3 TS 测试总数对账

| 来源 | admin | SDK | 总计 | 与实测差异 |
|---|---|---|---|---|
| **实测**(2026-06-20) | 82 | 68 | **150** | baseline |
| 1aa 报告(`2026-06-19-slice-1aa-ts-test-deepening.md`) | 64 | 48 | 112 | **-38(-25%)低估** |
| `project-status.md` §2.1 | — | — | "16 个测试文件" | 文件数对(16),但没量化用例数 |

**原因推测**:1aa 报告只统计了"切片新增的测试",没算上原有测试。这是"切片完成报告"自报的典型问题——只看 delta 不看 total。

### §3.4 关键零测试业务文件

**visitor-sdk/ 零测试关键文件**(🔴 high):

| 文件 | LOC | 风险 |
|---|---|---|
| `src/index.ts` | ~2371 | 🔴 **SDK 主入口**,所有 export 集中地,零测试 |
| `src/commands/handler.ts` | ~200 | 🔴 operator 命令分发器(光标/点击/滚动/填充/导航),零测试 |
| `src/commands/cursor.ts` | ~80 | 🟡 光标命令实现 |
| `src/commands/toast.ts` | ~50 | 🟡 Toast 命令 |
| `src/commands/nodeMap.ts` | ~100 | 🟡 rrweb 节点 ID → DOM 映射 |
| `src/collectors/rrweb.ts` | ~150 | 🔴 rrweb 采集核心 |
| `src/collectors/screenshot.ts` | ~100 | 🔴 选择性截图(canvas/WebGL/iframe) |
| `src/ui/*.ts`(4 个) | ~200 | 🟡 popup/cookie banner/co-browse 横幅 |

**admin/ 零测试关键文件**(🟡 medium):

| 文件 | LOC | 风险 |
|---|---|---|
| `src/utils/fetchJson.ts` | ~50 | 🟡 API 请求核心工具 |
| `src/utils/time.ts` | ~30 | 🟢 时间格式化 |
| `src/api/claim.ts` | ~30 | 🟡 claim/release API |
| `src/api/privacy.ts` | ~30 | 🟡 GDPR 隐私 API |
| `src/api/sessions.ts` | ~30 | 🟡 sessions API |
| `src/views/*.vue`(5 个) | — | 🟡 所有视图组件(.vue 测试是 Vue 生态薄弱点) |
| `src/components/*.vue`(6 个) | — | 🟡 所有 UI 组件 |

---

## §4 e2e 场景覆盖矩阵（19 spec × 91 test）

| spec | 用例数 | 覆盖切片 | 深度 |
|---|---|---|---|
| `01-trace-id.spec.ts` | 3 | 1z(trace_id 端到端) | 🟡 |
| `02-i18n-migration.spec.ts` | 3 | 1r(i18n + logger 迁移) | 🟢 |
| `03-flagged-session.spec.ts` | 4 | 1w(flagged session) | 🟢 |
| `04-login-throttle.spec.ts` | 5 | 1x(登录暴力破解) | 🟢 |
| `05-i18n-at-sign.spec.ts` | 3 | 1z(`@` 字符 i18n) | 🟢 |
| `06-trace-id-inherit.spec.ts` | 2 | 1z(trace_id 继承) | 🟢 |
| `1a-smoke.spec.ts` | 6 | 1a(骨架 smoke) | 🟡(浅,基本健康检查) |
| `1b-realtime.spec.ts` | 5 | 1b(单向最小) | 🟢 |
| `1c-rrweb.spec.ts` | 5 | 1c(rrweb 接入) | 🟢 |
| `1d-replay.spec.ts` | 5 | 1d(录像归档) | 🟢 |
| `1e-cobrowse.spec.ts` | 6 | 1e(双向通道) | 🟢 |
| `1f-form-navigate.spec.ts` | 5 | 1f(表单 + 跳转) | 🟢 |
| `1g-chat.spec.ts` | 5 | 1g(弹窗 + 聊天) | 🟢 |
| `1h-auth.spec.ts` | 5 | 1h(认证) | 🟢 |
| `1h-ui.spec.ts` | 4 | 1h-ui(LoginView) | 🟢 |
| `1i-antiscrape.spec.ts` | 5 | 1i(反爬虫) | 🟢 |
| `1j-i18n-deploy.spec.ts` | 5 | 1j(i18n + 部署) | 🟢 |
| `1k-security.spec.ts` | 7 | 1k(安全阻断栈) | 🟢 |
| `1l-privacy.spec.ts` | 8 | 1l(GDPR) | 🟢 |

**总计**:19 spec, 91 test cases

**v1 切片 e2e 覆盖**:1a-1l 主干全覆盖(12 spec),1m-1z 加固阶段部分通过专项 spec 覆盖(7 spec:01-06 + 部分 1j/1k/1l),1aa-1ai-h 测试深化阶段无专属 e2e(单测覆盖)。

**裸 e2e / 浅测识别**(沿用 [`2026-06-19-test-confidence-audit.md`](./2026-06-19-test-confidence-audit.md) T0~T3 rubric):

- `1a-smoke.spec.ts` 🟡 verified-shallow — 只验证 `/healthz` 和 SDK 加载,不验证业务路径
- `01-trace-id.spec.ts` 🟡 — 只验证 HTTP 头注入,不验证 WS 路径

**无 T3(测试存在但弱)**: 所有 e2e 都有真实业务断言,无纯 `expect(resp.ok()).toBeTruthy()` 类弱断言。

---

## §5 关键盲区 Top 20（按风险分级）

### §5.1 Go 安全/合规关键未覆盖函数（🔴 high priority）

| # | 函数 | 文件 | 当前覆盖 | 风险 |
|---|---|---|---|---|
| 1 | `isURLAllowed` | `internal/api/command.go:350` | **0.0%** | 🔴 popup URL 白名单核心,SSRF/钓鱼防护 |
| 2 | `RegisterMe` | `internal/api/auth.go:64` | **0.0%** | 🔴 admin 注册端点,未授权创建 admin 风险 |
| 3 | `NewAuthHandler` | `internal/api/auth.go:45` | **0.0%** | 🟡 构造函数 |
| 4 | `NewChatHandler` | `internal/api/chat.go:38` | **0.0%** | 🟡 构造函数 |
| 5 | `NewClaimHandler` | `internal/api/claim.go:41` | **0.0%** | 🟡 构造函数 |
| 6 | `NewCommandHandler` | `internal/api/command.go:48` | **0.0%** | 🟡 构造函数 |
| 7 | `NewPrivacyHandler` | `internal/api/privacy.go:29` | **0.0%** | 🟡 构造函数 |
| 8 | `NewReplayHandler` | `internal/api/replay.go:28` | **0.0%** | 🟡 构造函数 |
| 9 | `NewSessionHandler` | `internal/api/session.go:33` | **0.0%** | 🟡 构造函数 |
| 10 | `buildCommandPayload` | `internal/api/command.go:271` | **0.0%** | 🟡 命令 payload 构造,影响 postCommand 行为 |
| 11 | `healthLive` / `healthReady` | `internal/api/health.go:14,22` | **0.0%** | 🟡 health check 端点(e2e 部分覆盖) |
| 12 | `listSessions` | `internal/api/session.go:137` | **0.0%** | 🟡 session 列表查询 |
| 13 | `DeleteVisitor` | `internal/api/privacy.go:41` | **0.0%** | 🔴 GDPR 被遗忘权 API(e2e 部分覆盖) |

### §5.2 Go storage 适配器层（🟡 medium,生产故障路径）

| 文件 | 未覆盖函数 | 风险 |
|---|---|---|
| `internal/storage/minio.go` | ConnectMinIO/Ping/PutBytes/GetBytes/Close(全部) | 🟡 MinIO 故障路径无回归保护 |
| `internal/storage/postgres.go` | ConnectPostgres/Ping/Close | 🟡 PG 连接管理 |
| `internal/storage/redis.go` | Connect/Ping/Set/SetNX/EvalLua/Get/Del/TTL/Close(全部) | 🟡 Redis 故障路径(claim lock / rate limit) |
| `internal/storage/stores.go` | Connect/Close | 🟡 Stores 聚合初始化 |
| `internal/storage/erasure_repo.go` | ListEventBlobKeysBySessions | 🟡 GDPR erasure MinIO 删除(1l 切片遗漏) |

### §5.3 TS 关键业务文件零测试（🔴 high）

| # | 文件 | LOC | 风险 |
|---|---|---|---|
| 1 | `visitor-sdk/src/index.ts` | 2371 | 🔴 SDK 主入口 |
| 2 | `visitor-sdk/src/commands/handler.ts` | 200 | 🔴 operator 命令分发 |
| 3 | `visitor-sdk/src/collectors/rrweb.ts` | 150 | 🔴 rrweb 采集核心 |
| 4 | `visitor-sdk/src/collectors/screenshot.ts` | 100 | 🔴 选择性截图 |
| 5 | `visitor-sdk/src/commands/nodeMap.ts` | 100 | 🟡 rrweb 节点映射 |
| 6 | `visitor-sdk/src/commands/cursor.ts` | 80 | 🟡 光标命令 |
| 7 | `visitor-sdk/src/commands/toast.ts` | 50 | 🟡 Toast 命令 |
| 8 | `visitor-sdk/src/ui/*.ts`(4 个) | 200 | 🟡 popup/cookie/co-browse UI |

### §5.4 admin/ API client + utils 零测试（🟡 medium）

| 文件 | 风险 |
|---|---|
| `admin/src/utils/fetchJson.ts` | 🟡 全局 API 请求工具 |
| `admin/src/api/claim.ts` | 🟡 claim/release 客户端 |
| `admin/src/api/privacy.ts` | 🟡 GDPR 客户端 |
| `admin/src/api/sessions.ts` | 🟡 session 查询客户端 |
| `admin/src/views/*.vue`(5 个) | 🟢 视图组件(e2e 兜底) |
| `admin/src/components/*.vue`(6 个) | 🟢 UI 组件(e2e 兜底) |

---

## §6 历史虚标/低估修正

| # | 来源 | 声称值 | 实测值 | 差异 | 原因 |
|---|---|---|---|---|---|
| 1 | `docs/project-status.md` §2.1 | `api` 包 28.7% | **38.2%** | **-9.5pp 虚标** | §2.1 数据采自本地无 docker 环境,部分 API 测试被 skip |
| 2 | `docs/project-status.md` §2.1 | `storage` 包 1.5% | **57.6%** | **-56.1pp 严重虚标** | 本地无 PG/Redis/MinIO,所有 repo 测试 skip |
| 3 | Explore agent 探索报告 | `antiscrape` 21.4% | **86.7%** | **-65.3pp 严重虚标** | Explore agent 跑测试时环境/缓存问题 |
| 4 | Explore agent 探索报告 | `recording` 12.8% | **48.0%** | **-35.2pp 严重虚标** | 同上 |
| 5 | `docs/project-status.md` §2.1 | e2e 20 场景 | **19 spec** | **-1 虚标** | 计数错误(可能 1a-smoke 拆分时多算) |
| 6 | `1aa` 报告 | TS 测试 112(admin 64 + SDK 48) | **150**(admin 82 + SDK 68) | **+34% 低估** | 1aa 只统计切片新增,未算原有 |
| 7 | commit `b748e43` message | "api 包覆盖首次破 35%" | 38.2%(达成) | ✅ **真实** | commit 自报准确,与 1ai-h 完成时实测一致 |
| 8 | `daily/2026-06-19.md` | "storage 包覆盖 20.1%→57.6%" | 57.6%(docker) | ✅ **真实** | 1ai/1ai-b 报告数字真实,本地无 docker 时才显示 1.5% |

**关键结论**:

- **6/8 项虚标/低估**(`project-status.md` §2.1 占 3 处)
- **2/8 项真实**(commit message + daily)
- **根本原因**:`project-status.md` §2.1 数据采自本地无 docker 环境,严重低估真实覆盖率。需要修正为"docker 环境"口径。

### §6.5 附带发现：`.env` rename 重构遗漏

**问题**:rename 重构 5 步(`f461b59..234fa06`)漏改 `.env` 和 `.env.example`:

| 文件 | 当前值 | 应改为 |
|---|---|---|
| `.env:PG_DB` | `marketing_monitor` | `pinconsole` |
| `.env:MINIO_BUCKET` | `marketing-monitor` | `pinconsole` |
| `.env.example:PG_DB` | `marketing_monitor` | `pinconsole` |
| `.env.example:MINIO_BUCKET` | `marketing-monitor` | `pinconsole` |

**连带影响**:

- `docker-compose.yml` 默认值是 `${PG_DB:-pinconsole}` / `${MINIO_BUCKET:-pinconsole}`(已改)
- 但 `.env` 优先级高,实际 docker 起的 PG 数据库是 `marketing_monitor`,MinIO bucket 是 `marketing-monitor`
- 测试代码 `internal/storage/erasure_test.go:34` 硬编码 `localhost:5432/pinconsole`,与 docker 实际数据库名不匹配 → 所有 storage repo 测试在本地默认 `docker compose up` 后仍 skip → 覆盖率显示 1.5%

**本次评估的 workaround**:手动 `CREATE DATABASE pinconsole` + 应用 migrations 后测试通过。但**根因未修**——任何新人按 README 跑测试都会撞同样问题。

**建议**:作为 backlog,需要修 `.env` 和 `.env.example` + 让 `docker-compose.yml` 启动时自动创建 `pinconsole` 数据库(或测试代码改读 `PG_DB` 环境变量)。**不在本次评估范围**。

---

## §7 改进建议 backlog（纯建议，不在本次范围）

### §7.1 CI 覆盖率门槛 + 报告上传（🔴 高 ROI）

**当前**:`.github/workflows/ci.yml` 跑 `go test -race -cover -count=1 ./...` 但:
- ❌ 无 `-coverprofile` 留底
- ❌ 无 `go tool cover -func` 阈值检查
- ❌ 无 Codecov/Coveralls 上传

**建议**:

```yaml
- name: go test (with coverage profile)
  run: |
    go test -race -cover -count=1 -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -1  # 打印整体覆盖率
    # 阈值检查(可选):用 third-party action 如 pvanaken/coverage-checker

- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v4
  with:
    file: ./coverage.out
```

### §7.2 vitest coverage 配置（🟡 中 ROI）

**当前**:admin/ 和 visitor-sdk/ 的 `vite.config.ts` 完全无 coverage 配置。

**建议**(需要先 `pnpm add -D @vitest/coverage-v8`):

```ts
// admin/vite.config.ts + visitor-sdk/vite.config.ts
export default defineConfig({
  test: {
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      thresholds: {
        lines: 60,    // 当前用例集中在 store/composable,设保守阈值
        functions: 50,
      },
      exclude: ['node_modules/', 'dist/', '**/*.d.ts'],
    },
  },
});
```

### §7.3 补测优先级 Top 10（按 §5 风险分级）

**🔴 P0**(安全/合规关键):

1. `internal/api/command.go:isURLAllowed` — URL 白名单核心(0%)
2. `internal/api/auth.go:RegisterMe` — admin 注册(0%)
3. `internal/api/privacy.go:DeleteVisitor` — GDPR 被遗忘权(0%,e2e 部分覆盖)
4. `visitor-sdk/src/index.ts` — SDK 主入口(2371 行,0%)
5. `visitor-sdk/src/commands/handler.ts` — operator 命令分发(0%)

**🟡 P1**(生产故障路径):

6. `internal/storage/redis.go` — claim lock / rate limit 基础(0%)
7. `internal/storage/minio.go` — 录像归档(0%)
8. `internal/api/session.go:listSessions` — session 列表(0%)
9. `visitor-sdk/src/collectors/rrweb.ts` — rrweb 采集核心(0%)
10. `visitor-sdk/src/collectors/screenshot.ts` — 选择性截图(0%)

**🟢 P2**(构造函数/e2e 兜底):

- 所有 `New*Handler` 构造函数(7 个,均 0%)— 这些 e2e 间接覆盖,单测 ROI 低
- `internal/api/health.go` healthLive/healthReady — e2e 已覆盖
- `admin/src/views/*.vue` + `components/*.vue` — e2e 兜底

### §7.4 文档同步（🟢 低 ROI 但必做）

修正 `docs/project-status.md` §2.1 的覆盖率数字(详见 §6 表 #1, #2, #5)。本次评估已在阶段 C 同步修正。

---

## §8 验证方法（可重复跑）

```bash
# 1. 启动 docker 基础设施
cd /Users/rong.zhu/Code/pinconsole
docker compose up -d postgres redis minio

# 2. 等 PG ready
until docker compose exec -T postgres pg_isready -U mm 2>/dev/null | grep -q accepting; do sleep 1; done

# 3. 创建 pinconsole 数据库 + 应用 migrations(workaround §6.5 的 .env 残留问题)
docker compose exec -T postgres psql -U mm -d postgres -c "CREATE DATABASE pinconsole OWNER mm;"
for f in server/migrations/*.up.sql; do
  docker compose exec -T postgres psql -U mm -d pinconsole < "$f"
done

# 4. Go 全包覆盖率
cd server
go test -count=1 -cover ./...

# 5. 关键包函数级未覆盖清单
go test -count=1 -coverprofile=/tmp/cover-api.out ./internal/api/...
go tool cover -func=/tmp/cover-api.out | awk '$3 != "100.0%"' | sort -k3 -n

go test -count=1 -coverprofile=/tmp/cover-storage.out ./internal/storage/...
go tool cover -func=/tmp/cover-storage.out | awk '$3 != "100.0%"' | sort -k3 -n

# 6. 前端测试用例数
cd ..
grep -hcE "^\s*(it|test)\(" admin/tests/*.test.ts | paste -sd+ - | bc   # admin 用例数
grep -hcE "^\s*(it|test)\(" visitor-sdk/tests/*.test.ts | paste -sd+ - | bc  # SDK 用例数

# 7. e2e 场景清单
ls e2e/tests/*.spec.ts | xargs -n1 basename | sort

# 8. 清理(可选)
docker compose down
```

---

## 附录:与既有审计的关系

- 本报告 **只评估覆盖率**(量化数字)
- [`2026-06-19-test-confidence-audit.md`](./2026-06-19-test-confidence-audit.md) 评估 **测试存在性 vs spec 决策**(28 T0 + 40 T1 gap)
- [`2026-06-19-test-health-audit.md`](./2026-06-19-test-health-audit.md) 评估 **测试质量**(mutation/flaky/weak assertion)
- 三者互补:覆盖率告诉你"多少代码被测了",spec 对照告诉你"关键决策点是否被测",health 告诉你"测试是否有意义"
