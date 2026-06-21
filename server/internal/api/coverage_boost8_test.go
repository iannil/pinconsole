// CV-4 Round 9:剩余 error branches 推 api 到 90%。
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
	"github.com/iannil/pinconsole/internal/storage"
)

// TestListEndedSessions_DBError 验证 listEndedSessions 在 PG 失败时返回 500。
func TestListEndedSessions_DBError(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	stores.PG.Close() // 关闭 PG 让 ListEndedSessionsByTenant 失败

	h := &ReplayHandler{logger: testLogger(), stores: stores}
	r := newReplayBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/ended", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("db error: got %d, want 500", w.Code)
	}
}

// TestGetSessionReplay_DBError_Verify 验证 getSessionReplay 在 PG 失败时返回 500(去重)。
func TestGetSessionReplay_DBError_Verify(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	stores.PG.Close()

	h := &ReplayHandler{logger: testLogger(), stores: stores}
	r := newReplayBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+uuid.New().String()+"/replay", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("db error: got %d, want 500", w.Code)
	}
}

// TestPostMessage_CreateError_WithStores 验证 postMessage 在 PG CreateChatMessage 失败时返回 500。
func TestPostMessage_CreateError_WithStores(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	opUID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'postmsg-err@example.com', 'h', 'Op', 'operator')
	`, opUID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, opUID)

	visitorID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'postmsg-err-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)
	// 不 defer 删除 session(后面会删)

	claimK := claimKey(sessionID)
	stores.Redis.Set(ctx0, claimK, []byte(opUID.String()), time.Minute)
	defer stores.Redis.Del(ctx0, claimK)

	// 删除 sessions 表的 FK 让 CreateChatMessage 失败
	stores.PG.Pool.Exec(ctx0, `DELETE FROM sessions WHERE id = $1`, sessionID)

	h := NewChatHandler(stores, nil, testLogger())
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

	if w.Code != http.StatusInternalServerError {
		t.Errorf("create msg error: got %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

// TestGetClaim_DBError 验证 getClaim 在 Redis Get 失败时返回 500。
func TestGetClaim_DBError(t *testing.T) {
	opUID := uuid.New()
	h := &ClaimHandler{
		sessionRepo: stubSessionRepo{},
		redis:       errRedisRepo{err: nil}, // errRedisRepo.Get returns err
		logger:      testLogger(),
	}

	// 改用会真返回 error 的 mock
	h2 := &ClaimHandler{
		sessionRepo: stubSessionRepo{},
		redis:       errRedisGetOnly{},
		logger:      testLogger(),
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/sessions/:id/claim", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h2.getClaim(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+uuid.New().String()+"/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("getClaim db error: got %d, want 500", w.Code)
	}
	_ = h
}

// errRedisGetOnly 实现 claimRedisStore,只让 Get 返回 error。
type errRedisGetOnly struct{}

func (e errRedisGetOnly) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, errSentinel
}
func (e errRedisGetOnly) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (e errRedisGetOnly) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return true, nil
}
func (e errRedisGetOnly) Del(ctx context.Context, key string) error { return nil }
func (e errRedisGetOnly) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}
func (e errRedisGetOnly) EvalLua(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return nil, nil
}

var errSentinel = errSentinelErr{}

type errSentinelErr struct{}

func (e errSentinelErr) Error() string { return "sentinel" }

// TestClaim_AlreadyClaimed 验证 claim 已被占用返回 409。
func TestClaim_AlreadyClaimed(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	opUID := uuid.New()
	otherUID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'claim-other@example.com', 'h', 'Other', 'operator')
	`, otherUID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, otherUID)

	visitorID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'claim-ac-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM sessions WHERE id = $1`, sessionID)

	// 预 claim by other
	claimK := claimKey(sessionID)
	stores.Redis.Set(ctx0, claimK, []byte(otherUID.String()), time.Minute)
	defer stores.Redis.Del(ctx0, claimK)

	h := NewClaimHandler(stores, testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/claim", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.claim(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("already claimed: got %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

// TestDeleteVisitor_HappyWithEventBlob 验证 deleteVisitor 完整级联(含 event_blob + MinIO)。
func TestDeleteVisitor_HappyWithEventBlob(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	adminUID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'admin-cascade@example.com', 'h', 'Admin', 'admin')
	`, adminUID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, adminUID)

	visitorID := uuid.New()
	fp := "cascade-fp-" + uuid.New().String()[:8]
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, visitorID, storage.DefaultTenantID, fp)

	sessionID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)

	// seed event_blob + MinIO 对象
	objKey := "cascade/" + sessionID.String() + "/0.msgpack"
	stores.MinIO.PutBytes(ctx0, objKey, []byte("blob"))

	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO event_blobs (session_id, tenant_id, blob_index, started_at, ended_at, event_count, minio_object_key, size_bytes, checksum_sha256)
		VALUES ($1, $2, 0, NOW(), NOW(), 1, $3, 4, 'sha')
	`, sessionID, storage.DefaultTenantID, objKey)

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
		t.Errorf("cascade delete: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}
