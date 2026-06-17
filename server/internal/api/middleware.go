// Package api：认证中间件（切片 1h）。
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware 检查 cookie 中的 session_id，从 Redis 查 user_id，注入 context。
// 未认证返回 401。dev 模式（SERVER_ENV=dev）自动绕过，便于 e2e 测试。
func AuthMiddleware(getSession func(ctx context.Context, key string) ([]byte, error), devMode bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if devMode {
			// dev 模式：自动注入一个 mock user_id
			c.Set("user_id", uuid.Nil)
			c.Set("dev_mode", true)
			c.Next()
			return
		}

		sessionID, err := c.Cookie(sessionCookieName)
		if err != nil || sessionID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no_session"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		val, err := getSession(ctx, sessionRedisKey(sessionID))
		if err != nil || val == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
			return
		}

		userID, err := uuid.Parse(string(val))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_user_id"})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
