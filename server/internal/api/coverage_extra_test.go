// Go-6 Commit 1 切片补测:纯函数 + 构造函数 + health handlers + RegisterMe,
// 提升覆盖率 38.2% → ≥60%(Commit 2 WS handlers 再提升到 ≥90%)。
package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/config"
	"github.com/iannil/pinconsole/internal/proto"
	"github.com/iannil/pinconsole/internal/storage"
)

func apiDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// === 纯函数 ===

// TestTernary 验证 ternary 三元运算 helper。
func TestTernary(t *testing.T) {
	cases := []struct {
		cond   bool
		t, f   string
		want   string
	}{
		{true, "ready", "not_ready", "ready"},
		{false, "ready", "not_ready", "not_ready"},
		{true, "yes", "no", "yes"},
		{false, "yes", "no", "no"},
		{true, "", "fallback", ""},
	}
	for _, tc := range cases {
		got := ternary(tc.cond, tc.t, tc.f)
		if got != tc.want {
			t.Errorf("ternary(%v, %q, %q): got %q, want %q", tc.cond, tc.t, tc.f, got, tc.want)
		}
	}
}

// TestCommandHandler_IsURLAllowed 验证 URL 白名单核心逻辑(1f navigate 安全)。
func TestCommandHandler_IsURLAllowed(t *testing.T) {
	h := &CommandHandler{allowedDomains: []string{"example.com", "trusted.org"}}

	cases := []struct {
		name        string
		rawURL      string
		requestHost string
		want        bool
	}{
		{"empty URL", "", "host.com", false},
		{"invalid URL", "://invalid", "host.com", false},
		{"relative same-origin", "/path", "host.com", true},
		{"same host exact", "https://host.com/path", "host.com", true},
		{"same host with port", "https://host.com:8080/path", "host.com", true},
		{"localhost", "http://localhost:3000", "host.com", true},
		{"127.0.0.1", "http://127.0.0.1:3000", "host.com", true},
		{"allowed domain exact", "https://example.com/x", "host.com", true},
		{"allowed domain subdomain", "https://sub.example.com/x", "host.com", true},
		{"allowed domain trusted.org", "https://trusted.org/y", "host.com", true},
		{"disallowed domain", "https://evil.com/x", "host.com", false},
		{"disallowed domain similar", "https://notexample.com/x", "host.com", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := h.isURLAllowed(tc.rawURL, tc.requestHost)
			if got != tc.want {
				t.Errorf("isURLAllowed(%q, %q): got %v, want %v", tc.rawURL, tc.requestHost, got, tc.want)
			}
		})
	}
}

// TestCommandHandler_IsURLAllowed_EmptyAllowedDomains 验证无白名单时只有同源/localhost 通过。
func TestCommandHandler_IsURLAllowed_EmptyAllowedDomains(t *testing.T) {
	h := &CommandHandler{allowedDomains: nil}

	if h.isURLAllowed("https://example.com/x", "host.com") {
		t.Error("empty allowedDomains: example.com should be rejected")
	}
	if !h.isURLAllowed("https://host.com/x", "host.com") {
		t.Error("same host should always pass")
	}
}

// === 构造函数(全是字段赋值,简单验证非 nil) ===

// TestNewAuthHandler_NonNil 验证 NewAuthHandler 返回非 nil。
func TestNewAuthHandler_NonNil(t *testing.T) {
	h := NewAuthHandler(&storage.Stores{}, apiDiscardLogger(), false)
	if h == nil {
		t.Error("NewAuthHandler returned nil")
	}
}

// TestNewChatHandler_NonNil 验证 NewChatHandler 返回非 nil。
func TestNewChatHandler_NonNil(t *testing.T) {
	h := NewChatHandler(&storage.Stores{}, nil, apiDiscardLogger())
	if h == nil {
		t.Error("NewChatHandler returned nil")
	}
}

