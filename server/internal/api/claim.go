// Package api：claim/release 锁端点（切片 1h + 1k fail-secure）。
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/logging"
	"github.com/iannil/marketing-monitor/internal/observability"
	"github.com/iannil/marketing-monitor/internal/storage"
)

const claimTTL = 5 * time.Minute

// releaseClaimLua 是原子 release Lua 脚本：仅当 Redis 中 owner == caller UID 时才 DEL。
// 防止运营 A 误释放运营 B 的 claim（1k P0-4）。
const releaseClaimLua = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
else
	return 0
end
`

type ClaimHandler struct {
	// 1ai-e:用接口替代具体 *storage.Stores,解锁 mock 注入。
	// *storage.Postgres / *storage.Redis 自动满足。
	sessionRepo claimSessionRepo
	redis       claimRedisStore
	logger      *slog.Logger
}

// NewClaimHandler 创建 claim handler。
//
// 1ai-e:签名不变(仍接受 *storage.Stores),内部抽取 PG/Redis 适配接口。
func NewClaimHandler(stores *storage.Stores, logger *slog.Logger) *ClaimHandler {
	return &ClaimHandler{
		sessionRepo: stores.PG,
		redis:       stores.Redis,
		logger:      logger,
	}
}

func (h *ClaimHandler) Register(r gin.IRoutes) {
	r.POST("/api/sessions/:id/claim", h.claim)
	r.POST("/api/sessions/:id/release", h.release)
	r.GET("/api/sessions/:id/claim", h.getClaim)
}

func claimKey(sessionID uuid.UUID) string {
	return fmt.Sprintf("claim:session:%s", sessionID)
}

// claim 原子获取 session 锁（1k P0-4 修复 TOCTOU race）。
//
// 流程：
//  1. 校验 PG session 存在且未结束（404 / 409）
//  2. SET NX EX 300 原子获取（NX 失败时返回当前 owner 409）
func (h *ClaimHandler) claim(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 1s:Lifecycle 全链路埋点
	logger := logging.FromContext(ctx, h.logger)
	defer observability.Lifecycle(ctx, "Claim", logger)()

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return
	}

	// user_id（AuthMiddleware 注入；dev bypass 模式下为 uuid.Nil）
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uuid.UUID)

	// 1. PG session 存在性 + 未结束
	sess, err := h.sessionRepo.GetSession(ctx, sessionID)
	if err != nil || sess == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
		return
	}
	if sess.EndedAt.Valid {
		c.JSON(http.StatusConflict, gin.H{"error": "session_ended"})
		return
	}

	// 2. SET NX 原子获取
	ok, err := h.redis.SetNX(ctx, claimKey(sessionID), []byte(uid.String()), claimTTL)
	if err != nil {
		h.logger.ErrorContext(ctx, "claim SetNX failed", "session_id", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "claim_setnx_failed"})
		return
	}
	if !ok {
		// NX 失败 → 已被 claim；返回当前 owner
		ownerVal, _ := h.redis.Get(ctx, claimKey(sessionID))
		ownerUID := ""
		if ownerVal != nil {
			if parsed, err := uuid.Parse(string(ownerVal)); err == nil {
				ownerUID = parsed.String()
			}
		}
		c.JSON(http.StatusConflict, gin.H{
			"error":      "already_claimed",
			"claimed_by": ownerUID,
		})
		return
	}

	h.logger.InfoContext(ctx, "claim acquired", "session_id", sessionID, "user_id", uid)
	c.JSON(http.StatusOK, gin.H{"ok": true, "session_id": sessionID.String(), "claimed_by": uid.String()})
}

// release 原子释放（仅 owner 可释放，1k P0-4 修复 race）。
func (h *ClaimHandler) release(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 1s:Lifecycle
	logger := logging.FromContext(ctx, h.logger)
	defer observability.Lifecycle(ctx, "Release", logger)()

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return
	}

	userID, _ := c.Get("user_id")
	uid, _ := userID.(uuid.UUID)

	// Lua 原子：仅当 Redis owner == uid 才 DEL，返回删除条数（0 = 不匹配/不存在）
	result, err := h.redis.EvalLua(ctx, releaseClaimLua,
		[]string{claimKey(sessionID)}, uid.String())
	if err != nil {
		h.logger.ErrorContext(ctx, "release EvalLua failed", "session_id", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "release_failed"})
		return
	}
	deleted, _ := result.(int64)
	if deleted == 0 {
		// owner 不匹配或 key 不存在
		c.JSON(http.StatusForbidden, gin.H{"error": "not_claim_owner"})
		return
	}

	h.logger.InfoContext(ctx, "claim released", "session_id", sessionID, "user_id", uid)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// getClaim 查询当前 claim owner。任何认证用户均可查（不暴露 caller UID，仅 owner UID）。
func (h *ClaimHandler) getClaim(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return
	}

	val, err := h.redis.Get(ctx, claimKey(sessionID))
	if err != nil {
		h.logger.ErrorContext(ctx, "getClaim failed", "session_id", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "claim_lookup_failed"})
		return
	}
	if val == nil {
		c.JSON(http.StatusOK, gin.H{"claimed": false})
		return
	}
	ownerUID := ""
	if parsed, err := uuid.Parse(string(val)); err == nil {
		ownerUID = parsed.String()
	}
	c.JSON(http.StatusOK, gin.H{"claimed": true, "claimed_by": ownerUID})
}
