// 1ah 测试:claim.go HTTP 入口行为级测试。
//
// 补 1ac(原语层 SetNX/Lua)未覆盖的 HTTP handler 层:
//   - Register 路由 wire-up
//   - claim/release/getClaim UUID 拒绝路径(零依赖)
//   - getClaim claimed/not-claimed 状态(真 Redis seed)
//   - release non-owner → 403(真 Redis seed + 注入 user_id)
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
	"github.com/iannil/marketing-monitor/internal/storage"
)

// newClaimTestEngine 构造仅挂 claim 路由的 gin engine。
func newClaimTestEngine(h *ClaimHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)
	return r
}

// TestClaimRegister_Routes — 验证 Register 注册 3 个路由。
// 防 router 改名/漏挂。
func TestClaimRegister_Routes(t *testing.T) {
	h := &ClaimHandler{logger: testLogger()}
	r := newClaimTestEngine(h)

	wantRoutes := map[string]bool{
		"POST /api/sessions/:id/claim":   false,
		"POST /api/sessions/:id/release": false,
		"GET /api/sessions/:id/claim":    false,
	}
	for _, ri := range r.Routes() {
		k := ri.Method + " " + ri.Path
		if _, want := wantRoutes[k]; want {
			wantRoutes[k] = true
		}
	}
	for route, found := range wantRoutes {
		if !found {
			t.Errorf("claim route missing: %s", route)
		}
	}
}

// TestClaim_InvalidUUID_Returns400 — claim handler 非 UUID 必返 400。
//
// uuid.Parse 在 stores 调用前(claim.go:62-66)。
func TestClaim_InvalidUUID_Returns400(t *testing.T) {
	h := &ClaimHandler{logger: testLogger()}
	r := newClaimTestEngine(h)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/not-a-uuid/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_session_id") {
		t.Errorf("body = %q, want contains 'invalid_session_id'", w.Body.String())
	}
}

// TestRelease_InvalidUUID_Returns400 — release handler 非 UUID 必返 400。
func TestRelease_InvalidUUID_Returns400(t *testing.T) {
	h := &ClaimHandler{logger: testLogger()}
	r := newClaimTestEngine(h)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/not-a-uuid/release", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_session_id") {
		t.Errorf("body = %q, want contains 'invalid_session_id'", w.Body.String())
	}
}

// TestGetClaim_InvalidUUID_Returns400 — getClaim handler 非 UUID 必返 400。
//
// getClaim 此前完全 0% 覆盖,这是它的第一组测试。
func TestGetClaim_InvalidUUID_Returns400(t *testing.T) {
	h := &ClaimHandler{logger: testLogger()}
	r := newClaimTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/not-a-uuid/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_session_id") {
		t.Errorf("body = %q, want contains 'invalid_session_id'", w.Body.String())
	}
}

// TestGetClaim_NotClaimed_ReturnsFalse — 未 seed 的 session GET claim 必返 claimed:false。
//
// 真 Redis;不可用时 skip。沿用 1x/1ag 既定模式。
func TestGetClaim_NotClaimed_ReturnsFalse(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	sessionID := uuid.New()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	defer rdb.Del(ctx, claimKey(sessionID))

	h := &ClaimHandler{
		redis:  &storage.Redis{Client: rdb},
		logger: testLogger(),
	}
	r := newClaimTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID.String()+"/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"claimed":false`) {
		t.Errorf("body = %q, want contains 'claimed:false'", w.Body.String())
	}
}

// TestGetClaim_Claimed_ReturnsOwner — seed claim:session:<UUID>=<uid>,
// GET 必返 claimed:true + claimed_by=<uid>。
func TestGetClaim_Claimed_ReturnsOwner(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	sessionID := uuid.New()
	ownerUID := uuid.New()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	defer rdb.Del(ctx, claimKey(sessionID))

	if err := rdb.Set(ctx, claimKey(sessionID), ownerUID.String(), claimTTL).Err(); err != nil {
		t.Fatalf("seed claim: %v", err)
	}

	h := &ClaimHandler{
		redis:  &storage.Redis{Client: rdb},
		logger: testLogger(),
	}
	r := newClaimTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID.String()+"/claim", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"claimed":true`) {
		t.Errorf("body = %q, want contains 'claimed:true'", body)
	}
	if !strings.Contains(body, ownerUID.String()) {
		t.Errorf("body = %q, want contains owner UID %s", body, ownerUID.String())
	}
}

// TestRelease_NonOwner_Returns403 — 1k P0-5 关键回归:
// release 由 non-owner 调用必返 403 not_claim_owner(防运营 A 误释放 B 的 claim)。
//
// seed claim:session:<UUID>=<uid1>,以 uid2 调用 release,期望 403。
// 用 CreateTestContext 注入 user_id(模拟 AuthMiddleware)。
func TestRelease_NonOwner_Returns403(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	sessionID := uuid.New()
	ownerUID := uuid.New()
	callerUID := uuid.New()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	defer rdb.Del(ctx, claimKey(sessionID))

	if err := rdb.Set(ctx, claimKey(sessionID), ownerUID.String(), claimTTL).Err(); err != nil {
		t.Fatalf("seed claim: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/release", nil)
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", callerUID)

	h := &ClaimHandler{
		redis:  &storage.Redis{Client: rdb},
		logger: testLogger(),
	}
	h.release(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (non-owner release must be forbidden)", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not_claim_owner") {
		t.Errorf("body = %q, want contains 'not_claim_owner'", w.Body.String())
	}

	// 验证 claim 仍存在(release 未误删)
	got, err := rdb.Get(ctx, claimKey(sessionID)).Result()
	if err != nil {
		t.Fatalf("Get after non-owner release: %v", err)
	}
	if got != ownerUID.String() {
		t.Errorf("claim value = %q, want %s (non-owner release should not modify)", got, ownerUID.String())
	}
}