// TestNewClaimHandler_NonNil 验证 NewClaimHandler 返回非 nil。
func TestNewClaimHandler_NonNil(t *testing.T) {
	h := NewClaimHandler(&storage.Stores{}, apiDiscardLogger())
	if h == nil {
		t.Error("NewClaimHandler returned nil")
	}
}

// TestNewCommandHandler_NonNil 验证 NewCommandHandler 返回非 nil。
func TestNewCommandHandler_NonNil(t *testing.T) {
	h := NewCommandHandler(&storage.Stores{}, nil, "", apiDiscardLogger())
	if h == nil {
		t.Error("NewCommandHandler returned nil")
	}
}

// TestNewPrivacyHandler_NonNil 验证 NewPrivacyHandler 返回非 nil。
func TestNewPrivacyHandler_NonNil(t *testing.T) {
	h := NewPrivacyHandler(&storage.Stores{}, apiDiscardLogger())
	if h == nil {
		t.Error("NewPrivacyHandler returned nil")
	}
}

// TestNewReplayHandler_NonNil 验证 NewReplayHandler 返回非 nil。
func TestNewReplayHandler_NonNil(t *testing.T) {
	h := NewReplayHandler(&storage.Stores{}, apiDiscardLogger())
	if h == nil {
		t.Error("NewReplayHandler returned nil")
	}
}

// TestNewSessionHandler_NonNil 验证 NewSessionHandler 返回非 nil。
func TestNewSessionHandler_NonNil(t *testing.T) {
	h := NewSessionHandler(&storage.Stores{}, nil, apiDiscardLogger())
	if h == nil {
		t.Error("NewSessionHandler returned nil")
	}
}

// === health handlers ===

// TestHealthLive_ReturnsAlive 验证 healthLive 返回 200 + status=alive。
func TestHealthLive_ReturnsAlive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz", healthLive)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["status"] != "alive" {
		t.Errorf("status field: got %q, want alive", body["status"])
	}
}

// TestHealthReady_AllOK 验证 healthReady 在全部依赖正常时返回 200 + ready。
func TestHealthReady_AllOK(t *testing.T) {
	// 用真实 Stores 需要 docker,这里跳过如果不可用
	stores := helperStoresForAPITest(t)
	defer stores.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", healthReady(stores))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["status"] != "ready" {
		t.Errorf("status: got %v, want ready", body["status"])
	}
	comps, _ := body["components"].(map[string]any)
	if comps["postgres"] != "ok" {
		t.Errorf("postgres: got %v, want ok", comps["postgres"])
	}
}

// helperStoresForAPITest 返回真实 Stores,不可用 skip。
func helperStoresForAPITest(t *testing.T) *storage.Stores {
	t.Helper()
	stores, err := storage.Connect(context.Background(), helperAPIConfig(t), apiDiscardLogger())
	if err != nil {
		t.Skipf("stores not available: %v", err)
	}
	return stores
}

// helperAPIConfig 构造测试 Config。
func helperAPIConfig(t *testing.T) *config.Config {
	return &config.Config{
		Postgres: config.PostgresConfig{
			Host: "localhost", Port: "7032", User: "mm", Password: "mm_dev",
			Database: "pinconsole", SSLMode: "disable", MaxConns: 5,
		},
		Redis: config.RedisConfig{Addr: "localhost:7079", PoolSize: 5},
		MinIO: config.MinIOConfig{
			Endpoint: "localhost:7000", AccessKey: "mm_dev", SecretKey: "mm_dev_secret",
			Bucket: "test-api-" + uuid.New().String()[:8], UseSSL: false,
		},
	}
}

// === ws.go 纯函数 ===

// TestEventCountOf_SingleEvent 验证 single event 计数为 1。
func TestEventCountOf_SingleEvent(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		TS:   1,
		Payload: proto.EventPayload{
			Type: proto.EvClick,
			TS:   1,
		},
	}
	if got := eventCountOf(env); got != 1 {
		t.Errorf("single event: got %d, want 1", got)
	}
}

