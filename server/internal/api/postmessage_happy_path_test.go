// 1ai-g 测试:postMessage happy path 行为级测试。
//
// 1ai-g:requireClaimOwnership 接口化后,可用 mock 注入测 happy path。
// 复用 1ai-c/1ai-e 的 mockRedisStore + mockSessionRepo。
package api

import (
	"bytes"
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
)

// mockChatMessageCreator 实现 chatMessageCreator 接口(postMessage 用)。
type mockChatMessageCreator struct {
	mu           sync.Mutex
	lastSession  uuid.UUID
	lastSender   string
	lastContent  string
	createCalls  int
	listResult   []storage.ChatMessage
	createResult *storage.ChatMessage
	err          error
}

func (m *mockChatMessageCreator) ListChatMessagesBySession(ctx context.Context, sessionID uuid.UUID, sinceID int64, limit int32) ([]storage.ChatMessage, error) {
	return m.listResult, nil
}

func (m *mockChatMessageCreator) CreateChatMessage(ctx context.Context, tenantID uuid.UUID, sessionID uuid.UUID, sender, content string) (*storage.ChatMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCalls++
	m.lastSession = sessionID
	m.lastSender = sender
	m.lastContent = content
	if m.err != nil {
		return nil, m.err
	}
	if m.createResult != nil {
		return m.createResult, nil
	}
	return &storage.ChatMessage{
		ID:        int64(m.createCalls),
		TenantID:  tenantID,
		SessionID: sessionID,
		Sender:    sender,
		Content:   content,
		CreatedAt: time.Now(),
	}, nil
}

// mockCommandHub 实现 CommandHub 接口,记录 SendCommandToVisitor 调用。
type mockCommandHub struct {
	mu        sync.Mutex
	sendCalls int
	lastMsg   []byte
	result    bool
}

func (m *mockCommandHub) SendCommandToVisitor(sessionID uuid.UUID, msg []byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalls++
	m.lastMsg = msg
	return m.result
}

// TestPostMessage_Success_Returns200 — owner 调用 + mock ChatRepo +
// mock Hub → 200 + chatMessageItem body。
//
// requireClaimOwnership 接口化后,可用 mockRedisStore seed claim 验证 owner 匹配。
func TestPostMessage_Success_Returns200(t *testing.T) {
	sessionID := uuid.New()
	callerUID := uuid.New()

	mockRedis := newMockRedisStore()
	// seed claim:session:<UUID> = callerUID(自己是 owner)
	mockRedis.data[claimKey(sessionID)] = []byte(callerUID.String())

	mockMsg := &mockChatMessageCreator{}
	mockHub := &mockCommandHub{result: true}

	h := &ChatHandler{
		sessionRepo: nil, // postMessage 不要求 alive session
		redis:       mockRedis,
		createMsg:   mockMsg,
		hub:         mockHub,
		logger:      testLogger(),
	}

	// 构造 gin context + 注入 user_id + :id 参数
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/messages",
		bytes.NewReader([]byte(`{"content":"hello from 1ai-g"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", callerUID)

	h.postMessage(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"sender":"operator"`) {
		t.Errorf("body 缺 sender=operator: %s", body)
	}
	if !strings.Contains(body, "hello from 1ai-g") {
		t.Errorf("body 缺 content: %s", body)
	}

	// 验证 mock 调用
	if mockMsg.createCalls != 1 {
		t.Errorf("CreateChatMessage calls = %d, want 1", mockMsg.createCalls)
	}
	if mockMsg.lastSender != "operator" {
		t.Errorf("sender = %q, want 'operator' (1k P0-3 防 client-controllable)", mockMsg.lastSender)
	}
	if mockMsg.lastContent != "hello from 1ai-g" {
		t.Errorf("content = %q, want 'hello from 1ai-g'", mockMsg.lastContent)
	}
	if mockHub.sendCalls != 1 {
		t.Errorf("SendCommandToVisitor calls = %d, want 1 (下行到 visitor)", mockHub.sendCalls)
	}
}

// TestPostMessage_NotClaimOwner_Returns403 — seed claim as other_uid,
// 以 caller_uid 调 → 403 not_claim_owner + 不写 PG + 不下行。
func TestPostMessage_NotClaimOwner_Returns403(t *testing.T) {
	sessionID := uuid.New()
	ownerUID := uuid.New()
	callerUID := uuid.New()

	mockRedis := newMockRedisStore()
	mockRedis.data[claimKey(sessionID)] = []byte(ownerUID.String())

	mockMsg := &mockChatMessageCreator{}
	mockHub := &mockCommandHub{result: true}

	h := &ChatHandler{
		redis:     mockRedis,
		createMsg: mockMsg,
		hub:       mockHub,
		logger:    testLogger(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/messages",
		bytes.NewReader([]byte(`{"content":"hack"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", callerUID)

	h.postMessage(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not_claim_owner") {
		t.Errorf("body = %q, want contains 'not_claim_owner'", w.Body.String())
	}
	// 关键:不应写 PG,不应下行
	if mockMsg.createCalls != 0 {
		t.Errorf("CreateChatMessage calls = %d, want 0 (non-owner 不应写消息)", mockMsg.createCalls)
	}
	if mockHub.sendCalls != 0 {
		t.Errorf("SendCommandToVisitor calls = %d, want 0 (non-owner 不应下行)", mockHub.sendCalls)
	}
}

// TestPostMessage_InvalidJSON_Returns400 — owner 调用 + 非法 JSON → 400 invalid_json。
//
// 注:requireClaimOwnership 在 binding 之前,所以 owner 验证先过,然后 binding 失败。
func TestPostMessage_InvalidJSON_Returns400(t *testing.T) {
	sessionID := uuid.New()
	callerUID := uuid.New()

	mockRedis := newMockRedisStore()
	mockRedis.data[claimKey(sessionID)] = []byte(callerUID.String())

	h := &ChatHandler{
		redis: mockRedis,
		hub:   &mockCommandHub{result: true},
		logger: testLogger(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/messages",
		bytes.NewReader([]byte(`{not-json`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", callerUID)

	h.postMessage(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_json") {
		t.Errorf("body = %q, want contains 'invalid_json'", w.Body.String())
	}
}

// TestPostMessage_EmptyContent_Returns400 — content 是 required 字段,空 → 400。
func TestPostMessage_EmptyContent_Returns400(t *testing.T) {
	sessionID := uuid.New()
	callerUID := uuid.New()

	mockRedis := newMockRedisStore()
	mockRedis.data[claimKey(sessionID)] = []byte(callerUID.String())

	h := &ChatHandler{
		redis:  mockRedis,
		hub:    &mockCommandHub{result: true},
		logger: testLogger(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/messages",
		bytes.NewReader([]byte(`{"content":""}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	c.Set("user_id", callerUID)

	h.postMessage(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (empty content violates binding:required)", w.Code)
	}
}
