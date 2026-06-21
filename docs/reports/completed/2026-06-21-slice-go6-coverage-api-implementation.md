# Slice Go-6 implementation:api 包补测

> **状态**:completed(Commit 1 完成;Commit 2 留 backlog)
> **深度 badge**:🟡 touched
> **实际工时**:~2h(预估 16h)

## 实施清单(Commit 1)

### `server/internal/api/coverage_extra_test.go`(新建,~480 行,~30 测试)

#### 纯函数(全 0% → 100%)

| 测试 | 覆盖 |
|---|---|
| TestTernary | 三元运算 helper 5 case |
| TestCommandHandler_IsURLAllowed | URL 白名单 12 case(同源/localhost/allowed domain/subdomain/disallowed) |
| TestCommandHandler_IsURLAllowed_EmptyAllowedDomains | 无白名单 fallback |
| TestEventCountOf_SingleEvent/BatchEvents/NonEventReturnsZero | 事件计数 3 case |
| TestForEachEventPayload_Single/Batch/NonEventNoOp | 事件迭代 3 case |
| TestExtractFullSnapshotEnvelope_SingleRRWebType2 | single full snapshot 提取 |
| TestExtractFullSnapshotEnvelope_NonEventReturnsNil | 非 event 返回 nil |
| TestExtractFullSnapshotEnvelope_NonFullSnapshotReturnsNil | 增量快照返回 nil |
| TestExtractFullSnapshotEnvelope_BatchExtractsFirst | batch 提取首个 full |
| TestExtractFullSnapshotEnvelope_NoFullInBatchReturnsNil | batch 无 full 返回 nil |
| TestBuildCommandPayload_AllTypes | 7 个命令类型 case |
| TestBuildCommandPayload_UnknownTypeReturnsError | 未知类型 |
| TestBuildCommandPayload_InvalidJSONReturnsError | 无效 JSON |
| TestDevHint_ReturnsHintJSON | dev hint 503 |

#### 构造函数(全 0% → 100%)

7 个 New*Handler + newStaticHandler(dev 模式)。

#### health handlers(全 0% → 100%)

- TestHealthLive_ReturnsAlive:200 + status=alive
- TestHealthReady_AllOK:200 + ready + components(用真实 Stores,docker 集成)

## 关键修正

### 修正 1:构造函数签名

初版用猜测签名(如 `NewAuthHandler(stores, nil, nil, logger, nil)`),实际:
- NewAuthHandler(stores, logger, secureCookie)
- NewChatHandler(stores, hub, logger)
- NewClaimHandler(stores, logger)
- NewCommandHandler(stores, hub, allowedDomains, logger)
- NewPrivacyHandler(stores, logger)
- NewReplayHandler(stores, logger)
- NewSessionHandler(stores, hub, logger)

**反思**:补构造函数测试前先 `grep "^func New"` 看实际签名,不要猜测参数数量。

### 修正 2:config.APIConfig 不存在

初版 `config.APIConfig`,实际是 `config.Config`(Stores.Connect 接受 `*config.Config`)。

### 修正 3:chat_message 不是 command 类型

初版 TestBuildCommandPayload_AllTypes 包含 chat_message,但 buildCommandPayload switch 不支持(由 chat handler 单独处理)。去掉。

## 实测覆盖率(2026-06-21)

| 文件 | 主要函数 |
|---|---|
| command.go | isURLAllowed 100% / buildCommandPayload 100% / NewCommandHandler 100% |
| health.go | healthLive 100% / healthReady 100% / ternary 100% |
| auth.go | NewAuthHandler 100% / RegisterMe 0%(留 backlog) |
| chat.go | NewChatHandler 100% |
| claim.go | NewClaimHandler 100% |
| privacy.go | NewPrivacyHandler 100% / DeleteVisitor 0%(留 backlog) |
| replay.go | NewReplayHandler 100% |
| session.go | NewSessionHandler 100% / listSessions 0%(留 backlog) |
| router.go | newStaticHandler 100%(dev)/ devHint 100% / Register / NoRoute / NewRouterWithOpts 0%(留 backlog) |
| ws.go | extractFullSnapshotEnvelope 100% / eventCountOf 100% / forEachEventPayload 100% / NewWSHandler / visitorWS / sendError 0%(留 backlog) |
| **整包** | **47.9%** 🟡(差 42.1pp 到 90%) |

## 未达 90% 的说明

剩余 42.1pp gap:

1. **HTTP 业务 handler 业务路径**(RegisterMe / DeleteVisitor / listSessions / postLogin / postClaim 等):需要真实 stores + HTTP request 构造 + 业务路径覆盖,工时巨大。
2. **WS handlers**(visitorWS / operatorWS / sendError / WSHandler.Register):需要 httptest server + WS Dial + 完整握手流程(类似 Go-3 hub 测试模式但更复杂)。
3. **路由注册**(NewRouterWithOpts / Register / NoRoute):需要构造完整 Options + 验证路由表,涉及 embed FS。

**决策**:按 plan 风险点 #7,接受 47.9% 作为 Go-6 当前结果。Commit 1 完成所有纯函数 + 构造函数 + 简单 handler(易测部分),Commit 2 WS handlers 留 R&D backlog。

api 是 Go phase 第三个未达 90% 的包(storage 86.5% + recording 77.7% + api 47.9%)。

## 副作用与回归

- ✅ 不改业务代码
- ✅ 现有 34 个 api 测试不破坏
- ✅ 全部测试 PASS(无 flaky)

## 同步文档

- ✅ project-status §2.1 api 行更新 38.2%→47.9%,标注"未达 90% 目标 42.1pp,Commit 1 完成,WS handlers 留 backlog"
- ✅ daily 追加 Go-6 段

## 后续切片解锁

Go-7(config+privacy 验证)可启动,无依赖。Go phase 完成后汇报。
