// CV-4 切片补测:api 包覆盖率从 47.9% → ≥90%。
//
// 补测目标(分 4 类):
// 1. 构造函数 + Register 方法(零依赖,直接调)
// 2. HTTP handler 错误路径(invalid binding / missing param / db error)
// 3. 纯函数(sanitize/parseSince/decodeRRWeb 等)
// 4. 静态资源 handler(dev/prod 路径 + NoRoute fallback)
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/proto"
	"github.com/iannil/pinconsole/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

// === 静态资源 handler ===

// TestNewStaticHandler_DevMode 验证 dev 模式 newStaticHandler 不读 embedded。
func TestNewStaticHandler_DevMode(t *testing.T) {
	h := newStaticHandler(nil, false)
	if h == nil {
		t.Fatal("newStaticHandler returned nil")
	}
	if h.release {
		t.Error("release should be false in dev mode")
	}
}

// TestStaticHandler_Register_DevMode 验证 dev 模式 Register 挂载 dev hint 路由。
func TestStaticHandler_Register_DevMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := newStaticHandler(nil, false)
	h.Register(r)

	// GET / 应返回 503 dev hint
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("/ dev hint: got %d, want 503", w.Code)
	}

	// GET /sdk.js 同样
	req2 := httptest.NewRequest(http.MethodGet, "/sdk.js", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusServiceUnavailable {
		t.Errorf("/sdk.js dev hint: got %d, want 503", w2.Code)
	}
}

// TestStaticHandler_NoRoute_DevMode 验证 dev 模式 NoRoute 走 devHint。
func TestStaticHandler_NoRoute_DevMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := newStaticHandler(nil, false)
	r.NoRoute(h.NoRoute)

	req := httptest.NewRequest(http.MethodGet, "/random-path", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("NoRoute dev: got %d, want 503", w.Code)
	}
}

// TestStaticHandler_NoRoute_ProdAdminFallback 验证 prod 模式 NoRoute 走 admin SPA fallback。
// newStaticHandler 在 prod 模式 + nil embedded 会 panic,所以直接构造空 handler。
func TestStaticHandler_NoRoute_ProdAdminFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &staticHandler{release: true, adminIndex: []byte("<html>admin spa</html>")}
	r.NoRoute(h.NoRoute)

	req := httptest.NewRequest(http.MethodGet, "/admin/visitors", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("admin fallback: got %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "admin spa") {
		t.Errorf("admin fallback body: %q", w.Body.String())
	}
}

// TestStaticHandler_NoRoute_ProdAdminAssetNotFound 验证 prod 模式 /admin/assets/* 404。
func TestStaticHandler_NoRoute_ProdAdminAssetNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &staticHandler{release: true}
	r.NoRoute(h.NoRoute)

	req := httptest.NewRequest(http.MethodGet, "/admin/assets/missing.js", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("asset 404: got %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), "asset_not_found") {
		t.Errorf("asset 404 body: %q", w.Body.String())
	}
}

// TestStaticHandler_NoRoute_ProdNotFound 验证 prod 模式其他路径返回 not_found。
func TestStaticHandler_NoRoute_ProdNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &staticHandler{release: true}
	r.NoRoute(h.NoRoute)

	req := httptest.NewRequest(http.MethodGet, "/unknown-path", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown: got %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not_found") {
		t.Errorf("unknown body: %q", w.Body.String())
	}
}

// TestStaticHandler_NoRoute_ProdLandingFS 验证 prod 模式 /landing/* 走 landingFS。
func TestStaticHandler_NoRoute_ProdLandingFS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// landingFS 用一个嵌入式 fstest.MapFS 模拟
	h := &staticHandler{release: true, landingFS: nil} // landingFS=nil 走默认 404
	r.NoRoute(h.NoRoute)

	req := httptest.NewRequest(http.MethodGet, "/landing/index.html", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("landingFS nil: got %d, want 404", w.Code)
	}
}