// TestEventCountOf_BatchEvents 验证 batch 计数为 array 长度。
func TestEventCountOf_BatchEvents(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		TS:   1,
		Payload: []proto.EventPayload{
			{Type: proto.EvClick, TS: 1},
			{Type: proto.EvMouseMove, TS: 2},
			{Type: proto.EvScroll, TS: 3},
		},
	}
	if got := eventCountOf(env); got != 3 {
		t.Errorf("batch of 3: got %d, want 3", got)
	}
}

// TestEventCountOf_NonEventReturnsZero 验证非 event 类型返回 0。
func TestEventCountOf_NonEventReturnsZero(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgHello,
		TS:   1,
	}
	if got := eventCountOf(env); got != 0 {
		t.Errorf("non-event: got %d, want 0", got)
	}
}

// TestForEachEventPayload_Single 验证 single event 调用 fn 1 次。
func TestForEachEventPayload_Single(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		Payload: proto.EventPayload{
			Type: proto.EvClick,
		},
	}
	count := 0
	forEachEventPayload(env, func(ep proto.EventPayload) {
		count++
		if ep.Type != proto.EvClick {
			t.Errorf("fn got type %v, want EvClick", ep.Type)
		}
	})
	if count != 1 {
		t.Errorf("fn called %d times, want 1", count)
	}
}

// TestForEachEventPayload_Batch 验证 batch 调用 fn N 次。
func TestForEachEventPayload_Batch(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		Payload: []proto.EventPayload{
			{Type: proto.EvClick},
			{Type: proto.EvMouseMove},
		},
	}
	count := 0
	forEachEventPayload(env, func(ep proto.EventPayload) {
		count++
	})
	if count != 2 {
		t.Errorf("fn called %d times, want 2", count)
	}
}

// TestForEachEventPayload_NonEventNoOp 验证非 event 不调用 fn。
func TestForEachEventPayload_NonEventNoOp(t *testing.T) {
	env := proto.Envelope{
		Type: proto.MsgHello,
	}
	count := 0
	forEachEventPayload(env, func(ep proto.EventPayload) {
		count++
	})
	if count != 0 {
		t.Errorf("non-event: fn called %d times, want 0", count)
	}
}

// TestExtractFullSnapshotEnvelope_SingleRRWebType2 验证 single full snapshot 提取。
func TestExtractFullSnapshotEnvelope_SingleRRWebType2(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		TS:   100,
		Payload: proto.EventPayload{
			Type: proto.EvRRWeb,
			TS:   100,
			RRWeb: &proto.RRWebEvent{
				Type: 2, // FullSnapshot
				Data:  map[string]any{"node": "test"},
			},
		},
	}
	got := extractFullSnapshotEnvelope(context.Background(), env)
	if got == nil {
		t.Fatal("got nil, want envelope bytes")
	}
	// 验证返回的是合法 envelope
	decoded, err := proto.Decode(got)
	if err != nil {
		t.Fatalf("Decode returned bytes: %v", err)
	}
	if decoded.Type != proto.MsgEvent {
		t.Errorf("decoded type: got %v, want MsgEvent", decoded.Type)
	}
}

// TestExtractFullSnapshotEnvelope_NonEventReturnsNil 验证非 event 返回 nil。
func TestExtractFullSnapshotEnvelope_NonEventReturnsNil(t *testing.T) {
	env := proto.Envelope{
		Type: proto.MsgHello,
	}
	got := extractFullSnapshotEnvelope(context.Background(), env)
	if got != nil {
		t.Errorf("non-event: got %v, want nil", got)
	}
}

