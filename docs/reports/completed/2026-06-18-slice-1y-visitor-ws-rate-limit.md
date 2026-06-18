# 切片 1y-visitor-ws-rate-limit — 完成报告

**状态**:completed
**完成时间**:2026-06-18
**对应 progress**:(原文件已归档,见本文件)

## Summary

visitor WebSocket read loop 加 per-session 滑动窗口 rate limit:10 秒内最多 500 envelope 或 50 MiB(以先到为准),超限 force close + FlagSession。防止恶意 SDK 撑爆 Redis Stream + MinIO。

## Changes Delivered

- ✅ **`server/internal/api/ws.go`** — visitorWS read loop 在 envelope decode 后、stream.Append 前调 `checkWSRateLimit(ctx, h.stores.Redis.Client, sessionID, len(msg))`:
  - 允许:继续处理(写 stream、广播、PG touch)
  - 拒绝:`antiscrape.FlagSession(reason)` + `conn.Close(websocket.StatusPolicyViolation, "rate limit exceeded")` + `return`
  - Redis 故障:仅 warn 不阻断(fail-open,与 1i 一致)
- ✅ 常量与 Lua script 已存在(`wsRateLimitWindow` / `wsRateLimitMaxMsgs` / `wsRateLimitMaxBytes` / `wsRateLimitLua`),本次完成接线 + 测试
- ✅ **`server/internal/api/ws_ratelimit_test.go`**(新建)— 5 测试:
  - `TestCheckWSRateLimit_NormalTrafficAllows` — 100 个小 envelope 全允许
  - `TestCheckWSRateLimit_OverMsgCount` — 第 501 个超阈值被拒
  - `TestCheckWSRateLimit_OverBytes` — 第 6 个 10MB envelope 总和超 50MiB 被拒
  - `TestCheckWSRateLimit_SessionIsolation` — 不同 session 独立计数
  - `TestCheckWSRateLimit_RedisFailureFailOpen` — Redis 故障 fail-open

## Verification

```bash
cd server
go test ./internal/api/ -run TestCheckWSRateLimit -v -count=1
go test ./... -count=1                        # 12 packages ALL PASS
go vet ./...
go build -tags release ./cmd/server

cd ..
./ops.sh restart
cd e2e && SKIP_MM_WEBSERVER=1 npx playwright test --reporter=list
# 65 passed / 0 failed / 4 skipped
```

**预期结果**(2026-06-18 实测):
- 5 个新单测全 PASS
- 全 12 个 Go 包 ALL PASS
- e2e 全 65 测试 PASS(无回归)
- release binary 编译 OK

## 深度 badge

🟢 verified-deep:
- Go 单测覆盖正/负向 + 边界(500 阈值 + 50MiB 阈值 + session 隔离 + fail-open)
- 接线代码通过 e2e 全量回归(1b-1g 真实 visitor WS 流量未触发误拒)
- 阈值余量 5x/500x,只抓真攻击

## Follow-ups

- e2e 层验证恶意 SDK 触发限流:本次跳过(page.evaluate 拉原生 WS 刷 600 envelope 复杂);Go 单测已覆盖核心逻辑
- 监控:1y 触发后 server log "visitor ws rate limit exceeded, closing" 可作为运营告警源(后续 1z+ 可加 metrics)

## Notes

- 接线点选在 `stream.Append` **之前**,避免被限流的 envelope 也污染 Redis Stream/MinIO(节省存储 + 网络)
- fail-open 与 1i 一致:Redis 故障时业务继续跑,只 warn;但要注意这意味着 Redis 宕机期间限流失效——这是 trade-off,优于"Redis 宕机 → 全部 visitor 被踢"
- FlagSession reason 用结构化字符串 `ws_rate_exceeded_msgs:<count>` / `ws_rate_exceeded_bytes:<bytes>`,admin 看到 reason 可区分触发维度
- TTL 由 Lua 在首次 INCR 时设置(`c == 1` / `b == msgSize`),避免每次调 EXPIRE