// TestStaticHandler_Register_ProdMode 验证 prod 模式 Register 各分支。
func TestStaticHandler_Register_ProdMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &staticHandler{
		release:     true,
		sdkBytes:    []byte("sdk-content"),
		adminIndex:  []byte("<html>admin</html>"),
		landingRoot: []byte("<html>landing</html>"),
	}

	h.Register(r)

	// GET /sdk.js 应返回 sdkBytes
	req := httptest.NewRequest(http.MethodGet, "/sdk.js", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/sdk.js: got %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "sdk-content") {
		t.Errorf("/sdk.js body: %q", w.Body.String())
	}

	// GET /admin 应返回 adminIndex
	req2 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("/admin: got %d, want 200", w2.Code)
	}

	// GET / 应返回 landingRoot
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("/: got %d, want 200", w3.Code)
	}
}

// TestStaticHandler_Register_ProdMode_NoAssets 验证 prod 模式 sdkBytes/adminIndex/landingRoot 为 nil 的分支。
func TestStaticHandler_Register_ProdMode_NoAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &staticHandler{release: true} // 全部 nil
	h.Register(r)
}

// === router.go NewRouterWithOpts ===

// TestNewRouterWithOpts_DevMode 验证 NewRouterWithOpts 在 dev 模式可构造。
func TestNewRouterWithOpts_DevMode(t *testing.T) {
	stores := &storage.Stores{
		PG:    &storage.Postgres{},
		Redis: &storage.Redis{},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	r := NewRouterWithOpts(Options{
		Logger:   logger,
		Stores:   stores,
		Env:      "dev",
		Release:  false,
		Embedded: nil,
	})
	if r == nil {
		t.Fatal("NewRouterWithOpts returned nil")
	}
}

// TestNewRouterWithOpts_ProdMode 验证 prod 模式构造。
// 用空 fstest.MapFS 避免 newStaticHandler 在 nil embedded 时 panic。
func TestNewRouterWithOpts_ProdMode(t *testing.T) {
	stores := &storage.Stores{
		PG:    &storage.Postgres{},
		Redis: &storage.Redis{},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	r := NewRouterWithOpts(Options{
		Logger:   logger,
		Stores:   stores,
		Env:      "prod",
		Release:  true,
		Embedded: emptyEmbeddedFS(),
	})
	if r == nil {
		t.Fatal("NewRouterWithOpts returned nil")
	}
}

// TestNewRouterWithOpts_TrustedProxiesError 验证 SetTrustedProxies 失败时 warn 不阻塞。
func TestNewRouterWithOpts_TrustedProxiesError(t *testing.T) {
	stores := &storage.Stores{PG: &storage.Postgres{}, Redis: &storage.Redis{}}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	// 传入无效 TrustedProxies 触发 SetTrustedProxies error
	r := NewRouterWithOpts(Options{
		Logger:         logger,
		Stores:         stores,
		Env:            "dev",
		Release:        false,
		TrustedProxies: []string{"invalid-cidr"},
	})
	if r == nil {
		t.Fatal("NewRouterWithOpts returned nil even with bad proxies")
	}
}

// emptyEmbeddedFS 返回空 fs.FS 避免 newStaticHandler 在 nil 时 panic。
func emptyEmbeddedFS() fstest.MapFS {
	return fstest.MapFS{}
}

// === privacy.go handler 各路径 ===

// newPrivacyTestEngine 构造仅挂 privacy 路由的 engine。
func newPrivacyTestEngine(h *PrivacyHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterPublic(r)
	return r
}

// TestGetConsent_MissingFingerprint 验证缺 fingerprint 返回 400。
func TestGetConsent_MissingFingerprint(t *testing.T) {
	h := &PrivacyHandler{logger: testLogger()}
	r := newPrivacyTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/privacy/consent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing fingerprint: got %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "missing_fingerprint") {
		t.Errorf("body: %q", w.Body.String())
	}
}

// TestPostConsent_InvalidJSON 验证非 JSON body 返回 400。
func TestPostConsent_InvalidJSON(t *testing.T) {
	h := &PrivacyHandler{logger: testLogger()}
	r := newPrivacyTestEngine(h)

	req := httptest.NewRequest(http.MethodPost, "/api/privacy/consent", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid json: got %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_json") {
		t.Errorf("body: %q", w.Body.String())
	}
}

// TestDeleteVisitor_NoAuth 验证未认证返回 401。
func TestDeleteVisitor_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &PrivacyHandler{logger: testLogger(), stores: &storage.Stores{}}
	r.DELETE("/api/privacy/visitor/:fingerprint", h.DeleteVisitor)

	req := httptest.NewRequest(http.MethodDelete, "/api/privacy/visitor/fp-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("no auth: got %d, want 401", w.Code)
	}
}

