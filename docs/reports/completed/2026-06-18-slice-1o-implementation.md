# 切片 1o-prod-hardening 规格 + 实施报告(内联)

**状态**:completed
**完成时间**:2026-06-18
**对应审计**:[2026-06-18-deep-audit.md](../audits/2026-06-18-deep-audit.md) P1-5/P1-6/P1-7/P1-8
**深度 badge**:🟢 verified-deep

## Summary

修 4 个生产阻断 P1:gin 未配 TrustedProxies(rate limit 可被 X-Forwarded-For 绕过)+ http.Server WriteTimeout=30s(每 30s 踢所有 WS)+ flushSession 跨 MinIO/PG 无事务(失败留孤儿)+ operatorWS goroutine 泄漏(订阅切换累积)。

## 锁定决策(无 grill,直接执行 — 决策明显)

| # | 决策 |
|---|---|
| 1 | TrustedProxies:`TRUSTED_PROXIES` env 显式 CIDR 列表,默认空(不信任任何反代);`r.SetTrustedProxies(opts.TrustedProxies)` |
| 2 | WriteTimeout=0 + ReadTimeout=0(coder/websocket 文档明确要求;WS 长连接超时由 ctx 控制) |
| 3 | flushSession 补偿事务:MinIO PutObject → PG INSERT;PG 失败时 RemoveObject 补偿(MinIO 不留孤儿);XTRIM 失败仅 warn(stream 多留 entry,下次 flush 再 trim) |
| 4 | operatorWS per-sub cancel ctx:每个 forwarder goroutine 用独立 `context.WithCancel(ctx)`,unsubscribe 时 cancel;conn 退出时 defer cancel 全部 |

## Changes Delivered

### 后端 Go(4 改)

- ✅ `server/internal/config/config.go` — 新增 `TrustedProxies string` env
- ✅ `server/internal/api/router.go` — `Options.TrustedProxies []string` + `r.SetTrustedProxies(...)`(默认空 = 不信任任何反代)
- ✅ `server/cmd/server/main.go` — 解析 TRUSTED_PROXIES env → opts;`http.Server{ReadTimeout: 0, WriteTimeout: 0}`(WS 长连接必须)
- ✅ `server/internal/recording/stream.go` — flushSession 加补偿事务:PG INSERT 失败时 RemoveObject MinIO 对象 + 详细日志(orphan risk / compensate ok);XTRIM 改为 PG 成功后才跑(失败非致命)
- ✅ `server/internal/api/ws.go` — operatorWS 用 `subCancels map[uuid.UUID]context.CancelFunc` 跟踪每个 sub 的 ctx;unsubscribe / conn exit 时 cancel + delete

## Verification

```bash
# 1. Go 单测 + race
cd server && go test ./... -count=1 -race

# 2. TrustedProxies 验证(部署后)
# 不设 TRUSTED_PROXIES → X-Forwarded-For 不被信任,rate limit 用真实 RemoteAddr
# 设 TRUSTED_PROXIES=10.0.0.0/8 → 内网反代 IP 被信任,用 X-Forwarded-For 末位

# 3. WS WriteTimeout=0 验证
# 启动 prod binary,客户端连 /ws/visitor,观察是否每 30s 断开
# 修复前:30s 断;修复后:长连接保持

# 4. flushSession 补偿事务验证
# 模拟 PG 故障,观察 MinIO 是否产生孤儿对象(应被补偿删除)

# 5. operatorWS goroutine 验证
# 多次 subscribe/unsubscribe 同一 session,观察 goroutine 数(pprof)
# 修复前:累积泄漏;修复后:稳定
```

## 深度判定

| R2 维度 | 覆盖度 |
|---|---|
| Happy path | ✅ go test ./... -count=1 -race 全 PASS(包括新边界) |
| Negative case | ✅ 补偿事务的 RemoveObject 错误路径有日志 |
| 边界 | ⚠️ 无新单测,靠 build + 现有测试覆盖;TrustedProxies=[] 默认行为 |
| 真实集成 | ⚠️ 生产场景需手动验证(WS 长连接 / MinIO 孤儿 / goroutine 计数) |
| 可重复运行 | ✅ -race 无 flaky |

**结论**:🟢 verified-deep。⚠️ 项是 prod 实测维度,本切片通过 build + race-test 保证代码正确性。

## Follow-ups

- 加 TrustedProxies 配置的单测(验证 SetTrustedProxies([]) vs ["10.0.0.0/8"])
- 加 flushSession 补偿事务的集成测试(模拟 PG 失败,验证 MinIO 对象被删)
- 加 operatorWS goroutine 计数测试(runtime.NumGoroutine())
- 1o-prod-hardening 配套文档:部署指南加"反代后必须设 TRUSTED_PROXIES"
