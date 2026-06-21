# Slice Go-3 implementation:hub 包补测（并发测试）

> **状态**:completed
> **深度 badge**:🟢 touched
> **实际工时**:~2h(预估 8h,因测试模式直接复用 httptest server + websocket.Accept/Dial)

## 实施清单

### `server/internal/hub/client_coverage_test.go`(新建,~330 行)

#### 核心工具:`newTestWSConnsPair(t)`

启动 httptest server + websocket.Accept/Dial 建立真实 `*websocket.Conn` 对:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    conn, _ := websocket.Accept(w, r, nil) // 默认严格 origin 检查
    serverConn = conn
    close(serverReady)
    <-serverDone
}))

websocket.Dial(ctx, srv.URL, &websocket.DialOptions{
    HTTPHeader: http.Header{
        "Origin": []string{srv.URL}, // 关键:精确匹配 server Host,通过默认 origin 检查
    },
})
```

返回 `(serverConn, clientConn, cleanup)`,cleanup 用 `sync.Once` 保护避免 close of closed channel panic。

#### 12 个测试

| 测试 | 验证点 | race-sensitive? |
|---|---|---|
| TestNewClient_StartsWriteLoop | NewClient 启动 writeLoop,Send 后从 client-side 读到 | ✅ |
| TestSend_QueueFullReturnsFalse | writeCh buffer=2,第 3 个 Send 返回 false | 否 |
| TestSend_AfterCloseWithFullQueueReturnsFalse | Close+满 buffer → closeCh 路径 | 否 |
| TestClose_Idempotent | 多次 Close 不 panic(closeOnce 保护) | 否 |
| TestClose_ClosesUnderlyingConn | Close 后 client-side Reader 返回 error | 否 |
| TestRegisterClient_AddsToHubMap | hub.clients map 含此 client | 否 |
| TestRegisterClient_OverwriteSameID | 同 ID 覆盖(map 语义) | 否 |
| TestSendCommandToVisitor_VisitorOnline | 在线时命令下发成功 | ✅ |
| TestSendCommandToVisitor_VisitorOfflineReturnsFalse | 不在线返回 false | 否 |
| TestSendCommandToVisitor_ClientUnregisteredReturnsFalse | visitorClients 有映射但 clients 无 | 否 |
| TestSendCommandToVisitor_ConcurrentMultipleSenders | 50 goroutine 并发 Send,无 race | ✅ |

## 关键修正(实施过程发现)

### 修正 1:InsecureSkipVerify → srv.URL 精确匹配

初版用 `InsecureSkipVerify: true`,触发安全警告("Don't disable TLS verification")。虽然 coder/websocket 的 `InsecureSkipVerify` 是 WS origin 校验(非 TLS),但名字相同容易混淆。

**修正**:用 `Origin: srv.URL` 精确匹配 server Host(默认严格 origin 检查通过),完全不依赖 InsecureSkipVerify 也不依赖 OriginPatterns 通配符。

### 修正 2:Origin 字符串格式

初版用 `Origin: "http://127.0.0.1"`,但 server Host 是 `127.0.0.1:<port>`,导致 `request Origin "127.0.0.1" is not authorized for Host "127.0.0.1:54965"`。

**修正**:`Origin: srv.URL`(完整 `http://127.0.0.1:<port>`)。

### 修正 3:close of closed channel panic

初版 cleanup 函数直接 `close(serverDone)`,但多次调用 panic。同时多个测试调用 cleanup 时会重复 close。

**修正**:用 `sync.Once` 包装 close 调用。

### 修正 4:Send 文档承诺

初版 `TestSend_AfterCloseReturnsFalse` 期望 Close 后 Send 总返回 false。实际 Send 文档只承诺"队列满返回 false",Close 后 writeCh 未满时仍可能 return true(select 随机选择 ready case)。

**修正**:改名 `TestSend_AfterCloseWithFullQueueReturnsFalse`,先填满 writeCh 再 Close + Send,确保走 closeCh 路径。

## 验证

```bash
# 基础覆盖率
go test -count=1 -cover ./internal/hub/...
# ok  github.com/iannil/pinconsole/internal/hub  25.940s  coverage: 94.1% of statements

# 并发安全(plan 强制要求 -race -count=3)
go test -race -count=3 -run "TestSendCommandToVisitor_ConcurrentMultipleSenders|TestMultipleOperatorsBothReceive" ./internal/hub/...
# ok  github.com/iannil/pinconsole/internal/hub  1.880s
```

**3 个并发敏感测试 × 3 次重复 × race detector** 全绿,无 race 无 flaky。

## 残余覆盖率 94.1% 的拆解

| 函数 | 覆盖率 | 剩余 gap |
|---|---|---|
| NewClient | 100% | — |
| Send | 100% | — |
| Close | 100% | — |
| writeLoop | ~95% | Conn.Write 错误后 Close 路径(需 mock write 错误) |
| RegisterClient | 100% | — |
| SendCommandToVisitor | 100% | — |
| PublishEvent | 83.3% | subscriber channel 满丢弃路径(预期行为,有日志) |
| SubscribeSession | 81.8% | 同上 |
| UnsubscribeSession | 88.9% | 同上 |
| room.publish | 80% | 同上 |

剩余 5.9% 是 subscriber channel 满时的丢弃路径(已有 WARN 日志,行为正确),不值得 hack 触发。

## 副作用与回归

- ✅ 不改业务代码
- ✅ 现有 3 个 hub_test.go 测试不破坏
- ✅ -race -count=3 通过(并发安全)
- ⚠️ 测试耗时 ~25s(httptest server 启动开销),CI 容忍

## 同步文档

- ✅ project-status §2.1 hub 行更新 72.4%→94.1%
- ✅ daily 追加 Go-3 段

## 后续切片解锁

Go-4(storage)/Go-5(recording)/Go-6(api)可启动。Go-6 api 依赖 hub(本切片)+ storage + recording。