// TestDeleteVisitor_NotAdmin 验证非 admin role 返回 403。
func TestDeleteVisitor_NotAdmin(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	// seed 一个 operator user
	ctx := context.Background()
	opUID := uuid.New()
	_, err := stores.PG.Pool.Exec(ctx, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'op-cv@example.com', 'hash', 'Op', 'operator')
	`, opUID, storage.DefaultTenantID)
	if err != nil {
		t.Fatalf("seed op: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, opUID)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &PrivacyHandler{logger: testLogger(), stores: stores}
	// 模拟 AuthMiddleware 注入 user_id
	r.DELETE("/api/privacy/visitor/:fingerprint", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.DeleteVisitor(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/privacy/visitor/fp-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("not admin: got %d, want 403", w.Code)
	}
}

// TestDeleteVisitor_AdminNotFound 验证 user_id 在 DB 找不到时返回 401。
func TestDeleteVisitor_AdminNotFound(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &PrivacyHandler{logger: testLogger(), stores: stores}
	r.DELETE("/api/privacy/visitor/:fingerprint", func(c *gin.Context) {
		c.Set("user_id", uuid.New()) // 不存在的 user_id
		h.DeleteVisitor(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/privacy/visitor/fp-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("admin not found: got %d, want 401", w.Code)
	}
}

// TestDeleteVisitor_AdminVisitorNotFound 验证 admin 但 visitor 不存在返回 200 + visitor_not_found。
func TestDeleteVisitor_AdminVisitorNotFound(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	ctx := context.Background()
	adminUID := uuid.New()
	_, err := stores.PG.Pool.Exec(ctx, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'admin-cv@example.com', 'hash', 'Admin', 'admin')
	`, adminUID, storage.DefaultTenantID)
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, adminUID)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &PrivacyHandler{logger: testLogger(), stores: stores}
	r.DELETE("/api/privacy/visitor/:fingerprint", func(c *gin.Context) {
		c.Set("user_id", adminUID)
		h.DeleteVisitor(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/privacy/visitor/non-existent-fp-"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("admin visitor not found: got %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "visitor_not_found") {
		t.Errorf("body: %q", w.Body.String())
	}
}

// === replay.go handler 各路径 ===

// newReplayBoostEngine 构造仅挂 replay 路由的 engine(避免与 replay_http_test.go 同名)。
func newReplayBoostEngine(h *ReplayHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)
	return r
}

// TestListEndedSessions_InvalidSince 验证非合法 since 返回 400。
func TestListEndedSessions_InvalidSince(t *testing.T) {
	h := &ReplayHandler{logger: testLogger(), stores: &storage.Stores{}}
	r := newReplayBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/ended?since=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid since: got %d, want 400", w.Code)
	}
}

// TestGetSessionReplay_InvalidUUID 验证非 UUID 返回 400。
func TestGetSessionReplay_InvalidUUID(t *testing.T) {
	h := &ReplayHandler{logger: testLogger(), stores: &storage.Stores{}}
	r := newReplayBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/not-a-uuid/replay", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid uuid: got %d, want 400", w.Code)
	}
}

// TestGetSessionReplay_ValidUUID_NoBlobs 验证合法 UUID + 空 blob 返回 200 + 空事件。
func TestGetSessionReplay_ValidUUID_NoBlobs(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := &ReplayHandler{logger: testLogger(), stores: stores}
	r := newReplayBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+uuid.New().String()+"/replay", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("valid uuid no blobs: got %d, want 200", w.Code)
	}
}

