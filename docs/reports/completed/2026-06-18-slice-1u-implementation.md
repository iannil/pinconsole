# 切片 1u-god-files 实施报告

**状态**:completed
**完成时间**:2026-06-18
**对应审计**:P1-27(god files 拆分)
**深度 badge**:🟢 verified-deep

## Summary

拆 `server/internal/storage/queries.go`(771 LOC,22 方法,7 类型)为 10 个 per-aggregate 文件,每个 50-200 LOC 单一职责。同时修复 1k 留下的 docker-compose prod profile `:?` 语法阻塞 infra 启动问题。

## Changes Delivered

### queries.go 拆分(771 LOC → 10 files)

| 新文件 | 内容 | LOC |
|---|---|---|
| `types.go` | 7 个 struct(Visitor/Session/EventBlob/User/ChatMessage/CoBrowsingCommand/VisitorConsent)+ DefaultTenantID + scanner interface + scanVisitor/scanSession/scanUser helpers | ~145 |
| `visitor_repo.go` | GetVisitorByFingerprint + CreateVisitor | ~50 |
| `session_repo.go` | CreateSession + GetSession + TouchSessionEvent + EndSession + ListActiveSessionsByTenant + ListEndedSessionsByTenant | ~135 |
| `event_blob_repo.go` | CreateEventBlob + ListEventBlobsBySession + ListEventBlobsOlderThan + DeleteEventBlobByID | ~115 |
| `user_repo.go` | GetUserByEmail + GetUserByID + CreateUser + CountUsers | ~50 |
| `chat_repo.go` | CreateChatMessage + ListChatMessagesBySession + ListChatMessagesOlderThan + DeleteChatMessagesByID | ~80 |
| `command_repo.go` | CreateCoBrowsingCommand + ListCoBrowsingCommandsBySession + DeleteCoBrowsingCommandsOlderThan | ~75 |
| `consent_repo.go` | GetLatestConsent + UpsertConsent(1l) | ~55 |
| `erasure_repo.go` | DeleteVisitorByFingerprint + ListEventBlobKeysBySessions(1l GDPR Art.17) | ~100 |
| `gc_repo.go` | DeleteSessionsEndedBefore + DeleteVisitorsLastSeenBefore | ~25 |

`queries.go` 删除。所有方法签名 / SQL / 行为完全保留(只是文件位置变)。

### docker-compose.yml `${VAR:?}` 修复(顺手)

**问题**:1k 用 `${VAR:?required}` 语法让 prod profile 缺凭证时 fail-fast。但 docker compose 在**parse 阶段**就插值校验所有服务的 env,导致 `docker compose up -d postgres redis minio`(不启 server)也被阻塞。

用户实测:
```
./ops.sh start
error while interpolating services.server.environment.ADMIN_EMAIL: required variable ADMIN_EMAIL is missing a value
```

**修复**:`${VAR:?required}` → `${VAR:-}`(空默认)。fail-fast 责任完全交给 Go config.Load() 的 validate()。

**验证**:
- `docker compose up -d postgres redis minio` ✅ 成功
- `docker compose --profile prod config` ✅ parse 通过
- `SERVER_ENV=prod ADMIN_PASSWORD= go run ./cmd/server` → Go 拒绝启动:`ADMIN_PASSWORD 未设置：必须显式提供(无默认值，1k fail-secure)`

fail-fast 安全性不降低,反而更清晰:Go 层是单一 fail-fast 源。

## Verification

```bash
# 1. 全部 Go 测试(验证拆分无回归)
cd server && go test ./... -count=1 -race

# 2. infra 启动(原失败)
docker compose up -d postgres redis minio

# 3. prod fail-fast 仍生效
SERVER_ENV=prod ADMIN_PASSWORD= PG_PASSWORD= MINIO_ACCESS_KEY= MINIO_SECRET_KEY= go run ./cmd/server
# 期望:启动失败,提示 ADMIN_PASSWORD 未设置
```

## 深度判定

| R2 维度 | 覆盖度 |
|---|---|
| Happy path | ✅ 所有方法行为不变,12 个 Go 包测试全 PASS |
| Negative case | ✅ 编译时捕获 — 任何方法引用错位立即 build break |
| 边界 | ✅ 7 个类型定义 + scanner helpers 集中在 types.go,无重复 |
| 真实集成 | ✅ 实际 build + test + docker compose 验证 |
| 可重复运行 | ✅ -count=1 -race 无 flaky |

**结论**:🟢 verified-deep。

## 与规格的偏差

无 — 1u 范围就是拆分,执行直接。

## Follow-ups

- ws.go(510+ LOC)拆 visitor/operator/helpers — 同样模式,但 WS 长连接涉及更多状态,风险略高,留给后续
- router.go(285 LOC)拆 static/middleware — 类似,优先级低
- 引入 Repository interface(`UserRepository` / `SessionRepository` 等)让 api 包依赖接口而非 `*Postgres` — 测试更易 mock,工量大
