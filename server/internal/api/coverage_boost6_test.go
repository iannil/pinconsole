// CV-4 Round 7:auth error paths + claim ended-session branch + chat error。
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
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

// TestCheckLoginThrottle_ParseError 验证 checkLoginThrottle 非 int value 返回 error。
// 直接构造 AuthHandler 调内部 method。
func TestCheckLoginThrottle_ParseError(t *testing.T) {
	h := &AuthHandler{
		redis: fakeRedisClaim{claimVal: []byte("not-a-number")},
		logger: testLogger(),
	}
	ctx := context.Background()
	_, _, err := h.checkLoginThrottle(ctx, "any-key")
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}

// TestCheckLoginThrottle_LockedTTLFail 验证 locked 但 TTL fail 时用 fallback。
func TestCheckLoginThrottle_LockedTTLFail(t *testing.T) {
	h := &AuthHandler{
		redis: lockedRedisRepo{},
		logger: testLogger(),
	}
	ctx := context.Background()
	locked, retryAfter, err := h.checkLoginThrottle(ctx, "any-key")
	if err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	if !locked {
		t.Error("expected locked=true")
	}
	_ = retryAfter
}

// lockedRedisRepo 实现 authRedisStore,返回 locked 状态(value=10, TTL fail)。
type lockedRedisRepo struct{}

func (l lockedRedisRepo) Get(ctx context.Context, key string) ([]byte, error) {
	return []byte("10"), nil // count = 10 (>= 5 max attempts)
}
func (l lockedRedisRepo) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (l lockedRedisRepo) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return true, nil
}
func (l lockedRedisRepo) Del(ctx context.Context, key string) error { return nil }
func (l lockedRedisRepo) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, errors.New("ttl-fail")
}
func (l lockedRedisRepo) EvalLua(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return nil, nil
}

// TestCheckLoginThrottle_LockedTTLNegative 验证 locked 但 TTL < 0 用 fallback。
func TestCheckLoginThrottle_LockedTTLNegative(t *testing.T) {
	h := &AuthHandler{
		redis: negTTLRedisRepo{},
		logger: testLogger(),
	}
	ctx := context.Background()
	locked, _, err := h.checkLoginThrottle(ctx, "any-key")
	if err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	if !locked {
		t.Error("expected locked=true")
	}
}

// negTTLRedisRepo 实现 authRedisStore,返回 locked 但 TTL < 0。
type negTTLRedisRepo struct{}

func (l negTTLRedisRepo) Get(ctx context.Context, key string) ([]byte, error) {
	return []byte("10"), nil
}
func (l negTTLRedisRepo) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (l negTTLRedisRepo) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return true, nil
}
func (l negTTLRedisRepo) Del(ctx context.Context, key string) error { return nil }
func (l negTTLRedisRepo) TTL(ctx context.Context, key string) (time.Duration, error) {
	return -1 * time.Second, nil
}
func (l negTTLRedisRepo) EvalLua(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return nil, nil
}

// TestRecordLoginFailure_Error 验证 recordLoginFailure 在 EvalLua 失败时不 panic。
func TestRecordLoginFailure_Error(t *testing.T) {
	h := &AuthHandler{
		redis: errRedisRepo{err: errors.New("redis-down")},
		logger: testLogger(),
	}
	ctx := context.Background()
	h.recordLoginFailure(ctx, "any-key")
}

// TestLogin_RedisSetSessionFail 验证 login 在 Redis Set session 失败时返回 500。
func TestLogin_RedisSetSessionFail(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	email := "setfail@example.com"
	hash, _ := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.DefaultCost)
	userID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, $3, $4, 'SetFail', 'admin')
	`, userID, storage.DefaultTenantID, email, string(hash))
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, userID)

	// 用 errRedisRepo 替换 redis(让 Set 失败)
	h := &AuthHandler{
		userRepo:     stores.PG,
		redis:        errRedisRepo{err: errors.New("redis-set-fail")},
		logger:       testLogger(),
		secureCookie: false,
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/auth/login", h.login)

	body := `{"email":"` + email + `","password":"pw"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:55555"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("set fail: got %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

// TestLogin_ThrottleCheckFail_FailOpen 验证 login throttle check 失败时 fail-open 不阻塞。
func TestLogin_ThrottleCheckFail_FailOpen(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	email := "throttle-fail-" + uuid.New().String() + "@example.com"
	hash, _ := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.DefaultCost)
	userID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, $3, $4, 'Throttle', 'admin')
	`, userID, storage.DefaultTenantID, email, string(hash))
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, userID)

	// 用 failOpenRedis 让 Get 失败,但 Set 成功(throttle check fail-open)
	h := &AuthHandler{
		userRepo: stores.PG,
		redis:    failOpenRedis{},
		logger:   testLogger(),
		secureCookie: false,
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/auth/login", h.login)

	body := `{"email":"` + email + `","password":"pw"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:55556"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("throttle fail-open: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

// failOpenRedis 实现 authRedisStore,Get 失败让 throttle check fail-open,
// Set 成功让 login 主流程走完。
type failOpenRedis struct{}

func (f failOpenRedis) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, errors.New("redis-down")
}
func (f failOpenRedis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (f failOpenRedis) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return true, nil
}
func (f failOpenRedis) Del(ctx context.Context, key string) error { return nil }
func (f failOpenRedis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}
func (f failOpenRedis) EvalLua(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return nil, nil
}

// TestRequireClaimOwnership_SessionEnded 验证 requireClaimOwnership 在 ended session 返回 409。
func TestRequireClaimOwnership_SessionEnded(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/x", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Set("user_id", uuid.New())

	// stubSessionEnded 返回 EndedAt.Valid=true 的 session
	_, _, ok := requireClaimOwnership(c, stubEndedSessionRepo{}, errRedisRepo{err: nil}, testLogger(), true)
	if ok {
		t.Error("expected ok=false")
	}
	if w.Code != http.StatusConflict {
		t.Errorf("got %d, want 409", w.Code)
	}
}

// stubEndedSessionRepo 实现 claimSessionRepo,返回 EndedAt.Valid=true。
type stubEndedSessionRepo struct{}

func (s stubEndedSessionRepo) GetSession(ctx context.Context, id uuid.UUID) (*storage.Session, error) {
	now := time.Now()
	return &storage.Session{
		ID: id,
		EndedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}, nil
}
