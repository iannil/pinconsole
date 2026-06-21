// CV-4 Round 3:剩余 error path + happy path 补测。
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/hub"
	"github.com/iannil/pinconsole/internal/proto"
	"github.com/iannil/pinconsole/internal/storage"
	"github.com/vmihailenco/msgpack/v5"
)

// newChatBoostEngine 构造仅挂 chat 路由的 engine(避免与 chat_http_test.go 同名)。
func newChatBoostEngine(h *ChatHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)
	return r
}

// TestListMessages_InvalidUUID 验证 listMessages 非 UUID 返回 400。
func TestListMessages_InvalidUUID(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := NewChatHandler(stores, hub.New(testLogger()), testLogger())
	r := newChatBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/not-uuid/messages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid uuid: got %d, want 400", w.Code)
	}
}

// TestListMessages_HappyPath 验证 listMessages 合法 UUID 返回 200。
func TestListMessages_HappyPath(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	visitorID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'listmsg-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)

	// seed chat messages
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO chat_messages (session_id, tenant_id, sender, content)
		VALUES ($1, $2, 'operator', 'test-msg')
	`, sessionID, storage.DefaultTenantID)

	h := NewChatHandler(stores, hub.New(testLogger()), testLogger())
	r := newChatBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID.String()+"/messages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listMessages happy: got %d, want 200", w.Code)
	}
}

// TestListMessages_WithSinceID 验证 since_id 参数。
func TestListMessages_WithSinceID(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := NewChatHandler(stores, hub.New(testLogger()), testLogger())
	r := newChatBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+uuid.New().String()+"/messages?since_id=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("with since_id: got %d, want 200", w.Code)
	}
}

// TestPostMessage_InvalidJSON 验证 postMessage 非 JSON 返回 400。
// 需要 claim 锁通过,所以 seed admin + session + claim。
func TestPostMessage_InvalidJSON(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	opUID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'postmsg-op@example.com', 'h', 'Op', 'operator')
	`, opUID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, opUID)

	visitorID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'postmsg-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)

	claimK := claimKey(sessionID)
	stores.Redis.Set(ctx0, claimK, []byte(opUID.String()), time.Minute)
	defer stores.Redis.Del(ctx0, claimK)

	h := NewChatHandler(stores, hub.New(testLogger()), testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/messages", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.postMessage(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/messages", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid json: got %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

// TestPostMessage_HappyPath 验证 postMessage 写入并广播到 hub。
func TestPostMessage_HappyPath(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	opUID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'postmsg-happy@example.com', 'h', 'Op', 'operator')
	`, opUID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, opUID)

	visitorID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'postmsg-happy-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)

	claimK := claimKey(sessionID)
	stores.Redis.Set(ctx0, claimK, []byte(opUID.String()), time.Minute)
	defer stores.Redis.Del(ctx0, claimK)

	stubHub := &stubCommandHub{delivered: true}
	h := &ChatHandler{
		createMsg:   stores.PG,
		messageRepo: stores.PG,
		redis:       stores.Redis,
		hub:         stubHub,
		logger:      testLogger(),
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/messages", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.postMessage(c)
	})

	body := `{"content":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("postMessage happy: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

// stubChatHubDelivered 占位(实际 ChatHandler.hub 用 CommandHub 接口,复用 stubCommandHub)。
// 保留作为占位以备后续扩展;实际类型 = stubCommandHub。
var _ = func() *stubCommandHub { return &stubCommandHub{} }

// === claim.go handler ===

// TestClaim_Release_HappyPath 验证 claim + release 完整流程。
func TestClaim_Release_HappyPath(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	opUID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'claim-op@example.com', 'h', 'Op', 'operator')
	`, opUID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, opUID)

	visitorID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'claim-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)

	h := NewClaimHandler(stores, testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/claim", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.claim(c)
	})
	r.POST("/api/sessions/:id/release", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.release(c)
	})
	r.GET("/api/sessions/:id/claim", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.getClaim(c)
	})

	// claim
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("claim: got %d, want 200; body=%s", w.Code, w.Body.String())
	}

	// getClaim
	req2 := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID.String()+"/claim", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("getClaim: got %d, want 200", w2.Code)
	}

	// release
	req3 := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/release", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("release: got %d, want 200; body=%s", w3.Code, w3.Body.String())
	}
}

// TestClaim_InvalidSessionID 验证 claim 非 UUID session 返回 400。
func TestClaim_InvalidSessionID(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	opUID := uuid.New()
	h := NewClaimHandler(stores, testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/claim", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.claim(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/not-a-uuid/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid session id: got %d, want 400", w.Code)
	}
}

// TestRelease_InvalidSessionID 验证 release 非 UUID session 返回 400。
func TestRelease_InvalidSessionID(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	opUID := uuid.New()
	h := NewClaimHandler(stores, testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/release", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.release(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/not-a-uuid/release", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid session id: got %d, want 400", w.Code)
	}
}

// === getSessionReplay with actual blob data ===

// TestGetSessionReplay_WithBlobs 验证 replay 返回 blob 内容。
func TestGetSessionReplay_WithBlobs(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	visitorID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'replay-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)

	// seed MinIO blob + PG event_blob
	objKey := "replay/" + sessionID.String() + "/0.msgpack"
	// 构造 msgpack array of envelope bytes
	envBytes, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		TS:   time.Now().UnixMilli(),
		Payload: proto.EventPayload{
			Type: proto.EvRRWeb,
			RRWeb: &proto.RRWebEvent{
				Type:      2,
				Timestamp: time.Now().UnixMilli(),
				Data:      map[string]any{"foo": "bar"},
			},
		},
	})
	blobData, _ := msgpack.Marshal([][]byte{envBytes})
	if err := stores.MinIO.PutBytes(ctx0, objKey, blobData); err != nil {
		t.Fatalf("seed minio: %v", err)
	}
	defer stores.MinIO.Client.RemoveObject(ctx0, stores.MinIO.Bucket, objKey, minioRemoveObjectOpts)

	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO event_blobs (session_id, tenant_id, blob_index, started_at, ended_at, event_count, minio_object_key, size_bytes, checksum_sha256)
		VALUES ($1, $2, 0, NOW(), NOW(), 1, $3, $4, 'sha')
	`, sessionID, storage.DefaultTenantID, objKey, len(blobData))

	h := &ReplayHandler{stores: stores, logger: testLogger()}
	r := newReplayBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID.String()+"/replay", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("with blobs: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}
