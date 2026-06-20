// 1ai-h 测试:SessionHandler.initSession 行为级测试 + replay.go 纯函数测试。
//
// 1ai-h:SessionHandler 接口化(1ai-g 模式扩展)+ replay 纯函数零依赖测试。
package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
)

// ============ SessionHandler mock ============

// mockSessionInitRepo 实现 sessionInitRepo 接口。
type mockSessionInitRepo struct {
	mu              sync.Mutex
	visitorCalls    int
	sessionCalls    int
	lastVisitorUA   string
	lastVisitorIP   string
	lastSessionUA   string
	lastSessionIP   string
	visitorErr      error
	sessionErr      error
	visitorResult   *storage.Visitor
	sessionResult   *storage.Session
}

func (m *mockSessionInitRepo) CreateVisitor(ctx context.Context, tenantID uuid.UUID, fingerprint, ua, ip string) (*storage.Visitor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.visitorCalls++
	m.lastVisitorUA = ua
	m.lastVisitorIP = ip
	if m.visitorErr != nil {
		return nil, m.visitorErr
	}
	if m.visitorResult != nil {
		return m.visitorResult, nil
	}
	return &storage.Visitor{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Fingerprint: fingerprint,
	}, nil
}

func (m *mockSessionInitRepo) CreateSession(ctx context.Context, tenantID, visitorID uuid.UUID, ua, ip string) (*storage.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionCalls++
	m.lastSessionUA = ua
	m.lastSessionIP = ip
	if m.sessionErr != nil {
		return nil, m.sessionErr
	}
	if m.sessionResult != nil {
		return m.sessionResult, nil
	}
	return &storage.Session{
		ID:        uuid.New(),
		TenantID:  tenantID,
		VisitorID: visitorID,
		Status:    "active",
	}, nil
}

// ============ SessionHandler.initSession 测试 ============

// TestInitSession_Success_Returns200 — happy path:
// 提交合法 visitor_id → mock CreateVisitor + CreateSession → 200 + session_id。
func TestInitSession_Success_Returns200(t *testing.T) {
	mockRepo := &mockSessionInitRepo{}
	h := &SessionHandler{
		sessionRepo: mockRepo,
		logger:      testLogger(),
	}
	r := gin.New()
	gin.SetMode(gin.TestMode)
	r.POST("/api/session/init", h.initSession)

	body := []byte(`{"visitor_id":"fp-1aih-abc","ua":"Mozilla/5.0","ip":"10.0.0.1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/session/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.100:1234" // ClientIP 来源
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	respBody := w.Body.String()
	if !strings.Contains(respBody, `"session_id":`) {
		t.Errorf("body 缺 session_id: %s", respBody)
	}
	if !strings.Contains(respBody, `"visitor_id":`) {
		t.Errorf("body 缺 visitor_id: %s", respBody)
	}

	// 验证 mock 调用
	if mockRepo.visitorCalls != 1 {
		t.Errorf("CreateVisitor calls = %d, want 1", mockRepo.visitorCalls)
	}
	if mockRepo.sessionCalls != 1 {
		t.Errorf("CreateSession calls = %d, want 1", mockRepo.sessionCalls)
	}
}

// TestInitSession_MissingVisitorID_Returns400 — visitor_id 必填,缺 → 400 missing_visitor_id。
//
// 不触达 stores(检查在 stores 调用前)。
func TestInitSession_MissingVisitorID_Returns400(t *testing.T) {
	mockRepo := &mockSessionInitRepo{}
	h := &SessionHandler{
		sessionRepo: mockRepo,
		logger:      testLogger(),
	}
	r := gin.New()
	gin.SetMode(gin.TestMode)
	r.POST("/api/session/init", h.initSession)

	body := []byte(`{"ua":"Mozilla/5.0"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/session/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "missing_visitor_id") {
		t.Errorf("body = %q, want contains 'missing_visitor_id'", w.Body.String())
	}
	if mockRepo.visitorCalls != 0 {
		t.Errorf("CreateVisitor calls = %d, want 0 (visitor_id 检查在前)", mockRepo.visitorCalls)
	}
}

// TestInitSession_InvalidJSON_Returns400 — 非法 JSON → 400 invalid_json。
func TestInitSession_InvalidJSON_Returns400(t *testing.T) {
	h := &SessionHandler{
		sessionRepo: &mockSessionInitRepo{},
		logger:      testLogger(),
	}
	r := gin.New()
	gin.SetMode(gin.TestMode)
	r.POST("/api/session/init", h.initSession)

	req := httptest.NewRequest(http.MethodPost, "/api/session/init", bytes.NewReader([]byte(`{not-json`)))
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

// TestInitSession_CreateVisitorError_Returns500 — mock CreateVisitor 返 error → 500 db_error。
func TestInitSession_CreateVisitorError_Returns500(t *testing.T) {
	mockRepo := &mockSessionInitRepo{
		visitorErr: context.DeadlineExceeded,
	}
	h := &SessionHandler{
		sessionRepo: mockRepo,
		logger:      testLogger(),
	}
	r := gin.New()
	gin.SetMode(gin.TestMode)
	r.POST("/api/session/init", h.initSession)

	body := []byte(`{"visitor_id":"fp-err","ua":"","ip":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/session/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	if !strings.Contains(w.Body.String(), "db_error") {
		t.Errorf("body = %q, want contains 'db_error'", w.Body.String())
	}
	if mockRepo.sessionCalls != 0 {
		t.Errorf("CreateSession calls = %d, want 0 (visitor 创建失败应中止)", mockRepo.sessionCalls)
	}
}
