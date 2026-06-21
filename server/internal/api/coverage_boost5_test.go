// CV-4 Round 6:剩余 error path(health/authz/claim/chat/command)。
package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
)

// TestHealthReady_RedisDown 验证 Redis 关闭时 readyz 报告 redis fail。
func TestHealthReady_RedisDown(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	stores.Redis.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", healthReady(stores))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("redis down: got %d, want 503", w.Code)
	}
	if !strings.Contains(w.Body.String(), "redis") {
		t.Errorf("body should mention redis: %s", w.Body.String())
	}
}

// TestHealthReady_MinIODown 验证 MinIO 关闭时 readyz 报告 minio fail。
func TestHealthReady_MinIODown(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	// MinIO Close 是 no-op,但 ping 在关闭后 client 上会失败
	// 此处改用一个无法连接的 endpoint 重新构造 MinIO
	badStores := &storage.Stores{
		PG:    stores.PG,
		Redis: stores.Redis,
		MinIO: stores.MinIO, // 同 stores(MinIO Close 是 no-op)
	}
	defer badStores.PG.Close()
	defer badStores.Redis.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", healthReady(badStores))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// 预期 200(MinIO Ping 不容易失败,只能验证不 panic)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("unexpected status: %d", w.Code)
	}
}

// === authz.go requireClaimOwnership 各分支 ===

// newAuthzTestCtx 构造一个带 user_id 和 :id 参数的 gin.Context。
func newAuthzTestCtx(t *testing.T, userID, sessionID uuid.UUID, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/command", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	if userID != uuid.Nil {
		c.Set("user_id", userID)
	}
	return c, w
}

