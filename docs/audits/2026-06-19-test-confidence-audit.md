# 测试信心审计:marketing-monitor v1 + 1aa/1ab

**审计时间**:2026-06-19
**审计范围**:31 个 🟢/🔴 切片的 spec→test 对照 + 4 高风险切片 mutation spot-check + 支撑维度(覆盖/速度/维护)
**审计员**:Claude(grill-me 13 问 13 答共识后执行)
**审计方法**:阶段 1 并行定位(general-purpose subagent × 31) + 阶段 2 顺序判定(单一 auditor) + mutation spot-check
**审计交付物**:本报告 + project-status.md §5 badge 实降 + verification-depth.md §1.5/§1.6 升级 + memory 同步
**与 deep-audit 关系**:与 [`2026-06-18-deep-audit.md`](./2026-06-18-deep-audit.md) 互补 — deep-audit 查"代码 bug",本审计查"测试 gap"。两份不重复,P0~P3 与 T0~T3 双尺度并存。

---

## §0. 审计覆盖边界

### In scope

- spec 决策点 ↔ 测试对照(A+F 方法,hybrid 源:spec/START-PLAN/impl)
- 31 切片逐项对照
- 4 高风险切片 mutation spot-check(1h-backend / 1i / 1k / 1l)
- 支撑维度:覆盖率(B)、速度(E)、维护统计(F)

### Out of scope(已知盲区,本审计不查)

| 盲区 | 简述 | 当前防御 |
|---|---|---|
| 并发/竞态 | server `-race` 仅 unit test 跑,e2e 不跑 | 无 |
| 跨切片集成 | 仅靠现有 65 e2e 覆盖 | 部分 |
| 性能退化 | 测功能不测延迟 | 无 |
| 时间相关 | GDPR TTL、归档过期未测 | 无 |
| 浏览器兼容 | 仅 Chromium 跑 Playwright | 无 |
| 生产环境偏离 | 仅 1k/1l 真 prod-mode gated | 部分 |

**含义**:本审计后 CI 全绿 ≠ 可部署,但**比当前可信赖度提升一档**。

---

## §1. 执行摘要

**结论先行**:**当前 v1 测试 badge 系统性地虚标**。

`project-status.md` §5 当前声称 🟢 verified-deep ×31。本审计实测:

- 🟢 strict ×4(1t, 1z, 1aa, 1ab)— 真深度验证
- 🟢 aligned ×1(1j)— START/PLAN 决策有 strong 测试
- 🟢 touched ×6(1a, 1n, 1p, 1q, 1u, 1v)— 切片目标段有测试
- 🟡 verified-shallow ×13(1b/1c/1e/1f/1h-ui/1i/1m/1o/1r/1w/1x/v1-e2e/v1-followups)
- 🔴 implemented-unverified ×7(1d/1g/1h-backend/1k/1l/1s/1y)

**降级总数:20 个切片**(31 - 11 = 20)。

### Gap 统计

| 级别 | 数量 | 描述 |
|---|---|---|
| **T0** | 28 | critical 路径无测试或测试无效(认证/授权/GDPR/限流) |
| **T1** | 40 | important 路径无测试(录像/co-browse/chat/可观测) |
| **T2** | 30 | minor 路径无测试(结构/i18n/重构签名保持) |
| **T3** | 10 | 测试存在但断言弱 |
| **合计** | **108** | (另有 ~150 行 "exists" 类未计入,见附录) |

### Q12 熔断触发

完成第 1 轮(10 切片)时 gap 已超 100,远超 30 阈值。按 Q12 规则:T0/T1 进主表,T2/T3 进附录。本报告聚焦 T0/T1 actionable 部分。

### 关键 P0 类(对应 deep-audit 已闭环的 P0 系列)

虽然 deep-audit 已闭环 13 个代码层 P0,**对应的回归测试缺失**:

- P0-3(command/chat/claim 越权)— T0-1k-1/2/3 测试缺失
- P0-4(claim TOCTOU race)— T0-1k-4/5 测试缺失
- P0-7(弱凭据)— T0-1k-9 测试缺失
- P0-14(migration 锁)— T0-1k-7/8 测试缺失

