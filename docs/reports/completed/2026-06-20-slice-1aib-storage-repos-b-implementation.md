# 切片 1ai-b storage-repos-b — Implementation

**切片编号**:1ai-b
**类型**:测试深化(storage 包续)
**创建时间**:2026-06-20
**状态**:completed
**关联**:[spec](./2026-06-20-slice-1aib-storage-repos-b-spec.md)、[1ai impl](../completed/2026-06-19-slice-1ai-storage-repo-tests-implementation.md)

## Context

1ai 把 storage 覆盖推到 39.2%。本切片续做 visitor/command/event_blob 3 个 repo,目标 55%+。

## 执行摘要

| 维度 | 结果 |
|---|---|
| 新增测试 | **11 个**(visitor 3 + command 4 + event_blob 4) |
| 新增测试文件 | 2(`command_repo_test.go`、`event_blob_repo_test.go`)+ visitor_repo_test.go 追加 |
| storage 包覆盖率 | **39.2% → 57.8%**(+18.6pp,超 spec 目标 55%) |
| Go 全测试 | ✅ ALL PASS(12 包) |
| Mutation 抽样 | ✅ 2/2 KILLED |

## 新增测试列表

### `visitor_repo_test.go`(追加 3 测试到现有 1v 文件)

| 测试 | 验证 |
|---|---|
| `TestCreateVisitor_AndRetrieve_1aib` | Create → GetVisitorByFingerprint 字段一致 |
| `TestCreateVisitor_OnConflict_UpdatesLastSeen_1aib` | 重复触发 ON CONFLICT,last_seen_at + COALESCE ua 覆盖 |
| `TestGetVisitorByFingerprint_NotFound_ReturnsNil_1aib` | 真 PG 验证 (nil, nil) 行为 |

### `command_repo_test.go`(4 测试)

| 测试 | 验证 |
|---|---|
| `TestCreateCoBrowsingCommand_AndList` | Create + List 字段一致(含 JSON payload 语义比较) |
| `TestCreateCoBrowsingCommand_NullNodeID` | TargetNodeID=nil → DB NULL(非 0) |
| `TestListCoBrowsingCommandsBySession_OrderedByCreated` | 多条按 created_at 正序 |
| `TestDeleteCoBrowsingCommandsOlderThan` | 阈值前的删,后的留 |

### `event_blob_repo_test.go`(4 测试)

| 测试 | 验证 |
|---|---|
| `TestCreateEventBlob_AndList` | Create + ListBySession 字段一致 |
| `TestListEventBlobsBySession_OrderedByBlobIndex` | 乱序创建,期望按 blob_index ASC |
| `TestListEventBlobsOlderThan_FiltersAndLimits` | threshold 过滤 + LIMIT 生效 |
| `TestDeleteEventBlobByID_ExistingAndMissing` | 已存在 + 不存在 ID 都不报错(幂等) |

## 覆盖率前后对比(1ai → 1ai-b)

### visitor_repo.go

| 函数 | 1ai 后 | 1ai-b 后 |
|---|---|---|
| `GetVisitorByFingerprint` | 0% | **85.7%** |
| `CreateVisitor` | 0% | **100%** |

### command_repo.go

| 函数 | 1ai 后 | 1ai-b 后 |
|---|---|---|
| `CreateCoBrowsingCommand` | 0% | **90.0%** |
| `ListCoBrowsingCommandsBySession` | 0% | **84.6%** |
| `DeleteCoBrowsingCommandsOlderThan` | 0% | **100%** |

### event_blob_repo.go

| 函数 | 1ai 后 | 1ai-b 后 |
|---|---|---|
| `CreateEventBlob` | 0% | **83.3%** |
| `ListEventBlobsBySession` | 0% | **81.8%** |
| `ListEventBlobsOlderThan` | 0% | **81.8%** |
| `DeleteEventBlobByID` | 100% | 100%(1ad 已覆盖) |

### storage 包总体

**39.2% → 57.8%**(+18.6pp)

## Mutation 验证(抽样 2 项)

| Mutation | 应失败测试 | 结果 |
|---|---|---|
| `ListCoBrowsingCommandsBySession` SQL `ORDER BY ASC → DESC` | `TestListCoBrowsingCommandsBySession_OrderedByCreated` | ✅ KILLED(顺序反转) |
| `ListEventBlobsBySession` SQL `ORDER BY ASC → DESC` | `TestListEventBlobsBySession_OrderedByBlobIndex` | ✅ KILLED(blob_index 反序) |

## 关键模式

### PG JSONB 字节级比较会假阳性

PG JSONB 列存时会规范化空格/键序:
```
输入:{"x":123,"y":456}
存储:{"x": 123, "y": 456}  ← 注意 ": " 空格
```

直接 `bytes.Equal(stored, input)` 会假阳性失败。`command_repo_test.go` 引入
`assertJSONEqual` helper(用 `json.Unmarshal` 后递归比较 any / map / slice / scalar)。

### visitor_repo_test.go 1v + 1ai-b 并存

1v 已有 `TestGetVisitorByFingerprint_ErrNoRowsContract`(用 mock 验证编译时契约),
1ai-b 追加 3 个真 PG 测试(端到端验证)。源码契约 + 行为级双层防护,沿用 1af 既定模式。

### `TestListEventBlobsOlderThan_FiltersAndLimits` 边界

- 4 个 blob 顺序创建(0,1 在 threshold 前,2,3 在后)
- 验证只返 oldIDs(0,1),newIDs(2,3)被过滤
- LIMIT=1 应只返 1 条

测试方向曾被实现反(WHERE created_at < threshold 返 OLD,我误以为是 NEW),纠正后明确语义。

## Verification Depth Badge

**🟢 touched** — 3 个 repo PG 集成测试覆盖,2 项 mutation KILLED。

切片深度:
- **1b 单向最小**:🟢 touched → 维持(visitor_repo 全覆盖加深)
- **1d 录像归档**:🟢 touched → 维持(event_blob_repo 全覆盖加深)
- **1e 双向通道**:🟡 → **🟢 touched**(command_repo 全覆盖)

## Follow-up(留 backlog)

1. erasure_repo(67.7%)、consent_repo(80%+)、chat_repo(80%+)— 已较好,留 backlog
2. minio/postgres/redis.go Connect/Ping/Close — 启动期代码,价值低
3. **1ai-c**:Stores 接口重构(解锁 api 包 happy path 测试)

## 提交

建议 3 个 commit:

1. `test(1ai-b): visitor_repo PG 集成测试 — Create/Get/Upsert(3 测试追加)`
2. `test(1ai-b): command_repo PG 集成测试 — Create+List+Delete+JSON round-trip(4 测试)`
3. `test(1ai-b): event_blob_repo PG 集成测试 — Create/List/Filter+LIMIT/Delete 幂等(4 测试)`

## 下一步

### 立即可做

- 用户审阅 + commit 1ai-b
- 更新 `project-status.md` 加 1ai-b 行 + 1e 切片深度升级

### 短期 backlog

- **1ai-c**:Stores 接口重构(把 `*Postgres`/`*Redis`/`*MinIO` 改 interface,~5-8h,把 api 推到 50%+)
- 修剩余 gofmt 问题(~30min,纯格式)

### 长期(post-v1)

- 自定义域名 / 页面编辑器 / Tauri
- Mutation CI 集成(R7)
