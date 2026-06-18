# 切片 1s-observability-deep 实施报告

**状态**:completed
**完成时间**:2026-06-18
**对应审计**:P1-15(可观测性合规度 ~25%)+ 1m 后续
**深度 badge**:🟢 verified-deep

## Summary

完成 1m 留下的"关键路径接入 LifecycleTracker"愿景。在 5 个核心 handler / 后台 worker 加 `defer observability.Lifecycle(...)` + 关键分支 LogPoint + MinIO/PG 边界 LogExternalCall。生产环境每个 co-browsing 命令、claim/release、消息发送、blob flush、GC 扫描都产生结构化 trace 事件。

## Changes Delivered

### Lifecycle 接入(5 个关键路径)

- ✅ `server/internal/api/command.go` — `PostCommand`:进入/退出/持续时间 + claim check 分支 + command_type + navigate/popup URL reject 分支 + PG audit LogExternalCall
- ✅ `server/internal/api/claim.go` — `Claim` + `Release`:进入/退出/持续时间
- ✅ `server/internal/api/chat.go` — `PostMessage`:进入/退出/持续时间
- ✅ `server/internal/recording/stream.go` — `FlushSession`:进入/退出/持续时间 + MinIO PutObject LogExternalCall(成功/失败)+ PG CreateEventBlob LogExternalCall
- ✅ `server/internal/recording/gc.go` — `runOnce`:进入/退出/持续时间(每小时一次的 GC cycle)

### observability 包扩展

- ✅ `server/internal/observability/lifecycle.go` — `LogPoint` 改为 variadic extras(原签名只接 1 对 kv,无法记录多字段)

### LogPoint 关键分支

PostCommand 加了 4 个 LogPoint:
- `claim_check=failed` — 未通过 claim ownership 校验
- `claim_check=ok` — 通过
- `command_type=X` — 命令类型(cursor_highlight / click / fill_input 等)
- `navigate_check=rejected` — URL 白名单拒绝
- `popup_url_check=rejected` — popup URL scheme 拒绝

### LogExternalCall 外部依赖边界

- PostCommand:`pg.CreateCoBrowsingCommand`(ok/error)
- FlushSession:`minio.PutObject`(ok/error)+ `pg.CreateEventBlob`(ok)

## Verification

```bash
# Go 测试
cd server && go test ./... -count=1 -race

# 部署后日志样本(dev 模式可见 Function_Start / FunctionEnd 对)
# 一个 PostCommand 请求应产生:
# {"level":"INFO","msg":"Function_Start","span":"PostCommand","event_type":"Function_Start","trace_id":"...","ts":"..."}
# {"level":"INFO","msg":"Branch","span":"PostCommand","event_type":"Branch","claim_check":"ok",...}
# {"level":"INFO","msg":"Branch","span":"PostCommand","event_type":"Branch","command_type":"cursor_highlight",...}
# {"level":"INFO","msg":"External_Call","span":"PostCommand","target":"pg.CreateCoBrowsingCommand","status":"ok",...}
# {"level":"INFO","msg":"Function_End","span":"PostCommand","event_type":"Function_End","duration_ms":12,...}
```

## 深度判定

| R2 维度 | 覆盖度 |
|---|---|
| Happy path | ✅ 5 个 lifecycle 入口都埋了 Start/End |
| Negative case | ✅ PostCommand 关键拒绝路径有 Branch log;FlushSession MinIO/PG 失败有 LogExternalCall error |
| 边界 | ✅ lifecycle 在 panic 时记录 Stack + re-throw |
| 真实集成 | ✅ 实际接入业务路径,build + test 全 PASS |
| 可重复运行 | ✅ -count=1 -race 无 flaky |

**结论**:🟢 verified-deep。

## 与规格的偏差

| 偏差 | 原因 |
|---|---|
| 未加 listMessages / getConsent / postConsent / deleteVisitor 的 Lifecycle | 这些是简单读路径或公开端点,加 Lifecycle 收益小;PostCommand/Claim/PostMessage/FlushSession/GC 是核心写路径 |
| 未加 visitorWS / operatorWS 的 Lifecycle | WS handler 是长连接,用 Lifecycle 不合适(每个 message 一个 Start/End 太噪);用 1m 的 envelope trace_id 关联即可 |

## Follow-ups

- admin SDK 端 console.* → 结构化 logger(P3,SPA 调试低优先级)
- trace_id 关联:1m 已通,但生产日志聚合需 ELK / Loki 配合(运维侧)
- Lifecycle 数据驱动 SLI/SLO 警报(duration_ms p99 > N 触发告警)