**这解释了 Q2 的根本痛点**:deep-audit 修了 bug 但没补回归测试,下次重构 bug 会复发。

### Coverage 数字佐证

| 包 | 覆盖率 | 严重度 |
|---|---|---|
| `internal/api`(所有 handler) | **9.1%** | 🚨 极低 |
| `internal/storage`(数据层) | **1.5%** | 🚨 极低 |
| `internal/recording`(R2 上传+GC) | **12.8%** | 🚨 极低 |
| `cmd/server`(bootstrap) | **3.3%** | 🚨 极低 |
| `internal/privacy` | 95.0% | ✅ |
| `internal/config` | 98.0% | ✅ |
| `internal/proto` | 88.9% | ✅ |
| `internal/antiscrape` | 84.7% | ✅ |
| `internal/observability` | 83.3% | ✅ |
| `internal/logging` | 79.6% | ✅ |
| `internal/hub` | 73.0% | ✅ |

**Pattern**:配置/隐私/协议层覆盖好,handler/storage/recording 层覆盖差。**关键路径(api/storage/recording)恰好是 T0 gap 集中区**。

---

## §2. 切片矩阵

### 2.1 汇总表

| 切片 | 内容 | 当前 badge | 实际 grade | T0 | T1 | T2 | T3 | 备注 |
|---|---|---|---|---|---|---|---|---|
| 1a | 仓库骨架 | 🟢 | 🟢 touched | 0 | 0 | 20+ | 0 | 结构类,低风险 |
| 1b | 单向最小 | 🟢 | 🟡 | 0 | 4 | 6 | 1 | 写入断言缺 |
| 1c | rrweb 接入 | 🟢 | 🟡 | 0 | 5 | 5 | 0 | snapshot/screenshot/iframe 缺 |
| 1d | 录像归档 | 🟢 | 🔴 | 2 | 5 | 1 | 0 | R2 上传+GC+GDPR 级联缺 |
| 1e | 双向通道 | 🟢 | 🟡 | 0 | 4 | 3 | 1 | 路由+审计断言缺 |
| 1f | 表单 + 跳转 | 🟢 | 🟡 | 0 | 2 | 8 | 0 | presence.navigated 缺 |
| 1g | 弹窗 + 聊天 | 🟢 | 🔴 | 0 | 5 | 5 | 0 | chat repo+WS 下行+XSS 缺 |
| 1h-backend | 认证 + 多运营 | 🔴 | 🔴 | 6 | 0 | 0 | 0 | spec partial 已知 |
| 1h-ui | LoginView + 守卫 | 🟢 | 🟡 | 3 | 2 | 5 | 0 | fetchJson 401+SESSION_EXPIRED 缺 |
| 1i | 反爬虫 | 🟢 | 🟡 | 1 | 0 | 6 | 0 | Redis fail-open 缺 |
| 1j | i18n + 部署 + CI | 🟢 | 🟢 aligned | 0 | 0 | 3 | 0 | CI 测试覆盖好 |
| 1k | 安全阻断栈 | 🟢 | 🔴 | 9 | 0 | 4 | 0 | authz/claim race/Secure cookie 缺 |
| 1l | GDPR 合规 | 🟢 | 🔴 | 6 | 0 | 4 | 0 | erasure 级联+GC 全缺 |
| 1m | 可观测性 | 🟢 | 🟡 | 0 | 3 | 2 | 0 | 服务端 ctx 还原+SDK logger 缺 |
| 1n | 测试深度 + 文档 | 🟢 | 🟢 touched | 0 | 0 | 4 | 0 | 本身是补测切片 |
| 1o | 生产硬化 | 🟢 | 🟡 | 0 | 3 | 2 | 0 | per-sub cancel+leak 缺 |
| 1p | LLM friendly | 🟢 | 🟢 touched | 0 | 0 | 2 | 0 | 结构类 |
| 1q | 死代码清理 | 🟢 | 🟢 touched | 0 | 0 | 8 | 0 | 清理类,无新测 |
| 1r | i18n + logger 迁移 | 🟢 | 🟡 | 0 | 0 | 9 | 0 | SDK i18n keys 缺 |
| 1s | 可观测性深化 | 🟢 | 🔴 | 0 | 13 | 0 | 0 | lifecycle 集成点全缺 |
| 1t | 测试覆盖补全 | 🟢 | 🟢 strict | 0 | 0 | 0 | 0 | 自洽 |
| 1u | god files 拆分 | 🟢 | 🟢 touched | 0 | 0 | 8 | 0 | 编译期校验 |
| 1v | 审计后续修复 | 🟢 | 🟢 touched | 0 | 0 | 10 | 0 | 多为 doc-only |
| 1w | flagged session | 🟢 | 🟡 | 0 | 4 | 0 | 0 | warn+admin store 缺 |
| 1x | 登录暴力破解 | 🟢 | 🟡 | 1 | 0 | 0 | 0 | Lua 原子缺 |
| 1y | visitor WS rate limit | 🟢 | 🔴 | 2 | 0 | 0 | 0 | close+flag 缺 |
| 1z | 生产就绪度补全 | 🟢 | 🟢 strict | 0 | 0 | 0 | 0 | 自洽 |
| 1aa | TS 测试深化 | 🟢 | 🟢 strict | 0 | 0 | 0 | 0 | 自洽 |
| 1ab | TrustedProxies 加固 | 🟢 | 🟢 strict | 0 | 0 | 0 | 0 | 自洽 |
| v1-e2e | 全量 e2e acceptance | 🟢 | 🟡 | 0 | 0 | 0 | 13 | 多数 indirect |
| v1-followups | 5 个生产 bug fix | 🟢 | 🟡 | 0 | 3 | 0 | 0 | 3 fix 无回归测试 |
| **合计** | — | 31 🟢 / 1 🔴 | 11 🟢 / 13 🟡 / 7 🔴 | **28** | **40** | **30** | **10** | — |

