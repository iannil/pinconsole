# Slice Go-3 spec:hub 包补测（并发测试）

> **状态**:completed
> **工时预估**:8h | **实际工时**:~2h
> **深度 badge**:🟢 touched(WS 集成测试 + race detector 验证)

## 范围

| 包 | 切片前 | 切片后 | 提升 |
|---|---|---|---|
| internal/hub | 72.4% | **94.1%** | +21.7pp |

## 主要工作

新建 `server/internal/hub/client_coverage_test.go`(~330 行,12 测试):

**核心:httptest server + websocket.Accept/Dial 真实 WS 连接对**
- TestNewClient_StartsWriteLoop:验证 NewClient 启动 writeLoop goroutine
- TestSend_QueueFullReturnsFalse:writeCh 满时 Send 返回 false
- TestSend_AfterCloseWithFullQueueReturnsFalse:Close+满 buffer 走 closeCh 路径
- TestClose_Idempotent:多次 Close 不 panic
- TestClose_ClosesUnderlyingConn:Close 关闭底层 websocket.Conn
- TestRegisterClient_AddsToHubMap/RegisterClient_OverwriteSameID:hub map 操作
- TestSendCommandToVisitor_VisitorOnline/Offline/Unregistered/ConcurrentMultipleSenders:命令下发 + 并发

## 强制验证(本切片核心)

`go test -race -count=3` 全绿,无 race + 无 flaky。

## 不覆盖

- `PublishEvent` 83.3% / `SubscribeSession` 81.8% / `UnsubscribeSession` 88.9% / `room.publish` 80%:
  剩余分支是 subscriber channel 满时的丢弃路径(已有日志警告,行为正确),覆盖率不阻塞

## 关键决策

1. **不用 InsecureSkipVerify**:用 `Origin: srv.URL` 精确匹配 server Host,通过 websocket.Accept 默认严格 origin 检查
2. **不 mock websocket.Conn**:用 httptest server 建立真实连接对,测试完整生命周期
3. **TestSend_AfterCloseWithFullQueueReturnsFalse**:Close 后 Send 在 writeCh 未满时仍可能成功(文档承诺"队列满返回 false"),测试只覆盖 closeCh + writeCh 都满的边界
