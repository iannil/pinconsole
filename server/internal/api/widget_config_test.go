// widget_config_test.go — page-editor pe-1 API handler 测试。
//
// 覆盖目标:widget_config.go 的 4 个 0% 函数
//   - jsonRaw.UnmarshalJSON
//   - jsonUnmarshal
//   - getConfigs (GET /api/widget-config)
//   - upsertConfig (PUT /api/widget-config/:type)
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iannil/pinconsole/internal/storage"
)

// ---------- 纯函数测试 ----------

func TestJSONRaw_UnmarshalJSON(t *testing.T) {
	var r jsonRaw
	if err := r.UnmarshalJSON([]byte(`{"key":"value"}`)); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if string(r) != `{"key":"value"}` {
		t.Errorf("jsonRaw = %q, want %q", string(r), `{"key":"value"}`)
	}

	// 空对象
	var empty jsonRaw
	if err := empty.UnmarshalJSON([]byte(`{}`)); err != nil {
		t.Fatalf("UnmarshalJSON empty failed: %v", err)
	}
	if string(empty) != `{}` {
		t.Errorf("jsonRaw empty = %q, want %q", string(empty), `{}`)
	}
}

func TestJSONUnmarshal(t *testing.T) {
	var v map[string]string
	if err := jsonUnmarshal([]byte(`{"a":"b"}`), &v); err != nil {
		t.Fatalf("jsonUnmarshal failed: %v", err)
	}
	if v["a"] != "b" {
		t.Errorf("v[a] = %q, want %q", v["a"], "b")
	}

	// 无效 JSON
	var m map[string]any
	if err := jsonUnmarshal([]byte(`{invalid}`), &m); err == nil {
		t.Error("jsonUnmarshal should fail on invalid JSON")
	}
}

// ---------- handler 集成测试 ----------

func helperWidgetConfigHandler(t *testing.T) *WidgetConfigHandler {
	t.Helper()
	logger := apiDiscardLogger()
	stores := helperStoresForAPITest(t)
	return NewWidgetConfigHandler(stores, logger)
}

func TestWidgetConfig_GetConfigs_Empty(t *testing.T) {
	h := helperWidgetConfigHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/widget-config", h.getConfigs)

	req := httptest.NewRequest(http.MethodGet, "/api/widget-config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	// configs 字段应存在且为 map
	configs, ok := resp["configs"]
	if !ok {
		t.Fatal("response missing configs field")
	}
	_, ok = configs.(map[string]any)
	if !ok {
		t.Fatalf("configs type = %T, want map[string]any", configs)
	}
	// tenant_id 应为 default
	if tid := resp["tenant_id"]; tid != storage.DefaultTenantID.String() {
		t.Errorf("tenant_id = %v, want %s", tid, storage.DefaultTenantID)
	}
}

func TestWidgetConfig_UpsertAndGet(t *testing.T) {
	h := helperWidgetConfigHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/widget-config", h.getConfigs)
	r.PUT("/api/widget-config/:type", h.upsertConfig)

	// 1. 写入 popup 配置
	body := `{"config":{"title":"hello","color":"#333"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/widget-config/popup", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var putResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &putResp); err != nil {
		t.Fatalf("PUT json decode: %v", err)
	}
	if wt := putResp["widget_type"]; wt != "popup" {
		t.Errorf("widget_type = %v, want popup", wt)
	}

	// 2. 再 GET 确认
	req2 := httptest.NewRequest(http.MethodGet, "/api/widget-config", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("GET = %d, want 200; body: %s", w2.Code, w2.Body.String())
	}

	var getResp map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("GET json decode: %v", err)
	}
	configs := getResp["configs"].(map[string]any)
	popup, ok := configs["popup"]
	if !ok {
		t.Fatal("GET response missing popup config")
	}
	popupMap, ok := popup.(map[string]any)
	if !ok {
		t.Fatalf("popup type = %T, want map[string]any", popup)
	}
	if popupMap["title"] != "hello" {
		t.Errorf("popup.title = %v, want hello", popupMap["title"])
	}
}

func TestWidgetConfig_Upsert_InvalidJSON(t *testing.T) {
	h := helperWidgetConfigHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/api/widget-config/:type", h.upsertConfig)

	// 无效 JSON 语法触发 binding 错误
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPut, "/api/widget-config/popup", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("PUT invalid = %d, want 400; body: %s", w.Code, w.Body.String())
	}
}
