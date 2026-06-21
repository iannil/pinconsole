// Go-1 切片补测:LifecycleWithArgs + LogPoint/LogExternalCall extras 路径,
// 提升覆盖率 83.3% → 100%。
package observability

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestLifecycleWithArgs_OptionCreatesConfig 验证 LifecycleWithArgs 直接调用设置 args。
func TestLifecycleWithArgs_OptionCreatesConfig(t *testing.T) {
	opt := LifecycleWithArgs("arg1", 42, true)
	cfg := &lifecycleConfig{}
	opt(cfg)

	if len(cfg.args) != 3 {
		t.Fatalf("args len: got %d, want 3", len(cfg.args))
	}
	if cfg.args[0] != "arg1" {
		t.Errorf("args[0]: got %v, want arg1", cfg.args[0])
	}
	if cfg.args[1] != 42 {
		t.Errorf("args[1]: got %v, want 42", cfg.args[1])
	}
	if cfg.args[2] != true {
		t.Errorf("args[2]: got %v, want true", cfg.args[2])
	}
}

// TestLifecycle_WithArgsOption 验证 Lifecycle 传 LifecycleWithArgs 不 panic
// (虽然当前实现未真正读 args,但 option 应被接受且不破坏 lifecycle)。
func TestLifecycle_WithArgsOption(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)
	ctx := context.Background()

	done := Lifecycle(ctx, "WithArgsHandler", logger, LifecycleWithArgs("visitor_id", "s-123"))
	// defer 触发结束函数
	done()

	logs := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 log entries (start+end), got %d", len(logs))
	}
	// 第一行是 Function_Start,最后一行是 Function_End
	if !strings.Contains(logs[0], string(EventFunctionStart)) {
		t.Errorf("first log: missing %s, got %q", EventFunctionStart, logs[0])
	}
	if !strings.Contains(logs[len(logs)-1], string(EventFunctionEnd)) {
		t.Errorf("last log: missing %s, got %q", EventFunctionEnd, logs[len(logs)-1])
	}
}

// TestLogPoint_WithMultipleExtras 验证 LogPoint 多 extras 字段正确写入。
func TestLogPoint_WithMultipleExtras(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)
	ctx := context.Background()

	LogPoint(ctx, logger, EventBranch, "navigate_check",
		"url_allowed", true,
		"reason", "match_host",
		"latency_ms", int64(5),
	)

	logs := parseLogs(&buf)
	if len(logs) != 1 {
		t.Fatalf("log entries: got %d, want 1", len(logs))
	}
	m := logs[0]
	if m["event_type"] != string(EventBranch) {
		t.Errorf("event_type: got %v, want %v", m["event_type"], EventBranch)
	}
	if m["span"] != "navigate_check" {
		t.Errorf("span: got %v, want navigate_check", m["span"])
	}
	if m["url_allowed"] != true {
		t.Errorf("url_allowed: got %v, want true", m["url_allowed"])
	}
	if m["reason"] != "match_host" {
		t.Errorf("reason: got %v, want match_host", m["reason"])
	}
}

// TestLogExternalCall_WithMultipleExtras 验证 LogExternalCall 多 extras 字段。
func TestLogExternalCall_WithMultipleExtras(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)
	ctx := context.Background()

	LogExternalCall(ctx, logger, "minio.PutObject", "ok",
		"size_bytes", int64(1024),
		"duration_ms", int64(15),
		"bucket", "pinconsole",
	)

	logs := parseLogs(&buf)
	if len(logs) != 1 {
		t.Fatalf("log entries: got %d, want 1", len(logs))
	}
	m := logs[0]
	if m["event_type"] != string(EventExternalCall) {
		t.Errorf("event_type: got %v, want %v", m["event_type"], EventExternalCall)
	}
	if m["target"] != "minio.PutObject" {
		t.Errorf("target: got %v, want minio.PutObject", m["target"])
	}
	if m["status"] != "ok" {
		t.Errorf("status: got %v, want ok", m["status"])
	}
	if m["bucket"] != "pinconsole" {
		t.Errorf("bucket: got %v, want pinconsole", m["bucket"])
	}
}

// TestLogPoint_NoExtras 验证 LogPoint 无 extras 也能正确写入。
func TestLogPoint_NoExtras(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	LogPoint(context.Background(), logger, EventFunctionStart, "SoloHandler")

	logs := parseLogs(&buf)
	if len(logs) != 1 {
		t.Fatalf("log entries: got %d, want 1", len(logs))
	}
	if logs[0]["event_type"] != string(EventFunctionStart) {
		t.Errorf("event_type: got %v, want %v", logs[0]["event_type"], EventFunctionStart)
	}
}

// TestLogExternalCall_NoExtras 验证 LogExternalCall 无 extras 也能正确写入。
func TestLogExternalCall_NoExtras(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	LogExternalCall(context.Background(), logger, "redis.Set", "ok")

	logs := parseLogs(&buf)
	if len(logs) != 1 {
		t.Fatalf("log entries: got %d, want 1", len(logs))
	}
	if logs[0]["target"] != "redis.Set" {
		t.Errorf("target: got %v, want redis.Set", logs[0]["target"])
	}
}

// TestLifecycle_LoggerNil 但 opts 非空,验证 nil logger 下 opts 不破坏流程。
func TestLifecycle_NilLoggerWithOptions(t *testing.T) {
	ctx := context.Background()

	// 不应 panic
	done := Lifecycle(ctx, "NilLoggerHandler", nil, LifecycleWithArgs("x"))
	done()
}