// TestExtractFullSnapshotEnvelope_NonFullSnapshotReturnsNil 验证非 full snapshot 返回 nil。
func TestExtractFullSnapshotEnvelope_NonFullSnapshotReturnsNil(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		Payload: proto.EventPayload{
			Type: proto.EvRRWeb,
			RRWeb: &proto.RRWebEvent{
				Type: 3, // IncrementalSnapshot,不是 full snapshot
			},
		},
	}
	got := extractFullSnapshotEnvelope(context.Background(), env)
	if got != nil {
		t.Errorf("incremental snapshot: got %v, want nil", got)
	}
}

// TestExtractFullSnapshotEnvelope_BatchExtractsFirst 验证 batch 提取第一个 full snapshot。
func TestExtractFullSnapshotEnvelope_BatchExtractsFirst(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		TS:   100,
		Payload: []proto.EventPayload{
			{Type: proto.EvClick},
			{
				Type: proto.EvRRWeb,
				TS:   200,
				RRWeb: &proto.RRWebEvent{
					Type: 2, // FullSnapshot
				},
			},
		},
	}
	got := extractFullSnapshotEnvelope(context.Background(), env)
	if got == nil {
		t.Fatal("batch with full snapshot: got nil, want envelope bytes")
	}
}

// TestExtractFullSnapshotEnvelope_NoFullInBatchReturnsNil 验证 batch 无 full snapshot 返回 nil。
func TestExtractFullSnapshotEnvelope_NoFullInBatchReturnsNil(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		Payload: []proto.EventPayload{
			{Type: proto.EvClick},
			{Type: proto.EvMouseMove},
		},
	}
	got := extractFullSnapshotEnvelope(context.Background(), env)
	if got != nil {
		t.Errorf("batch without full: got %v, want nil", got)
	}
}

// === extractMetaEnvelope ===
// 2026-06-21 新增:服务端需缓存 meta event (rrweb type=4) 让 admin 收到完整事件流,
// 否则 rrweb-player 收不到 meta 无法触发 handleResize,iframe 始终 display:none。

// TestExtractMetaEnvelope_SingleMeta 验证 single meta event (type=4) 提取。
func TestExtractMetaEnvelope_SingleMeta(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		TS:   100,
		Payload: proto.EventPayload{
			Type: proto.EvRRWeb,
			TS:   100,
			RRWeb: &proto.RRWebEvent{
				Type: 4, // Meta
				Data: map[string]any{
					"href":   "https://example.com",
					"width":  1579,
					"height": 904,
				},
			},
		},
	}
	got := extractMetaEnvelope(context.Background(), env)
	if got == nil {
		t.Fatal("got nil, want envelope bytes for meta event")
	}
	decoded, err := proto.Decode(got)
	if err != nil {
		t.Fatalf("decoded failed: %v", err)
	}
	// 验证返回的 envelope payload 是 meta event
	var ep proto.EventPayload
	if err := proto.DecodePayload(decoded.Payload, &ep); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if ep.RRWeb == nil || ep.RRWeb.Type != 4 {
		t.Errorf("not a meta event: %+v", ep.RRWeb)
	}
}

// TestExtractMetaEnvelope_NonEventReturnsNil 验证非 event 类型返回 nil。
func TestExtractMetaEnvelope_NonEventReturnsNil(t *testing.T) {
	env := proto.Envelope{Type: proto.MsgHello}
	if got := extractMetaEnvelope(context.Background(), env); got != nil {
		t.Errorf("non-event: got %v, want nil", got)
	}
}

// TestExtractMetaEnvelope_FullSnapshotReturnsNil 验证 full snapshot (type=2) 不被当 meta 提取。
func TestExtractMetaEnvelope_FullSnapshotReturnsNil(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		Payload: proto.EventPayload{
			Type: proto.EvRRWeb,
			RRWeb: &proto.RRWebEvent{
				Type: 2, // FullSnapshot, 不是 Meta
			},
		},
	}
	if got := extractMetaEnvelope(context.Background(), env); got != nil {
		t.Errorf("full snapshot误识为 meta: got %v, want nil", got)
	}
}

