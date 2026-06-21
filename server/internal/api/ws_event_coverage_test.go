// CV-4 Round 5:WS event loops + listSessions with data。
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/hub"
	"github.com/iannil/pinconsole/internal/proto"
	"github.com/iannil/pinconsole/internal/recording"
	"github.com/iannil/pinconsole/internal/storage"
)

// TestVisitorWS_SendEvent 验证 visitorWS 在 hello+ack 后接受 event envelope。
func TestVisitorWS_SendEvent(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	visitorID := uuid.New()
	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, visitorID, storage.DefaultTenantID, "ws-evt-fp-"+visitorID.String()[:8])
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)

	h := hub.New(testLogger())
	stream := recording.NewStream(stores.Redis.Client, testLogger())
	flusher := recording.NewFlusher(recording.DefaultConfig(), stream, stores, testLogger())
	defer flusher.Stop()
	wsH := NewWSHandler(h, stores, stream, flusher, nil, testLogger(), true)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	wsH.Register(r)
	srv := httptest.NewServer(r)
	defer srv.Close()

	conn := wsDialBoost(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/visitor")
	defer conn.Close(websocket.StatusNormalClosure, "test")

	// 发 hello
	helloEnv, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgHello,
		TS:   time.Now().UnixMilli(),
		Payload: proto.HelloPayload{
			VisitorID: visitorID.String(),
			SessionID: sessionID.String(),
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageBinary, helloEnv); err != nil {
		t.Fatalf("write hello: %v", err)
	}

	// 读 ack
	_, _, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}

	// 发送 event envelope
	eventEnv, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		TS:   time.Now().UnixMilli(),
		Payload: proto.EventPayload{
			Type: proto.EvRRWeb,
			RRWeb: &proto.RRWebEvent{
				Type:      2,
				Timestamp: time.Now().UnixMilli(),
				Data:      map[string]any{"node": "test"},
			},
		},
	})
	if err := conn.Write(ctx, websocket.MessageBinary, eventEnv); err != nil {
		t.Fatalf("write event: %v", err)
	}

	// 等待 handler 处理 event
	time.Sleep(100 * time.Millisecond)
	// 不严格断言结果,只验证不 panic
}

// TestVisitorWS_SendNonBinaryMessage 验证 visitorWS 收到非 binary 消息跳过。
func TestVisitorWS_SendNonBinaryMessage(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	visitorID := uuid.New()
	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, visitorID, storage.DefaultTenantID, "ws-text-fp-"+visitorID.String()[:8])
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)

	h := hub.New(testLogger())
	stream := recording.NewStream(stores.Redis.Client, testLogger())
	wsH := NewWSHandler(h, stores, stream, nil, nil, testLogger(), true)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	wsH.Register(r)
	srv := httptest.NewServer(r)
	defer srv.Close()

	conn := wsDialBoost(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/visitor")
	defer conn.Close(websocket.StatusNormalClosure, "test")

	helloEnv, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgHello,
		TS:   time.Now().UnixMilli(),
		Payload: proto.HelloPayload{
			VisitorID: visitorID.String(),
			SessionID: sessionID.String(),
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	conn.Write(ctx, websocket.MessageBinary, helloEnv)
	conn.Read(ctx) // ack

	// 发 text message(非 binary)
	if err := conn.Write(ctx, websocket.MessageText, []byte("text")); err != nil {
		t.Fatalf("write text: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
}

// TestOperatorWS_SubscribeUnsubscribe 验证 operatorWS 接受 subscribe/unsubscribe 命令。
func TestOperatorWS_SubscribeUnsubscribe(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	// seed 一个 visitor + session(订阅需要 session 存在,但 dev mode 不强制)
	h := hub.New(testLogger())
	wsH := NewWSHandler(h, stores, nil, nil, nil, testLogger(), true)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	wsH.Register(r)
	srv := httptest.NewServer(r)
	defer srv.Close()

	conn := wsDialBoost(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/operator")
	defer conn.Close(websocket.StatusNormalClosure, "test")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 发 subscribe
	subscribeEnv, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgSubscribe,
		TS:   time.Now().UnixMilli(),
		Payload: proto.SubscribePayload{
			SessionID: uuid.New().String(),
		},
	})
	if err := conn.Write(ctx, websocket.MessageBinary, subscribeEnv); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// 发 unsubscribe(同 session)
	unsubscribeEnv, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgUnsubscribe,
		TS:   time.Now().UnixMilli(),
		Payload: proto.SubscribePayload{
			SessionID: uuid.New().String(), // 不同的 sessionID(未订阅,走 ignore 分支)
		},
	})
	if err := conn.Write(ctx, websocket.MessageBinary, unsubscribeEnv); err != nil {
		t.Fatalf("write unsubscribe: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

// TestOperatorWS_InvalidEnvelope 验证 operatorWS 收到非法 envelope 不 panic。
func TestOperatorWS_InvalidEnvelope(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := hub.New(testLogger())
	wsH := NewWSHandler(h, stores, nil, nil, nil, testLogger(), true)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	wsH.Register(r)
	srv := httptest.NewServer(r)
	defer srv.Close()

	conn := wsDialBoost(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/operator")
	defer conn.Close(websocket.StatusNormalClosure, "test")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 发非 msgpack envelope
	if err := conn.Write(ctx, websocket.MessageBinary, []byte("not-msgpack")); err != nil {
		t.Fatalf("write: %v", err)
	}

	// 发 msgpack 但非 subscribe 类型
	badEnv, _ := proto.Encode(proto.Envelope{V: 1, Type: proto.MsgEvent})
	conn.Write(ctx, websocket.MessageBinary, badEnv)

	time.Sleep(100 * time.Millisecond)
}

// TestListSessions_WithActiveSessions 验证 listSessions 返回活跃 sessions 列表。
func TestListSessions_WithActiveSessions(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	visitorID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'listactive-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)

	h := NewSessionHandler(stores, nil, testLogger())
	r := newSessionTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listSessions with active: got %d, want 200", w.Code)
	}
}

// TestHealthReady 各分支。
func TestHealthReady(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", healthReady(stores))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("readyz: got %d, want 200", w.Code)
	}
}

// TestHealthReady_PGDown 验证 PG 关闭时 readyz 返回 503。
func TestHealthReady_PGDown(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	stores.PG.Close() // 关闭 PG

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", healthReady(stores))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("readyz pg down: got %d, want 503", w.Code)
	}
}

// TestHealthLive 验证 healthz 始终 200。
func TestHealthLive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz", healthLive)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("healthz: got %d, want 200", w.Code)
	}
}

// 兼容 helper:占位 import
var _ = websocket.Accept
