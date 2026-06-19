# 切片 1ai-c auth-handler-interface — Spec(Phase 1)

**切片编号**:1ai-c
**类型**:重构 + 测试深化(api 包)
**创建时间**:2026-06-20
**状态**:approved
**关联**:[1ag impl](../completed/2026-06-19-slice-1ag-api-handler-behavioral-tests-implementation.md)、[1ai-b impl](../completed/2026-06-20-slice-1aib-storage-repos-b-implementation.md)

## Context

为什么做这个切片?

- **触发原因**:api 包覆盖 29.3%,login/logout/me 等 happy path 全靠 e2e 兜底。1ag 给了拒绝路径测试,但 happy path(成功登录、正确密码 bcrypt 验证)无单测。原因:`*storage.Stores` 是具体类型,无法注入 mock。
- **业务/技术价值**:Phase 1 仅重构 AuthHandler,验证"accept interfaces, return structs"模式在 api 包可行;后续 1ai-d/1ai-e 同模式扩展到 claim/chat/replay 等,最终把 api 推到 50%+。
- **不做的代价**:post-v1 改 auth 逻辑(如加 MFA、加 SSO)时,无 happy path 单测保护,bcrypt 验证流回归无人发现。

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 范围 | A: 全 6 handler / B: 仅 auth / C: auth + claim | **B** | A 工时 ~10h 超 budget + 大 churn;B 验证模式 PoC,~3h;C 中间但 claim 的 requireClaimOwnership 增加 mock 复杂度 |
| 2 | 接口位置 | A: storage 包定义 / B: api 包定义 | **B** | "accept interfaces, return structs" 原则;api 包作为消费者定义自己需要的最小接口;storage 不污染 |
| 3 | AuthHandler 字段 | A: 单一 stores interface / B: 拆 userRepo + redisStore | **B** | 单一 stores 太宽(暴露 100+ 方法),违反 ISP;拆分让 mock 简单 |
| 4 | TTL 路径 | A: 加 TTL 到 storage.Redis / B: 接口暴露 Client | **A** | 暴露 Client 等于泄漏实现;加 TTL 到 storage.Redis 让接口干净 |
| 5 | 既有调用方 | A: 改 NewAuthHandler 签名 / B: 保持 + 内部转换 | **B** | 保持外部 API 不变;构造函数从 *storage.Stores 抽出 PG/Redis 适配接口 |
| 6 | Mock 实现 | A: 手写 / B: mockgen / C: gomock | **A** | 手写 2 个 mock(UserRepo + RedisStore),简单可控;B/C 引入新工具链超 scope |

## Acceptance(可验证的成功标准)

### 必须满足

- [ ] **A1**:`internal/api/auth_interfaces.go` 定义 2 接口:
  - `authUserRepo`:GetUserByEmail / GetUserByID
  - `authRedisStore`:Get / Set / Del / EvalLua / TTL
- [ ] **A2**:`storage/redis.go` 加 `TTL(ctx, key) (time.Duration, error)` 方法(替代 auth.go 直接调 `Client.TTL`)
- [ ] **A3**:`auth.go` AuthHandler 字段改为接口类型;`NewAuthHandler` 从 `*storage.Stores` 抽取 PG/Redis(签名不变)
- [ ] **A4**:`router.go` 调用方无需改动(NewAuthHandler 仍接受 *storage.Stores)
- [ ] **A5**:新增 `auth_happy_path_test.go`,3 测试用 mock:
  - `TestLogin_Success_Returns200_SetCookie_Body` — mock user + bcrypt hash 匹配 → 200 + Set-Cookie + meResponse
  - `TestLogin_WrongPassword_Returns401_NoCookie` — mock user + bcrypt 不匹配 → 401 + 无 Set-Cookie
  - `TestLogin_UserNotFound_Returns401_RecordsFailure` — mock 返回 error → 401 + recordLoginFailure 被调

### 验证维度

- [ ] `go test ./...` 全绿(无 regression,现有 12 个 auth 相关测试仍 PASS)
- [ ] api 包覆盖率:**29.3% → ≥35%**(login happy path 3 个分支覆盖)
- [ ] auth.go `login` 覆盖率:**37.8% → ≥70%**
- [ ] Mutation 抽样 2 项 KILLED
- [ ] `go vet` + `gofmt` 干净(仅本切片新文件)

### 不在本切片范围

- claim.go / chat.go / replay.go / session.go / command.go 接口重构 — 留 1ai-d+ 后续切片
- storage.Redis 全部方法接口化(仅加 TTL) — 留后续按需扩展
- mockgen / gomock 工具链 — 留 post-v1 评估

## 工时预算

| 项 | 工时 |
|---|---|
| 写 spec | 15min |
| 定义接口 + storage/redis.go 加 TTL | 30min |
| 改 auth.go + 验证编译 | 45min |
| 写 3 happy path 测试 + mock | 1.5h |
| 跑测试 + mutation + 报告 | 30min |
| **合计** | **~3.5h** |

## 完成后预期

- api 包覆盖率 29.3% → ≥35%
- auth.go login 37.8% → ≥70%(happy path + 错密码 + 用户不存在)
- 1h 切片深度:🟢 touched → 维持(happy path 行为级覆盖加深)
- 接口化模式 PoC 完成,1ai-d+ 可复用

## Verification Depth Badge 目标

🟢 touched — AuthHandler 接口化 + 3 happy path 测试 + mutation KILLED。

升 🟢 strict 需 mutation CI(R7),留 post-v1。

## 关联

- 前置:1ag(auth HTTP 拒绝路径)、1ai-b(storage repo 全覆盖)
- 后续:1ai-d(claim handler 接口化)、1ai-e(chat/replay handler 接口化)
- 终态:api 包覆盖 ≥50%,所有 happy path 单测覆盖

## Notes

**为什么不直接用真 PG + 真 Redis seed 写 happy path**(跳过重构)?

- 优点:无生产代码改动,~1.5h
- 缺点:依赖 docker PG/Redis 在 CI 跑(CI 配置未确认),且 bcrypt hash seed 较慢
- 取舍:重构虽然贵但解锁**所有** handler 的 happy path,长期 ROI 高;真 infra 测试是 1ai-c 失败时的 plan B

**为什么不引入 mockgen?**

- 当前 mock 简单(2 接口、~5 方法),手写 ~50 行可控
- 引入 mockgen 加 toolchain 复杂度,违反 LLM friendly
- 未来 mock 数量 >5 个时再评估
