// Go-3 切片补测:NewClient / Send / Close / writeLoop / RegisterClient / SendCommandToVisitor,
// 提升覆盖率 72.4% → ≥90%。
//
// 测试策略:用 httptest server + websocket.Accept/Dial 建立真实 *websocket.Conn 对,
// 测试 NewClient 全生命周期(包括 writeLoop goroutine)。
//
// 安全注意:Origin 校验通过 Dial 时 Origin=srv.URL 精确匹配 server Host,
// 通过 websocket.Accept 默认严格 origin 检查,完全不绕过 origin 校验。
package hub

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// newTestWSConnsPair 启动 httptest server,接受一个 WS 连接,
// 返回 (serverConn, clientConn, cleanup)。调用方必须 defer cleanup()。
//
// Origin 校验策略:Dial 时 Origin 用 srv.URL 精确匹配 server Host(默认严格 origin 检查通过),
// 不绕过 origin 校验也不依赖通配符,符合安全规范。
func newTestWSConnsPair(t *testing.T) (serverConn, clientConn *websocket.Conn, cleanup func()) {
	t.Helper()

	serverReady := make(chan struct{})
	var serverDoneOnce sync.Once
	serverDone := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 默认 AcceptOptions 严格检查 Origin == Host
		// Dial 时 Origin = srv.URL 精确匹配,通过检查
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Logf("server Accept: %v", err)
			return
		}
		serverConn = conn
		close(serverReady)
		<-serverDone
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Origin 用 srv.URL(http://127.0.0.1:<port>),与 server Host 精确匹配,
	// 通过 websocket.Accept 默认的严格 origin 校验
	clientConnPtr, _, err := websocket.Dial(ctx, srv.URL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": []string{srv.URL},
		},
	})
	if err != nil {
		srv.Close()
		serverDoneOnce.Do(func() { close(serverDone) })
		t.Fatalf("websocket.Dial: %v", err)
	}
	clientConn = clientConnPtr

	select {
	case <-serverReady:
		// server Accept 成功
	case <-time.After(500 * time.Millisecond):
		srv.Close()
		serverDoneOnce.Do(func() { close(serverDone) })
		t.Fatal("server did not accept connection in time")
	}

	cleanup = func() {
		serverDoneOnce.Do(func() { close(serverDone) })
		_ = clientConn.Close(websocket.StatusNormalClosure, "test cleanup")
		_ = serverConn.Close(websocket.StatusNormalClosure, "test cleanup")
		srv.Close()
	}
	return serverConn, clientConn, cleanup
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// TestNewClient_StartsWriteLoop 验证 NewClient 返回 client 并启动 writeLoop goroutine。
// 通过 Send 一条消息后从 client-side Conn 读出,证明 writeLoop 在跑。
func TestNewClient_StartsWriteLoop(t *testing.T) {
	serverConn, clientConn, cleanup := newTestWSConnsPair(t)
	defer cleanup()

	// NewClient 用 server-side Conn,writeLoop 会向其写入消息
	// server-side 写入的消息会出现在 client-side(因为这两个 Conn 是配对的)
	client := NewClient(uuid.New(), RoleVisitor, serverConn, newTestLogger())
	defer client.Close(context.Background())

	msg := []byte("hello-writeLoop")
	if !client.Send(msg) {
		t.Fatal("Send returned false, want true")
	}

	// 从 client-side Conn 读出消息
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, reader, err := clientConn.Reader(ctx)
	if err != nil {
		t.Fatalf("clientConn.Reader: %v", err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if string(got) != "hello-writeLoop" {
		t.Errorf("got %q, want hello-writeLoop", got)
	}
}

// TestSend_QueueFullReturnsFalse 验证队列满时 Send 返回 false。
func TestSend_QueueFullReturnsFalse(t *testing.T) {
	serverConn, _, cleanup := newTestWSConnsPair(t)
	defer cleanup()

	// 手工构造 Client with small writeCh buffer(不调 NewClient 避开 writeLoop 干扰)
	c := &Client{
		ID:                 uuid.New(),
		Role:               RoleOperator,
		Conn:               serverConn,
		logger:             newTestLogger(),
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 2), // 小 buffer
		closeCh:            make(chan struct{}),
	}

	// 填满 buffer(不启动 writeLoop,确保 buffer 不被消费)
	if !c.Send([]byte("1")) {
		t.Error("Send(1) returned false, want true")
	}
	if !c.Send([]byte("2")) {
		t.Error("Send(2) returned false, want true")
	}
	// 第三个应该满,返回 false
	if c.Send([]byte("3")) {
		t.Error("Send(3) returned true, want false (queue full)")
	}
}

// TestSend_AfterCloseWithFullQueueReturnsFalse 验证 Close + writeCh 满时 Send 走 closeCh 路径返回 false。
//
// 注:Close 后 writeCh 未满时 Send 仍可能成功(buffer 还有空间,select 走 writeCh case)。
// 这是 hub 设计:Send 不保证 Close 后立即失败,依赖调用方在 Close 后不再 Send。
// 本测试只覆盖 writeCh 满 + closeCh 关闭时 Send 的 closeCh 路径。
func TestSend_AfterCloseWithFullQueueReturnsFalse(t *testing.T) {
	serverConn, _, cleanup := newTestWSConnsPair(t)
	defer cleanup()

	c := &Client{
		ID:                 uuid.New(),
		Role:               RoleOperator,
		Conn:               serverConn,
		logger:             newTestLogger(),
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 1), // 小 buffer
		closeCh:            make(chan struct{}),
	}

	// 先填满 writeCh(不启动 writeLoop)
	c.Send([]byte("fill"))
	// Close 关闭 closeCh
	c.Close(context.Background())

	// 此时 Send:writeCh 满 + closeCh 关闭 → select 走 closeCh 返回 false
	if c.Send([]byte("after-close-full")) {
		t.Error("Send after Close with full queue returned true, want false (should hit closeCh path)")
	}
}