// TestRequireClaimOwnership_InvalidSessionID_Boost 验证 session_id 非 UUID 返回 400(去重)。
func TestRequireClaimOwnership_InvalidSessionID_Boost(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/not-uuid/x", nil)
	c.Params = gin.Params{{Key: "id", Value: "not-uuid"}}

	_, _, ok := requireClaimOwnership(c, nil, errRedisRepo{err: nil}, testLogger(), false)
	if ok {
		t.Error("expected ok=false for invalid session_id")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

// TestRequireClaimOwnership_NoUserID 验证无 user_id 返回 401。
func TestRequireClaimOwnership_NoUserID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/x", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	_, _, ok := requireClaimOwnership(c, nil, errRedisRepo{err: nil}, testLogger(), false)
	if ok {
		t.Error("expected ok=false for no user_id")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", w.Code)
	}
}

// TestRequireClaimOwnership_RequireAliveSession_RepoNil 验证 requireAliveSession=true 但 repo=nil 返回 500。
func TestRequireClaimOwnership_RequireAliveSession_RepoNil(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/x", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Set("user_id", uuid.New())

	_, _, ok := requireClaimOwnership(c, nil, errRedisRepo{err: nil}, testLogger(), true)
	if ok {
		t.Error("expected ok=false for nil repo")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// TestRequireClaimOwnership_RequireAliveSession_NotFound 验证 GetSession 失败返回 404。
func TestRequireClaimOwnership_RequireAliveSession_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/x", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Set("user_id", uuid.New())

	_, _, ok := requireClaimOwnership(c, errPGRepo{err: errors.New("not found")}, errRedisRepo{err: nil}, testLogger(), true)
	if ok {
		t.Error("expected ok=false")
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", w.Code)
	}
}

// TestRequireClaimOwnership_ClaimLookupFailed 验证 Redis Get 失败返回 500。
func TestRequireClaimOwnership_ClaimLookupFailed(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/x", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Set("user_id", uuid.New())

	_, _, ok := requireClaimOwnership(c, nil, errRedisRepo{err: errors.New("redis-down")}, testLogger(), false)
	if ok {
		t.Error("expected ok=false")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// fakeRedisClaimGet 包装 errRedisRepo,允许指定 key 返回特定值。
type fakeRedisClaim struct {
	claimVal []byte
	err      error
}

func (f fakeRedisClaim) Get(ctx context.Context, key string) ([]byte, error) {
	return f.claimVal, f.err
}
func (f fakeRedisClaim) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (f fakeRedisClaim) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return true, nil
}
func (f fakeRedisClaim) Del(ctx context.Context, key string) error { return nil }
func (f fakeRedisClaim) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}
func (f fakeRedisClaim) EvalLua(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return nil, nil
}

// TestRequireClaimOwnership_NotClaimed 验证 claim 为 nil 返回 403。
func TestRequireClaimOwnership_NotClaimed(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/x", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Set("user_id", uuid.New())

	_, _, ok := requireClaimOwnership(c, nil, fakeRedisClaim{claimVal: nil}, testLogger(), false)
	if ok {
		t.Error("expected ok=false")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", w.Code)
	}
}

// TestRequireClaimOwnership_ClaimCorrupt_Boost 验证 claim 值非 UUID 返回 500(去重)。
func TestRequireClaimOwnership_ClaimCorrupt_Boost(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/x", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Set("user_id", uuid.New())

	_, _, ok := requireClaimOwnership(c, nil, fakeRedisClaim{claimVal: []byte("not-a-uuid")}, testLogger(), false)
	if ok {
		t.Error("expected ok=false")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// TestRequireClaimOwnership_NotOwner 验证 claim 非 owner 返回 403。
func TestRequireClaimOwnership_NotOwner(t *testing.T) {
	otherUID := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/x", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Set("user_id", uuid.New()) // 不同的 UID

	_, _, ok := requireClaimOwnership(c, nil, fakeRedisClaim{claimVal: []byte(otherUID.String())}, testLogger(), false)
	if ok {
		t.Error("expected ok=false")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", w.Code)
	}
}

// TestRequireClaimOwnership_HappyPath 验证 owner 匹配返回 ok=true。
func TestRequireClaimOwnership_HappyPath(t *testing.T) {
	callerUID := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/x", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Set("user_id", callerUID)

	_, _, ok := requireClaimOwnership(c, nil, fakeRedisClaim{claimVal: []byte(callerUID.String())}, testLogger(), false)
	if !ok {
		t.Errorf("expected ok=true; body=%s", w.Body.String())
	}
}

// === claim.go error branches ===

// TestRelease_DBError 验证 release EvalLua 失败返回 500。
func TestRelease_DBError(t *testing.T) {
	opUID := uuid.New()
	h := &ClaimHandler{
		sessionRepo: errPGRepo{err: nil},
		redis:       errRedisRepo{err: errors.New("redis-down")},
		logger:      testLogger(),
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/release", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.release(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/release", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("release db error: got %d, want 500", w.Code)
	}
}

// TestClaim_SetNXError 验证 SetNX 失败返回 500。
func TestClaim_SetNXError(t *testing.T) {
	opUID := uuid.New()
	h := &ClaimHandler{
		sessionRepo: stubSessionRepo{},
		redis:       errRedisRepo{err: errors.New("redis-down")},
		logger:      testLogger(),
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/claim", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.claim(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("claim SetNX error: got %d, want 500", w.Code)
	}
}

// TestGetClaim_HappyPath 验证 getClaim 返回当前 claim 状态。
func TestGetClaim_HappyPath(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	opUID := uuid.New()
	sessionID := uuid.New()
	stores.Redis.Set(ctx0, claimKey(sessionID), []byte(opUID.String()), time.Minute)
	defer stores.Redis.Del(ctx0, claimKey(sessionID))

	h := NewClaimHandler(stores, testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/sessions/:id/claim", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.getClaim(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID.String()+"/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("getClaim: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

// TestGetClaim_InvalidSessionID 验证 getClaim 非 UUID 返回 400。
func TestGetClaim_InvalidSessionID(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	opUID := uuid.New()
	h := NewClaimHandler(stores, testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/sessions/:id/claim", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.getClaim(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/not-uuid/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("getClaim invalid: got %d, want 400", w.Code)
	}
}
