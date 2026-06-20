// 1ac + 1ac-final 测试:WS 同源校验 + operatorWS auth(审计 T0-1h-6 + T0-1h-2)。
//
// T0-1h-6:websocket.Accept 必须设 InsecureSkipVerify: false(同源校验),
//
//	防止跨域 WS 滥用(CSWSH)。
//
// T0-1h-2(1ac-final 已修复):operatorWS 此前完全无认证检查。
//
//	修复:handler 内调 authenticateOperatorWS,校验 cookie session,失败返回 401。
//	测试:无 cookie / 无效 session / 有效 session / devMode bypass / 接线源码契约。
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
)

// TestWS_VisitorOriginCheck_Enabled — T0-1h-6:
// visitorWS 的 websocket.Accept 必须设 InsecureSkipVerify: false。
// 否则跨域连接可读取 visitor 录像数据。
func TestWS_VisitorOriginCheck_Enabled(t *testing.T) {
	src, err := os.ReadFile("ws.go")
	if err != nil {
		t.Fatalf("read ws.go: %v", err)
	}
	body := string(src)

	// 找 visitorWS 函数体内的 Accept 调用
	idx := strings.Index(body, "func (h *WSHandler) visitorWS")
	if idx < 0 {
		t.Fatal("找不到 visitorWS 函数")
	}
	end := strings.Index(body[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(body) - idx - 1
	}
	fnBody := body[idx : idx+1+end]

	if !strings.Contains(fnBody, "InsecureSkipVerify: false") {
		t.Errorf("visitorWS 缺失 InsecureSkipVerify: false — 同源校验破坏(CSWSH 风险):\n%s", fnBody)
	}
	// 反模式:InsecureSkipVerify: true 等同禁用同源校验
	if strings.Contains(fnBody, "InsecureSkipVerify: true") {
		t.Errorf("visitorWS 设了 InsecureSkipVerify: true — 跨域 WS 滥用风险")
	}
}

// TestWS_OperatorOriginCheck_Enabled — T0-1h-6 operator 侧:
// operatorWS 也必须设 InsecureSkipVerify: false。
func TestWS_OperatorOriginCheck_Enabled(t *testing.T) {
	src, err := os.ReadFile("ws.go")
	if err != nil {
		t.Fatalf("read ws.go: %v", err)
	}
	body := string(src)

	idx := strings.Index(body, "func (h *WSHandler) operatorWS")
	if idx < 0 {
		t.Fatal("找不到 operatorWS 函数")
	}
	end := strings.Index(body[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(body) - idx - 1
	}
	fnBody := body[idx : idx+1+end]

	if !strings.Contains(fnBody, "InsecureSkipVerify: false") {
		t.Errorf("operatorWS 缺失 InsecureSkipVerify: false — 同源校验破坏")
	}
	if strings.Contains(fnBody, "InsecureSkipVerify: true") {
		t.Errorf("operatorWS 设了 InsecureSkipVerify: true — 跨域 WS 滥用风险")
	}
}

// TestWS_AuthenticateOperatorWS_NoCookie_Returns401 — T0-1h-2 修复验证:
// 无 cookie 的请求应被拒绝(401 no_session),authenticateOperatorWS 返回 ok=false。
func TestWS_AuthenticateOperatorWS_NoCookie_Returns401(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ws/operator", nil)
	// 不加 cookie

	stores := &storage.Redis{Client: rdb}
	_, ok := authenticateOperatorWS(c, stores, false /* prod mode */)
	if ok {
		t.Errorf("ok=true want false (no cookie should reject)")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401 (no_session)", w.Code)
	}
	if !contains(w.Body.String(), "no_session") {
		t.Errorf("body should contain no_session, got: %s", w.Body.String())
	}
}

// TestWS_AuthenticateOperatorWS_InvalidSession_Returns401 —
// 有 cookie 但 Redis 中 session 不存在/已过期 → 401 invalid_session。
func TestWS_AuthenticateOperatorWS_InvalidSession_Returns401(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ws/operator", nil)
	c.Request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "nonexistent-session-id"})

	stores := &storage.Redis{Client: rdb}
	_, ok := authenticateOperatorWS(c, stores, false)
	if ok {
		t.Errorf("ok=true want false (invalid session should reject)")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401 (invalid_session)", w.Code)
	}
	if !contains(w.Body.String(), "invalid_session") {
		t.Errorf("body should contain invalid_session, got: %s", w.Body.String())
	}
}

// TestWS_AuthenticateOperatorWS_ValidSession_OK —
// 有效 cookie + Redis 命中 → 返回 (user_id, true)。
func TestWS_AuthenticateOperatorWS_ValidSession_OK(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	userID := uuid.New()
	sessionID := "test-ws-auth-valid-session-1ac-final"
	ctx := context.Background()
	defer rdb.Del(ctx, sessionRedisKey(sessionID))
	if err := rdb.Set(ctx, sessionRedisKey(sessionID), userID.String(), 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ws/operator", nil)
	c.Request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})

	stores := &storage.Redis{Client: rdb}
	gotUID, ok := authenticateOperatorWS(c, stores, false)
	if !ok {
		t.Errorf("ok=false want true (valid session)")
	}
	if gotUID != userID {
		t.Errorf("gotUID=%s want %s", gotUID, userID)
	}
	// user_id 应已写入 gin ctx(供下游 handler 用)
	ctxUID, exists := c.Get("user_id")
	if !exists {
		t.Errorf("c.Get(user_id) not exists after auth")
	}
	if uid, _ := ctxUID.(uuid.UUID); uid != userID {
		t.Errorf("ctx user_id=%v want %s", ctxUID, userID)
	}
}