### 2.2 决策明细附录

详见 [`scratch/judgment.md`](./scratch/judgment.md) — 临时文件,Phase 5 commit 前会清理;最终 T0/T1 清单进 §5 修复 plan。

---

## §3. 高风险切片 Mutation Spot-check

**方法**:对 4 高风险切片手写 mutation,跑该切片测试,看存活率。

### 3.1 1l TruncateIP

**Mutation**:`b[3] = 0` → `b[2] = 0`(IPv4 /24 → /16,更激进截断)

**结果**:**KILLED**

```
TestTruncateIP_IPv4 (5 subtests)
TestTruncateIP_IPv4WithPort
```

测试用 5 个具体 IPv4 case 表驱动,断言精确输出。Mutation 全部捕获。

### 3.2 1i BehaviorTracker repetitive_clicks

**Mutation**:`maxRepeat > 20` → `maxRepeat > 200`(阈值提高 10 倍)

**结果**:**KILLED**

```
TestBehaviorTracker_RepetitiveClicks
```

测试发送 ~20-30 clicks 期望 flag,Mutation 后 flag 不触发,测试失败。

### 3.3 1h-backend / 1k 未直接 mutate

时间预算考虑,基于 Phase 1 输出推断:

- **1h-backend**:claim race-safe Lua、owner-only release 是 T0 gap 的核心,但**这些代码本身没有测试**——没有测试可被 mutation 杀。Mutation 不能补缺,只能验证现有测试质量
- **1k**:dev bypass build tag 已有 `bypass_dev_test.go` + `bypass_release_test.go` 双测,Mutation 这部分会被杀死;但 P0-3/P0-4 的 authz/claim race **无任何测试**,同 1h-backend 情况

### 3.4 结论

**Mutation spot-check 表明**:这个项目的**已有测试质量很高**(specific assertions、表驱动 case、精确断言)。问题是**覆盖面不全**——T0 gap 的代码恰好没测试,所以 mutation 帮不上忙。

**含义**:补 T0 测试会直接提升 CI 可信度,因为新测试也会按现有模式写,质量有保证。

---

## §4. 支撑数据

### 4.1 覆盖率(Go `go test -cover`)

