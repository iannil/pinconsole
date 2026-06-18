// 1t 测试:logging 包纯函数 + 中间件行为。
package logging

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},     // 默认
		{"DEBUG", slog.LevelInfo}, // 大小写敏感,默认
		{"unknown", slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := parseLevel(tt.in)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestNewID_LengthAndHex(t *testing.T) {
	id := newID()
	if len(id) != 32 {
		t.Errorf("newID() length = %d, want 32 (16 bytes hex)", len(id))
	}
	// 全部应为 hex 字符
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("newID() contains non-hex char %q in %s", c, id)
			break
		}
	}
}

func TestNewID_UniqueAcrossCalls(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := newID()
		if ids[id] {
			t.Fatalf("newID() duplicate at iteration %d: %s", i, id)
		}
		ids[id] = true
	}
}

func TestWithTraceID_AndTraceID(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-abc-123")
	if got := TraceID(ctx); got != "trace-abc-123" {
		t.Errorf("TraceID() = %q, want 'trace-abc-123'", got)
	}
}

func TestWithSpanID_AndSpanID(t *testing.T) {
	ctx := context.Background()
	ctx = WithSpanID(ctx, "span-xyz")
	if got := SpanID(ctx); got != "span-xyz" {
		t.Errorf("SpanID() = %q, want 'span-xyz'", got)
	}
}

func TestTraceID_EmptyWhenNotSet(t *testing.T) {
	ctx := context.Background()
	if got := TraceID(ctx); got != "" {
		t.Errorf("TraceID() on bare ctx = %q, want empty", got)
	}
}

func TestFromContext_NoTraceID_ReturnsBase(t *testing.T) {
	base := slog.Default()
	ctx := context.Background()
	got := FromContext(ctx, base)
	if got != base {
		t.Errorf("FromContext without trace_id should return base logger")
	}
}

func TestFromContext_WithTraceID_HasAttr(t *testing.T) {
	// 用 test handler 检查 logger 输出含 trace_id
	var buf strings.Builder
	testLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := WithTraceID(context.Background(), "trace-test-123")
	ctx = WithSpanID(ctx, "span-test-456")
	enriched := FromContext(ctx, testLogger)
	enriched.InfoContext(ctx, "test_msg")

	output := buf.String()
	if !strings.Contains(output, "trace_id=trace-test-123") {
		t.Errorf("output should contain trace_id, got: %s", output)
	}
	if !strings.Contains(output, "span_id=span-test-456") {
		t.Errorf("output should contain span_id, got: %s", output)
	}
}

// TestTraceMiddleware 注入 trace_id + span_id 到 gin.Context + 响应头
func TestTraceMiddleware_InjectsIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceMiddleware(slog.Default()))
	var capturedTraceID, capturedSpanID string
	r.GET("/test", func(c *gin.Context) {
		// gin.Context 的 Set 注入
		v1, _ := c.Get("trace_id")
		v2, _ := c.Get("span_id")
		capturedTraceID, _ = v1.(string)
		capturedSpanID, _ = v2.(string)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedTraceID == "" {
		t.Errorf("trace_id should be set in gin.Context")
	}
	if capturedSpanID == "" {
		t.Errorf("span_id should be set in gin.Context")
	}
	if len(capturedTraceID) != 32 {
		t.Errorf("trace_id length = %d, want 32", len(capturedTraceID))
	}
	// 响应头
	if w.Header().Get("X-Trace-Id") != capturedTraceID {
		t.Errorf("X-Trace-Id response header = %q, want %q",
			w.Header().Get("X-Trace-Id"), capturedTraceID)
	}
}

func TestTraceMiddleware_PreservesClientHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceMiddleware(slog.Default()))
	var captured string
	r.GET("/test", func(c *gin.Context) {
		v, _ := c.Get("trace_id")
		captured, _ = v.(string)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Trace-Id", "client-provided-trace-id")
	r.ServeHTTP(httptest.NewRecorder(), req)

	if captured != "client-provided-trace-id" {
		t.Errorf("trace_id = %q, want client-provided-trace-id (preserved from header)", captured)
	}
}
