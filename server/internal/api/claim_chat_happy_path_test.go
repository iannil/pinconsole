// 1ai-e 测试:ClaimHandler + ChatHandler happy path 行为级测试。
//
// 复用 1ai-c 的 mockRedisStore(支持 SetNX/Get/EvalLua/TTL),
// 加 2 新 mock:mockSessionRepo + mockChatMessageRepo。
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
	"github.com/iannil/marketing-monitor/internal/storage"
	"github.com/jackc/pgx/v5/pgtype"
)

// ============ 新 mock ============

// mockSessionRepo 是 claimSessionRepo 的手写 mock。
type mockSessionRepo struct {
	mu       sync.Mutex
	sessions map[uuid.UUID]*storage.Session
	err      error
	calls    int
}

func (m *mockSessionRepo) GetSession(ctx context.Context, id uuid.UUID) (*storage.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.sessions[id], nil
}

// mockChatMessageRepo 是 chatMessageRepo 的手写 mock。
type mockChatMessageRepo struct {
	mu        sync.Mutex
	messages  []storage.ChatMessage
	err       error
	listCalls int
}

func (m *mockChatMessageRepo) ListChatMessagesBySession(ctx context.Context, sessionID uuid.UUID, sinceID int64, limit int32) ([]storage.ChatMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listCalls++
	if m.err != nil {
		return nil, m.err
	}
	return m.messages, nil
}

// newMockRedisStoreWithSetNX 扩展支持 SetNX 返回值控制(测试 already-claimed 用)。
// 复用 1ai-c 的 mockRedisStore,加 setnxResult 字段。
type mockRedisStoreForClaim struct {
	*mockRedisStore
	mu           sync.Mutex
	setnxResult  bool // 控制 SetNX 返回值
	setnxCalls   int
}

func (m *mockRedisStoreForClaim) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setnxCalls++
	if m.setnxResult {
		m.data[key] = value
		m.ttl[key] = ttl
	}
	return m.setnxResult, nil
}

// Get/Set/Del/EvalLua/TTL 透传给内嵌 mockRedisStore。
// 注意:SetNX 用本类型的覆盖版,其他方法继承。

// ============ ClaimHandler happy path ============

// TestClaim_Success_Returns200_ClaimedBy — mock session active + SetNX 成功 →
// 200 + claimed_by = caller UID。
func TestClaim_Success_Returns200_ClaimedBy(t *testing.T) {
	sessionID := uuid.New()
	callerUID := uuid.New()

	mockSessions := &mockSessionRepo{
		sessions: map[uuid.UUID]*storage.Session{
			sessionID: {
				ID:       sessionID,
				Status:   "active",
				EndedAt:  pgtype.Timestamptz{}, // Valid=false 表示未结束
			},
		},
	}
	mockRedis := &mockRedisStoreForClaim{
		mockRedisStore: newMockRedisStore(),
		setnxResult:    true,
	}

	h := &ClaimHandler{
		sessionRepo: mockSessions,
		redis:       mockRedis,
		logger:      testLogger(),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/claim", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()

	// 注入 user_id(模拟 AuthMiddleware)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", callerUID)

	// 直接调 handler(避免 gin engine 不传 user_id 的复杂性)
	h.claim(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, callerUID.String()) {
		t.Errorf("body 缺 caller UID: %s", body)
	}
	if !strings.Contains(body, sessionID.String()) {
		t.Errorf("body 缺 session ID: %s", body)
	}

	// 验证 mock 调用:GetSession 1 次,SetNX 1 次
	if mockSessions.calls != 1 {
		t.Errorf("GetSession calls = %d, want 1", mockSessions.calls)
	}
	if mockRedis.setnxCalls != 1 {
		t.Errorf("SetNX calls = %d, want 1", mockRedis.setnxCalls)
	}
}

// TestClaim_AlreadyClaimed_Returns409 — SetNX 失败 + Redis.Get 返现有 owner →
// 409 already_claimed + claimed_by=owner。
//
// 1k P0-4 race-safety 核心验证:SetNX 失败时不应覆盖现有 owner。
func TestClaim_AlreadyClaimed_Returns409(t *testing.T) {
	sessionID := uuid.New()
	callerUID := uuid.New()
	ownerUID := uuid.New()

	mockSessions := &mockSessionRepo{
		sessions: map[uuid.UUID]*storage.Session{
			sessionID: {ID: sessionID, Status: "active"},
		},
	}
	mockRedis := &mockRedisStoreForClaim{
		mockRedisStore: newMockRedisStore(),
		setnxResult:    false, // 模拟已被 claim
	}
	// 预设现有 owner
	mockRedis.data[claimKey(sessionID)] = []byte(ownerUID.String())

	h := &ClaimHandler{
		sessionRepo: mockSessions,
		redis:       mockRedis,
		logger:      testLogger(),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/claim", nil)
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", callerUID)

	h.claim(c)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "already_claimed") {
		t.Errorf("body 缺 already_claimed: %s", body)
	}
	if !strings.Contains(body, ownerUID.String()) {
		t.Errorf("body 缺 owner UID: %s", body)
	}
	// 不应包含 caller UID(否则 owner 被覆盖)
	if strings.Contains(body, callerUID.String()) {
		t.Errorf("body 不应含 caller UID(owner 应保留): %s", body)
	}
}

// ============ ChatHandler.listMessages happy path ============

// TestListMessages_Success_ReturnsArray — mock 返多条消息 → 200 + JSON array。
func TestListMessages_Success_ReturnsArray(t *testing.T) {
	sessionID := uuid.New()
	now := time.Now()
	mockMsgs := &mockChatMessageRepo{
		messages: []storage.ChatMessage{
			{ID: 1, Sender: "operator", Content: "hello", CreatedAt: now},
			{ID: 2, Sender: "visitor", Content: "hi", CreatedAt: now.Add(time.Second)},
		},
	}

	h := &ChatHandler{
		messageRepo: mockMsgs,
		logger:      testLogger(),
	}
	r := newChatTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID.String()+"/messages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"id":1`) || !strings.Contains(body, `"id":2`) {
		t.Errorf("body 缺消息 ID: %s", body)
	}
	if !strings.Contains(body, "hello") || !strings.Contains(body, "hi") {
		t.Errorf("body 缺消息内容: %s", body)
	}
	if !strings.Contains(body, `"sender":"operator"`) {
		t.Errorf("body 缺 sender: %s", body)
	}
	if mockMsgs.listCalls != 1 {
		t.Errorf("ListChatMessagesBySession calls = %d, want 1", mockMsgs.listCalls)
	}
}

// TestListMessages_Empty_ReturnsEmptyArray — mock 返空 slice → 200 + "[]"。
//
// 防 JSON 序列化为 null(admin 前端期望 []).
func TestListMessages_Empty_ReturnsEmptyArray(t *testing.T) {
	sessionID := uuid.New()
	mockMsgs := &mockChatMessageRepo{
		messages: []storage.ChatMessage{}, // 显式空 slice
	}

	h := &ChatHandler{
		messageRepo: mockMsgs,
		logger:      testLogger(),
	}
	r := newChatTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID.String()+"/messages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	// 必须是 [] 不是 null(防 admin 前端 .map 报错)
	if !strings.Contains(body, `"messages":[]`) {
		t.Errorf("body 缺 'messages:[]': %s", body)
	}
}

// 编译时契约:防 pgtype import 被裁剪。
var _ = pgtype.Timestamptz{}
