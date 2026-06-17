// Package logging：HTTP 中间件，为每个请求注入 trace_id。
package logging

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// newID 生成 16 字节 hex ID（32 字符）。
// 切片 1a 用 crypto/rand；后续可换 ULID/Snowflake（见 spec 推迟清单）。
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 极罕见；fallback 用时间戳
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}

// TraceMiddleware 为每个请求注入 trace_id + span_id，并放入 gin.Context。
// gin.Context 底层 context.Context 用于 slog.WithContext。
func TraceMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = newID()
		}
		spanID := newID()

		ctx := WithTraceID(c.Request.Context(), traceID)
		ctx = WithSpanID(ctx, spanID)
		c.Request = c.Request.WithContext(ctx)

		c.Set("trace_id", traceID)
		c.Set("span_id", spanID)

		// 响应头也带上，便于客户端关联日志
		c.Header("X-Trace-Id", traceID)

		start := time.Now()
		c.Next()

		reqLogger := FromContext(c.Request.Context(), logger)
		reqLogger.Info("http_request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		)
	}
}
