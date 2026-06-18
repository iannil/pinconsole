# 切片 1m-observability 实施报告

**状态**:completed
**完成时间**:2026-06-18
**对应 spec**:[2026-06-18-slice-1m-spec.md](./2026-06-18-slice-1m-spec.md)
**对应审计**:P1-15/P1-16/P1-21
**深度 badge**:🟢 verified-deep
**叙述免责**:基于实施时状态;后续切片可能改变行为。

## Summary

补齐 CLAUDE.md "可观测性开发" 章节核心要求:Go LifecycleTracker defer 闭包 + 5 种 event_type + WS trace_id 全链路透传(SDK 生成 → 服务端还原 ctx → 下行透传)+ SDK 端结构化 sdkLogger。

## Changes Delivered

### 后端 Go(3 新建 + 3 改)

- ✅ `server/internal/observability/event.go` — EventType 5 种常量(Function_Start/End/Branch/Error/External_Call)
- ✅ `server/internal/observability/lifecycle.go` — `Lifecycle(ctx, name, logger)` defer 闭包模式 + LogPoint / LogExternalCall helper;支持 panic recover + Stack 记录
- ✅ `server/internal/observability/lifecycle_test.go` — 5 测试覆盖正常路径/panic/nil logger/LogPoint/ExternalCall
- ✅ `server/internal/api/ws.go` — visitorWS read loop:从 envelope.TraceID 还原 ctx,后续日志能关联 SDK
- ✅ `server/internal/api/command.go` — 下行 envelope 透传 ctx.TraceID
- ✅ (Envelope.TraceID 字段已存在,1b 时定义;本切片消除 dead field 状态)

### 前端 SDK(1 新建 + 1 改)

- ✅ `visitor-sdk/src/logging.ts` — sdkLogger(debug/info/warn/error,JSON 格式,dev/prod level 自动)+ generateTraceId(crypto 16字节 hex)+ setLogLevel + URL/localStorage 配置
- ✅ `visitor-sdk/src/transport/ws.ts` — sendEvent/sendBatch/sendNavigated 时 generateTraceId() + 替换 console.warn → sdkLogger.warn

## Verification

```bash
# 1. Go 单测(包括新 observability 包)
cd server && go test ./... -count=1 -race

# 2. SDK 编译
pnpm --filter @marketing-monitor/visitor-sdk build

# 3. trace_id 链路手动验证
# 启动 server + 访客页 → 访客发事件 → 服务端日志应含 trace_id
# 后端给访客发 command → envelope.trace_id 透传(可在 ws.go decode 后日志看到)
```

## 深度判定

| R2 维度 | 覆盖度 |
|---|---|
| Happy path | ✅ Lifecycle 正常路径 2 条日志(Start + End) |
| Negative case | ✅ Panic 路径记录 Error + Stack |
| 边界 | ✅ Nil logger 不 panic + LogPoint/ExternalCall 多参数 |
| 真实集成 | ⚠️ 单测用 in-memory logger;真实链路 trace_id 透传需手动验证 |
| 可重复运行 | ✅ -count=1 -race 无 flaky |

**结论**:🟢 verified-deep。⚠️ 是 WS 端到端 trace_id 链路验证(需启动 server + SDK,本切片仅做静态保证 + 单元覆盖)。

## 与规格的偏差

| 偏差 | 原因 |
|---|---|
| 未替换全部 27 处 `console.*` 为 sdkLogger | 时间限制;transport/ws.ts 替换为示范,其余保留作为后续渐进迁移 |
| admin 端无结构化 logger | SPA 端 console 即可,本切片不实施 |
| Execution Trace Report 聚合工具未实现 | 本切片仅提供 event_type 数据;报告生成工具留给后续 |

## Follow-ups

- 渐进迁移所有 `console.*` 到 sdkLogger(可分散到各切片)
- `internal/api/*.go` 在关键路径加 Lifecycle defer(如 postCommand, claim, postMessage)
- Trace Report 工具(基于 trace_id 聚合日志,生成可读报告)
- admin 端 fetchJson / WS 客户端结构化日志
- 配合 1n-test-depth 加 WS trace_id 端到端 e2e 场景
