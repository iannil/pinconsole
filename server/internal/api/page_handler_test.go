// page_handler_test.go — page-editor pe-1 API handler 测试。
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iannil/pinconsole/internal/pages"
	"github.com/iannil/pinconsole/internal/storage"
)

// ---------- 辅助函数 ----------

func helperEnsurePageTables(t *testing.T, stores *storage.Stores) {
	t.Helper()
	// 确保测试用表存在（独立迁移，不依赖外部 migration runner）
	queries := []string{
		`CREATE TABLE IF NOT EXISTS pages (
			id BIGSERIAL PRIMARY KEY,
			tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
			slug VARCHAR(64) NOT NULL,
			title VARCHAR(255) NOT NULL DEFAULT '',
			status VARCHAR(16) NOT NULL DEFAULT 'draft',
			schema JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(tenant_id, slug)
		)`,
		`CREATE TABLE IF NOT EXISTS page_leads (
			id BIGSERIAL PRIMARY KEY,
			tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
			page_slug VARCHAR(64) NOT NULL,
			fields JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}
	for _, q := range queries {
		if _, err := stores.PG.Pool.Exec(t.Context(), q); err != nil {
			t.Fatalf("ensure page tables: %v", err)
		}
	}
}

func helperPageHandler(t *testing.T) *PageHandler {
	t.Helper()
	logger := apiDiscardLogger()
	stores := helperStoresForAPITest(t)
	helperEnsurePageTables(t, stores)
	renderer, err := pages.NewRenderer()
	if err != nil {
		t.Skipf("renderer not available: %v", err)
	}
	return NewPageHandler(stores, renderer, logger)
}

func helperPageSetup(t *testing.T, h *PageHandler) string {
	t.Helper()
	tenantID := storage.DefaultTenantID

	// 清理历史数据
	_ = h.stores.PG.DeletePageByID(t.Context(), 0) // no-op if 0

	// 创建一个测试页面
	page, err := h.stores.PG.CreatePage(t.Context(), tenantID, "test-page", "测试页面")
	if err != nil {
		t.Fatalf("create page: %v", err)
	}

	// 写一个基本 schema
	schema := `{"meta":{"title":"测试页面"},"style":{"primary_color":"#0f766e","background":"#fff"},"sections":[{"id":"s1","type":"hero","props":{"title":"Hello","subtitle":"World"}}]}`
	_, err = h.stores.PG.UpdatePage(t.Context(), tenantID, "test-page", nil, []byte(schema), nil)
	if err != nil {
		t.Fatalf("update page schema: %v", err)
	}

	return page.Slug
}

func helperPageCleanup(t *testing.T, h *PageHandler, slug string) {
	t.Helper()
	_ = h.stores.PG.DeletePage(t.Context(), storage.DefaultTenantID, slug)
	// 清理该页面的所有 leads
	leads, _ := h.stores.PG.ListPageLeads(t.Context(), storage.DefaultTenantID, slug)
	for _, l := range leads {
		_ = h.stores.PG.DeletePageLeadByID(t.Context(), l.ID)
	}
}

// ---------- ListPages ----------

func TestPage_ListPages_Empty(t *testing.T) {
	h := helperPageHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/pages", h.listPages)

	req := httptest.NewRequest(http.MethodGet, "/api/pages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var res []any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// 可能已有其他测试数据，只检查是数组
	t.Logf("list pages count: %d", len(res))
}

func TestPage_CreatePage_Success(t *testing.T) {
	h := helperPageHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/pages", h.createPage)

	body := `{"title":"My Landing","slug":"my-landing"}`
	req := httptest.NewRequest(http.MethodPost, "/api/pages", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res["slug"] != "my-landing" {
		t.Errorf("slug = %v, want my-landing", res["slug"])
	}
	if res["status"] != "draft" {
		t.Errorf("status = %v, want draft", res["status"])
	}

	// 清理
	_ = h.stores.PG.DeletePage(t.Context(), storage.DefaultTenantID, "my-landing")
}

func TestPage_CreatePage_DuplicateSlug(t *testing.T) {
	h := helperPageHandler(t)
	slug := helperPageSetup(t, h)
	defer helperPageCleanup(t, h, slug)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/pages", h.createPage)

	body := `{"title":"Duplicate","slug":"` + slug + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/pages", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestPage_CreatePage_NoSlug(t *testing.T) {
	h := helperPageHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/pages", h.createPage)

	body := `{"title":"Auto Slug"}`
	req := httptest.NewRequest(http.MethodPost, "/api/pages", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	slug, ok := res["slug"].(string)
	if !ok || slug == "" {
		t.Errorf("slug should be auto-generated, got %v", res["slug"])
	}

	// 清理
	_ = h.stores.PG.DeletePage(t.Context(), storage.DefaultTenantID, slug)
}

// ---------- GetPage ----------

func TestPage_GetPage_Success(t *testing.T) {
	h := helperPageHandler(t)
	slug := helperPageSetup(t, h)
	defer helperPageCleanup(t, h, slug)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/pages/:slug", h.getPage)

	req := httptest.NewRequest(http.MethodGet, "/api/pages/"+slug, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res["slug"] != slug {
		t.Errorf("slug = %v, want %s", res["slug"], slug)
	}
	if res["schema"] == nil {
		t.Error("schema should not be nil")
	}
}

func TestPage_GetPage_NotFound(t *testing.T) {
	h := helperPageHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/pages/:slug", h.getPage)

	req := httptest.NewRequest(http.MethodGet, "/api/pages/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ---------- UpdatePage ----------

func TestPage_UpdatePage_Schema(t *testing.T) {
	h := helperPageHandler(t)
	slug := helperPageSetup(t, h)
	defer helperPageCleanup(t, h, slug)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/api/pages/:slug", h.updatePage)

	newSchema := `{"meta":{"title":"Updated"},"style":{"primary_color":"#000","background":"#fff"},"sections":[]}`
	body := `{"schema":` + newSchema + `}`
	req := httptest.NewRequest(http.MethodPut, "/api/pages/"+slug, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ---------- PublishPage ----------

func TestPage_PublishPage(t *testing.T) {
	h := helperPageHandler(t)
	slug := helperPageSetup(t, h)
	defer helperPageCleanup(t, h, slug)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/pages/:slug/publish", h.publishPage)

	body := `{"status":"published"}`
	req := httptest.NewRequest(http.MethodPost, "/api/pages/"+slug+"/publish", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res["status"] != "published" {
		t.Errorf("status = %v, want published", res["status"])
	}
}

// ---------- DeletePage ----------

func TestPage_DeletePage(t *testing.T) {
	h := helperPageHandler(t)
	slug := helperPageSetup(t, h)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/api/pages/:slug", h.deletePage)

	req := httptest.NewRequest(http.MethodDelete, "/api/pages/"+slug, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// 验证已删除
	page, err := h.stores.PG.GetPageBySlug(t.Context(), storage.DefaultTenantID, slug)
	if err != nil {
		t.Fatalf("get page after delete: %v", err)
	}
	if page != nil {
		t.Error("page should be nil after delete")
	}
}

// ---------- Form Submit ----------

func TestPage_FormSubmit_Success(t *testing.T) {
	h := helperPageHandler(t)
	slug := helperPageSetup(t, h)
	defer helperPageCleanup(t, h, slug)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/pages/:slug/form", h.submitForm)

	body := "name=张三&email=test@example.com"
	req := httptest.NewRequest(http.MethodPost, "/api/pages/"+slug+"/form", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	// 验证存储
	leads, err := h.stores.PG.ListPageLeads(t.Context(), storage.DefaultTenantID, slug)
	if err != nil {
		t.Fatalf("list leads: %v", err)
	}
	if len(leads) != 1 {
		t.Errorf("leads count = %d, want 1", len(leads))
	}
}

func TestPage_FormSubmit_Honeypot(t *testing.T) {
	h := helperPageHandler(t)
	slug := helperPageSetup(t, h)
	defer helperPageCleanup(t, h, slug)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/pages/:slug/form", h.submitForm)

	// honeypot 有值表示机器人提交
	body := "_pin=robot&name=bot"
	req := httptest.NewRequest(http.MethodPost, "/api/pages/"+slug+"/form", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// 验证没有存储
	leads, err := h.stores.PG.ListPageLeads(t.Context(), storage.DefaultTenantID, slug)
	if err != nil {
		t.Fatalf("list leads: %v", err)
	}
	if len(leads) != 0 {
		t.Errorf("leads count = %d, want 0 (honeypot should filter)", len(leads))
	}
}

// ---------- SSR Render ----------

func TestPage_RenderPage_Success(t *testing.T) {
	h := helperPageHandler(t)
	slug := helperPageSetup(t, h)
	defer helperPageCleanup(t, h, slug)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/p/:slug", h.renderPage)

	req := httptest.NewRequest(http.MethodGet, "/p/"+slug, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "Hello") {
		t.Error("rendered HTML should contain hero title 'Hello'")
	}
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("rendered HTML should be a complete page")
	}
}

func TestPage_RenderPage_NotFound(t *testing.T) {
	h := helperPageHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/p/:slug", h.renderPage)

	req := httptest.NewRequest(http.MethodGet, "/p/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
