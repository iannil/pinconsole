// 1k 测试：AuthMiddleware + dev bypass compile-tag 隔离。
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// stubGetSession 返回固定的 user_id bytes 模拟 Redis 命中。
func stubGetSession(userUUID string) func(ctx context.Context, key string) ([]byte, error) {
	return func(ctx context.Context, key string) ([]byte, error) {
		if key == "auth:session:valid-session-id" {
			return []byte(userUUID), nil
		}
		return nil, nil
	}
}

func newTestRouter(mw gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", mw, func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"ok": true, "user_id": uid})
	})
	return r
}

func TestAuthMiddleware_NoCookie_Returns401(t *testing.T) {
	mw := AuthMiddleware(stubGetSession(uuid.New().String()), false) // prod mode
	r := newTestRouter(mw)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (no cookie)", w.Code)
	}
}

func TestAuthMiddleware_InvalidSession_Returns401(t *testing.T) {
	mw := AuthMiddleware(stubGetSession(uuid.New().String()), false) // prod mode
	r := newTestRouter(mw)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "invalid-session"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (invalid session)", w.Code)
	}
}

func TestAuthMiddleware_ValidSession_SetsUserID(t *testing.T) {
	uid := uuid.New()
	mw := AuthMiddleware(stubGetSession(uid.String()), false) // prod mode
	r := newTestRouter(mw)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "valid-session-id"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	// 响应体应包含正确的 user_id
	body := w.Body.String()
	if !contains(body, uid.String()) {
		t.Errorf("response body should contain user_id %s, got: %s", uid, body)
	}
}

func TestAuthMiddleware_DevMode_AppliesBypassWhenCompiledIn(t *testing.T) {
	// devMode=true;在 dev build 下应绕过 (tryDevBypass 返回 true)
	// 在 release build 下应仍要求 cookie (tryDevBypass 返回 false)
	mw := AuthMiddleware(stubGetSession(uuid.New().String()), true)
	r := newTestRouter(mw)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if isReleaseBuild {
		// release build: dev bypass 不应生效,无 cookie 应 401
		if w.Code != http.StatusUnauthorized {
			t.Errorf("release build: status = %d, want 401 (dev bypass not compiled in)", w.Code)
		}
	} else {
		// dev build: dev bypass 应生效,无 cookie 也 200
		if w.Code != http.StatusOK {
			t.Errorf("dev build: status = %d, want 200 (dev bypass)", w.Code)
		}
	}
}

// contains 简化字符串包含检查。
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
