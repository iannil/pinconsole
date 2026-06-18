# 切片 1t-test-coverage 实施报告

**状态**:completed
**完成时间**:2026-06-18
**对应审计**:P2-10(5 Go 包零单测)
**深度 badge**:🟢 verified-deep

## Summary

补齐审计 §7 C 阶段点名的"5 Go 包零单测"尾巴。1k 已修 config,1i 已修 antiscrape,本切片补 logging / storage / api(privacy handler)/ migrations。所有 12 个 Go 包现在都有测试。

## Changes Delivered

### logging 包(0 → 9 tests)

- ✅ `server/internal/logging/handler_test.go`:
  - `TestParseLevel`(7 cases:debug/info/warn/error/空/大写/未知)
  - `TestNewID_LengthAndHex`(32 字符 hex 校验)
  - `TestNewID_UniqueAcrossCalls`(100 次无重复)
  - `TestWithTraceID_AndTraceID` / `TestWithSpanID_AndSpanID`(ctx round-trip)
  - `TestTraceID_EmptyWhenNotSet`(默认空)
  - `TestFromContext_NoTraceID_ReturnsBase`(无 trace 返回 base logger)
  - `TestFromContext_WithTraceID_HasAttr`(text handler 验证输出含 trace_id/span_id)
  - `TestTraceMiddleware_InjectsIDs`(gin.Context + 响应头注入)
  - `TestTraceMiddleware_PreservesClientHeader`(X-Trace-Id 优先用客户端传入)

### storage 包(0 → 2 tests)

- ✅ `server/internal/storage/postgres_test.go`:
  - `TestPostgresConfig_DSN`(3 cases:default / require ssl / empty password)验证拼接 + sslmode 参数化
  - `TestDefaultTenantID` 验证 uuid.Nil 是默认 tenant(业务约定)

### api 包(privacy handler,4 tests)

- ✅ `server/internal/api/privacy_test.go`:
  - GET consent 缺 fingerprint → 400
  - POST consent invalid JSON → 400
  - POST consent 缺 fingerprint 字段 → 400
  - 路由注册完整性(GET + POST 都注册)

### migrations 包(0 → 2 tests)

- ✅ `server/migrations/embed_test.go`:
  - embed FS 非空
  - up/down 数量一致

## Verification

```bash
cd server && go test ./... -count=1
# 全部 12 个包都 ok,无 "no test files" 警告
```

## 深度判定

| R2 维度 | 覆盖度 |
|---|---|
| Happy path | ✅ logging 全部纯函数 + middleware 行为 |
| Negative case | ✅ privacy handler 缺参/坏 JSON 路径 |
| 边界 | ✅ DSN 3 种 sslmode + empty password |
| 真实集成 | ⚠️ httptest + testLogger,无真实 PG/Redis;SQL 集成测试仍空 |
| 可重复运行 | ✅ -count=1 无 flaky |

**结论**:🟢 verified-deep(纯函数层);⚠️ 真实 SQL/集成测试仍空,留给 testcontainers 后续切片。

## 与规格的偏差

| 偏差 | 原因 |
|---|---|
| storage 包仅 DSN 纯函数测试,SQL 方法无测试 | 真实 PG 集成测试需 testcontainers 基建,本切片明确不做(用户选项 1 范围) |
| api 包仅 privacy handler 测试,未覆盖 command/chat/claim | 这些 handler 已被 1k/1l 的 e2e 覆盖关键负向路径;单测层重复劳动收益低 |
| observability 已在 1m 测试 | 不重复 |

## Follow-ups

- testcontainers 基建(选项 3):真实 PG + Redis + MinIO 集成测试,可覆盖 storage 22 方法的真实 SQL + 1b SDK 重连 + 1e/1g 真实场景
- api 包剩余 handler 单测覆盖率从 ~30%(本切片后)升到 ~80%,需 ~50 个测试(投入大,留低优)
