# 测试健康度审计（2026-06-19）

> 4 维度综合判定 v1 测试套件健康度。本审计在 2026-06-19 测试信心审计 + 1ac + 1ac-final + 1ad 之后做"二阶验证"——验证 1ac/1ad 的 68 项闭包声明是否真实有效。
>
> **审计方法**:rubric 判定 + 弱断言扫描 + 手动 mutation testing + 多次重复运行 + CI 历史分析。
>
> **本审计为独立审计**:不修代码,只诊断 + 推荐。修复走 [`1ae test-health-followup`](#8-follow-up-slice-proposal) 切片。

---

## TL;DR

| 维度 | 阈值 | 实测 | Verdict |
|---|---|---|---|
| **D1** Badge 准确性 | PASS ≥ 90% 🟢 / 80-89% 🟡 / <80% 🔴 | **38.2%**（26/68） | 🔴 |
| **D2** 弱断言数 | ≤20 🟢 / 21-50 🟡 / >50 🔴 | **20** | 🟢(边界) |
| **D3** Mutation Score | ≥80% 🟢 / 60-79% 🟡 / <60% 🔴 | **71.4%**（5/7 killed） | 🟡 |
| **D4** Flakiness | 0% 🟢 / <5% 🟡 / ≥5% 🔴 | **e2e 25%(1/4), Go/TS 0%** | 🔴 |

### 整体 verdict = worst-of-4 = 🔴

**测试套件存在系统性问题,refactor 前必修**。但问题不是"测试少",而是"测试虚"——大量源码契约 grep 测试只验证"调用了 X",不验证"X 行为正确"。

### 核心发现（按风险排序）

1. **🚨 源码契约测试疲劳（41/68 = 60% 闭包是 PARTIAL）**——`strings.Contains(fnBody, "X(")` 类断言占主导。1ad 报告自陈"60% 源码契约",但**没有任何 grep 测试升级为行为测试**(除 ws-trace-inherit.test.ts 例外)。
2. **🚨 erasure 级联测试被 PG CASCADE 欺骗**——`TestDeleteVisitorByFingerprint_CascadesAllTables` 看似验证 5 表删除,实际是 PG `ON DELETE CASCADE` 在工作。Mutation 跳过显式 `DELETE FROM chat_messages` 后测试仍通过(附录 M3)。
3. **🚨 operatorWS auth 接线测试不能检测 dead code**——`TestWS_OperatorWS_WiresAuthentication` grep `authenticateOperatorWS(` 字符串。Mutation 把调用包进 `if false { ... }` 后测试仍通过(附录 M4)。安全风险:任何重构误删 auth 调用都不会被测试捕获。
4. **🚨 e2e 1d-replay 长 session 分页测试 flaky**——Run 1 失败(Expected > 0 Received 0),Run 2-4 通过。可能是测试数据累积或 GC timing 问题。
5. **🟡 Badge 系统性虚标持续存在**——`T0-1h-ui-3` 报告声明"已 cover SESSION_EXPIRED i18n key",实际是 inline 重写 conditional 的 trivial test(等价 `expect('a').toBe('a')`),不能防 LoginView 重构。

### 通过的部分（不要再破坏）

- **真 PG 集成测试**(erasure_test.go、gc_test.go、chat_repo_test.go、ratelimit_failopen_test.go):seed 真数据 + 断言行级 COUNT。模式正确,只是被 CASCADE 削弱。
- **真 Redis 集成测试**(authz_test.go、claim_test.go、ws_ratelimit_test.go):真 SetNX/Set/Get + 行为断言。
- **真 WebSocket mock 测试**(ws-trace-inherit.test.ts):decode sentBytes + 字符串相等断言。**这是其他源码契约测试应对齐的目标**。
- **D2 弱断言控制良好**:仅 20 处,主要在源码契约测试本身(预期内)。
- **D4 Go/TS 单测零 flakiness**:20/20 + 20/20。
- **隔离性好**:无共享 Redis key、无未 mock 时间、无固定路径;唯一遗憾是 0 处 `t.Parallel()`(慢但安全)。

---

## 1. Methodology

### 1.1 4 维度方法

| 维度 | 方法 | 样本量 | 工具 |
|---|---|---|---|
| **D1** Badge 准确性 | 全量审计 1ac+1ac-final+1ad 68 项闭包,3 subagent 并行 + 主代理 10% 抽样复核 | 68/68 | rubric 判定 + 源码阅读 |
| **D2** 弱断言扫描 | piggyback D1,扫 5 类反模式(truthy/resp.ok/silent skip/explicit skip/empty-as-passed) | 全量 | grep + 人工读测 |
| **D3** Mutation testing | Tier 1(D1 PASS 项)+ Tier 2(6 安全包随机选),手动 sed/Edit + git 回滚 + 跑测试 | 7 mutations | 5 类算子 |
| **D4** 运行可靠性 | Go test ×20(`-shuffle=on`)+ TS test ×20 + e2e ×10 + CI 90 天 + 隔离 grep | 20+20+10 runs | bash loop + gh |

### 1.2 判定阈值

| 维度 | 🟢 健康 | 🟡 警戒 | 🔴 不健康 |
|---|---|---|---|
| **D1 PASS 率** | ≥90% | 80-89% | <80% |
| **D2 弱断言** | ≤20 | 21-50 | >50 |
| **D3 Mutation Score** | ≥80% | 60-79% | <60% |
| **D4 Flakiness** | 0% | <5% | ≥5% |

**整体 verdict = worst-of-4**(保守:任一维 🔴 整体即 🔴)

### 1.3 安全网执行

- **L1**:68 项清单 + 测试 file:line 通过 subagent 报告固化
- **L2**:subagent 输出固定结构(rubric + 5 反模式 + JSON-like 表格)
- **L3**:主代理对 T0-1k-1 等抽样复核,确认 subagent 判定合理(无误报)
- **Mutation**:每个 mutation 用 Edit + 显式 git diff 验证 + Edit 回滚,working tree 始终 clean
- **D4 后台**:日志归档到 `/tmp/d4-{go,ts,e2e}.log`,失败 run 完整保留

### 1.4 已知 limitation

- **CI 90 天历史未分析**:`gh auth` 未登录,无法访问 GitHub Actions 历史。改为本地跑替代(D4 重复运行)。
- **e2e ×10 部分完成**:Run 5 时报告撰写;已捕获 1 次失败,足够判定 🔴。
- **Mutation 样本偏小**:7 个变异点不足以做统计泛化,但每个 survivor 都是"系统性问题"的强信号。

---

## 2. D1 Findings — Badge 准确性

### 2.1 总览

| Subagent | 范围 | 文件数 | 闭包数 | PASS | PARTIAL | FAIL |
|---|---|---|---|---|---|---|
| A | server/internal/api + cmd/server + migrations | 11 | 24 | 13 | 10 | 1(误判) |
| B | storage + recording + antiscrape | 5 | 9 | 5 | 4 | 0 |
| C | observability + admin/tests + visitor-sdk/tests | 9 | 35 | 8 | 27 | 0 |
| **合计** | — | **25** | **68** | **26** | **41** | **0** |

> **A 的 T0-1i-1 "FAIL" 是误判**:A 在 server/internal/api/ 没找到 `TestRateLimitMiddleware_RedisUnavailable_FailOpen`,但该测试在 `server/internal/antiscrape/ratelimit_failopen_test.go:23`(B 已确认 PASS)。这是 subagent 范围切分导致,非真实 gap。

### 2.2 PASS 率 = 38.2% → 🔴

**核心问题**:1ad 报告自陈"60% 源码契约主导",但**这 60% 全部是 PARTIAL**——只验证"调用了 X",不验证"X 行为正确"。

### 2.3 真 PASS（26 项,模式正确）

#### 真 PG 集成（6 项,Subagent B 范围）

- `TestDeleteVisitorByFingerprint_CascadesAllTables`(erasure_test.go:44)— ⚠️ 受 PG CASCADE 削弱,见 D3 M3
- `TestDeleteVisitorByFingerprint_NonExistent_NoError`(erasure_test.go:140)
- `TestGC_DeleteSessionsEndedBefore / _DeleteVisitorsLastSeenBefore / _DeleteEventBlobByID`(gc_test.go:24/110/153)
- `TestConsent_UpsertAndGetLatest / _GetLatest_VersionScoped`(gc_test.go:198/262)
- `TestChat_CreateAndList / _SenderIsOperatorOrVisitor / _GC_ListOlderThanAndDelete / _DeleteEmptyIDs_NoOp`(chat_repo_test.go)

#### 真 Redis 集成（10 项,Subagent A 范围）

- `TestRequireClaimOwnership_*` 6 子测试(authz_test.go:89+)
- `TestClaim_SetNX_RaceSafety`(claim_test.go:27)
- `TestClaim_ReleaseLua_OwnerOnlyDelete` 4 子测试(claim_test.go:83)
- `TestWS_AuthenticateOperatorWS_*` 5 子测试(ws_auth_test.go:84)— ⚠️ 函数级测试强,但接线测试弱,见 D3 M4

#### 真行为测试（10 项,跨范围）

- `TestRateLimitMiddleware_RedisUnavailable_FailOpen`(ratelimit_failopen_test.go:23)
- `TestRateLimitMiddleware_RedisUnavailable_NoPanic`(ratelimit_failopen_test.go:72)— ⚠️ 弱,只查 panic
- `TestLoginThrottle_LockAfter5Failures`(auth_test.go:40)
- `TestWSRateLimit_*` 5 子测试(ws_ratelimit_test.go)— 真 Redis 行为
- `TestSDK_PopupXSS_TextContent`(chat_wiring_test.go:82)— 跨包源码契约
- `fetchJson 401 triggers unauthorizedHandler + 200/500/credentials`(session-expired.test.ts:34-138)
- 6 cases `ws-trace-inherit.test.ts`(94-181)— **真 mock WebSocket,黄金标准**

### 2.4 PARTIAL 主导（41 项,系统性问题）

**根因模式**:`strings.Contains(fnBody, "X(")` 类断言占主导。例:

```go
// T0-1h-5 PARTIAL:auth_bcrypt_test.go:22
if !strings.Contains(loginHandlerBody, "bcrypt.CompareHashAndPassword(") {
    t.Errorf("...")
}
```

**此类测试能捕获**:重构把方法名改了(如 `bcrypt.CompareHashAndPassword` → `bcrypt.Compare`)。

**此类测试不能捕获**:
1. **参数顺序错**:`bcrypt.CompareHashAndPassword(hash, password)` 写成 `(password, hash)` 仍 PASS
2. **Dead code**:`if false { bcrypt.CompareHashAndPassword(...) }` 仍 PASS(因字符串存在)
3. **逻辑分支错**:`if user != nil { 检查密码 }` 写成 `if user == nil { 检查密码 }` 仍 PASS
4. **错误处理 broken**:返回 nil err 即使密码错仍 PASS

### 2.5 严重虚标（1 项）

**T0-1h-ui-3**:`session-expired.test.ts:71-89` 声明"已 cover SESSION_EXPIRED i18n key",实际是:

```ts
const error = 'SESSION_EXPIRED';
const expectedKey = error === 'SESSION_EXPIRED' ? 'login.error_session_expired' : 'login.error_invalid_credentials';
expect(expectedKey).toBe('login.error_session_expired');
```

等价 `expect('a').toBe('a')`——**完全不能防 LoginView 重构**。

### 2.6 跨范围未覆盖（2 项）

- 1ac 报告声明 `T0-1l-2/3/5` 在 `server/internal/api/privacy_handler_test.go + privacy_admin_test.go`——Subagent A 审计了但归到 1k 范围,实际是 1l GDPR handler 测试。需后续补审计。
- 1ad 报告声明 `T1-1g-2/3/4/5` 在 `server/internal/api/chat_wiring_test.go + visitor-sdk TS`——A 部分覆盖,C 部分覆盖。需后续合并。

---

## 3. D2 Findings — 弱断言扫描

### 3.1 总览

| Subagent | 高危 | 中危 | 低危 | INFO | 合计 |
|---|---|---|---|---|---|
| A | 1 | 3 | 1 | 0 | 5 |
| B | 2 | 0 | 1 | 1(`t.Skip` 合理) | 3(+1 INFO) |
| C | 4 | 6 | 2 | 0 | 12 |
| **合计** | **7** | **9** | **4** | **1** | **20**(+1 INFO) |

**Verdict = 🟢(边界,正好 20)**

### 3.2 高危弱断言（7 项）

| 文件:行 | pattern | 风险 | 建议 |
|---|---|---|---|
| `flusher_wiring_test.go:36-75` | 3 个 `strings.Contains(fnBody, "X(")` 当断言 | 测试虚 | 改用 NewFlusher with mock MinIO/PG/Redis → 调 flushSession → 断言 mock.PutCalled |
| `flusher_wiring_test.go:79-102` | 5 个方法名 grep | 同上 | 改用 NewGC with mock repos → 调 runOnce → 断言 5 个 mock.Delete |
| `session-expired.test.ts:88` | inline 重写 LoginView 逻辑 + 断言常量相等 | 虚标 PASS | mount LoginView + i18n spy |
| `collectors_wiring.test.ts:38-41` | `toBeDefined()` + `length > 0` + regex canvas/webgl/iframe | 太弱 | 改 grep 具体 config 字段 |
| `transport_recovery.test.ts:23` | `backoff\|retry` keyword | 太弱 | 改 grep `Math.min(... reconnectMaxBackoffMs ...)` |
| `transport_recovery.test.ts:28` | `suppress\|closed` keyword | 太弱 | 改 grep `this.closed = true` + close handler 注销 |
| `chat_wiring_test.go:65,85` | 2 处 `t.Skip` 静默跳过 | 隐藏未来 rename | 改 `t.Fatal` |

### 3.3 反模式分布

- **5 类反模式中,源码契约 grep 占 16/20 = 80%**——与 1ad 报告自陈"60% 源码契约"一致(略高)
- **0 个 `.only` / `.skip` 滥用**(e2e forbidOnly 在 CI 生效)
- **0 个 `expect(arr).toEqual([])` 当 pass**
- **2 个 `t.Skip` 静默**(chat_wiring_test.go 已记)

### 3.4 D2 Verdict

🟢(边界)。20 处全部集中在源码契约测试本身——这是预期的(源码契约就是 grep)。**不是新增风险,是已知 PARTIAL 的另一面**。

---

## 4. D3 Findings — Mutation Testing

### 4.1 Tier 1 + Tier 2 共 7 个变异点

| ID | 目标 | 变异 | 结果 | 说明 |
|---|---|---|---|---|
| **M1** | T0-1k-1 authz owner check | `if ownerUID != uid` → `==` | ✅ KILLED | authz_test.go:109-119 4 个断言全失败 |
| **M2** | T0-1k-5 release Lua owner-only | `if redis.call('GET') == ARGV[1]` → `if true` | ✅ KILLED | claim_test.go:133/138/176 3 个子测试失败 |
| **M3** | T0-1l-1 erasure cascade | 跳过 `DELETE FROM chat_messages` | ❌ **SURVIVED** | PG `ON DELETE CASCADE` 自动级联,测试无法捕获 |
| **M4** | T0-1h-2 operatorWS auth 接线 | 把 auth 调用包进 `if false { ... }` | ❌ **SURVIVED** | `TestWS_OperatorWS_WiresAuthentication` grep 字符串仍命中 |
| **M5** | T0-1i-1 rate limit fail-open | `c.Next()` → `c.AbortWithStatus(503)` | ✅ KILLED(半) | FailOpen 测试失败;NoPanic 测试仍通过(只查 panic) |
| **M6** | T1-1c-5 rrweb maxRetries | `?? 3` → `?? 0` | ✅ KILLED | collectors_wiring.test.ts:14-16 失败 |
| **M7** | T1-1b-4 rrweb maskAllInputs | `?? true` → `?? false` | ✅ KILLED | collectors_wiring.test.ts:46 失败 |

**Mutation Score = 5/7 = 71.4% → 🟡**

### 4.2 M3 SURVIVED 详解（erasure cascade）

**变异**:在 `erasure_repo.go:68-72` 把 `DELETE FROM chat_messages WHERE session_id = ANY($1)` 包进 `if false { ... }`。

**结果**:`TestDeleteVisitorByFingerprint_CascadesAllTables` 仍 PASS。

**根因**:`000001_init.up.sql` 中所有 FK 都用 `ON DELETE CASCADE`:
```sql
sessions.visitor_id     REFERENCES visitors(id) ON DELETE CASCADE
chat_messages.session_id     REFERENCES sessions(id) ON DELETE CASCADE
co_browsing_commands.session_id REFERENCES sessions(id) ON DELETE CASCADE
event_blobs.session_id     REFERENCES sessions(id) ON DELETE CASCADE
```

`DELETE FROM visitors WHERE id = $1`(erasure_repo.go:93-97)执行后,PG 自动级联清空所有子表。**显式 DELETE 是冗余的死代码**。

**测试盲点**:测试 seed 5 表数据 → 调 erasure → 断言 5 表 COUNT=0。看似严密,实际:
- 即使删掉所有显式 DELETE(只留 visitor delete),测试仍 PASS
- 即使 erasure 函数完全不实现级联,测试仍 PASS
- 显式 DELETE 的错误处理(如 chat_messages 删失败)永远不会被测试发现

**风险**:中等。函数行为正确(PG 保证级联),但代码层有"防御性"显式 DELETE 是 misleading,可能误导未来开发者认为"显式 DELETE 是必须的"。

### 4.3 M4 SURVIVED 详解（operatorWS auth）

**变异**:在 `ws.go:339-346` 把 `_, authOK := authenticateOperatorWS(...)` 包进 `if false { ... }`。

**结果**:`TestWS_OperatorWS_WiresAuthentication` 仍 PASS。

**根因**:`ws_auth_test.go:222` 用 `strings.Contains(fnBody, "authenticateOperatorWS(")` 验证接线。变异后字符串仍存在(在 dead code 块内),grep 仍命中。

**测试盲点**:源码契约测试**只能验证字符串存在,不能验证代码可达**。

**风险**:**高**。安全相关。任何重构误删 auth 调用(或包进 dead code)都不会被测试捕获。例:
- `if devMode { authCheck } else { skipAuth }` — 字符串存在但 prod mode 完全跳过
- `// authenticateOperatorWS(...)` — 注释也算
- 把整个调用移到不可达分支

### 4.4 M5 半 KILLED

- `TestRateLimitMiddleware_RedisUnavailable_FailOpen` ✅ 失败(status=503 want 200)
- `TestRateLimitMiddleware_RedisUnavailable_NoPanic` ❌ 仍通过(只查 panic)

NoPanic 测试是 D2 弱断言(只查 panic 不查 status),建议升级为查 status + next handler。

### 4.5 Mutation Score 解读

**71.4% 处于 🟡 区间**(60-79%)。但**两个 survivor 都是系统性问题**:

1. **源码契约不能检测 dead code**(M4):整个项目 41 个源码契约 PARTIAL 测试都有此风险
2. **集成测试可能被 DB 特性欺骗**(M3):其他依赖 FK CASCADE 的测试也有此风险

**结论**:Mutation Score 数字本身不算糟糕,但**survivor 的性质是系统性**,需要根因修复。

---

## 5. D4 Findings — 运行可靠性

### 5.1 单测重复运行（Go + TS）

| 套件 | 跑次 | 通过率 | Flakiness |
|---|---|---|---|
| Go test (`go test ./... -count=1 -shuffle=on`) | 20 | **20/20 = 100%** | **0%** 🟢 |
| TS test (vitest admin + SDK) | 20 | **20/20 = 100%** | **0%** 🟢 |

**Go test 加 `-shuffle=on`** 验证包内测试无顺序依赖——全绿。

### 5.2 e2e 重复运行

| Run | 结果 | 失败测试 |
|---|---|---|
| 1 | ❌ FAIL | `1d 场景3:长 session 分页 replay(1000+ 事件)` — Expected > 0 Received 0 |
| 2 | ✅ PASS | — |
| 3 | ✅ PASS | — |
| 4 | ✅ PASS | — |
| 5 | 进行中 | — |
| 6-10 | 待跑 | — |

**Flakiness(已观察):1/4 = 25%** → 🔴

**风险**:`1d-replay.spec.ts:71` 长会话分页测试在 Run 1 失败,Run 2-4 通过。可能原因:
1. **测试数据累积**:Run 1 后 DB 残留数据,影响后续 Run 的 pagination 阈值
2. **GC timing**:1000+ 事件可能触发异步 GC,导致 page1 在查询时已被部分清理
3. **MinIO/PG 一致性延迟**:大 session 写入后立即查询,可能未达 strong consistency

**待办**:等 Run 5-10 完成后取最终通过率。如 ≥5% flakiness,需 1ae 切片修复。

### 5.3 隔离性扫描

| 检查项 | 结果 | 说明 |
|---|---|---|
| 共享 Redis key 模式 | ✅ 无 | grep `"visitor:"` 在 production code 0 处 |
| 未 mock 的 time.Now() | 🟡 15 处 | 全部在 production code(预期);测试中 0 处未 mock 时间 |
| 固定端口 | ✅ 仅 vite.config + SDK 默认 | `localhost:8080` 是开发默认,非测试问题 |
| 固定文件路径 | ✅ 无 | 测试中无 `/tmp/test.txt` 类硬编码 |
| `t.Parallel()` | 🟡 0 处 | 所有 Go 测试串行——慢但安全,无并行竞态 |
| 测试间共享 DB 状态 | ⚠️ 潜在 | 测试用 `uuid.New()` 生成 unique key(好),但 e2e 1d 失败可能源于累积 |

**Verdict**:🟢(隔离性整体良好,e2e flaky 是单点问题非系统性)

### 5.4 CI 历史(未完成)

**Limitation**:`gh auth` 未登录,无法访问 GitHub Actions 历史。改为本地重复运行(D4.1 + D4.2)。

**建议**:用户后续 `gh auth login` 后,跑 `gh run list --workflow=release.yml --limit=50` 取 90 天 pass rate 补全。

### 5.5 D4 Verdict = 🔴(因 e2e flakiness)

Go/TS 单测完美(0% flakiness),但 e2e flaky(25%)拉低整体。worst-of-4 已生效。

---

## 6. Recommendations(按优先级)

### 🔴 P0 — 安全/合规风险,必修

#### R1. 升级 operatorWS auth 测试为行为级(M4 修复)

**当前**:`TestWS_OperatorWS_WiresAuthentication`(ws_auth_test.go:204)用 `strings.Contains` 验证 `authenticateOperatorWS(` 字符串。

**目标**:加一个**行为级**测试,真正调 `operatorWS` handler,验证无 cookie 时 upgrade 失败 + 返回 401。

**实现**:
```go
func TestOperatorWS_NoCookie_Returns401_Behavioral(t *testing.T) {
    // 真 httptest recorder,真调 operatorWS,真断言 status=401 + 无 upgrade
    gin.SetMode(gin.TestMode)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = httptest.NewRequest("GET", "/ws/operator", nil)
    // 不设 cookie
    h := &WSHandler{stores: &storage.Stores{Redis: &storage.Redis{Client: rdb}}, devMode: false}
    h.operatorWS(c)
    if w.Code != http.StatusUnauthorized { t.Errorf("...") }
}
```

**影响**:安全相关。如果不修,任何重构误删 auth 调用都不会被测试捕获(包括未来引入的 devMode/prodMode 分支误判)。

#### R2. 升级 erasure 测试避免 PG CASCADE 误导(M3 修复)

**当前**:`TestDeleteVisitorByFingerprint_CascadesAllTables`(erasure_test.go:44)依赖 PG `ON DELETE CASCADE` 自动级联,显式 DELETE 是冗余。

**目标**:加测试**显式验证 erasure 函数自己的 DELETE 被调用**。

**实现选项**:
- **A**:在测试 DB 用 schema **不带 CASCADE**(专门为测试建的 schema),强制 erasure 函数自己删
- **B**:在 erasure 函数加 trace 日志,测试 grep 日志确认每步执行
- **C**:用 mock PG pool,断言每个 Exec 调用的 SQL string

**推荐 A**(最干净,但需测试 schema 设置)或 C(最快,但维护 mock 成本高)。

**影响**:GDPR 合规。当前测试通过 = "数据被删"(因 CASCADE),但不能验证"erasure 函数逻辑正确"。如果未来迁移到 NoCASCADE schema 或换 DB,erasure 可能 silently 不工作。

### 🟡 P1 — 测试质量,refactor 前应修

#### R3. 源码契约测试批量升级为行为测试

**当前**:41/68 = 60% 是 `strings.Contains(fnBody, "X(")` 类断言。

**目标**:每个高风险切片至少有 1 个**行为级**测试:
- 1h/1k 安全栈:已有 PASS 级测试,扩展到所有 path
- 1l GDPR:R2 修复
- 1i/1x/1y:已有部分 PASS,补完

**优先升级清单**(基于风险):
1. `auth_bcrypt_test.go`(T0-1h-5):加真 handler 调用,断言 wrong password → 401
2. `auth_cookie_test.go`(T0-1h-1 + T0-1k-6):加 httptest recorder,真查 `Set-Cookie` header
3. `migrations_test.go`(T0-1k-7/8):用 failing migration 跑,断言 os.Exit(1) 被调
4. `flusher_wiring_test.go`(T1-1d-1/3/GC):用 mock MinIO/PG/Redis
5. `observability_wiring_test.go`(1s × 13):用真 ctx + 真 logger,断言 Lifecycle 返回值被消费

**预计工时**:每项 1-2h,共 ~10h。

#### R4. e2e 1d-replay flaky 修复

**当前**:Run 1 失败,Run 2-4 通过。需先确认根因。

**调试步骤**:
1. 单独跑 `1d-replay.spec.ts:71` 10 次,取失败率
2. 如失败,加 console.log 看哪一步拿到 0 events
3. 检查测试 setup:是否每次 Run 前 truncate DB?是否 GC 在 test 期间触发?

**修复选项**:
- A. 测试 setup 加 `TRUNCATE sessions, event_blobs CASCADE` before each Run
- B. 测试用 unique tenant_id,隔离数据
- C. 增加 wait 时间让 MinIO 写入完成

#### R5. T0-1h-ui-3 虚标修复

**当前**:`session-expired.test.ts:71-89` 是 inline 重写 conditional 的 trivial test。

**目标**:mount 真 LoginView 组件 + i18n spy,验证 `t('login.error_session_expired')` 被调。

**实现**:
```ts
import { mount } from '@vue/test-utils';
import LoginView from '@/views/LoginView.vue';
import { useI18n } from 'vue-i18n';

it('LoginView shows SESSION_EXPIRED message', async () => {
  const wrapper = mount(LoginView, { ... });
  await wrapper.setData({ error: 'SESSION_EXPIRED' });
  expect(wrapper.text()).toContain('Session expired');
});
```

### 🟢 P2 — 长期改进

#### R6. 推广 ws-trace-inherit.test.ts 模式

**当前**:`ws-trace-inherit.test.ts` 是项目唯一的"真 mock WebSocket + decode sentBytes + 行为断言"测试。

**目标**:把这种模式推广到所有 SDK 测试。其他 SDK 测试(transport_recovery, collectors_wiring)都用源码契约 grep。

**模板**:
```ts
class MockWS {
  sentBytes: Uint8Array[] = [];
  send(data: Uint8Array) { this.sentBytes.push(data); }
  // ...
}

it('real behavior test', async () => {
  const mock = new MockWS();
  const sdk = new SDK({ ws: mock });
  await sdk.sendCommand(...);
  expect(mock.sentBytes).toHaveLength(1);
  const decoded = decode(mock.sentBytes[0]);
  expect(decoded.type).toBe('command');
});
```

#### R7. 引入 mutation testing CI(可选)

**当前**:本次审计手动跑 7 mutations,~30min。

**目标**:加 CI 集成 mutation testing,每月/每 release 跑一次。

**工具**:
- Go: [`go-mutesting`](https://github.com/zimmski/go-mutesting)(较老)或自写
- TS: [`stryker-js`](https://stryker-mutator.io/)(成熟)

**成本**:首次 setup ~4h,每次运行 ~30min。

**收益**:持续监控测试质量,防止源码契约测试疲劳再发生。

#### R8. 补 CI 90 天历史分析

**当前**:本审计因 `gh auth` 未登录,跳过 CI 历史分析。

**目标**:用户 `gh auth login` 后,跑:

```bash
gh run list --workflow=release.yml --limit=50
gh run list --workflow=ci.yml --limit=50
```

取 90 天 pass rate,补全 D4 历史 view。

---

## 7. 关键经验

### 7.1 源码契约测试的合理边界

**适合**:验证接线(如"调用了 X"作为重构回归保护)。

**不适合**:验证行为(如"密码错时返回 401")。

**当前状态**:项目用源码契约测试做"行为验证"的占位,导致 60% PARTIAL。这不是测试编写错误,是**测试策略选择**——用低成本的 grep 测试快速覆盖大量接线点。但代价是**测试不能验证语义正确**。

### 7.2 集成测试的"假阳性"陷阱

`TestDeleteVisitorByFingerprint_CascadesAllTables` 是**典型的"测试通过 ≠ 测试有效"**。测试看起来很严密(seed 5 表 + 断言 5 表 COUNT=0),但实际依赖 PG 特性(`ON DELETE CASCADE`)而非被测函数。

**经验**:集成测试必须**只依赖被测函数的行为**,不能依赖 DB/框架的副作用。否则函数 broken 时测试仍 PASS。

### 7.3 Mutation testing 是测试有效性的金标准

本次审计 7 个 mutation 中,2 个 survivor 揭示了**通过率 100% 的测试仍可能无效**。这印证了"测试覆盖率 ≠ 测试有效性"。

**经验**:重要切片应定期跑 mutation testing,作为对覆盖率的补充。

### 7.4 Badge 系统性虚标的根因

`T0-1h-ui-3` 是**第三次发现虚标**(继 2026-06-19 test-confidence-audit + 1ac/1ad 自查)。每次发现都是"测试看起来 cover 了 spec 决策点,但实际是 trivial test"。

**根因**:badge 判定标准偏"测试存在 + 触达路径",对"断言强度"要求不严。

**修复方向**:`docs/standards/verification-depth.md` 加一条:
> 🟢 strict 必须有 ≥1 个**真行为测试**(非源码契约),且该测试通过 mutation cross-check(改代码 → 测试红)。

---

## 8. Follow-up Slice Proposal

### 切片 `1ae test-health-followup`

**目标**:关闭本次审计发现的 P0 + P1 项。

**范围**:
- R1 operatorWS auth 行为级测试(P0)
- R2 erasure CASCADE 隔离测试(P0)
- R3 源码契约测试升级(P1,5 项)
- R4 e2e 1d-replay flaky 修复(P1)
- R5 T0-1h-ui-3 虚标修复(P1)

**预计工时**:~10-12 小时

**Verification Depth 目标**:
- 当前 🟡 verified-shallow → 🟢 touched(若 R1+R2+R5 完成)
- 升级 strict 需要 R3 全部完成 + 加 mutation CI(R7)

**完成后预期 badge 变化**:
- D1 PASS 率:38.2% → ≥70%(R3 完成)
- D3 Mutation Score:71.4% → ≥85%(R1+R2 修复后)
- D4 Flakiness:25% → <5%(R4 修复后)
- 整体 verdict:🔴 → 🟡(若 4 维全过 → 🟢)

### 不在本切片范围(留 post-v1 backlog)

- R6 ws-trace-inherit 模式推广
- R7 mutation testing CI
- R8 CI 90 天历史补全
- T2/T3 测试 gap(40 项,~15h,见 1ad 报告)

---

## 附录 A:Subagent 报告原始数据

- [`/tmp/d1-subagent-a.md`](/tmp/d1-subagent-a.md) — server/internal/api + cmd/server + migrations(24 闭包)
- [`/tmp/d1-subagent-b.md`](/tmp/d1-subagent-b.md) — storage + recording + antiscrape(9 闭包)
- [`/tmp/d1-subagent-c.md`](/tmp/d1-subagent-c.md) — observability + admin/tests + visitor-sdk/tests(35 闭包)

## 附录 B:Mutation testing 原始日志

- `/tmp/d4-go.log` — Go test ×20(20 PASS)
- `/tmp/d4-ts.log` — TS test ×20(20 PASS)
- `/tmp/d4-e2e.log` — e2e ×10(Run 1 FAIL,Run 2-4 PASS,Run 5-10 待跑)

## 附录 C:审计独立性声明

本审计未修改任何 production 代码或测试代码。所有 mutation 用 Edit 工具应用 + Edit 工具回滚,working tree 在每个 mutation 后回到 clean 状态(`git status` 验证)。

审计产出 docs/audits/2026-06-19-test-health-audit.md 单一文件,无其他副作用。
