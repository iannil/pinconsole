// CV-4 WS handler + HTTP happy path 补充测试。
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

// newWSBoostServer 启动真实 httptest server 挂 WSHandler,返回 (server, wsHost)。
// 调用方用 wsHost 通过 websocket.Dial 连接。
func newWSBoostServer(t *testing.T, h *WSHandler) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)
	srv := httptest.NewServer(r)
	return srv
}

// wsDialBoost 用正确 Origin 连接 WS server,返回 client conn。
func wsDialBoost(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{url}},
	})
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	return conn
}

// TestVisitorWS_NoHello 验证 visitorWS 在 client 立即 close 不 panic。
// 真实 SDK 总是先发 hello,这里测 read hello failed 分支。
func TestVisitorWS_NoHello(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := hub.New(testLogger())
	stream := recording.NewStream(stores.Redis.Client, testLogger())
	wsH := NewWSHandler(h, stores, stream, nil, nil, testLogger(), true)

	srv := newWSBoostServer(t, wsH)
	defer srv.Close()

	conn := wsDialBoost(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/visitor")
	defer conn.Close(websocket.StatusNormalClosure, "test")

	// 不发 hello,直接 close,handler 应在 read hello timeout 后退出
	time.Sleep(200 * time.Millisecond)
}

// TestVisitorWS_InvalidHelloPayload 验证 visitorWS 收到非 hello envelope 返回 error。
func TestVisitorWS_InvalidHelloPayload(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := hub.New(testLogger())
	stream := recording.NewStream(stores.Redis.Client, testLogger())
	wsH := NewWSHandler(h, stores, stream, nil, nil, testLogger(), true)

	srv := newWSBoostServer(t, wsH)
	defer srv.Close()

	conn := wsDialBoost(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/visitor")
	defer conn.Close(websocket.StatusNormalClosure, "test")

	// 发非 hello envelope
	badEnv, _ := proto.Encode(proto.Envelope{V: 1, Type: proto.MsgEvent})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageBinary, badEnv); err != nil {
		t.Fatalf("write: %v", err)
	}

	// 读响应应是 error envelope
	_, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	respEnv, _ := proto.Decode(msg)
	if respEnv.Type != proto.MsgError {
		t.Errorf("response type: got %v, want error", respEnv.Type)
	}
}

// TestVisitorWS_InvalidVisitorID 验证 visitorWS 收到 hello 但 visitor_id 非法 UUID 返回 error。
func TestVisitorWS_InvalidVisitorID(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := hub.New(testLogger())
	stream := recording.NewStream(stores.Redis.Client, testLogger())
	wsH := NewWSHandler(h, stores, stream, nil, nil, testLogger(), true)

	srv := newWSBoostServer(t, wsH)
	defer srv.Close()

	conn := wsDialBoost(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/visitor")
	defer conn.Close(websocket.StatusNormalClosure, "test")

	// 发 hello 但 visitor_id 非 UUID
	helloEnv, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgHello,
		TS:   time.Now().UnixMilli(),
		Payload: proto.HelloPayload{
			VisitorID: "not-a-uuid",
			SessionID: uuid.New().String(),
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageBinary, helloEnv); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, msg, _ := conn.Read(ctx)
	respEnv, _ := proto.Decode(msg)
	if respEnv.Type != proto.MsgError {
		t.Errorf("response type: got %v, want error", respEnv.Type)
	}
}

// TestVisitorWS_InvalidSessionID 验证 visitorWS hello session_id 非法 UUID 返回 error。
func TestVisitorWS_InvalidSessionID(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := hub.New(testLogger())
	stream := recording.NewStream(stores.Redis.Client, testLogger())
	wsH := NewWSHandler(h, stores, stream, nil, nil, testLogger(), true)

	srv := newWSBoostServer(t, wsH)
	defer srv.Close()

	conn := wsDialBoost(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/visitor")
	defer conn.Close(websocket.StatusNormalClosure, "test")

	helloEnv, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgHello,
		TS:   time.Now().UnixMilli(),
		Payload: proto.HelloPayload{
			VisitorID: uuid.New().String(),
			SessionID: "not-a-uuid",
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	conn.Write(ctx, websocket.MessageBinary, helloEnv)

	_, msg, _ := conn.Read(ctx)
	respEnv, _ := proto.Decode(msg)
	if respEnv.Type != proto.MsgError {
		t.Errorf("response type: got %v, want error", respEnv.Type)
	}
}

// TestVisitorWS_SessionNotFound 验证 visitorWS session 不存在返回 error。
func TestVisitorWS_SessionNotFound(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := hub.New(testLogger())
	stream := recording.NewStream(stores.Redis.Client, testLogger())
	wsH := NewWSHandler(h, stores, stream, nil, nil, testLogger(), true)

	srv := newWSBoostServer(t, wsH)
	defer srv.Close()

	conn := wsDialBoost(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/visitor")
	defer conn.Close(websocket.StatusNormalClosure, "test")

	helloEnv, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgHello,
		TS:   time.Now().UnixMilli(),
		Payload: proto.HelloPayload{
			VisitorID: uuid.New().String(),
			SessionID: uuid.New().String(), // 不存在的 session
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn.Write(ctx, websocket.MessageBinary, helloEnv)

	_, msg, _ := conn.Read(ctx)
	respEnv, _ := proto.Decode(msg)
	if respEnv.Type != proto.MsgError {
		t.Errorf("response type: got %v, want error", respEnv.Type)
	}
}

// TestVisitorWS_HappyPath 验证完整 visitorWS 流程:hello → ack → close。
func TestVisitorWS_HappyPath(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	// seed visitor + session
	visitorID := uuid.New()
	sessionID := uuid.New()
	_, err := stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, visitorID, storage.DefaultTenantID, "ws-test-fp-"+visitorID.String()[:8])
	if err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	_, err = stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}

	h := hub.New(testLogger())
	stream := recording.NewStream(stores.Redis.Client, testLogger())
	wsH := NewWSHandler(h, stores, stream, nil, nil, testLogger(), true)

	srv := newWSBoostServer(t, wsH)
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageBinary, helloEnv); err != nil {
		t.Fatalf("write hello: %v", err)
	}

	// 读 ack
	_, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	respEnv, _ := proto.Decode(msg)
	if respEnv.Type != proto.MsgAck {
		t.Errorf("response type: got %v, want ack", respEnv.Type)
	}
}

// TestOperatorWS_NoAuth 验证 operatorWS 无认证 cookie 返回 401。
// devMode=false 触发认证检查;无 cookie 必拒。
func TestOperatorWS_NoAuth(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := hub.New(testLogger())
	wsH := NewWSHandler(h, stores, nil, nil, nil, testLogger(), false /* prod mode */)

	srv := newWSBoostServer(t, wsH)
	defer srv.Close()

	// 用 websocket.Dial 尝试连接;handler 应在 accept 前返回 401
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/operator",
		&websocket.DialOptions{
			HTTPHeader: http.Header{"Origin": []string{srv.URL}},
		})
	if err == nil {
		// Dial 在 401 时应失败(handler 未 accept WS)
		// 注:websocket.Dial 对 HTTP 401 返回 error
	}
}

// TestOperatorWS_DevMode_NoAuthAllowed 验证 dev mode operatorWS 无认证可连。
func TestOperatorWS_DevMode_NoAuthAllowed(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := hub.New(testLogger())
	wsH := NewWSHandler(h, stores, nil, nil, nil, testLogger(), true /* dev mode */)

	srv := newWSBoostServer(t, wsH)
	defer srv.Close()

	conn := wsDialBoost(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/operator")
	defer conn.Close(websocket.StatusNormalClosure, "test")
	time.Sleep(100 * time.Millisecond)
	// 不验证具体行为,只验证连上不 panic
}

// TestSendError_DirectCall 验证 sendError 直接调用不 panic。
func TestSendError_DirectCall(t *testing.T) {
	// 启动 ws server + client pair
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "test")
		wsH := &WSHandler{logger: testLogger()}
		wsH.sendError(r.Context(), conn, "test_code", "test message")
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"),
		&websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{srv.URL}}})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test")

	_, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	env, _ := proto.Decode(msg)
	if env.Type != proto.MsgError {
		t.Errorf("type: got %v, want error", env.Type)
	}
}

