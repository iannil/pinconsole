// 1ai-d 测试:me + logout happy path 行为级测试(1ai-c 续)。
//
// 复用 1ai-c 既有的 mockUserRepo + mockRedisStore,补:
//   - me happy path + user-not-found
//   - logout 验证 Redis.Del 被调(session 失效副作用)
package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
)

// TestMe_Success_Returns200_Body — 注入 user_id + mock userRepo 返 user →
// 200 + meResponse body(ID/Email/DisplayName/Role)。
func TestMe_Success_Returns200_Body(t *testing.T) {
	uid := uuid.New()
	user := &storage.User{
		ID:          uid,
		Email:       "1aid-me@example.com",
		DisplayName: "Me Test",
		Role:        "admin",
	}
	mockUsers := &mockUserRepo{
		byID: map[uuid.UUID]*storage.User{uid: user},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	c.Set("user_id", uid) // 模拟 AuthMiddleware 注入

	h := &AuthHandler{userRepo: mockUsers, logger: testLogger()}
	h.me(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		uid.String(),
		user.Email,
		user.DisplayName,
		`"role":"admin"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body 缺 %q: %s", want, body)
		}
	}
	if mockUsers.idCalls != 1 {
		t.Errorf("GetUserByID calls = %d, want 1", mockUsers.idCalls)
	}
}

// TestMe_UserNotFound_Returns401 — mock userRepo 返 error → 401 user_not_found。
//
// 场景:用户被 admin 删除但 cookie 仍有效,下次请求 me 应返 401(强制重新登录)。
func TestMe_UserNotFound_Returns401(t *testing.T) {
	uid := uuid.New()
	mockUsers := &mockUserRepo{
		byID:    map[uuid.UUID]*storage.User{}, // 空
		byIDErr: fmt.Errorf("pgx.ErrNoRows: user deleted"),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	c.Set("user_id", uid)

	h := &AuthHandler{userRepo: mockUsers, logger: testLogger()}
	h.me(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "user_not_found") {
		t.Errorf("body = %q, want contains 'user_not_found'", w.Body.String())
	}
}

// TestLogout_DeletesRedisSession — 带 cookie 的 logout 应:
//  1. Redis.Del 被调 1 次(删除 session:sessionID)
//  2. 返 200 + 清 cookie(MaxAge<0)
//
// 此前 1ag 的 TestLogout_ClearsCookie_AndReturns200 未注入 redis,
// 无法断言 Del 副作用(防 logout 不删 Redis 的回归)。
func TestLogout_DeletesRedisSession(t *testing.T) {
	mockRedis := newMockRedisStore()
	// 预设 session key,模拟 login 后的状态
	sessionID := "test-session-1aid-abc123"
	sessionKey := sessionRedisKey(sessionID)
	mockRedis.data[sessionKey] = []byte(uuid.New().String())

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	// 加 cookie(模拟已登录)
	c.Request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})

	h := &AuthHandler{redis: mockRedis, logger: testLogger(), secureCookie: false}
	h.logout(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	// Del 必须被调 1 次,且参数是 sessionRedisKey(sessionID)
	if mockRedis.delCalls != 1 {
		t.Errorf("Redis.Del calls = %d, want 1 (清 session)", mockRedis.delCalls)
	}
	if _, stillExists := mockRedis.data[sessionKey]; stillExists {
		t.Errorf("session key 未被删:仍存在 data map")
	}
	// Set-Cookie 清空 cookie(MaxAge<0 / value 空)
	setCookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "mm_session=") {
		t.Errorf("Set-Cookie 缺 mm_session: %s", setCookie)
	}
}

// TestLogout_NoCookie_NoRedisDel — 无 cookie 的 logout 不应调 Del(no-op 路径)。
//
// 覆盖 logout.go:207 `if err == nil && sessionID != ""` 的 false 分支。
func TestLogout_NoCookie_NoRedisDel(t *testing.T) {
	mockRedis := newMockRedisStore()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	// 故意无 cookie

	h := &AuthHandler{redis: mockRedis, logger: testLogger()}
	h.logout(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if mockRedis.delCalls != 0 {
		t.Errorf("Redis.Del calls = %d, want 0 (无 cookie 不应调 Del)", mockRedis.delCalls)
	}
}