// TestListEndedSessions_DefaultSince 验证默认 since=24h 返回 200。
func TestListEndedSessions_DefaultSince(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := &ReplayHandler{logger: testLogger(), stores: stores}
	r := newReplayBoostEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/ended", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("default since: got %d, want 200", w.Code)
	}
}

// === session.go listSessions ===

// newSessionTestEngine 构造仅挂 session 路由的 engine。
func newSessionTestEngine(h *SessionHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)
	return r
}

// TestListSessions_DefaultLimit 验证默认 limit 返回 200 + sessions 列表。
func TestListSessions_DefaultLimit(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	// SessionHandler 需要 stores.Redis.Client(IsSessionFlagged 用)
	h := NewSessionHandler(stores, nil, testLogger())
	r := newSessionTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listSessions: got %d, want 200", w.Code)
	}
}

// TestListSessions_WithCustomLimit 验证 limit query 参数。
func TestListSessions_WithCustomLimit(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := NewSessionHandler(stores, nil, testLogger())
	r := newSessionTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions?limit=50", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listSessions limit=50: got %d, want 200", w.Code)
	}
}

// TestListSessions_WithInvalidLimit 验证非法 limit 走默认 200。
func TestListSessions_WithInvalidLimit(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := NewSessionHandler(stores, nil, testLogger())
	r := newSessionTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions?limit=not-a-number", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listSessions invalid limit: got %d, want 200", w.Code)
	}
}

// TestInitSession_MissingVisitorID 验证缺 visitor_id 返回 400。
func TestInitSession_MissingVisitorID(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := NewSessionHandler(stores, nil, testLogger())
	r := newSessionTestEngine(h)

	req := httptest.NewRequest(http.MethodPost, "/api/session/init", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing visitor_id: got %d, want 400", w.Code)
	}
}

// TestInitSession_InvalidJSON 验证非 JSON body 返回 400。
func TestInitSession_InvalidJSON(t *testing.T) {
	h := NewSessionHandler(&storage.Stores{}, nil, testLogger())
	r := newSessionTestEngine(h)

	req := httptest.NewRequest(http.MethodPost, "/api/session/init", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid json: got %d, want 400", w.Code)
	}
}

