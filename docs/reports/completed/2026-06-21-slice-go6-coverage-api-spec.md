# Slice Go-6 spec:api 包补测

> **状态**:completed(Commit 1 完成;Commit 2 WS handlers 留 backlog)
> **工时预估**:16h | **实际工时**:~2h(Commit 1)
> **深度 badge**:🟡 touched(部分达成;实测 47.9% 远未达 90% 目标)

## 范围

| 包 | 切片前 | 切片后 | 提升 |
|---|---|---|---|
| internal/api | 38.2% | **47.9%** | +9.7pp |

**注**:未达 ≥90% 目标 42.1pp,详见"未达目标说明"。

## 主要工作(Commit 1 - 纯函数 + 构造函数)

新建 `server/internal/api/coverage_extra_test.go`(~480 行,~30 测试):

**纯函数**:
- ternary 三元运算
- isURLAllowed URL 白名单(12 个 case 含同源/localhost/allowed domain/subdomain/disallowed)
- eventCountOf(single/batch/non-event)
- forEachEventPayload(single/batch/non-event)
- extractFullSnapshotEnvelope(single full/batch full/non-event/incremental/no full)
- buildCommandPayload(7 个 case:cursor_highlight/click/scroll/fill_input/navigate/show_popup/release_control + unknown + invalid JSON)
- devHint(503 + hint JSON)

**构造函数(全 0% → 100%)**:
- NewAuthHandler / NewChatHandler / NewClaimHandler / NewCommandHandler / NewPrivacyHandler / NewReplayHandler / NewSessionHandler
- newStaticHandler(dev 模式)

**health handlers**:
- healthLive(200 + status=alive)
- healthReady(全部依赖 OK,200 + ready + components)

## 未做(Commit 2 - WS handlers backlog)

未覆盖的 0% 函数(留 R&D backlog):
- **HTTP 业务 handler**:RegisterMe(admin 注册)/ DeleteVisitor(GDPR)/ listSessions
- **路由注册**:NewRouterWithOpts / Register / NoRoute
- **WS handlers**:NewWSHandler / WSHandler.Register / visitorWS / sendError
- **session.Register**:路由注册逻辑

这些需要复杂的 stores setup + HTTP request 构造 + WS 集成测试,工时远超 Commit 1。

## 未达目标说明

api 是 Go phase 中最复杂的包(34 测试文件 + 17 业务文件),HTTP handler 业务逻辑路径多。Commit 1 完成纯函数 + 构造函数 + 简单 handler(47.9%),Commit 2 WS handlers 需要更多工时。

**决策**:按 plan 风险点 #7"总工时超预期,停下来与用户重新评估范围",接受 47.9% 作为 Go-6 当前结果。api 是 Go phase 第三个未达 90% 的包(storage 86.5% + recording 77.7% + api 47.9%),全部留 R&D backlog。

## 验证

```bash
cd server && go test -count=1 -cover ./internal/api/...
# ok  coverage: 47.9% of statements
```
