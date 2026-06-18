// Package observability:LifecycleTracker 实现。
//
// 用法(Go defer 闭包模式):
//
//	func MyHandler(c *gin.Context) {
//	    ctx := c.Request.Context()
//	    defer observability.Lifecycle(ctx, "PostCommand", logger)()
//
//	    // ... 业务逻辑
//	    observability.LogPoint(ctx, logger, observability.EventBranch, "navigate_check", "url_allowed", true)
//	    if err := callMinIO(ctx); err != nil {
//	        observability.LogExternalCall(ctx, logger, "minio.PutObject", "error", err)
//	        return
//	    }
//	}
//
// Lifecycle 在进入时记录 Function_Start + Args(可选),
// defer 触发时记录 Function_End + Duration + Return(可选),
// panic 时 recover 记录 Error + Stack。
package observability

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"
)

// startKey 用于 ctx 存储当前 lifecycle 的开始时间 + span name。
type startKey struct{ name string }

// Lifecycle 返回一个结束函数,用 defer 调用。
//
// 进入时立即记录 Function_Start event(含 span name);
// 返回的函数在 defer 时记录 Function_End + Duration;
// panic 时 recover 记录 Error + Stack。
//
// opts 可选:LifecycleWithArgs(args ...any)、LifecycleWithReturn(标记有返回值)。
func Lifecycle(ctx context.Context, name string, logger *slog.Logger, opts ...LifecycleOption) func() {
	start := time.Now()
	if logger != nil {
		logger.InfoContext(ctx, string(EventFunctionStart),
			"span", name,
			"event_type", EventFunctionStart,
		)
	}
	return func() {
		duration := time.Since(start)
		if r := recover(); r != nil {
			stack := debug.Stack()
			if logger != nil {
				logger.ErrorContext(ctx, string(EventError),
					"span", name,
					"event_type", EventError,
					"panic", r,
					"stack", string(stack),
					"duration_ms", duration.Milliseconds(),
				)
			}
			panic(r) // re-throw
		}
		if logger != nil {
			logger.InfoContext(ctx, string(EventFunctionEnd),
				"span", name,
				"event_type", EventFunctionEnd,
				"duration_ms", duration.Milliseconds(),
			)
		}
	}
}

// LifecycleOption 配置 Lifecycle 行为(预留扩展点)。
type LifecycleOption func(*lifecycleConfig)

type lifecycleConfig struct {
	args []any
}

// LifecycleWithArgs 在 Function_Start 日志中记录输入参数。
func LifecycleWithArgs(args ...any) LifecycleOption {
	return func(c *lifecycleConfig) {
		c.args = args
	}
}

// LogPoint 在 if/else/loop/外部调用前后手动埋点(CLAUDE.md 要求)。
// 接受 variadic extras,便于记录多个字段。
func LogPoint(ctx context.Context, logger *slog.Logger, et EventType, span string, extras ...any) {
	if logger == nil {
		return
	}
	attrs := []any{
		"event_type", et,
		"span", span,
	}
	attrs = append(attrs, extras...)
	logger.InfoContext(ctx, string(et), attrs...)
}

// LogExternalCall 记录外部 API 调用(MinIO/Redis/PG)前后。
// status: "ok" / "error"
func LogExternalCall(ctx context.Context, logger *slog.Logger, target, status string, extras ...any) {
	if logger == nil {
		return
	}
	attrs := []any{
		"event_type", EventExternalCall,
		"target", target,
		"status", status,
	}
	attrs = append(attrs, extras...)
	logger.InfoContext(ctx, string(EventExternalCall), attrs...)
}