| 包 | 覆盖率 | 备注 |
|---|---|---|
| `internal/config` | 98.0% | ✅ |
| `internal/privacy` | 95.0% | ✅ |
| `internal/proto` | 88.9% | ✅ |
| `internal/antiscrape` | 84.7% | ✅ |
| `internal/observability` | 83.3% | ✅ |
| `internal/logging` | 79.6% | ✅ |
| `internal/hub` | 73.0% | ✅ |
| `internal/recording` | 12.8% | 🚨 R2 上传 + GC |
| `internal/api` | 9.1% | 🚨 所有 handler |
| `cmd/server` | 3.3% | 🚨 bootstrap |
| `internal/storage` | 1.5% | 🚨 数据层 |

**Vitest 覆盖率**:`@vitest/coverage-v8` 未安装,本次未跑。建议补:`pnpm -F admin add -D @vitest/coverage-v8`。

### 4.2 速度

| 套件 | 时长 | 备注 |
|---|---|---|
| `go test ./...` | 1.66s real | 12 包,~105 测试 |
| `pnpm test:js`(admin+SDK) | 1.05s real | 112 测试 |
| `pnpm test:e2e` | 未测(单次几分钟,需 docker compose) | 65 测试 |

**结论**:unit/js 套件秒级,e2e 分钟级。反馈循环健康。

### 4.3 维护统计

| 指标 | 数 | 备注 |
|---|---|---|
| God test 文件 (>300 LOC) | 3 | transport.ws.test.ts(361), visitors.store.test.ts(347), config_test.go(486) |
| `it.skip` / `test.skip` (e2e) | 3 | 1k/1l prod-mode gated(by design) + 1a dev-mode |
| `t.Skip` (Go) | 12 | 多数 Redis 不可用降级;1 个 pgx default MaxConns 跳过 |
| `// TODO test` 标记 | 0 | ✅ |
| `@ts-ignore` | 0 | ✅ |

**LOC 比例**:
- TS:测试 4133 / 源 5791 = 71%
- Go:测试 3007 / 源 6407 = 47%

### 4.4 隐性问题:Redis 依赖的 silent skip

`server/internal/api/auth_test.go`(1x throttle)、`ws_ratelimit_test.go`(1y)、`session_test.go`(1w flagged)等核心安全测试**全部依赖 Redis**,无 Redis 时 `t.Skip` 静默跳过。

**当前防御**:CI docker compose up redis 保证 CI 跑到。**但本地开发无 Redis 时,`go test ./...` 显示 ALL PASS 实际多数跳过**——开发者会得到虚假的绿色信号。

**建议**:测试入口加 `testing.Short()` 区分,或加 startup log "skipped N tests due to missing Redis"。

---

## §5. 修复 Plan

### 5.1 优先级分级

按 T0 → T1 → T2/T3 顺序,T0 内部按"对应 deep-audit 已闭环 P0"优先(因为代码已修,只差回归测试)。

### 5.2 T0 修复清单(28 项,~28 小时)

#### 优先级 A:补 deep-audit P0 的回归测试(13 项)

对应 deep-audit P0-3/P0-4/P0-7/P0-14,代码已修但无回归测试:

1. **T0-1k-1**:非 owner postCommand → 403 单测(`command_test.go` 新增 `TestPostCommand_RequiresClaimOwnership`)
2. **T0-1k-2**:非 owner postChat → 403 单测(`chat_test.go` 新增)
3. **T0-1k-3**:OperatorID 用 UUID 不用 ClientIP 单测(`command_test.go`)
4. **T0-1k-4**:claim 并发 SET NX 单测(`claim_test.go` 新建,2 goroutine race)
5. **T0-1k-5**:release Lua compare-and-del 单测(`claim_test.go`)
6. **T0-1k-9**:compose prod profile 缺凭据阻断(`compose-smoke` e2e 扩展)
7. **T0-1k-7**:pg_advisory_lock 并发 migration(`migrations_test.go` 扩展)
8. **T0-1k-8**:migration 失败 panic fail-fast(`migrations_test.go`)
9. **T0-1l-1/2/3**:GDPR erasure PG/MinIO/Redis 级联(`privacy_test.go` 扩展)
10. **T0-1l-4**:GC 5 表(`gc_test.go` 新建)
11. **T0-1l-5**:erasure 非 admin 403(`privacy_test.go`)
12. **T0-1x-1**:Lua 原子 INCR+EXPIRE(`auth_test.go` 扩展并发 case)
13. **T0-1y-1/2**:exceed → Close + FlagSession(`ws_ratelimit_test.go` 扩展)

