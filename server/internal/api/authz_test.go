// 1ac 测试:requireClaimOwnership 横向越权防护(审计 T0-1k-1/2)。
//
// 验证:
//   - 非 owner 调用 → 403 not_claim_owner
//   - 未被 claim 的 session → 403 not_claimed
//   - owner 调用 → ok=true
//
// 关联:command.go / chat.go 都通过 requireClaimOwnership 做授权,
// 此处直接测 helper 等同于覆盖 command + chat 两路(T0-1k-1 + T0-1k-2)。
package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
	"github.com/redis/go-redis/v9"
)

// testLogger 返回一个丢弃所有输出的 slog logger,测试用。
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// helperRedisIfAvailable 返回真 Redis client,不可用时 skip。
func helperRedisIfAvailable(t *testing.T) *redis.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("需要 Redis")
	}
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis 不可用(%v),跳过", err)
	}
	return rdb
}

// newAuthzTestContext 构造一个带 user_id 和 :id 参数的 gin.Context,
// 配合 httptest.Recorder 捕获 requireClaimOwnership 写的响应。
func newAuthzTestContext(sessionID, callerUID uuid.UUID) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/"+sessionID.String()+"/command", nil)
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", callerUID)
	return c, w
}

// TestRequireClaimOwnership_NotClaimed_Returns403 — T0-1k-1/2 基线:
// session 未被任何运营 claim → 403 not_claimed(防匿名调用 command/chat)。
func TestRequireClaimOwnership_NotClaimed_Returns403(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	sessionID := uuid.New()
	callerUID := uuid.New()

	// 确保 claim key 不存在
	ctx := context.Background()
	rdb.Del(ctx, claimKey(sessionID))

	stores := &storage.Stores{Redis: &storage.Redis{Client: rdb}}
	c, w := newAuthzTestContext(sessionID, callerUID)

	_, _, ok := requireClaimOwnership(c, stores, nil, false)
	if ok {
		t.Errorf("ok=true want false (session not claimed)")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status=%d want 403 (not_claimed)", w.Code)
	}
	if !contains(w.Body.String(), "not_claimed") {
		t.Errorf("body should contain not_claimed, got: %s", w.Body.String())
	}
}

// TestRequireClaimOwnership_OwnerMismatch_Returns403 — T0-1k-1/2 核心:
// session 被 A claim,B 调用 → 403 not_claim_owner + 暴露 claimed_by(便于审计)
func TestRequireClaimOwnership_OwnerMismatch_Returns403(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	sessionID := uuid.New()
	ownerUID := uuid.New()
	callerUID := uuid.New() // 不同的运营

	ctx := context.Background()
	key := claimKey(sessionID)
	defer rdb.Del(ctx, key)
	if err := rdb.Set(ctx, key, ownerUID.String(), 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed claim: %v", err)
	}

	stores := &storage.Stores{Redis: &storage.Redis{Client: rdb}}
	c, w := newAuthzTestContext(sessionID, callerUID)

	_, _, ok := requireClaimOwnership(c, stores, nil, false)
	if ok {
		t.Errorf("ok=true want false (caller != owner)")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status=%d want 403 (not_claim_owner)", w.Code)
	}
	body := w.Body.String()
	if !contains(body, "not_claim_owner") {
		t.Errorf("body should contain not_claim_owner, got: %s", body)
	}
	if !contains(body, ownerUID.String()) {
		t.Errorf("body should expose claimed_by for audit, got: %s", body)
	}
}

// TestRequireClaimOwnership_Owner_Ok — owner 调用应通过,返回 callerUID 用于后续审计。
func TestRequireClaimOwnership_Owner_Ok(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	sessionID := uuid.New()
	ownerUID := uuid.New()

	ctx := context.Background()
	key := claimKey(sessionID)
	defer rdb.Del(ctx, key)
	if err := rdb.Set(ctx, key, ownerUID.String(), 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed claim: %v", err)
	}

	stores := &storage.Stores{Redis: &storage.Redis{Client: rdb}}
	c, _ := newAuthzTestContext(sessionID, ownerUID)

	gotSID, gotCaller, ok := requireClaimOwnership(c, stores, nil, false)
	if !ok {
		t.Errorf("ok=false want true (owner should pass)")
	}
	if gotSID != sessionID {
		t.Errorf("returned sessionID=%s want %s", gotSID, sessionID)
	}
	if gotCaller != ownerUID {
		t.Errorf("returned callerUID=%s want %s (for OperatorID audit)", gotCaller, ownerUID)
	}
}

// TestRequireClaimOwnership_InvalidSessionID — 不合法 UUID 返回 400。
func TestRequireClaimOwnership_InvalidSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/not-a-uuid/command", nil)
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	c.Set("user_id", uuid.New())

	_, _, ok := requireClaimOwnership(c, &storage.Stores{}, nil, false)
	if ok {
		t.Errorf("ok=true want false (invalid session_id)")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d want 400 (invalid_session_id)", w.Code)
	}
}

// TestRequireClaimOwnership_NoUserID_Returns401 — 兜底:AuthMiddleware 应已拦截匿名,
// 但若 ctx 缺 user_id 也要 fail-secure(不静默放行)。
func TestRequireClaimOwnership_NoUserID_Returns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/"+uuid.New().String()+"/command", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	// 不 Set user_id,模拟 AuthMiddleware 失效

	_, _, ok := requireClaimOwnership(c, &storage.Stores{}, nil, false)
	if ok {
		t.Errorf("ok=true want false (no user_id in ctx)")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401 (not_authenticated)", w.Code)
	}
}

// TestRequireClaimOwnership_ClaimCorrupt — claim 值非 UUID 形态 → 500(防数据损坏绕过)。
func TestRequireClaimOwnership_ClaimCorrupt(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	sessionID := uuid.New()
	callerUID := uuid.New()

	ctx := context.Background()
	key := claimKey(sessionID)
	defer rdb.Del(ctx, key)
	// 写入损坏的 claim 值(非 UUID 字符串)
	if err := rdb.Set(ctx, key, "not-a-uuid-corrupt", 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed corrupt: %v", err)
	}

	stores := &storage.Stores{Redis: &storage.Redis{Client: rdb}}
	c, w := newAuthzTestContext(sessionID, callerUID)

	_, _, ok := requireClaimOwnership(c, stores, testLogger(), false)
	if ok {
		t.Errorf("ok=true want false (corrupt claim value)")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status=%d want 500 (claim_corrupt)", w.Code)
	}
}
