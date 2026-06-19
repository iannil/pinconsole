# 切片 1ai-b storage-repos-b — Spec

**切片编号**:1ai-b
**类型**:测试深化(storage 包续)
**创建时间**:2026-06-19
**状态**:approved
**关联**:[1ai impl](../completed/2026-06-19-slice-1ai-storage-repo-tests-implementation.md)

## Context

为什么做这个切片?

- **触发原因**:1ai 把 storage 覆盖推到 39.2%,但仍有 3 个 repo 全 0%:
  - visitor_repo:GetVisitorByFingerprint / CreateVisitor
  - command_repo:CreateCoBrowsingCommand / ListBySession / DeleteOlderThan
  - event_blob_repo:CreateEventBlob / ListBySession / ListOlderThan(DeleteByID 已 100%)
- **业务/技术价值**:补完后 storage 覆盖应到 60%+,基本覆盖所有核心 CRUD;1b/1e/1d 切片深度受益。
- **不做的代价**:post-v1 重构 SQL 时无回归保护。

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 范围 | A: 全 3 repo(8 函数) / B: 仅 visitor+command / C: 仅 visitor | **A** | A 工时 ~2-3h 在 budget;B/C 漏关键路径 |
| 2 | visitor CreateVisitor ON CONFLICT | A: 仅测首次创建 / B: + 重复触发 last_seen_at 更新 | **B** | CreateVisitor 是 upsert,关键回归点是 ON CONFLICT DO UPDATE 触发 last_seen_at 刷新 |
| 3 | command payload | A: 简单 {} / B: 复杂 JSON | **B** | JSONB 序列化/反序列化是 command_repo 的关键路径,需测真实 round-trip |
| 4 | event_blob threshold 测试 | A: 仅 created_at < threshold / B: + LIMIT 生效 | **B** | 1d GC 依赖 LIMIT 防 batch 过大,需覆盖 |

## Acceptance(可验证的成功标准)

### 必须满足

- [ ] **A1**:新增 `visitor_repo_test.go`,至少 3 测试:
  - `TestCreateVisitor_AndRetrieve` — Create → GetVisitorByFingerprint 字段一致
  - `TestCreateVisitor_OnConflict_UpdatesLastSeen` — 重复触发 last_seen_at 刷新
  - `TestGetVisitorByFingerprint_NotFound_ReturnsNil` — 不存在 → (nil, nil)(非 error)
- [ ] **A2**:新增 `command_repo_test.go`,至少 4 测试:
  - `TestCreateCoBrowsingCommand_AndList` — Create + List 字段一致(含 JSON payload round-trip)
  - `TestCreateCoBrowsingCommand_NullNodeID` — TargetNodeID=nil → 数据库 NULL
  - `TestListCoBrowsingCommandsBySession_OrderedByCreated` — 多条按 created_at 正序
  - `TestDeleteCoBrowsingCommandsOlderThan` — 阈值前的删,后的留
- [ ] **A3**:新增 `event_blob_repo_test.go`,至少 4 测试:
  - `TestCreateEventBlob_AndList` — Create + ListBySession 字段一致
  - `TestListEventBlobsBySession_OrderedByBlobIndex` — 多 blob 按 blob_index 正序
  - `TestListEventBlobsOlderThan_FiltersAndLimits` — threshold + LIMIT 生效
  - `TestDeleteEventBlobByID` — 已 100% 但加深:verify 不存在的 ID 不报错

### 验证维度

- [ ] `go test ./internal/storage/` 全绿
- [ ] `go test ./...` 全绿
- [ ] storage 包覆盖率:**39.2% → ≥55%**(目标 60% 可达,留余量)
- [ ] Mutation 抽样 2 项 KILLED

### 不在本切片范围

- erasure_repo(67.7% 已较好)— 留 backlog
- consent_repo(80%+ 已较好)— 留 backlog
- chat_repo(80%+ 已较好)— 留 backlog
- gc_repo(100%)— 已覆盖
- minio/postgres/redis.go(Connect/Ping/Close)— 启动期代码,价值低

## 工时预算

| 项 | 工时 |
|---|---|
| 写 spec | 15min |
| visitor_repo_test.go 3 测试 | 30min |
| command_repo_test.go 4 测试 | 1h |
| event_blob_repo_test.go 4 测试 | 1h |
| 跑测试 + mutation + 报告 | 30min |
| **合计** | **~3h** |

## 完成后预期

- storage 包覆盖率 39.2% → ≥55%
- visitor_repo 0% → 100%
- command_repo 0% → 90%+
- event_blob_repo 大部分函数 0% → 90%+
- 1b 切片深度维持 🟢 touched
- 1d 切片深度维持 🟢 touched
- 1e 切片深度 🟡 → 🟢 touched(command_repo 全覆盖)

## Verification Depth Badge 目标

🟢 touched — 3 个 repo PG 集成测试覆盖。

## 关联

- 前置:1ai(user+session repo)
- 后续:1ai-c(Stores 接口重构)
