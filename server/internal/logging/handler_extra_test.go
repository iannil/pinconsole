// Go-2 切片补测:NewLogger + parseLevel 边界,
// 提升覆盖率 79.6% → ≥90%。
package logging

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// TestNewLogger_DevReturnsTextHandler 验证 dev 模式返回 TextHandler。
// 通过捕获 stdout + 验证输出格式(文本不是 JSON)间接判断。
func TestNewLogger_DevReturnsTextHandler(t *testing.T) {
	// 暂存原 stdout
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	// 暂存原 default logger
	origDefault := slog.Default()
	defer func() {
		os.Stdout = origStdout
		slog.SetDefault(origDefault)
	}()

	logger := NewLogger("debug", "dev")
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	// 写一条日志
	logger.Info("dev_test_msg")

	// 恢复 stdout 并读取
	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	// TextHandler 输出格式: time=... level=INFO msg="dev_test_msg"
	// JSONHandler 输出格式: {"time":...,"level":"INFO","msg":"dev_test_msg"}
	if !strings.Contains(out, "msg=") {
		t.Errorf("dev mode should use TextHandler (msg= key=value), got: %s", out)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("dev mode should NOT use JSONHandler, got JSON: %s", out)
	}
}

// TestNewLogger_ProdReturnsJSONHandler 验证非 dev 模式返回 JSONHandler。
func TestNewLogger_ProdReturnsJSONHandler(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	origDefault := slog.Default()
	defer func() {
		os.Stdout = origStdout
		slog.SetDefault(origDefault)
	}()

	logger := NewLogger("info", "prod")
	logger.Info("prod_test_msg")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := strings.TrimSpace(buf.String())

	if !strings.HasPrefix(out, "{") {
		t.Errorf("prod mode should use JSONHandler (output starts with '{'), got: %s", out)
	}
	if !strings.Contains(out, "\"msg\":\"prod_test_msg\"") {
		t.Errorf("prod mode output missing msg field, got: %s", out)
	}
}

// TestNewLogger_SetsDefault 验证 NewLogger 调用 slog.SetDefault 设置全局 logger。
func TestNewLogger_SetsDefault(t *testing.T) {
	origDefault := slog.Default()
	defer slog.SetDefault(origDefault)

	NewLogger("info", "prod")
	// slog.Default() 应该返回 NewLogger 创建的 logger
	// 不能直接比较指针(slog.Default 可能返回包装),通过行为验证
	if slog.Default() == origDefault {
		t.Errorf("slog.Default() should be updated by NewLogger")
	}
}

// TestNewLogger_DevLevelDebug 验证 dev 模式 level=debug 能输出 Debug 日志。
func TestNewLogger_DevLevelDebug(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	origDefault := slog.Default()
	defer func() {
		os.Stdout = origStdout
		slog.SetDefault(origDefault)
	}()

	logger := NewLogger("debug", "dev")
	logger.Debug("debug_visible")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "debug_visible") {
		t.Errorf("debug level should output Debug msg, got: %s", out)
	}
}

// TestNewLogger_ProdLevelInfo 验证 prod 模式 level=info 屏蔽 Debug 日志。
func TestNewLogger_ProdLevelInfo(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	origDefault := slog.Default()
	defer func() {
		os.Stdout = origStdout
		slog.SetDefault(origDefault)
	}()

	logger := NewLogger("info", "prod")
	logger.Debug("debug_should_be_hidden")
	logger.Info("info_should_be_visible")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	if strings.Contains(out, "debug_should_be_hidden") {
		t.Errorf("info level should hide Debug msg, got: %s", out)
	}
	if !strings.Contains(out, "info_should_be_visible") {
		t.Errorf("info level should show Info msg, got: %s", out)
	}
}

// TestNewLogger_DevAddSource 验证 dev 模式 AddSource=true(输出含 source)。
func TestNewLogger_DevAddSource(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	origDefault := slog.Default()
	defer func() {
		os.Stdout = origStdout
		slog.SetDefault(origDefault)
	}()

	logger := NewLogger("info", "dev")
	logger.Info("with_source")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	// AddSource=true 时 TextHandler 输出含 source=<file>:<line>
	if !strings.Contains(out, "source=") {
		t.Errorf("dev mode AddSource=true should output source=, got: %s", out)
	}
}

// TestNewLogger_ProdNoAddSource 验证 prod 模式 AddSource=false(无 source 字段)。
func TestNewLogger_ProdNoAddSource(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	origDefault := slog.Default()
	defer func() {
		os.Stdout = origStdout
		slog.SetDefault(origDefault)
	}()

	logger := NewLogger("info", "prod")
	logger.Info("no_source")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	if strings.Contains(out, `"source":`) {
		t.Errorf("prod mode AddSource=false should not output source, got: %s", out)
	}
}

// TestNewLogger_TestEnvUsesJSON 验证 env=test 等非 dev 值也用 JSONHandler。
func TestNewLogger_TestEnvUsesJSON(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	origDefault := slog.Default()
	defer func() {
		os.Stdout = origStdout
		slog.SetDefault(origDefault)
	}()

	for _, env := range []string{"test", "staging", "production", ""} {
		// 重新 redirect 每次
		logger := NewLogger("info", env)
		logger.Info("env_" + env)

		w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		out := strings.TrimSpace(buf.String())

		if !strings.HasPrefix(out, "{") {
			t.Errorf("env=%q should use JSONHandler, got: %s", env, out)
		}

		// 重新 setup pipe for next iteration
		r, w, _ = os.Pipe()
		os.Stdout = w
		_ = logger
	}
	w.Close()
}

// TestNewLogger_FromContextAfterInit 验证 NewLogger 创建的 logger 与 FromContext 协同工作。
func TestNewLogger_FromContextAfterInit(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	origDefault := slog.Default()
	defer func() {
		os.Stdout = origStdout
		slog.SetDefault(origDefault)
	}()

	logger := NewLogger("info", "prod")
	ctx := WithTraceID(context.Background(), "trace-integration-123")
	enriched := FromContext(ctx, logger)
	enriched.InfoContext(ctx, "with_trace")

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "trace_id") {
		t.Errorf("FromContext + NewLogger should output trace_id, got: %s", out)
	}
	if !strings.Contains(out, "trace-integration-123") {
		t.Errorf("trace_id value missing, got: %s", out)
	}
}