// TestExtractMetaEnvelope_BatchExtractsMeta 验证 batch 中提取 meta event。
func TestExtractMetaEnvelope_BatchExtractsMeta(t *testing.T) {
	env := proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgEvent,
		TS:   100,
		Payload: []proto.EventPayload{
			{
				Type: proto.EvRRWeb,
				TS:   100,
				RRWeb: &proto.RRWebEvent{
					Type: 4, // Meta
					Data: map[string]any{"width": 1024, "height": 768},
				},
			},
			{Type: proto.EvClick},
			{
				Type: proto.EvRRWeb,
				RRWeb: &proto.RRWebEvent{
					Type: 2, // FullSnapshot
				},
			},
		},
	}
	got := extractMetaEnvelope(context.Background(), env)
	if got == nil {
		t.Fatal("batch with meta: got nil, want envelope bytes")
	}
}

// === buildCommandPayload ===

// TestBuildCommandPayload_AllTypes 验证所有命令类型 payload 解析。
func TestBuildCommandPayload_AllTypes(t *testing.T) {
	cases := []struct {
		name     string
		reqType  string
		payload  string
	}{
		{"cursor_highlight", "cursor_highlight", `{"x":100,"y":200,"name":"op"}`},
		{"click", "click", `{"node_id":5,"x":10,"y":20}`},
		{"scroll", "scroll", `{"x":0,"y":500}`},
		{"fill_input", "fill_input", `{"node_id":3,"value":"hello"}`},
		{"navigate", "navigate", `{"url":"https://example.com"}`},
		{"show_popup", "show_popup", `{"title":"t","body":"b","dismissible":true}`},
		{"release_control", "release_control", `null`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cp, err := buildCommandPayload(commandRequest{
				Type:    tc.reqType,
				Payload: []byte(tc.payload),
			})
			if err != nil {
				t.Fatalf("buildCommandPayload(%s): %v", tc.reqType, err)
			}
			if cp.Type != tc.reqType {
				t.Errorf("Type: got %q, want %q", cp.Type, tc.reqType)
			}
			if cp.TS == 0 {
				t.Errorf("TS should be non-zero")
			}
		})
	}
}

// TestBuildCommandPayload_UnknownTypeReturnsError 验证未知类型返回 error。
func TestBuildCommandPayload_UnknownTypeReturnsError(t *testing.T) {
	_, err := buildCommandPayload(commandRequest{
		Type:    "totally_unknown_command",
		Payload: []byte(`{}`),
	})
	if err == nil {
		t.Error("unknown type: expected error, got nil")
	}
}

// TestBuildCommandPayload_InvalidJSONReturnsError 验证无效 JSON 返回 error。
func TestBuildCommandPayload_InvalidJSONReturnsError(t *testing.T) {
	_, err := buildCommandPayload(commandRequest{
		Type:    "cursor_highlight",
		Payload: []byte(`not valid json`),
	})
	if err == nil {
		t.Error("invalid JSON: expected error, got nil")
	}
}

// === newStaticHandler + devHint ===

// TestNewStaticHandler_DevModeReturnsBasic 验证 dev 模式返回 basic handler(无 admin FS 加载)。
func TestNewStaticHandler_DevModeReturnsBasic(t *testing.T) {
	// 用 nil embedded(测试 release=false 分支)
	h := newStaticHandler(nil, false)
	if h == nil {
		t.Fatal("newStaticHandler returned nil")
	}
	if h.release {
		t.Error("dev mode: release should be false")
	}
}

// TestDevHint_ReturnsHintJSON 验证 devHint 返回 503 + hint 字段。
func TestDevHint_ReturnsHintJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	staticH := &staticHandler{}
	r := gin.New()
	r.GET("/admin/*any", staticH.devHint("/admin", "run pnpm --filter @pinconsole/admin build"))

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["hint"] == "" {
		t.Error("hint field is empty")
	}
	if body["path"] != "/admin" {
		t.Errorf("path: got %q, want /admin", body["path"])
	}
}
