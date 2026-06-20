// 1ag 测试:auth.go HTTP 入口行为级测试。
//
// 补 1x/1ac/1ae 没覆盖的 handler 层路径:
//   - login 无效 JSON / 缺字段 → 400(binding 失败,不触达 stores)
//   - login 已 throttle 锁定 → 429 + Retry-After(真 Redis seed + 真 gin engine)
//   - me 无 user_id 注入 → 401(不触达 stores)
//   - logout 始终清 cookie + 200
//
// 模式:r.POST(...) + r.ServeHTTP(recorder, req),走 gin 真路由 + 真 binding,
// 比 CreateTestContext 直调 handler 更接近生产(覆盖 router 级 binding/middleware)。
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iannil/pinconsole/internal/storage"
)

// newAuthTestEngine 构造仅挂 auth 路由的 gin engine,不含 middleware。
// 用于隔离测试 handler 路由 + binding,不引入 router.go 的 stores 依赖。
func newAuthTestEngine(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)
	return r
}

// TestLogin_InvalidJSON_Returns400 — POST 非法 JSON body 必返 400 invalid_json。
//
// 不触达 stores(binding 在 stores 调用前失败),所以无需 mock PG/Redis。
func TestLogin_InvalidJSON_Returns400(t *testing.T) {
	h := &AuthHandler{logger: testLogger()}
	r := newAuthTestEngine(h)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_json") {
		t.Errorf("body = %q, want contains 'invalid_json'", w.Body.String())
	}
}

// TestLogin_MissingFields_Returns400 — POST 缺 email 或 password 必返 400(binding required)。
//
// binding:"required" 标签在 gin binding 阶段拒绝,不触达 stores。
func TestLogin_MissingFields_Returns400(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing_password", `{"email":"a@b.com"}`},
		{"missing_email", `{"password":"secret"}`},
		{"empty_email", `{"email":"","password":"secret"}`},
		{"empty_password", `{"email":"a@b.com","password":""}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &AuthHandler{logger: testLogger()}
			r := newAuthTestEngine(h)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 (case %s)", w.Code, tc.name)
			}
		})
	}
}

// TestLogin_Locked_Returns429_WithRetryAfter — 真 Redis seed 计数到阈值,
// POST login 必返 429 + Retry-After header + too_many_attempts body。
//
// 覆盖 1x throttle 在 HTTP 入口的实际拦截(此前 1x 仅测纯函数 recordLoginFailure/checkLoginThrottle)。
// 需要 Redis,不可用时 skip。
func TestLogin_Locked_Returns429_WithRetryAfter(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	email := "1ag-locked@example.com"
	ip := "10.99.99.42"
	key := loginThrottleKey(email, ip)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	defer rdb.Del(ctx, key)

	h := &AuthHandler{
		redis:        &storage.Redis{Client: rdb},
		logger:       testLogger(),
		secureCookie: false,
	}
	// seed:INCR 到 loginMaxAttempts 触发锁定
	for i := 0; i < loginMaxAttempts; i++ {
		h.recordLoginFailure(ctx, key)
	}

	r := newAuthTestEngine(h)
	// gin ClientIP 默认从 RemoteAddr 取,此处 IP 仅用于 throttle key 组合;
	// 测试通过直接 seed 该 email+IP 维度,gin engine 路由层不感知 IP。
	body, _ := json.Marshal(map[string]string{"email": email, "password": "any"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// X-Forwarded-For 让 ClientIP 取到 seed 的 IP(默认 gin 信任所有反代;
	// 测试环境无 TrustedProxies 配置,RemoteAddr 优先,此处设 RemoteAddr)
	req.RemoteAddr = ip + ":1234"
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429 (login should be locked)", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got == "" {
		t.Error("Retry-After header missing")
	}
	if !strings.Contains(w.Body.String(), "too_many_attempts") {
		t.Errorf("body = %q, want contains 'too_many_attempts'", w.Body.String())
	}
}

// TestMe_NoUserIDContext_Returns401 — me handler 不存在 user_id 时必返 401 not_authenticated。
//
// 生产环境由 AuthMiddleware 注入 user_id;此处直接调 handler 模拟未挂 middleware 的场景。
// 不触达 stores(user_id 检查在前)。
func TestMe_NoUserIDContext_Returns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)

	h := &AuthHandler{logger: testLogger()}
	h.me(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not_authenticated") {
		t.Errorf("body = %q, want contains 'not_authenticated'", w.Body.String())
	}
}

// TestLogout_ClearsCookie_AndReturns200 — logout 始终清 cookie + 200,
// 无论 Redis 是否可用(代码用 _ = 容错)。
//
// 直接调 handler(不挂路由),验证 Set-Cookie MaxAge<0 + status 200。
func TestLogout_ClearsCookie_AndReturns200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)

	h := &AuthHandler{logger: testLogger(), secureCookie: false}
	h.logout(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	setCookie := w.Header().Get("Set-Cookie")
	if setCookie == "" {
		t.Fatal("Set-Cookie header missing")
	}
	// Max-Age<0 表示删除 cookie gin 用 "Max-Age=0" 或省略;
	// 关键:cookie value 必须为空(等效删除)
	if !strings.Contains(setCookie, "mm_session=") {
		t.Errorf("Set-Cookie 不含 mm_session: %s", setCookie)
	}
	// 验证 cookie 被清空(value 段为空或立即过期)
	if strings.Contains(setCookie, "mm_session=;") || strings.Contains(setCookie, "mm_session= ") {
		// OK: value empty
	} else {
		// 兜底:必须含 Max-Age=0 或 Max-Age=-1(gin logout 用 SetCookie maxAge=-1)
		if !strings.Contains(setCookie, "Max-Age=0") && !strings.Contains(setCookie, "Max-Age=-1") {
			t.Errorf("Set-Cookie 未清空 cookie(value 非空 + 无 Max-Age<=0): %s", setCookie)
		}
	}
}
