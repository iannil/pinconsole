// 1ai-f 测试:CommandHandler.postCommand 行为级测试。
//
// 范围(精简版,1ai-g 做完整 happy path):
//   - postCommand Register wireup
//   - postCommand 无 user_id 注入 → 401(requireClaimOwnership 拒绝)
//   - postCommand 非 owner → 403
//
// postCommand happy path 需 requireClaimOwnership 接口化(1ai-g)。
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

// newCommandTestEngine 构造仅挂 command 路由的 gin engine。
func newCommandTestEngine(h *CommandHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)
	return r
}

// TestCommandRegister_Routes — 验证 Register 注册 1 个路由。
func TestCommandRegister_Routes(t *testing.T) {
	h := &CommandHandler{logger: testLogger()}
	r := newCommandTestEngine(h)

	want := "POST /api/sessions/:id/command"
	found := false
	for _, ri := range r.Routes() {
		if ri.Method+" "+ri.Path == want {
			found = true
		}
	}
	if !found {
		t.Errorf("command route missing: %s", want)
	}
}

// TestPostCommand_NoUserID_Returns401 — 无 user_id 注入(模拟未挂 AuthMiddleware)
// → requireClaimOwnership 返 401 not_authenticated。
//
// 不触达 stores(user_id 检查在前)。
func TestPostCommand_NoUserID_Returns401(t *testing.T) {
	h := &CommandHandler{
		// stores 完整(供 requireClaimOwnership 用),但因 user_id 检查在前不会触达
		stores:      &storage.Stores{},
		logger:      testLogger(),
	}
	r := newCommandTestEngine(h)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+uuid.New().String()+"/command", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not_authenticated") {
		t.Errorf("body = %q, want contains 'not_authenticated'", w.Body.String())
	}
}

// TestPostCommand_NotClaimOwner_Returns403 — seed Redis claim:session:<UUID>=other_uid,
// 以 caller_uid 调用 → 403 not_claim_owner。
//
// 1k P0-3 核心:非 owner 不能发命令(防越权操作访客)。
// 用真 Redis seed(claim_key) + CreateTestContext 注入 user_id。
func TestPostCommand_NotClaimOwner_Returns403(t *testing.T) {
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

	h := &CommandHandler{
		stores:      &storage.Stores{Redis: &storage.Redis{Client: rdb}},
		redis:       &storage.Redis{Client: rdb},
		logger:      testLogger(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/command", nil)
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", callerUID)

	h.postCommand(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not_claim_owner") {
		t.Errorf("body = %q, want contains 'not_claim_owner'", w.Body.String())
	}

	// claim 不应被修改(防误改 owner)
	got, _ := rdb.Get(ctx, claimKey(sessionID)).Result()
	if got != ownerUID.String() {
		t.Errorf("claim value = %q, want %s (postCommand 不应改 claim)", got, ownerUID.String())
	}
}

// 编译时契约:防 CommandHandler 字段在 1ai-g 重构时被误删。
func TestCommandHandler_FieldsContract(t *testing.T) {
	h := &CommandHandler{}
	_ = h.stores       // 1ai-g 删除
	_ = h.sessionRepo  // 1ai-f 新增
	_ = h.redis        // 1ai-f 新增
	_ = h.commandRepo  // 1ai-f 新增
	_ = h.hub
	_ = h.allowedDomains
}
