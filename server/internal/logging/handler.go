// Package logging 提供 slog 日志处理器与 trace_id 上下文传播。
//
// 约定（详见 CLAUDE.md "可观测性开发"）：
//   - 所有日志结构化 JSON
//   - 字段含 timestamp, level, msg, trace_id, span_id, event_type, payload
//   - trace_id 从 context 中读取，由中间件注入
package logging

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey int

const (
	ctxKeyTraceID ctxKey = iota
	ctxKeySpanID
)

// NewLogger 创建 slog.Logger。
// dev 模式输出 TextHandler（彩色，便于本地查看），其余为 JSONHandler。
func NewLogger(level, env string) *slog.Logger {
	lvl := parseLevel(level)
	opts := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: env == "dev",
	}

	var handler slog.Handler
	if env == "dev" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithTraceID 将 trace_id 注入 context。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKeyTraceID, traceID)
}

// WithSpanID 将 span_id 注入 context。
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, ctxKeySpanID, spanID)
}

// TraceID 从 context 取出 trace_id（若不存在返回空字符串）。
func TraceID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyTraceID).(string)
	return v
}

// SpanID 从 context 取出 span_id。
func SpanID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeySpanID).(string)
	return v
}

// FromContext 返回附带 trace_id 与 span_id 的 logger。
// 若 ctx 无 trace_id 则返回原 logger。
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	traceID := TraceID(ctx)
	spanID := SpanID(ctx)
	if traceID == "" && spanID == "" {
		return base
	}
	attrs := make([]any, 0, 4)
	if traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}
	if spanID != "" {
		attrs = append(attrs, "span_id", spanID)
	}
	return base.With(attrs...)
}
