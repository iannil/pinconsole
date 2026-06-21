// Claim refresh 端点测试(P1-claim-TTL 修复)。
//
// 场景:claim TTL=5min,无续期机制 → 长会话静默失去 claim。
// 修复:加 POST /api/sessions/:id/claim/refresh,owner 可调,EXPIRE 续 TTL。
//
// 测试覆盖:
//   - owner 调 refresh → 200,TTL 被续期
//   - 非 owner 调 refresh → 403,TTL 不变
//   - 无 claim(key 不存在)→ 403
//   - Lua 错误 → 500
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
	"github.com/jackc/pgx/v5/pgtype"
)

// refreshMockRedis 专门处理 refreshClaimLua 语义:仅当 GET keys[1] == args[0] 时 EXPIRE。
// 不复用 mockRedisStore,因为其 EvalLua 是 INCR 硬编码。
type refreshMockRedis struct {
	mu       sync.Mutex
	data     map[string][]byte
	ttl      map[string]string // iso duration string for testing visibility
	evalErr  error
	expCalls int
	failEx   bool // 模拟 owner 不匹配
}

func newRefreshMockRedis() *refreshMockRedis {
	return &refreshMockRedis{
		data: make(map[string][]byte),
		ttl:  make(map[string]string),
	}
}

func (m *refreshMockRedis) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data[key], nil
}

func (m *refreshMockRedis) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.data[key]; exists {
		return false, nil
	}
	m.data[key] = value
	m.ttl[key] = ttl.String()
	return true, nil
}

func (m *refreshMockRedis) Del(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *refreshMockRedis) EvalLua(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.evalErr != nil {
		return nil, m.evalErr
	}
	// 仿 releaseClaimLua / refreshClaimLua 语义:
	// 仅当 GET keys[1] == args[0] 时执行操作(DEL 或 EXPIRE),返回 1;否则返回 0。
	key := keys[0]
	caller, _ := args[0].(string)
	current, exists := m.data[key]
	if !exists || string(current) != caller {
		return int64(0), nil
	}
	// 区分 DEL(release)和 EXPIRE(refresh):看脚本内容含 "EXPIRE"。
	if strings.Contains(script, "EXPIRE") {
		m.expCalls++
		m.ttl[key] = "300s" // 续期
		return int64(1), nil
	}
	delete(m.data, key)
	delete(m.ttl, key)
	return int64(1), nil
}

func (m *refreshMockRedis) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, _ := time.ParseDuration(m.ttl[key])
	return d, nil
}

// --- tests ---

// TestRefresh_Success_OwnerRenewsTTL — owner 调 refresh → 200,TTL 续期。
func TestRefresh_Success_OwnerRenewsTTL(t *testing.T) {
	sessionID := uuid.New()
	ownerUID := uuid.New()

	mockSessions := &mockSessionRepo{
		sessions: map[uuid.UUID]*storage.Session{
			sessionID: {ID: sessionID, Status: "active", EndedAt: pgtype.Timestamptz{}},
		},
	}
	mockRedis := newRefreshMockRedis()
	// 预设 claim 存在,owner=ownerUID
	mockRedis.data[claimKey(sessionID)] = []byte(ownerUID.String())
	mockRedis.ttl[claimKey(sessionID)] = "1s" // 模拟 TTL 快到了

	h := &ClaimHandler{
		sessionRepo: mockSessions,
		redis:       mockRedis,
		logger:      testLogger(),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/claim/refresh", nil)
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", ownerUID)

	h.refresh(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), sessionID.String()) {
		t.Errorf("body 缺 session_id: %s", w.Body.String())
	}
	if mockRedis.expCalls != 1 {
		t.Errorf("EXPIRE calls = %d, want 1", mockRedis.expCalls)
	}
	gotTTL, _ := time.ParseDuration(mockRedis.ttl[claimKey(sessionID)])
	if gotTTL != claimTTL {
		t.Errorf("TTL after refresh = %v, want %v", gotTTL, claimTTL)
	}
}

// TestRefresh_NotOwner_Returns403 — 非 owner 调 refresh → 403,TTL 不变。
func TestRefresh_NotOwner_Returns403(t *testing.T) {
	sessionID := uuid.New()
	ownerUID := uuid.New()
	otherUID := uuid.New()

	mockSessions := &mockSessionRepo{
		sessions: map[uuid.UUID]*storage.Session{
			sessionID: {ID: sessionID, Status: "active"},
		},
	}
	mockRedis := newRefreshMockRedis()
	mockRedis.data[claimKey(sessionID)] = []byte(ownerUID.String())
	mockRedis.ttl[claimKey(sessionID)] = "1s"

	h := &ClaimHandler{
		sessionRepo: mockSessions,
		redis:       mockRedis,
		logger:      testLogger(),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/claim/refresh", nil)
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", otherUID) // 非 owner

	h.refresh(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "not_claim_owner") {
		t.Errorf("body 缺 not_claim_owner: %s", w.Body.String())
	}
	if mockRedis.expCalls != 0 {
		t.Errorf("EXPIRE should not be called for non-owner, got %d", mockRedis.expCalls)
	}
}

// TestRefresh_NoClaim_Returns403 — claim 不存在(key 过期)→ 403。
func TestRefresh_NoClaim_Returns403(t *testing.T) {
	sessionID := uuid.New()
	callerUID := uuid.New()

	mockSessions := &mockSessionRepo{
		sessions: map[uuid.UUID]*storage.Session{
			sessionID: {ID: sessionID, Status: "active"},
		},
	}
	mockRedis := newRefreshMockRedis() // 空,无 claim

	h := &ClaimHandler{
		sessionRepo: mockSessions,
		redis:       mockRedis,
		logger:      testLogger(),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/claim/refresh", nil)
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", callerUID)

	h.refresh(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "not_claim_owner") {
		t.Errorf("body 缺 not_claim_owner: %s", w.Body.String())
	}
}

// TestRefresh_InvalidSessionID_Returns400 — UUID 不合法 → 400。
func TestRefresh_InvalidSessionID_Returns400(t *testing.T) {
	mockRedis := newRefreshMockRedis()
	h := &ClaimHandler{
		sessionRepo: &mockSessionRepo{sessions: map[uuid.UUID]*storage.Session{}},
		redis:       mockRedis,
		logger:      testLogger(),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/not-a-uuid/claim/refresh", nil)
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	c.Set("user_id", uuid.New())

	h.refresh(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_session_id") {
		t.Errorf("body 缺 invalid_session_id: %s", w.Body.String())
	}
}

// TestRefresh_LuaError_Returns500 — Redis EvalLua 失败 → 500。
func TestRefresh_LuaError_Returns500(t *testing.T) {
	sessionID := uuid.New()
	ownerUID := uuid.New()

	mockSessions := &mockSessionRepo{
		sessions: map[uuid.UUID]*storage.Session{
			sessionID: {ID: sessionID, Status: "active"},
		},
	}
	mockRedis := newRefreshMockRedis()
	mockRedis.data[claimKey(sessionID)] = []byte(ownerUID.String())
	mockRedis.evalErr = context.DeadlineExceeded

	h := &ClaimHandler{
		sessionRepo: mockSessions,
		redis:       mockRedis,
		logger:      testLogger(),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/claim/refresh", nil)
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", ownerUID)

	h.refresh(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "refresh_failed") {
		t.Errorf("body 缺 refresh_failed: %s", w.Body.String())
	}
}
