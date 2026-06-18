// 1m 测试:LifecycleTracker 行为覆盖。
package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func parseLogs(buf *bytes.Buffer) []map[string]any {
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func TestLifecycle_Normal_RecordsStartAndEnd(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)
	ctx := context.Background()

	func() {
		defer Lifecycle(ctx, "MyFunc", logger)()
		// 业务逻辑
	}()

	logs := parseLogs(&buf)
	if len(logs) != 2 {
		t.Fatalf("expected 2 log entries (start+end), got %d: %v", len(logs), logs)
	}
	if logs[0]["event_type"] != string(EventFunctionStart) {
		t.Errorf("first log event_type = %v, want %s", logs[0]["event_type"], EventFunctionStart)
	}
	if logs[1]["event_type"] != string(EventFunctionEnd) {
		t.Errorf("second log event_type = %v, want %s", logs[1]["event_type"], EventFunctionEnd)
	}
	if logs[0]["span"] != "MyFunc" {
		t.Errorf("first log span = %v, want MyFunc", logs[0]["span"])
	}
	if logs[1]["duration_ms"] == nil {
		t.Errorf("duration_ms should be present in end log")
	}
}

func TestLifecycle_Panic_RecordsError(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)
	ctx := context.Background()

	defer func() {
		// recover the re-thrown panic
		_ = recover()
	}()

	func() {
		defer Lifecycle(ctx, "PanickyFunc", logger)()
		panic("test panic")
	}()

	logs := parseLogs(&buf)
	// 应有 Function_Start + Error(panic)两条;Function_End 不应记录(panic 路径)
	hasStart := false
	hasError := false
	for _, l := range logs {
		if l["event_type"] == string(EventFunctionStart) {
			hasStart = true
		}
		if l["event_type"] == string(EventError) {
			hasError = true
			if l["panic"] != "test panic" {
				t.Errorf("panic field = %v, want 'test panic'", l["panic"])
			}
			if l["stack"] == nil {
				t.Errorf("stack should be present in error log")
			}
		}
	}
	if !hasStart {
		t.Errorf("should have Function_Start log")
	}
	if !hasError {
		t.Errorf("should have Error log on panic")
	}
}

func TestLifecycle_NilLogger_NoCrash(t *testing.T) {
	ctx := context.Background()
	func() {
		defer Lifecycle(ctx, "Func", nil)()
	}()
	// 不 panic 即通过
}

func TestLogPoint_RecordsEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)
	ctx := context.Background()

	LogPoint(ctx, logger, EventBranch, "PostCommand", "navigate_allowed", true)

	logs := parseLogs(&buf)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}
	if logs[0]["event_type"] != string(EventBranch) {
		t.Errorf("event_type = %v, want %s", logs[0]["event_type"], EventBranch)
	}
	if logs[0]["navigate_allowed"] != true {
		t.Errorf("custom field navigate_allowed = %v, want true", logs[0]["navigate_allowed"])
	}
}

func TestLogExternalCall_OK(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)
	ctx := context.Background()

	LogExternalCall(ctx, logger, "minio.PutObject", "ok", "key", "abc", "size", 1024)

	logs := parseLogs(&buf)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}
	if logs[0]["event_type"] != string(EventExternalCall) {
		t.Errorf("event_type = %v, want %s", logs[0]["event_type"], EventExternalCall)
	}
	if logs[0]["target"] != "minio.PutObject" {
		t.Errorf("target = %v, want minio.PutObject", logs[0]["target"])
	}
	if logs[0]["status"] != "ok" {
		t.Errorf("status = %v, want ok", logs[0]["status"])
	}
}