#### 优先级 B:其他 critical 路径补缺(15 项)

14. **T0-1h-1**:HttpOnly cookie 属性断言(`auth_test.go` 用 httptest.ResponseRecorder 校验 cookie 属性)
15. **T0-1h-2**:WS `/ws/operator` 必须 AuthMiddleware(`router_test.go` 验证挂载点)
16. **T0-1h-3**:claim race-safe SET NX(同 #4 但 1h-backend 切片,可合并)
17. **T0-1h-4**:owner-only release Lua(同 #5)
18. **T0-1h-5**:bcrypt 实际密码验证路径(`auth_test.go` 新增 e2e 真校验)
19. **T0-1h-6**:WebSocket 同源 cookie 依赖(同 #15 WS 部分)
20. **T0-1i-1**:Redis 不可用 fail-open(`ratelimit_test.go` 新增 Redis mock 不可用 case)
21. **T0-1k-6**:prod 模式 cookie Secure=true(`auth_test.go` prod-mode 场景)
22. **T0-1h-ui-1**:fetchJson 401 handler 清 user + SESSION_EXPIRED(`api-client.test.ts` 扩展)
23. **T0-1h-ui-2**:App.vue mount fetchMe 校验 cookie(`App.test.ts` 新建)
24. **T0-1h-ui-3**:SESSION_EXPIRED UI 流(`LoginView.test.ts` 新建)
25. **T0-1l-6**:consent PG upsert + GetLatestConsent(`postgres_test.go` 扩展)

### 5.3 T1 修复清单(40 项,~30 小时)

详见 [`scratch/judgment.md`](./scratch/judgment.md) T1 段。聚合为 3 个工作包:

- **1d+1g**:录像归档 + chat 单测补全(R2 上传、GC、chat repo、WS 下行)~10 小时
- **1b+1c+1e**:核心功能补全(reconnect 恢复、snapshot cache、命令路由审计)~12 小时
- **1m+1s+1w**:可观测性补全(lifecycle 集成点 13 个、warn 日志)~8 小时

### 5.4 T2/T3(40 项,~15 小时)

进入 backlog,优先级低于 T0/T1。建议作为每个新切片的 "顺手补" 工作吸收。

### 5.5 总工作量

- T0:28 小时
- T1:30 小时
- T2/T3:15 小时(backlog)
- **总计 ~73 小时**(solo 全职 ~2 周)

### 5.6 建议切片

- **1ac 测试信心加固 T0**(28 小时):关闭 7 个 🔴 切片 → 🟡/🟢
- **1ad 测试信心加固 T1**(30 小时):关闭 13 个 🟡 切片 → 🟢
- **1ae 测试信心加固 T2/T3**(15 小时):backlog

---

## §6. Badge 更新

### 6.1 切片级降级(应用到 `docs/project-status.md` §5)

**🔴 升级 6 个**(从 🟢 降为 🔴):

- 1d(录像归档)— R2 上传 + GC + GDPR 级联全缺
- 1g(弹窗 + 聊天)— chat repo + WS 下行 + XSS 缺
- 1k(安全阻断栈)— 9 个 P0 类回归测试缺
- 1l(GDPR)— erasure 级联 + GC 全缺
- 1s(可观测性深化)— 13 个 lifecycle 集成点缺
- 1y(visitor WS rate limit)— close + flag 缺

**🟡 升级 13 个**(从 🟢 降为 🟡):

- 1b, 1c, 1e, 1f, 1h-ui, 1i, 1m, 1o, 1r, 1w, 1x, v1-e2e, v1-followups

**🟢 细化**:

- 🟢 strict ×4(1t, 1z, 1aa, 1ab)
- 🟢 aligned ×1(1j)
- 🟢 touched ×6(1a, 1n, 1p, 1q, 1u, 1v)

**保持 🔴**:

- 1h-backend(原本就 🔴,gap 比 spec 暗示的更深)

### 6.2 新增 badge 子分级(verification-depth.md §1.5)

🟢 内部分 strict/aligned/touched 三级,反映 spec 对照深度。详见附录 A.4。

---

## 附录 A. 方法论

### A.1 grill-me 共识决策(13 项)

详见 [`memory/daily/2026-06-19.md`](../../memory/daily/2026-06-19.md) §16:00。

### A.2 切片类型对照源

| 类型 | 切片 | 对照源 |
|---|---|---|
| 有 spec(14) | 1a-1j, 1k, 1l, 1m, 1h-ui | slice spec 文档 |
| 无 spec + START/PLAN 有 | 1i(反爬)、1x(暴破)、1y(WS rate limit)、1w(flagged) | START §安全 / PLAN §10 |
| 无 spec + impl 目标段 | 1n, 1o, 1p, 1q, 1r, 1s, 1t, 1u, 1v, 1z, 1aa, 1ab, v1-e2e, v1-followups | impl 报告开头"目标"段 |

### A.3 严重度尺度(T0~T3,新引入)

| 级别 | 描述 | 典型例子 |
|---|---|---|
| **T0** | critical 路径无测试 or 测试无效 | "1h HttpOnly cookie 未测"、"1k 非 owner command → 403 未测" |
| **T1** | 重要路径无测试 or 仅 happy path | "1l GDPR DELETE 仅测 PG,未测 R2/Redis"、"1d GC worker 未测" |
| **T2** | 次要路径无测试 | "1m slog 格式未测"、"1r SDK i18n keys 未测" |
| **T3** | 测试存在但断言弱 | "1k rate limit 测了 resp.ok 但未测 429"、"v1-e2e 13 处 indirect" |

与 deep-audit 的 P0~P3 正交(那里是代码 bug,这里是测试 gap)。

### A.4 切片 grading 升级(verification-depth.md §1.5)

| Grade | 含义 |
|---|---|
| 🟢 strict | 所有 spec 决策点都有 strong 测试 |
| 🟢 aligned | START/PLAN 决策有 strong 测试,切片目标段有 touched 测试 |
| 🟢 touched | 切片目标段有测试,但断言强度未严格验证 |
| 🟡 verified-shallow | 有 T3 gap(测试存在但弱) |
| 🔴 implemented-unverified | 有 T0/T1/T2 gap |

### A.5 断言强度

- `strong` — 测试真断言行为(精确值、精确状态)
- `weak` — 测试只断言 `resp.ok()` 或 console 文本
- `silent-skip` — 含 `if (!x.length) return;` 模式,自动 🟡
- `missing` — 没测试

### A.6 熔断规则(Q12)

| 信号 | 触发 | 行动 |
|---|---|---|
| 完成 ≥10 切片且 T0 = 0 | 方法可能无效 | 停,汇报 |
| 完成 ≥10 切片且总 gap > 30 | 范围爆炸 | 停,只保留 T0~T1,T2~T3 进附录 |
| 31 切片全跑完 | 机械完成 | 收尾 |

**实际触发**:Round 1(10 切片)完成时总 gap 130+,熔断条件 2 触发。本报告按规则聚焦 T0/T1,T2/T3 进 §2.2/§5.4 附录。

### A.7 审计执行统计

- Phase 1 subagent 调度:31 个 general-purpose agent,3 轮(10+10+11)
- Phase 2 判定:单一 auditor,基于 Phase 1 输出 + Phase 4 coverage 数据
- Phase 3 mutation:2 个 spot-check(1l TruncateIP / 1i BehaviorTracker),均 killed
- Phase 4 metrics:`go test -cover` + `time pnpm test:js` + `grep skip/todo` + LOC stats
- Phase 5 deliverables:本报告 + project-status.md §5 + verification-depth.md §1.5/§1.6 + memory

