// 1ah 测试:chat.go HTTP 入口行为级测试。
//
// 补 1g/1k 未覆盖的 chat handler 路径:
//   - Register 路由 wire-up
//   - listMessages UUID 拒绝路径(零依赖,uuid.Parse 在 PG 调用前)
//
// postMessage 第一行 requireClaimOwnership 需 Redis+PG,留 backlog(见 1ah spec)。
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// newChatTestEngine 构造仅挂 chat 路由的 gin engine。
// 注:chat Register 不需 CommandHub(只在 postMessage 用),listMessages 不触达 hub。
func newChatTestEngine(h *ChatHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)
	return r
}

// TestChatRegister_Routes — 验证 Register 注册 2 个路由。
func TestChatRegister_Routes(t *testing.T) {
	h := &ChatHandler{logger: testLogger()}
	r := newChatTestEngine(h)

	wantRoutes := map[string]bool{
		"GET /api/sessions/:id/messages":  false,
		"POST /api/sessions/:id/messages": false,
	}
	for _, ri := range r.Routes() {
		k := ri.Method + " " + ri.Path
		if _, want := wantRoutes[k]; want {
			wantRoutes[k] = true
		}
	}
	for route, found := range wantRoutes {
		if !found {
			t.Errorf("chat route missing: %s", route)
		}
	}
}

// TestListMessages_InvalidUUID_Returns400 — listMessages 非 UUID 必返 400。
//
// uuid.Parse 在 PG 调用前(chat.go:57-61)。listMessages 此前完全 0% 覆盖。
func TestListMessages_InvalidUUID_Returns400(t *testing.T) {
	h := &ChatHandler{logger: testLogger()}
	r := newChatTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/not-a-uuid/messages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_session_id") {
		t.Errorf("body = %q, want contains 'invalid_session_id'", w.Body.String())
	}
}

// TestListMessages_ValidUUIDPassesParsing — 合法 UUID 通过解析阶段。
//
// stores 未注入会在 PG 调用时 panic;用 defer recover 兜底,
// 关键断言:不返 400(uuid.Parse 误拒合法 UUID 是回归)。
func TestListMessages_ValidUUIDPassesParsing(t *testing.T) {
	h := &ChatHandler{logger: testLogger()}
	r := newChatTestEngine(h)
	defer func() { _ = recover() }()

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/550e8400-e29b-41d4-a716-446655440000/messages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest {
		t.Errorf("status = 400 for valid UUID — uuid.Parse 误拒合法 UUID")
	}
}
