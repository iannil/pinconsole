// Package api:WebSocket 鉴权 helper(1ac-final T0-1h-2 修复)。
//
// operatorWS 必须校验 cookie session,否则任意匿名客户端可连 /ws/operator
// 接收全部 visitor 事件流(录像 / co-browsing / 聊天)。
//
// 为什么不用 AuthMiddleware:
//   - gin group middleware 在 handler 前跑,但 WS upgrade 在 handler 内
//   - 中间件失败时 AbortWithStatusJSON 写 JSON body,但 WS 客户端期望 401 状态码
//   - 拆 visitorWS(public)和 operatorWS(protected)需要拆 Register 方法
//
// 解决:handler 内显式调用 authenticateOperatorWS,失败时直接返回 401(不 Accept)。
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
)

// authenticateOperatorWS 校验调用方 cookie session。
//
// 返回 (user_id, ok)。
// ok=false 时已写 401 响应,调用方应直接 return(不要 websocket.Accept)。
//
// 流程:
//  1. dev build + devMode → 注入 uuid.Nil,放行(便于 e2e)
//  2. 读 mm_session cookie,缺失 → 401 no_session
//  3. Redis 查 auth:session:{id},缺失/错 → 401 invalid_session
//  4. 解析 user_id UUID,失败 → 401 invalid_user_id
func authenticateOperatorWS(c *gin.Context, rdb *storage.Redis, devMode bool) (uuid.UUID, bool) {
	// 1. dev bypass(与 AuthMiddleware 一致,release build 编译为 false)
	if devMode && tryDevBypass(c) {
		return uuid.Nil, true
	}

	// 2. cookie
	sessionID, err := c.Cookie(sessionCookieName)
	if err != nil || sessionID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no_session"})
		return uuid.Nil, false
	}

	// 3. Redis 查
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	val, err := rdb.Get(ctx, sessionRedisKey(sessionID))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
		return uuid.Nil, false
	}
	if val == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
		return uuid.Nil, false
	}

	// 4. 解析 user_id
	userID, err := uuid.Parse(string(val))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_user_id"})
		return uuid.Nil, false
	}

	c.Set("user_id", userID)
	return userID, true
}
