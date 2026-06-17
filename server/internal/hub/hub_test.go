package hub

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// newTestHub 创建测试用 hub。
func newTestHub() *Hub {
	return New(slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{Level: slog.LevelError})))
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
func (discardWriter) Close() error                { return nil }

// TestVisitorOnlineOfflineSubscription 验证 visitor 上线/下线、operator 订阅/退订的核心路由。
func TestVisitorOnlineOfflineSubscription(t *testing.T) {
	h := newTestHub()
	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建 operator（不需要真实 WS Conn，仅 hub 内存测试）
	opClient := &Client{
		ID:                 uuid.New(),
		Role:               RoleOperator,
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 16),
		closeCh:            make(chan struct{}),
		logger:             slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{Level: slog.LevelError})),
	}
	go func() {
		for range opClient.writeCh {
		}
	}()
	defer close(opClient.closeCh)

	// operator 加入 tenant room
	presenceCh := h.JoinTenantRoom(opClient, tenantID)

	// visitor 上线 → operator 应收到 presence online
	h.VisitorOnline(tenantID, sessionID, uuid.New(), []byte("online"))
	select {
	case msg := <-presenceCh:
		if string(msg) != "online" {
			t.Errorf("got %q, want online", msg)
		}
	default:
		t.Fatal("did not receive presence online")
	}

	// operator 订阅 session → 应能收到事件
	eventCh := h.SubscribeSession(opClient, sessionID)
	h.PublishEvent(sessionID, []byte("event-1"))
	select {
	case msg := <-eventCh:
		if string(msg) != "event-1" {
			t.Errorf("got %q, want event-1", msg)
		}
	default:
		t.Fatal("operator did not receive event after subscribe")
	}

	// operator 退订 → 不再收
	h.UnsubscribeSession(opClient, sessionID)
	h.PublishEvent(sessionID, []byte("event-2"))
	select {
	case msg, ok := <-eventCh:
		if ok {
			t.Errorf("channel should be closed, got msg %q", msg)
		}
	default:
		// chan 已关闭，符合预期
	}

	// visitor 下线 → operator 收到 presence offline
	h.VisitorOffline(tenantID, sessionID, []byte("offline"))
	select {
	case msg := <-presenceCh:
		if string(msg) != "offline" {
			t.Errorf("got %q, want offline", msg)
		}
	default:
		t.Fatal("did not receive presence offline")
	}
}

// TestMultipleOperatorsBothReceive 验证两个 operator 同时订阅同一 session 都收到事件。
func TestMultipleOperatorsBothReceive(t *testing.T) {
	h := newTestHub()
	tenantID := uuid.New()
	sessionID := uuid.New()
	h.VisitorOnline(tenantID, sessionID, uuid.New(), nil)

	makeClient := func() *Client {
		c := &Client{
			ID:                 uuid.New(),
			Role:               RoleOperator,
			subscribedSessions: make(map[uuid.UUID]struct{}),
			joinedTenants:      make(map[uuid.UUID]struct{}),
			writeCh:            make(chan []byte, 16),
			closeCh:            make(chan struct{}),
			logger:             slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{Level: slog.LevelError})),
		}
		return c
	}

	op1, op2 := makeClient(), makeClient()
	defer close(op1.closeCh)
	defer close(op2.closeCh)

	ch1 := h.SubscribeSession(op1, sessionID)
	ch2 := h.SubscribeSession(op2, sessionID)

	const N = 100
	consumed1 := make(chan int, 1)
	consumed2 := make(chan int, 1)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		count := 0
		for range ch1 {
			count++
		}
		consumed1 <- count
	}()
	go func() {
		defer wg.Done()
		count := 0
		for range ch2 {
			count++
		}
		consumed2 <- count
	}()

	// 给消费者足够时间 ready
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < N; i++ {
		h.PublishEvent(sessionID, []byte("evt"))
	}

	h.VisitorOffline(tenantID, sessionID, nil)
	wg.Wait()

	// publish 用非阻塞发送，缓冲满时会丢弃（这是 hub 的语义）。
	// 验证：两边都收到了事件（>0），且数量相当（差异 < 50%）。
	got1 := <-consumed1
	got2 := <-consumed2
	if got1 == 0 || got2 == 0 {
		t.Errorf("expected both ops to receive events; got op1=%d op2=%d", got1, got2)
	}
	diff := got1 - got2
	if diff < 0 {
		diff = -diff
	}
	if diff > N/2 {
		t.Errorf("expected roughly equal; got op1=%d op2=%d", got1, got2)
	}
}

// TestUnregisterClientCleansUp 验证 UnregisterClient 清理订阅。
func TestUnregisterClientCleansUp(t *testing.T) {
	h := newTestHub()
	tenantID := uuid.New()
	sessionID := uuid.New()
	h.VisitorOnline(tenantID, sessionID, uuid.New(), nil)

	c := &Client{
		ID:                 uuid.New(),
		Role:               RoleOperator,
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 16),
		closeCh:            make(chan struct{}),
		logger:             slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{Level: slog.LevelError})),
	}
	defer close(c.closeCh)

	_ = h.SubscribeSession(c, sessionID)
	_ = h.JoinTenantRoom(c, tenantID)

	if len(c.subscribedSessions) != 1 {
		t.Errorf("expected 1 subscription, got %d", len(c.subscribedSessions))
	}

	h.UnregisterClient(c)

	if len(c.subscribedSessions) != 0 {
		t.Errorf("expected 0 subscriptions after unregister, got %d", len(c.subscribedSessions))
	}
}
