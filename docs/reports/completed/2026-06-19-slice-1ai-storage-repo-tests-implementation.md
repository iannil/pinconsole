# 切片 1ai storage-repo-tests — Implementation

**切片编号**:1ai
**类型**:测试深化(storage 包)
**创建时间**:2026-06-19
**状态**:completed
**关联**:[spec](./2026-06-19-slice-1ai-storage-repo-tests-spec.md)、[1ae impl](../completed/2026-06-19-slice-1ae-implementation.md)、[1ag~1aj 累计](../completed/2026-06-19-slice-1aj-followup-bugs-implementation.md)

## Context

storage 包覆盖率 20.1%,核心 repo 几乎全 0%。本切片用 1ae 既定 PG 集成测试模式补 user_repo + session_repo,把 storage 覆盖推到 ≥35%。

## 执行摘要

| 维度 | 结果 |
|---|---|
| 新增测试 | **11 个**(user_repo 5 + session_repo 6) |
| 新增测试文件 | 2(`user_repo_test.go`、`session_repo_test.go`) |
| storage 包覆盖率 | **20.1% → 39.2%**(+19.1pp,超 spec 目标 35%) |
| Go 全测试 | ✅ ALL PASS(12 包) |
| Mutation 抽样 | ✅ 2/2 KILLED |

## 新增测试列表

### `user_repo_test.go`(5 测试)

| 测试 | 验证 |
|---|---|
| `TestCreateUser_AndRetrieve` | CreateUser → GetUserByEmail + GetUserByID 字段一致 |
| `TestCreateUser_OnConflict_NoOp` | 同 email 二次插入 → ErrNoRows + 原行未覆盖 |
| `TestGetUserByEmail_NotFound_ReturnsError` | 不存在 → error |
| `TestGetUserByID_NotFound_ReturnsError` | 不存在 → error |
| `TestCountUsers_ReturnsCount` | 创建后计数 +1 |

### `session_repo_test.go`(6 测试)

| 测试 | 验证 |
|---|---|
| `TestCreateSession_AndRetrieve` | CreateSession → GetSession 字段一致 |
| `TestCreateSession_NullUA_NullIP` | 空字符串 → NULL(防 `if ua != ""` 误改) |
| `TestTouchSessionEvent_IncrementsCount` | Touch 5+10+3 → EventCount=18(防"覆盖"而非"累加"回归) |
| `TestEndSession_SetsEndedAt` | EndSession 后 ended_at 非 null + status 更新 |
| `TestListActiveSessionsByTenant_FiltersByStatus` | 只返 active,ended 不混入 |
| `TestListEndedSessionsByTenant_FiltersByWindow` | 窗口过滤生效(24h 命中,1ms 边界排除) |

## 覆盖率前后对比

### `storage/user_repo.go`

| 函数 | 1ai 前 | 1ai 后 |
|---|---|---|
| `GetUserByEmail` | 0% | **100%** |
| `GetUserByID` | 0% | **100%** |
| `CreateUser` | 0% | **100%** |
| `CountUsers` | 0% | **100%** |

### `storage/session_repo.go`

| 函数 | 1ai 前 | 1ai 后 |
|---|---|---|
| `CreateSession` | 0% | **100%** |
| `GetSession` | 0% | **100%** |
| `TouchSessionEvent` | 0% | **100%** |
| `EndSession` | 0% | **100%** |
| `ListActiveSessionsByTenant` | 0% | **78.9%**(rows.Err() 路径未覆盖,可接受) |
| `ListEndedSessionsByTenant` | 0% | **78.9%**(同上) |

### storage 包总体

**20.1% → 39.2%**(+19.1pp)

## Mutation 验证(抽样 2 项)

| Mutation | 应失败测试 | 结果 |
|---|---|---|
| `GetUserByEmail` SQL `$2 → $1`(email 改为按 tenant_id 二次绑定) | `TestCreateUser_AndRetrieve` | ✅ KILLED(PG 触发 `operator does not exist: text = uuid`) |
| `TouchSessionEvent` SQL 删 `event_count +`(覆盖而非累加) | `TestTouchSessionEvent_IncrementsCount` | ✅ KILLED(EventCount=3 而非 18) |

## 关键模式

### 真 PG + skip 模式

沿用 1ac `helperPGPool` helper:
- PG 可用 → 真跑 SQL,捕获 WHERE/JOIN/CASCADE 回归
- PG 不可用 → `t.Skipf`,不阻塞 CI
- 数据隔离:`uuid.New()` 后缀的 email/fingerprint + `defer cleanup`

### `TestCreateUser_OnConflict_NoOp` 期望 ErrNoRows

PG 行为:`ON CONFLICT DO NOTHING` + `RETURNING` 冲突时**不返行**,
`pgx.QueryRow.Scan` 返回 `pgx.ErrNoRows`。这与直觉(期望返回旧行)不同,
值得专门测试覆盖(防回归把 `DO NOTHING` 改成 `DO UPDATE`)。

### `TestTouchSessionEvent_IncrementsCount` 累加验证

 Touch 3 次(5+10+3 = 18),验证 EventCount=18。这是防"覆盖"回归:
 ```sql
 UPDATE sessions SET event_count = event_count + $2  -- ← 累加,正确
 UPDATE sessions SET event_count = $2                -- ← 覆盖,错误
 ```

### `TestListEndedSessionsByTenant_FiltersByWindow` 边界用 1ms

`time.Sleep(2ms)` + 窗口 1ms → 刚结束的 session 已超出窗口,不应返。
这种"小窗口边界"测试比"大窗口命中"更能捕获 WHERE 子句的边界 bug。

## Verification Depth Badge

**🟢 touched** — user_repo + session_repo 有 PG 集成测试,2 项 mutation KILLED。

切片深度:
- **1h 认证 + 多运营 后端**:🟢 touched(加深:user_repo 100% 覆盖)
- **1b 单向最小**:🟡 → 🟢 touched(session_repo 100% 覆盖,创建/读取/touch/end 全测)

## Follow-up(留 backlog)

1. **visitor_repo / command_repo / event_blob_repo** — 同模式可推 storage 覆盖到 60%+(留 1ai-b)
2. **storage Connect/Ping/Close** — 0% 覆盖,但属于"启动期"代码,价值低
3. **Stores 接口重构** — 解锁 api 包 happy path 测试,推 api 包到 50%+(留 1ai-c)

## 提交

建议 2 个 commit:

1. `test(1ai): user_repo PG 集成测试 — CRUD + ON CONFLICT(5 测试)`
2. `test(1ai): session_repo PG 集成测试 — CreateSession/Get/Touch/End/List(6 测试)`

## 下一步

### 立即可做

- 用户审阅 + commit 1ag/1ah/1aj/1ai(共 8 个建议拆分)
- 把 1ai spec + impl 移到 `docs/reports/completed/`
- 更新 `project-status.md` 加 1ai 行 + 升 1b 切片深度

### 短期 backlog

- **1ai-b**:visitor_repo / command_repo / event_blob_repo 测试(同模式,~2-3h)
- **1ai-c**:Stores 接口重构(解锁 api happy path,~5-8h)
- 修剩余 gofmt 问题(~30min,纯格式)

### 长期(post-v1)

- 自定义域名 / 页面编辑器 / Tauri
- Mutation CI 集成(R7)
