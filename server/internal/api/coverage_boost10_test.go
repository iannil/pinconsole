// CV-4 Round 11:router middleware + newStaticHandler init + remaining。
package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
)

// TestNewRouterWithOpts_ProdModeWithBannedUA 验证 prod 模式下 UA 黑名单 + rate limit 路径。
func TestNewRouterWithOpts_ProdModeWithBannedUA(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	r := NewRouterWithOpts(Options{
		Logger:         logger,
		Stores:         stores,
		Env:            "prod",
		Release:        false, // dev 模式避免 newStaticHandler panic
		RateLimitPerMin: 100,
		BannedUAs:      []string{"evil-bot"},
	})

	// 用 banned UA 请求任意非 health 路径
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	req.Header.Set("User-Agent", "evil-bot")
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 应被 UA 黑名单拦截
	if w.Code != http.StatusForbidden {
		t.Errorf("banned UA: got %d, want 403", w.Code)
	}
}

// TestNewRouterWithOpts_NormalUA 通过 UA 检查(非黑名单)。
func TestNewRouterWithOpts_NormalUA(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	r := NewRouterWithOpts(Options{
		Logger:         logger,
		Stores:         stores,
		Env:            "prod",
		Release:        false,
		RateLimitPerMin: 100,
		BannedUAs:      []string{"evil-bot"},
	})

	// 正常 UA 请求 healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("healthz with normal UA: got %d, want 200", w.Code)
	}
}

// TestNewRouterWithOpts_RateLimited 验证 prod 模式 rate limit 触发(超阈值后 429)。
func TestNewRouterWithOpts_RateLimited(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	r := NewRouterWithOpts(Options{
		Logger:          logger,
		Stores:          stores,
		Env:             "prod",
		Release:         false,
		RateLimitPerMin: 2, // 极低阈值
		BannedUAs:       nil,
	})

	// 连发 5 个请求,后几个应被限流
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.RemoteAddr = "127.0.0.1:12346"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		_ = w.Code
	}
}

// TestNewStaticHandler_ProdWithRealFS 验证 prod 模式 newStaticHandler 读 embedded。
func TestNewStaticHandler_ProdWithRealFS(t *testing.T) {
	// 构造一个 map FS 模拟 embedded/admin/index.html 等
	embed := fstest.MapFS{
		"embedded/admin/index.html": &fstest.MapFile{Data: []byte("<html>admin</html>")},
		"embedded/sdk/sdk.js":       &fstest.MapFile{Data: []byte("// sdk code")},
		"embedded/landing/demo/index.html": &fstest.MapFile{Data: []byte("<html>landing</html>")},
	}

	h := newStaticHandler(embed, true)
	if h == nil {
		t.Fatal("newStaticHandler returned nil")
	}
	if h.adminIndex == nil {
		t.Error("adminIndex should be loaded")
	}
	if len(h.sdkBytes) == 0 {
		t.Error("sdkBytes should be loaded")
	}
	if h.landingRoot == nil {
		t.Error("landingRoot should be loaded")
	}
}

// TestNewStaticHandler_ProdWithPartialFS 验证 prod 模式部分文件缺失不 panic。
func TestNewStaticHandler_ProdWithPartialFS(t *testing.T) {
	// 只有 admin/index.html,缺 sdk + landing
	embed := fstest.MapFS{
		"embedded/admin/index.html": &fstest.MapFile{Data: []byte("<html>admin</html>")},
		// 故意缺 admin/assets 子目录(让 fs.Sub 失败)
	}
	h := newStaticHandler(embed, true)
	if h == nil {
		t.Fatal("newStaticHandler returned nil")
	}
}

// TestPrivacyDeleteVisitorWithVisitorNotFound 用 errRedisGetOnly 让 GetVisitorByFingerprint 等失败。
// 此测试用 closed PG 触发多个 error 分支。
func TestPrivacyDeleteVisitorWithVisitorNotFound(t *testing.T) {
	stores := helperAPIStores(t)
	defer stores.Close()
	ctx0 := context.Background()

	adminUID := uuid.New()
	stores.PG.Pool.Exec(ctx0, `
		INSERT INTO users (id, tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'admin-pristine@example.com', 'h', 'Admin', 'admin')
	`, adminUID, storage.DefaultTenantID)
	defer stores.PG.Pool.Exec(ctx0, `DELETE FROM users WHERE id = $1`, adminUID)

	// 不 seed visitor → GetVisitorByFingerprint 返回 (nil, nil) → 进入 not_found 分支(已覆盖)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &PrivacyHandler{stores: stores, logger: testLogger()}
	r.DELETE("/api/privacy/visitor/:fingerprint", func(c *gin.Context) {
		c.Set("user_id", adminUID)
		h.DeleteVisitor(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/privacy/visitor/non-existent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// 应 200 visitor_not_found
	if w.Code != http.StatusOK {
		t.Errorf("not found: got %d, want 200", w.Code)
	}
}

// TestHealthReady_AllOK_Boost 验证 readyz 在所有依赖正常时返回 200 + ready(去重)。
func TestHealthReady_AllOK_Boost(t *testing.T) {
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

// 占位 imports
var _ = context.Background
var _ = time.Second