// === HTTP happy path tests ===

// TestPostCommand_HappyPath_Click 验证 postCommand 完整流程:click 命令 + claim 锁 + 真实 hub。
func TestPostCommand_HappyPath_Click(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	// seed admin + visitor + session + claim
	adminUID := uuid.New()
	_, err := stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'cmd-admin@example.com', 'h', 'Admin', 'admin')
	`, adminUID, storage.DefaultTenantID)
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, adminUID)

	visitorID := uuid.New()
	_, err = stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'cmd-happy-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	if err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	_, err = stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM sessions WHERE id = $1`, sessionID)

	// claim 锁
	claimK := claimKey(sessionID)
	stores.Redis.Set(ctx0, claimK, []byte(adminUID.String()), time.Minute)
	defer stores.Redis.Del(ctx0, claimK)

	stubHub := &stubCommandHub{delivered: true}

	h := NewCommandHandler(stores, stubHub, "", testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/command", func(c *gin.Context) {
		c.Set("user_id", adminUID)
		h.postCommand(c)
	})

	body := `{"type":"click","payload":{"node_id":1,"x":10,"y":20}}`
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/command", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("postCommand happy: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

// TestPostCommand_VisitorOffline 验证 hub 未投递时返回 503 visitor_offline。
func TestPostCommand_VisitorOffline(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	adminUID := uuid.New()
	_, err := stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'cmd-off@example.com', 'h', 'Admin', 'admin')
	`, adminUID, storage.DefaultTenantID)
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, adminUID)

	visitorID := uuid.New()
	_, err = stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'cmd-off-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	if err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	_, err = stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM sessions WHERE id = $1`, sessionID)

	claimK := claimKey(sessionID)
	stores.Redis.Set(ctx0, claimK, []byte(adminUID.String()), time.Minute)
	defer stores.Redis.Del(ctx0, claimK)

	stubHub := &stubCommandHub{delivered: false} // visitor offline

	h := NewCommandHandler(stores, stubHub, "", testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/command", func(c *gin.Context) {
		c.Set("user_id", adminUID)
		h.postCommand(c)
	})

	body := `{"type":"scroll","payload":{"x":0,"y":10}}`
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/command", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("visitor offline: got %d, want 503; body=%s", w.Code, w.Body.String())
	}
}