// TestWS_AuthenticateOperatorWS_DevModeBypass —
// devMode=true 且 dev build → 旁路(返回 uuid.Nil, true)。
// 注:release build 下 tryDevBypass 恒 false,此测试在 release build 下不 bypass。
func TestWS_AuthenticateOperatorWS_DevModeBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ws/operator", nil)
	// 不加 cookie,但 devMode=true

	// stores 可以是 nil,因 devMode bypass 不会触达 Redis
	uid, ok := authenticateOperatorWS(c, nil, true /* devMode */)
	if !isReleaseBuild {
		// dev build: bypass 生效
		if !ok {
			t.Errorf("dev build + devMode: ok=false want true (bypass)")
		}
		if uid != uuid.Nil {
			t.Errorf("dev build + devMode: uid=%s want uuid.Nil", uid)
		}
	} else {
		// release build: bypass 不生效,无 cookie 应 401
		if ok {
			t.Errorf("release build: ok=true want false (no bypass, no cookie)")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("release build: status=%d want 401", w.Code)
		}
	}
}

// TestWS_OperatorWS_WiresAuthentication — 源码契约:
// operatorWS 必须在 websocket.Accept 前调 authenticateOperatorWS。
// 防止后续重构误删 auth 检查。
func TestWS_OperatorWS_WiresAuthentication(t *testing.T) {
	src, err := os.ReadFile("ws.go")
	if err != nil {
		t.Fatalf("read ws.go: %v", err)
	}
	body := string(src)

	idx := strings.Index(body, "func (h *WSHandler) operatorWS")
	if idx < 0 {
		t.Fatal("找不到 operatorWS 函数")
	}
	end := strings.Index(body[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(body) - idx - 1
	}
	fnBody := body[idx : idx+1+end]

	// 必须调 authenticateOperatorWS
	if !strings.Contains(fnBody, "authenticateOperatorWS(") {
		t.Errorf("operatorWS 缺失 authenticateOperatorWS 调用 — auth 接线破坏")
	}

	// 必须在 websocket.Accept 之前
	idxAuth := strings.Index(fnBody, "authenticateOperatorWS(")
	idxAccept := strings.Index(fnBody, "websocket.Accept(")
	if idxAuth < 0 || idxAccept < 0 {
		t.Fatal("找不到 auth 或 Accept 调用")
	}
	if idxAuth > idxAccept {
		t.Errorf("authenticateOperatorWS (idx=%d) 在 websocket.Accept (idx=%d) 之后 — 顺序错,先 Accept 后 auth 等于无效", idxAuth, idxAccept)
	}

	// auth 失败时必须 return(不 Accept)
	authCallBlock := fnBody[idxAuth:]
	if !strings.Contains(authCallBlock, "if !authOK") {
		t.Errorf("operatorWS 缺失 `if !authOK { return }` 守护")
	}
}

// TestOperatorWS_NoCookie_Returns401_Behavioral — 1ae 升级 T0-1h-2 行为级:
// 真调 operatorWS handler(不是 authenticateOperatorWS 函数本身),
// 断言无 cookie 时整个 handler 返回 401 + 不进入 websocket.Accept。
//
// 此测试解决 audit M4 SURVIVED:源码契约测试不能检测 dead code(如把 auth 调用
// 包进 if false)。行为级测试必须真正触发 handler 执行路径。
func TestOperatorWS_NoCookie_Returns401_Behavioral(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ws/operator", nil)
	c.Request.Header.Set("Origin", "http://localhost:8080") // 同源,过 OriginCheck

	h := &WSHandler{
		stores:  &storage.Stores{Redis: &storage.Redis{Client: rdb}},
		devMode: false, // prod mode,无 bypass
		// hub/stream/flusher/snapshots 不需要:auth 失败应在到达 Accept 前返回
	}

	h.operatorWS(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("operatorWS no-cookie: status=%d want 401 (behavioral: handler must invoke auth check)", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "no_session") && !strings.Contains(body, "invalid_session") {
		t.Errorf("operatorWS no-cookie: body should contain no_session/invalid_session error, got: %s", body)
	}
	// 关键:WebSocket Upgrade 不应发生 — 响应不应有 Upgrade headers
	if got := w.Header().Get("Upgrade"); got != "" {
		t.Errorf("operatorWS no-cookie: Upgrade header set (%q) — WebSocket Accept 不应在 auth 失败时被调用", got)
	}
	if got := w.Header().Get("Connection"); got == "Upgrade" {
		t.Errorf("operatorWS no-cookie: Connection=Upgrade — 同上")
	}
}

// TestOperatorWS_InvalidSession_Returns401_Behavioral — 1ae 新增:
// 带无效 cookie session 时,handler 也必须 401,不进入 Accept。
func TestOperatorWS_InvalidSession_Returns401_Behavioral(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ws/operator", nil)
	c.Request.Header.Set("Origin", "http://localhost:8080")
	// 加一个不在 Redis 中的 session cookie
	c.Request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "nonexistent-session-id"})

	h := &WSHandler{
		stores:  &storage.Stores{Redis: &storage.Redis{Client: rdb}},
		devMode: false,
	}

	h.operatorWS(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("operatorWS invalid-session: status=%d want 401", w.Code)
	}
	if got := w.Header().Get("Upgrade"); got != "" {
		t.Errorf("operatorWS invalid-session: Upgrade header set — Accept 不应被调用")
	}
}
