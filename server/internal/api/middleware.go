// Package api：认证中间件（切片 1h + 1k fail-secure）。
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware 检查 cookie 中的 session_id，从 Redis 查 user_id，注入 context。
// 未认证返回 401。
//
// 1k fail-secure：dev 模式（SERVER_ENV=dev）自动绕过便于 e2e 测试，
// 但绕过代码本身在 release 构建下不存在（//go:build !release），
// 因此 release 二进制结构上无法走 dev bypass，即使误配 SERVER_ENV=dev 也安全。
//
// dev bypass 仅在"无 cookie"时生效:有 cookie 时仍走真实 session 校验,
// 这样浏览器登录后的会话能正常解析出 user_id,避免 /api/auth/me 永远
// 401 user_not_found(dev 模式刷新即登出的根因)。
func AuthMiddleware(getSession func(ctx context.Context, key string) ([]byte, error), devMode bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// dev bypass 仅在 dev build 中编译进来（见 bypass_dev.go）
		if devMode {
			if cookie, err := c.Cookie(sessionCookieName); err != nil || cookie == "" {
				if tryDevBypass(c) {
					return
				}
			}
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