// TestClose_Idempotent 验证多次调用 Close 不 panic。
func TestClose_Idempotent(t *testing.T) {
	serverConn, _, cleanup := newTestWSConnsPair(t)
	defer cleanup()

	c := &Client{
		ID:                 uuid.New(),
		Role:               RoleOperator,
		Conn:               serverConn,
		logger:             newTestLogger(),
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 4),
		closeCh:            make(chan struct{}),
	}

	// 多次 Close 不 panic(用 closeOnce 保护)
	c.Close(context.Background())
	c.Close(context.Background())
	c.Close(context.Background())
}

// TestClose_ClosesUnderlyingConn 验证 Close 关闭底层 websocket.Conn。
func TestClose_ClosesUnderlyingConn(t *testing.T) {
	serverConn, clientConn, cleanup := newTestWSConnsPair(t)
	defer cleanup()

	c := &Client{
		ID:                 uuid.New(),
		Role:               RoleOperator,
		Conn:               serverConn,
		logger:             newTestLogger(),
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 4),
		closeCh:            make(chan struct{}),
	}

	c.Close(context.Background())

	// client-side 应观察到连接关闭(读会返回 error)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, _, err := clientConn.Reader(ctx)
	if err == nil {
		t.Error("expected error after Close, got nil")
	}
}

// TestRegisterClient_AddsToHubMap 验证 RegisterClient 把 client 加进 hub.clients。
func TestRegisterClient_AddsToHubMap(t *testing.T) {
	h := newTestHub()
	c := &Client{
		ID:                 uuid.New(),
		Role:               RoleOperator,
		logger:             newTestLogger(),
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 4),
		closeCh:            make(chan struct{}),
	}

	h.RegisterClient(c)

	h.mu.RLock()
	_, ok := h.clients[c.ID]
	h.mu.RUnlock()
	if !ok {
		t.Errorf("client not registered in hub.clients")
	}
}