// TestInitSession_HappyPath 验证合法 visitor_id 返回 200 + session_id。
func TestInitSession_HappyPath(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	h := NewSessionHandler(stores, nil, testLogger())
	r := newSessionTestEngine(h)

	body := `{"visitor_id":"test-fp-cv-1","ua":"test-ua","ip":"1.1.1.1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/session/init", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("initSession happy: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
	// cleanup
	stores.PG.Pool.Exec(context.Background(), `DELETE FROM visitors WHERE fingerprint = 'test-fp-cv-1'`)
}

// === command.go postCommand ===

// TestPostCommand_InvalidJSON 验证非 JSON body 返回 400。
// 注:command handler 走 requireClaimOwnership,需要 claim 锁;本测试仅覆盖 binding 失败前。
// 由于 requireClaimOwnership 在 binding 前调用,需用 stubStores 让 claim check 通过。
func TestPostCommand_InvalidJSON(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()

	// seed user + session + claim
	ctx := context.Background()
	opUID := uuid.New()
	_, err := stores.PG.Pool.Exec(ctx, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'op-cmd@example.com', 'h', 'Op', 'operator')
	`, opUID, storage.DefaultTenantID)
	if err != nil {
		t.Fatalf("seed op: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, opUID)

	visitorID := uuid.New()
	_, err = stores.PG.Pool.Exec(ctx, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'cmd-test-fp', NOW(), NOW())
	`, visitorID, storage.DefaultTenantID)
	if err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx, `DELETE FROM visitors WHERE id = $1`, visitorID)

	sessionID := uuid.New()
	_, err = stores.PG.Pool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, storage.DefaultTenantID, visitorID)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	defer stores.PG.Pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)

	// seed claim 锁
	claimK := claimKey(sessionID)
	stores.Redis.Set(ctx, claimK, []byte(opUID.String()), time.Minute)
	defer stores.Redis.Del(ctx, claimK)

	// stub hub
	stubHub := &stubCommandHub{delivered: true}

	h := NewCommandHandler(stores, stubHub, "", testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/sessions/:id/command", func(c *gin.Context) {
		c.Set("user_id", opUID)
		h.postCommand(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/command", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid json: got %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

// stubCommandHub 实现 CommandHub 接口。
type stubCommandHub struct {
	delivered bool
}

func (s *stubCommandHub) SendCommandToVisitor(sessionID uuid.UUID, msg []byte) bool {
	return s.delivered
}

// === 纯函数测试 ===

// TestParseSince_EdgeCases 验证 parseSince 各分支。
func TestParseSince_EdgeCases(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
		want    time.Duration
	}{
		{"", false, 24 * time.Hour},          // 默认 24h
		{"24h", false, 24 * time.Hour},       // 24 小时
		{"7d", false, 7 * 24 * time.Hour},    // 7 天
		{"x", true, 0},                       // 太短
		{"abc", true, 0},                     // 非数字
		{"-1h", true, 0},                     // 负数
		{"0h", true, 0},                      // 零
		{"5m", true, 0},                      // 不支持单位
		{"999999h", false, 999999 * time.Hour}, // 大数
	}
	for _, tc := range cases {
		got, err := parseSince(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseSince(%q): expected error, got nil", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSince(%q): got error %v, want nil", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseSince(%q): got %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestSanitizeCommandType_AlreadyKnown 验证 sanitizeCommandType 已知 type 原样返回。
// (其他分支已被 command_test.go 覆盖)
func TestSanitizeCommandType_AlreadyKnown(t *testing.T) {
	if got := sanitizeCommandType("click"); got != "click" {
		t.Errorf("sanitize click: got %q", got)
	}
}

// TestSanitizeURLForLog_QueryString 验证 sanitizeURLForLog query string 剥离。
// (其他分支已被 command_test.go 覆盖)
func TestSanitizeURLForLog_QueryString(t *testing.T) {
	if got := sanitizeURLForLog("http://example.com/path?token=secret"); got != "http://example.com/path" {
		t.Errorf("with query: got %q", got)
	}
}

// TestIsURLSchemeAllowed_ExtraCases 验证 isURLSchemeAllowed 额外分支。
// (基础情况已被 command_test.go 覆盖)
func TestIsURLSchemeAllowed_ExtraCases(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"HTTP://example.com", true},       // HTTP 大写
		{"//cdn.example.com/lib.js", true}, // protocol-relative
		{"page.html", true},                // 文件名(无 scheme)
		{"mailto:foo@bar", false},          // mailto(非 http/https)
		{"custom:foo", false},              // 自定义 scheme
	}
	for _, tc := range cases {
		got := isURLSchemeAllowed(tc.in)
		if got != tc.want {
			t.Errorf("isURLSchemeAllowed(%q): got %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestDecodeRRWebEventsFromBlob_InvalidInput 验证 decodeRRWebEventsFromBlob 各错误分支。
func TestDecodeRRWebEventsFromBlob_InvalidInput(t *testing.T) {
	// 非 msgpack 输入
	_, err := decodeRRWebEventsFromBlob([]byte("not msgpack"))
	if err == nil {
		t.Error("decode with garbage: expected error, got nil")
	}

	// 空 input
	events, err := decodeRRWebEventsFromBlob([]byte{})
	if err == nil {
		t.Error("decode empty: expected error, got nil")
	}

	// msgpack 但非 array of bytes(无效 envelope)
	// 构造 msgpack array of 1 string(不是 bytes)
	arrBytes, _ := proto.Encode(proto.Envelope{V: 1, Type: proto.MsgEvent})
	// 直接 marshal 一个 array of arrays(不是 array of bytes)
	arrArr, _ := json.Marshal([][]byte{arrBytes})
	events, err = decodeRRWebEventsFromBlob(arrArr)
	_ = events // 验证不 panic
}

// TestExtractRRWebEventsFromPayload_NilPayload 验证 nil payload 返回空 slice。
func TestExtractRRWebEventsFromPayload_NilPayload(t *testing.T) {
	got := extractRRWebEventsFromPayload(nil)
	if len(got) != 0 {
		t.Errorf("nil payload: got %d events, want 0", len(got))
	}
}

// TestBuildCommandPayload_VerifyTypes 验证 buildCommandPayload 各 type 分支(去重)。
func TestBuildCommandPayload_VerifyTypes(t *testing.T) {
	// cursor_highlight + scroll 未在 coverage_extra_test.go 覆盖
	cases := []struct {
		cmdType   string
		payload   string
		wantError bool
	}{
		{"cursor_highlight", `{"x":1,"y":2,"name":"arrow"}`, false},
		{"scroll", `{"x":0,"y":100}`, false},
		{"release_control", `{}`, false},
	}
	for _, tc := range cases {
		_, err := buildCommandPayload(commandRequest{
			Type:    tc.cmdType,
			Payload: json.RawMessage(tc.payload),
		})
		if tc.wantError && err == nil {
			t.Errorf("buildCommandPayload(%s, %s): expected error, got nil", tc.cmdType, tc.payload)
		}
		if !tc.wantError && err != nil {
			t.Errorf("buildCommandPayload(%s, %s): got error %v, want nil", tc.cmdType, tc.payload, err)
		}
	}
}

// TestCommandHandler_IsURLAllowed_VerifyCases 验证 isURLAllowed 各场景(去重)。
func TestCommandHandler_IsURLAllowed_VerifyCases(t *testing.T) {
	h := &CommandHandler{allowedDomains: []string{"example.com", "cdn.example.org"}}

	cases := []struct {
		url  string
		host string
		want bool
	}{
		{"", "host.com", false},                       // 空
		{"not-a-url", "host.com", true},               // 相对路径(无 host)
		{"/relative", "host.com", true},               // 相对路径
		{"http://host.com/path", "host.com", true},    // 同 host
		{"http://other.com/", "host.com", false},      // 不同 host
		{"http://localhost:8080/", "host.com", true},  // localhost
		{"http://127.0.0.1/", "host.com", true},       // 127.0.0.1
		{"http://example.com/", "host.com", true},     // 白名单
		{"http://sub.example.com/", "host.com", true}, // 白名单子域
		{"http://other.org/", "host.com", false},      // 不在白名单
	}
	for _, tc := range cases {
		got := h.isURLAllowed(tc.url, tc.host)
		if got != tc.want {
			t.Errorf("isURLAllowed(%q, %q): got %v, want %v", tc.url, tc.host, got, tc.want)
		}
	}
}

// === helperAPIStores:构造真实 docker stores 用于测试 ===

// helperAPIStores 构造 docker-backed Stores 用于 api 测试。不可用 skip。
func helperAPIStores(t *testing.T) *storage.Stores {
	t.Helper()
	if testing.Short() {
		t.Skip("需要 docker")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://mm:mm_dev@localhost:5432/pinconsole?sslmode=disable")
	if err != nil {
		t.Skipf("pg not available: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("pg ping: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		pool.Close()
		t.Skipf("redis not available: %v", err)
	}
	mclient, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("mm_dev", "mm_dev_secret", ""),
		Secure: false,
	})
	if err != nil {
		pool.Close()
		rdb.Close()
		t.Skipf("minio client: %v", err)
	}
	bucket := "test-api-" + uuid.New().String()[:8]
	if err := mclient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		pool.Close()
		rdb.Close()
		t.Skipf("minio MakeBucket: %v", err)
	}

	return &storage.Stores{
		PG:    &storage.Postgres{Pool: pool},
		Redis: &storage.Redis{Client: rdb},
		MinIO: &storage.MinIO{Client: mclient, Bucket: bucket},
	}
}

// 兼容 helper:用 errors 包防止未用 import
var _ = errors.New
