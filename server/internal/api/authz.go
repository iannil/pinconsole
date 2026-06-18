// Package api：授权检查 helper（1k P0-3）。
//
// claim ownership 检查模式：
//   1. 从 ctx 取 user_id（AuthMiddleware 注入）
//   2. 从 URL 取 session_id
//   3. （可选）查 PG 确认 session 存在且未结束
//   4. 查 Redis claim:session:{id}，比对 owner UID
//
// 失败时写入 HTTP 响应并返回 ok=false，调用方应直接 return。
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
)

// requireClaimOwnership 校验调用方拥有 session 的 claim。
//
// 参数：
//   - requireAliveSession: 是否要求 PG 中 session 存在且 ended_at IS NULL
//     (claim/release 必须；command/chat 可选，因为命令可能针对刚结束的 session)
//
// 返回：
//   - sessionID 解析后的 UUID
//   - callerUID 调用方 user_id
//   - ok 是否通过；若 false 调用方应直接 return
func requireClaimOwnership(
	c *gin.Context,
	stores *storage.Stores,
	logger *slog.Logger,
	requireAliveSession bool,
) (sessionID uuid.UUID, callerUID uuid.UUID, ok bool) {
	// 1. session_id 解析
	idStr := c.Param("id")
	sid, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return uuid.Nil, uuid.Nil, false
	}

	// 2. user_id（AuthMiddleware 注入；dev bypass 模式下为 uuid.Nil）
	uidAny, exists := c.Get("user_id")
	if !exists {
		// AuthMiddleware 应已拒绝匿名；此处兜底
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not_authenticated"})
		return uuid.Nil, uuid.Nil, false
	}
	uid, _ := uidAny.(uuid.UUID)

	// 3. （可选）PG session 存在性
	if requireAliveSession {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		sess, err := stores.PG.GetSession(ctx, sid)
		if err != nil || sess == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
			return uuid.Nil, uuid.Nil, false
		}
		if sess.EndedAt.Valid {
			c.JSON(http.StatusConflict, gin.H{"error": "session_ended"})
			return uuid.Nil, uuid.Nil, false
		}
	}

	// 4. Redis claim ownership
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	claimVal, err := stores.Redis.Get(ctx, claimKey(sid))
	if err != nil {
		logger.ErrorContext(ctx, "claim lookup failed", "session_id", sid, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "claim_lookup_failed"})
		return uuid.Nil, uuid.Nil, false
	}
	if claimVal == nil {
		// 未被任何运营 claim
		c.JSON(http.StatusForbidden, gin.H{"error": "not_claimed"})
		return uuid.Nil, uuid.Nil, false
	}
	ownerUID, err := uuid.Parse(string(claimVal))
	if err != nil {
		// claim 数据损坏
		logger.ErrorContext(ctx, "claim value malformed", "session_id", sid, "raw", string(claimVal))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "claim_corrupt"})
		return uuid.Nil, uuid.Nil, false
	}
	if ownerUID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "not_claim_owner", "claimed_by": ownerUID.String()})
		return uuid.Nil, uuid.Nil, false
	}

	return sid, uid, true
}