// TestRegisterClient_OverwriteSameID 验证 RegisterClient 同 ID 覆盖(map 语义)。
func TestRegisterClient_OverwriteSameID(t *testing.T) {
	h := newTestHub()
	id := uuid.New()

	c1 := &Client{
		ID: id, Role: RoleOperator, logger: newTestLogger(),
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 4),
		closeCh:            make(chan struct{}),
	}
	c2 := &Client{
		ID: id, Role: RoleOperator, logger: newTestLogger(),
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 4),
		closeCh:            make(chan struct{}),
	}

	h.RegisterClient(c1)
	h.RegisterClient(c2) // 覆盖

	h.mu.RLock()
	got := h.clients[id]
	count := len(h.clients)
	h.mu.RUnlock()
	if count != 1 {
		t.Errorf("client count: got %d, want 1", count)
	}
	if got != c2 {
		t.Errorf("registered client: got %p, want c2 %p", got, c2)
	}
}

// TestSendCommandToVisitor_VisitorOnline 验证 visitor 在线时命令下发成功。
func TestSendCommandToVisitor_VisitorOnline(t *testing.T) {
	h := newTestHub()
	tenantID := uuid.New()
	sessionID := uuid.New()
	visitorClientID := uuid.New()

	// 创建 visitor client(用手工 Client + 大 buffer writeCh)
	visitor := &Client{
		ID: visitorClientID, Role: RoleVisitor, logger: newTestLogger(),
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 16),
		closeCh:            make(chan struct{}),
	}
	defer close(visitor.closeCh)

	h.RegisterClient(visitor)
	h.VisitorOnline(tenantID, sessionID, visitorClientID, nil)

	cmd := []byte("navigate:https://example.com")
	if !h.SendCommandToVisitor(sessionID, cmd) {
		t.Error("SendCommandToVisitor returned false, want true (visitor online)")
	}

	// 验证消息到达 visitor writeCh
	select {
	case got := <-visitor.writeCh:
		if string(got) != "navigate:https://example.com" {
			t.Errorf("got %q, want navigate:...", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("visitor did not receive command")
	}
}

// TestSendCommandToVisitor_VisitorOfflineReturnsFalse 验证 visitor 不在线返回 false。
func TestSendCommandToVisitor_VisitorOfflineReturnsFalse(t *testing.T) {
	h := newTestHub()
	sessionID := uuid.New()

	if h.SendCommandToVisitor(sessionID, []byte("cmd")) {
		t.Error("SendCommandToVisitor returned true for unknown session, want false")
	}
}

// TestSendCommandToVisitor_ClientUnregisteredReturnsFalse 验证 visitor offline 后(SendCommandToVisitor 路径中的 visitor client = nil)。
func TestSendCommandToVisitor_ClientUnregisteredReturnsFalse(t *testing.T) {
	h := newTestHub()
	tenantID := uuid.New()
	sessionID := uuid.New()
	visitorClientID := uuid.New()

	h.VisitorOnline(tenantID, sessionID, visitorClientID, nil)
	// 故意不 RegisterClient,visitorClients 有映射但 clients map 没有该 client

	if h.SendCommandToVisitor(sessionID, []byte("cmd")) {
		t.Error("SendCommandToVisitor returned true for unregistered client, want false")
	}
}

// TestSendCommandToVisitor_ConcurrentMultipleSenders 验证多 goroutine 并发 Send 不出 race。
// 这是 -race -count=5 强制检查的 flaky 防护。
func TestSendCommandToVisitor_ConcurrentMultipleSenders(t *testing.T) {
	h := newTestHub()
	tenantID := uuid.New()
	sessionID := uuid.New()
	visitorClientID := uuid.New()

	visitor := &Client{
		ID: visitorClientID, Role: RoleVisitor, logger: newTestLogger(),
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 1024), // 大 buffer 容纳并发
		closeCh:            make(chan struct{}),
	}
	defer close(visitor.closeCh)

	h.RegisterClient(visitor)
	h.VisitorOnline(tenantID, sessionID, visitorClientID, nil)

	var success int32
	var wg sync.WaitGroup
	const N = 50
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			if h.SendCommandToVisitor(sessionID, []byte("cmd")) {
				atomic.AddInt32(&success, 1)
			}
		}(i)
	}
	wg.Wait()

	// 至少一些成功(可能因 close 路径或满 queue 失败)
	if atomic.LoadInt32(&success) == 0 {
		t.Errorf("expected some SendCommandToVisitor to succeed, got 0")
	}
}
