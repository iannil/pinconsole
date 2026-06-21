// 1g-visitor 测试:postVisitorMessage 公开端点行为级测试。
//
// 验证 P1 fix:visitor 无 admin cookie 也能发消息;sender 固定 "visitor"。
// 复用 postmessage_happy_path_test.go 的 mockChatMessageCreator 和
// claim_chat_happy_path_test.go 的 mockSessionRepo。
package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
	"github.com/jackc/pgx/v5/pgtype"
)

// newSessionRepoMockWithSession 构造一个已 seed 了指定 session 的 mock。
// ended=true 时 EndedAt.Valid=true,模拟会话已结束。
func newSessionRepoMockWithSession(sessionID uuid.UUID, ended bool) *mockSessionRepo {
	sess := &storage.Session{
		ID:     sessionID,
		Status: "active",
	}
	if ended {
		sess.EndedAt = pgtype.Timestamptz{Valid: true}
	}
	return &mockSessionRepo{
		sessions: map[uuid.UUID]*storage.Session{sessionID: sess},
	}
}

func newEmptySessionRepoMock() *mockSessionRepo {
	return &mockSessionRepo{
		sessions: map[uuid.UUID]*storage.Session{},
	}
}

// TestPostVisitorMessage_Success_NoAuth_Returns200 — visitor 无 admin cookie,
// session 存在 + 未结束 → 200,sender 固定 visitor。
func TestPostVisitorMessage_Success_NoAuth_Returns200(t *testing.T) {
	sessionID := uuid.New()
	mockSessions := newSessionRepoMockWithSession(sessionID, false)
	mockMsg := &mockChatMessageCreator{}

	h := &ChatHandler{
		sessionRepo: mockSessions,
		createMsg:   mockMsg,
		logger:      testLogger(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+sessionID.String()+"/visitor-message",
		bytes.NewReader([]byte(`{"content":"你好,请问价格?"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
	// 关键:不 c.Set("user_id", ...) —— 模拟 visitor 无 admin cookie

	h.postVisitorMessage(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"sender":"visitor"`) {
		t.Errorf("body 缺 sender=visitor: %s", body)
	}
	if !strings.Contains(body, "你好,请问价格?") {
		t.Errorf("body 缺 content: %s", body)
	}
	if mockMsg.createCalls != 1 {
		t.Errorf("CreateChatMessage calls = %d, want 1", mockMsg.createCalls)
	}
	if mockMsg.lastSender != "visitor" {
		t.Errorf("sender = %q, want 'visitor'", mockMsg.lastSender)
	}
}

// TestPostVisitorMessage_UnknownSession_Returns404 — session 不存在 → 404。
func TestPostVisitorMessage_UnknownSession_Returns404(t *testing.T) {
	sessionID := uuid.New()
	h := &ChatHandler{
		sessionRepo: newEmptySessionRepoMock(),
		createMsg:   &mockChatMessageCreator{},
		logger:      testLogger(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+sessionID.String()+"/visitor-message",
		bytes.NewReader([]byte(`{"content":"hi"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}

	h.postVisitorMessage(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "session_not_found") {
		t.Errorf("body = %q, want contains 'session_not_found'", w.Body.String())
	}
}

// TestPostVisitorMessage_EmptyContent_Returns400 — 空 content → 400 empty_content。
func TestPostVisitorMessage_EmptyContent_Returns400(t *testing.T) {
	sessionID := uuid.New()
	h := &ChatHandler{
		sessionRepo: newSessionRepoMockWithSession(sessionID, false),
		createMsg:   &mockChatMessageCreator{},
		logger:      testLogger(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+sessionID.String()+"/visitor-message",
		bytes.NewReader([]byte(`{"content":""}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}

	h.postVisitorMessage(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "empty_content") {
		t.Errorf("body = %q, want contains 'empty_content'", w.Body.String())
	}
}

// TestPostVisitorMessage_SessionEnded_Returns409 — session 已结束 → 409。
func TestPostVisitorMessage_SessionEnded_Returns409(t *testing.T) {
	sessionID := uuid.New()
	h := &ChatHandler{
		sessionRepo: newSessionRepoMockWithSession(sessionID, true),
		createMsg:   &mockChatMessageCreator{},
		logger:      testLogger(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/api/sessions/"+sessionID.String()+"/visitor-message",
		bytes.NewReader([]byte(`{"content":"hi"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}

	h.postVisitorMessage(c)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "session_ended") {
		t.Errorf("body = %q, want contains 'session_ended'", w.Body.String())
	}
}

// TestPostVisitorMessage_InvalidUUID_Returns400 — 非 UUID → 400 invalid_session_id。
func TestPostVisitorMessage_InvalidUUID_Returns400(t *testing.T) {
	h := &ChatHandler{
		sessionRepo: newEmptySessionRepoMock(),
		createMsg:   &mockChatMessageCreator{},
		logger:      testLogger(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/api/sessions/not-a-uuid/visitor-message",
		bytes.NewReader([]byte(`{"content":"hi"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}

	h.postVisitorMessage(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_session_id") {
		t.Errorf("body = %q, want contains 'invalid_session_id'", w.Body.String())
	}
}
