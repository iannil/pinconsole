# 切片 1ai storage-repo-tests — Spec

**切片编号**:1ai
**类型**:测试深化(storage 包)
**创建时间**:2026-06-19
**状态**:approved
**关联**:[1ae impl](../completed/2026-06-19-slice-1ae-implementation.md)、[1ag~1aj 累计](../completed/2026-06-19-slice-1aj-followup-bugs-implementation.md)

## Context

为什么做这个切片?

- **触发原因**:storage 包覆盖率 20.1%,核心 repo 几乎全 0%:
  - user_repo:GetUserByEmail/GetUserByID/CreateUser/CountUsers 全 0%
  - session_repo:CreateSession/GetSession/TouchSessionEvent/EndSession/ListActiveSessionsByTenant 全 0%
  - 这些函数被 auth.go(login)、visitor WS、admin dashboard 高频调用
- **业务/技术价值**:补 PG 集成测试,捕获 SQL 回归(WHERE 条件、JOIN、CASCADE);为 1ai-b(把 api 包覆盖推到 50%+)打底。
- **不做的代价**:post-v1 refactor SQL 时,无回归保护,可能 silently 破坏认证/会话。

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 范围 | A: 全 storage repo(13 个函数) / B: user+session(9 个) / C: 仅 user(4 个) | **B** | A 工时 ~5h 超 budget;C 漏 session 核心路径;B 是认证+会话双核心,~2h |
| 2 | 测试模式 | A: 真 PG + skip(沿用 erasure_test.go) / B: pgxmock / C: testcontainers | **A** | A 已被 1ae/1ac 验证,与现有 helperPGPool 兼容;B 引入新依赖且 SQL 字符串校验弱;C 启动慢 |
| 3 | 数据隔离 | A: 真 uuid.New() 唯一 seed + cleanup / B: 测试 schema + TRUNCATE / C: 事务回滚 | **A** | A 沿用 erasure_test.go 模式;B 需建测试 schema 维护;C 不适用(测试 CREATE/INSERT) |
| 4 | 边界覆盖 | A: 仅 happy path / B: happy + not-found / C: + 边界值(null ua/ip) | **C** | A 太浅;B 中等;C 覆盖 scan 函数 null 处理(scanUser/scanSession 关键回归点) |
| 5 | 引入接口重构 | A: 同步做 Stores 接口化 / B: 不动 | **B** | 接口化是大手术,影响所有 handler;本切片只补测试,LLM friendly 原则;留 1ai-b |

## Acceptance(可验证的成功标准)

### 必须满足

- [ ] **A1**:新增 `server/internal/storage/user_repo_test.go`,含至少 5 个测试:
  - `TestCreateUser_AndRetrieve` — CreateUser → GetUserByEmail + GetUserByID 返回一致
  - `TestCreateUser_OnConflict_NoOp` — 同 email 二次插入返回旧行(`ON CONFLICT DO NOTHING`)
  - `TestGetUserByEmail_NotFound_ReturnsError` — 不存在 → error
  - `TestGetUserByID_NotFound_ReturnsError` — 不存在 → error
  - `TestCountUsers_ReturnsCount` — 至少 +1 后计数增加
- [ ] **A2**:新增 `server/internal/storage/session_repo_test.go`,含至少 6 个测试:
  - `TestCreateSession_AndRetrieve` — CreateSession → GetSession 字段一致
  - `TestCreateSession_NullUA_NullIP` — ua/ip 空字符串 → 字段为 nil
  - `TestTouchSessionEvent_IncrementsCount` — Touch 后 event_count 增加
  - `TestEndSession_SetsEndedAt` — EndSession 后 ended_at 非 null + status 更新
  - `TestListActiveSessionsByTenant_FiltersByStatus` — 只返 active
  - `TestListActiveSessionsByTenant_LimitClamped` — limit 生效

### 验证维度

- [ ] `go test ./internal/storage/` 全绿(预计 +11 测试,真 PG 不可用时全 skip)
- [ ] `go test ./...` 全绿(无 regression)
- [ ] storage 包覆盖率:**20.1% → ≥35%**(user+session 函数从 0% → 80%+)
- [ ] PG 可用性:本地 docker PG(`marketing-monitor-postgres-1`)在跑,测试真执行

### 不在本切片范围

- visitor_repo / command_repo / event_blob_repo / chat_repo / consent_repo(已被 1ad 部分覆盖)— 留 backlog
- Stores 接口重构(解锁 happy path 测试)— 留 1ai-b
- miniredis 替代真 Redis — 留 backlog
- testcontainers PG — 留 backlog

## 工时预算

| 项 | 工时 |
|---|---|
| 写 spec | 15min |
| user_repo_test.go 5 测试 | 45min |
| session_repo_test.go 6 测试 | 1h |
| 跑测试 + 覆盖率 + 报告 | 30min |
| **合计** | **~2.5h** |

## 完成后预期

- storage 包覆盖率:20.1% → ≥35%
- user_repo 函数:GetUserByEmail/GetUserByID/CreateUser 0% → 80%+
- session_repo 函数:CreateSession/GetSession/TouchSessionEvent/EndSession 0% → 80%+
- 1h 切片深度:🟢 touched → 维持(repo 层行为级覆盖加深)

## Verification Depth Badge 目标

🟢 touched — user_repo + session_repo 有 PG 集成测试,真 SQL 路径覆盖。

## 关联

- 前置:1aj(follow-up bugs 修完)
- 后续:1ai-b(Stores 接口重构,解锁 api happy path 测试)
