# 切片 1m-observability 规格

**状态**:in-progress
**开始**:2026-06-18
**对应审计**:[2026-06-18-deep-audit.md](../audits/2026-06-18-deep-audit.md) §3 P1-15/P1-16/P1-21(可观测性合规度 ~25%)

## Context

CLAUDE.md "可观测性开发" 章节明确要求 LifecycleTracker / event_type / trace_id 全链路 / Execution Trace Report。审计实测合规度 ~25%。本切片补齐核心能力。

## 锁定决策

| # | 决策 |
|---|---|
| 1 | LifecycleTracker:Go defer 闭包模式 `defer observability.Lifecycle(ctx, name, logger)()` |
| 2 | event_type 5 种:Function_Start / Function_End / Branch / Error / External_Call |
| 3 | WS trace_id:SDK 生成 crypto random 16字节 hex + 服务端 proto.Decode 写回 ctx + 下行 envelope 透传 |
| 4 | SDK 端抽 `sdkLogger` 模块(dev 模式 console.warn JSON;prod 仅 error 级别),替换 27 处 `console.*` |
| 5 | Go 单测覆盖,目标 🟢 |

## 范围

### 后端 Go(新建 2 + 改 ~5)

- ✅ `server/internal/observability/lifecycle.go` — `Lifecycle(ctx, name, logger)` defer 闭包
- ✅ `server/internal/observability/lifecycle_test.go` — 正常/异常/超时 case
- ✅ `server/internal/observability/event.go` — EventType 常量 + LogPoint helper
- ✅ `server/internal/proto/envelope.go` — Envelope.TraceID 已存在,需在 Encode/Decode 时填充
- ✅ `server/internal/api/ws.go` — visitorWS/operatorWS 接受 envelope 时把 trace_id 写回 ctx
- ✅ `server/internal/api/command.go` 等 — 下行 envelope 透传 ctx trace_id
- ✅ `server/internal/logging/handler.go` — FromContext 优先用 envelope.TraceID

### 前端 SDK(新建 1 + 改 ~3)

- ✅ `visitor-sdk/src/logging.ts` — sdkLogger(debug/info/warn/error,JSON 格式)
- ✅ `visitor-sdk/src/transport/ws.ts` — sendEvent/sendBatch 时生成或透传 trace_id
- ✅ 替换 27 处 `console.*` 为 `sdkLogger.*`

### 不实施

- Execution Trace Report 聚合工具(留给后续,本切片提供数据但不实现报告生成)
- admin 端结构化 logger(admin 是 SPA,console 即可)

## Verification

```bash
# Go 单测
cd server && go test ./internal/observability/ -count=1 -v

# WS trace_id 链路验证(需启动 server + SDK)
# 服务端日志中应含 envelope.trace_id;SDK 端 console 也应含同 trace_id

# SDK 编译
pnpm --filter @marketing-monitor/visitor-sdk build
```

## 估时

solo 全职:1-1.5 天