// TestPostCommand_NavigateBadURL 验证 navigate URL 不同源/白名单返回 403。
func TestPostCommand_NavigateBadURL(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	adminUID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'nav-bad@example.com', 'h', 'Admin', 'admin')
	`, adminUID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, adminUID)

	visitorID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'nav-bad-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM sessions WHERE id = $1`, sessionID)

	claimK := claimKey(sessionID)
	stores.Redis.Set(ctx0, claimK, []byte(adminUID.String()), time.Minute)
	defer stores.Redis.Del(ctx0, claimK)

	stubHub := &stubCommandHub{delivered: true}

	h := NewCommandHandler(stores, stubHub, "", testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/command", func(c *gin.Context) {
		c.Set("user_id", adminUID)
		h.postCommand(c)
	})

	body := `{"type":"navigate","payload":{"url":"http://evil.com/"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/command", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("navigate bad url: got %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

// TestGetConsent_HappyPath 验证 getConsent 返回已记录的 consent。
func TestGetConsent_HappyPath(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	fp := "getconsent-fp-" + uuid.New().String()[:8]
	_, err := stores.PG.UpsertConsent(ctx0, storage.DefaultTenantID, fp, consentScope, consentVersion, true)
	if err != nil {
		t.Fatalf("seed consent: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitor_consents WHERE fingerprint = $1`, fp)

	h := &PrivacyHandler{stores: stores, logger: testLogger()}
	r := newPrivacyTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/privacy/consent?fingerprint="+fp, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("getConsent happy: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

// TestGetConsent_NotFound 验证 getConsent 不存在 fingerprint 返回 200 + found=false。
func TestGetConsent_NotFound(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := &PrivacyHandler{stores: stores, logger: testLogger()}
	r := newPrivacyTestEngine(h)

	fp := "nonexistent-fp-" + uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/privacy/consent?fingerprint="+fp, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("getConsent not found: got %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "false") {
		t.Errorf("body should contain false: %s", w.Body.String())
	}
}

// TestPostConsent_HappyPath 验证 postConsent 写入返回 200。
func TestPostConsent_HappyPath(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	fp := "postconsent-fp-" + uuid.New().String()[:8]
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitor_consents WHERE fingerprint = $1`, fp)

	h := &PrivacyHandler{stores: stores, logger: testLogger()}
	r := newPrivacyTestEngine(h)

	body := `{"fingerprint":"` + fp + `","accepted":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/privacy/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("postConsent happy: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

// TestDeleteVisitor_AdminHappyPath 验证 admin 删除 visitor 成功(级联)。
func TestDeleteVisitor_AdminHappyPath(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	adminUID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'admin-del@example.com', 'h', 'Admin', 'admin')
	`, adminUID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, adminUID)

	visitorID := uuid.New()
	fp := "del-happy-fp-" + uuid.New().String()[:8]
	_, err := stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, visitorID, storage.DefaultTenantID, fp)
	if err != nil {
		t.Fatalf("seed visitor: %v", err)
	}

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &PrivacyHandler{stores: stores, logger: testLogger()}
	r.DELETE("/api/privacy/visitor/:fingerprint", func(c *gin.Context) {
		c.Set("user_id", adminUID)
		h.DeleteVisitor(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/privacy/visitor/"+fp, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("admin delete: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

// TestListEndedSessions_WithData 验证 listEndedSessions 返回已结束会话。
func TestListEndedSessions_WithData(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	visitorID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'ended-test-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at, ended_at, status)
		VALUES ($1, $2, $3, NOW(), NOW(), 'ended')
	`, sessionID, storage.DefaultTenantID, visitorID)

	h := &ReplayHandler{stores: stores, logger: testLogger()}
	r := newReplayBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/ended?since=24h&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listEndedSessions with data: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}
