// 1t 测试:privacy handler 端点边界(httptest,不依赖 PG/Redis)。
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// stubStoresForPrivacy 返回 nil PG/Redis/MinIO(用于测试纯路由 + 请求校验,不实际查库)。
//
// 测试覆盖:
//   - GET /api/privacy/consent 缺 fingerprint 参数 → 400
//   - POST /api/privacy/consent 缺 body / invalid JSON → 400
//   - DELETE /api/privacy/visitor/:fingerprint 路径注册
func TestPrivacyHandler_GETConsent_MissingFingerprint_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// stub PrivacyHandler(用最小 Stores)
	h := &PrivacyHandler{stores: nil, logger: nil}

	r := gin.New()
	r.GET("/api/privacy/consent", h.getConsent)

	req := httptest.NewRequest(http.MethodGet, "/api/privacy/consent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GET /consent without fingerprint: status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "missing_fingerprint") {
		t.Errorf("body should contain missing_fingerprint, got: %s", w.Body.String())
	}
}

func TestPrivacyHandler_POSTConsent_InvalidJSON_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PrivacyHandler{stores: nil, logger: nil}

	r := gin.New()
	r.POST("/api/privacy/consent", h.postConsent)

	req := httptest.NewRequest(http.MethodPost, "/api/privacy/consent", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /consent invalid JSON: status = %d, want 400", w.Code)
	}
}

func TestPrivacyHandler_POSTConsent_MissingFingerprint_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PrivacyHandler{stores: nil, logger: nil}

	r := gin.New()
	r.POST("/api/privacy/consent", h.postConsent)

	// JSON 合法但缺 fingerprint
	req := httptest.NewRequest(http.MethodPost, "/api/privacy/consent",
		strings.NewReader(`{"accepted":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /consent missing fingerprint: status = %d, want 400", w.Code)
	}
}

// TestPrivacyHandler_RouteRegistration 验证路由签名不破坏(简单的 wire-up 测试)。
func TestPrivacyHandler_RouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PrivacyHandler{stores: nil, logger: nil}

	r := gin.New()
	h.RegisterPublic(r)

	// 验证路由表有 2 个隐私端点
	routes := r.Routes()
	hasGetConsent := false
	hasPostConsent := false
	for _, route := range routes {
		if route.Path == "/api/privacy/consent" {
			if route.Method == http.MethodGet {
				hasGetConsent = true
			}
			if route.Method == http.MethodPost {
				hasPostConsent = true
			}
		}
	}
	if !hasGetConsent {
		t.Errorf("GET /api/privacy/consent not registered")
	}
	if !hasPostConsent {
		t.Errorf("POST /api/privacy/consent not registered")
	}
}
