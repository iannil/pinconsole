// Package api：claim/release 锁端点（切片 1h）。
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
)

const claimTTL = 5 * time.Minute

type ClaimHandler struct {
	stores *storage.Stores
	logger *slog.Logger
}

func NewClaimHandler(stores *storage.Stores, logger *slog.Logger) *ClaimHandler {
	return &ClaimHandler{stores: stores, logger: logger}
}

func (h *ClaimHandler) Register(r gin.IRoutes) {
	r.POST("/api/sessions/:id/claim", h.claim)
	r.POST("/api/sessions/:id/release", h.release)
	r.GET("/api/sessions/:id/claim", h.getClaim)
}

func claimKey(sessionID uuid.UUID) string {
	return fmt.Sprintf("claim:session:%s", sessionID)
}

func (h *ClaimHandler) claim(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return
	}

	userID, _ := c.Get("user_id")
	uid, ok := userID.(uuid.UUID)
	if !ok {
		uid = uuid.Nil // dev mode
	}

	// 检查是否已被其他运营 claim
	existing, _ := h.stores.Redis.Get(ctx, claimKey(sessionID))
	if existing != nil {
		var existingUID uuid.UUID
		fmt.Sscanf(string(existing), "%s", &existingUID)
		if existingUID != uid {
			c.JSON(http.StatusConflict, gin.H{"error": "already_claimed", "claimed_by": string(existing)})
			return
		}
	}

	// claim
	_ = h.stores.Redis.Set(ctx, claimKey(sessionID), []byte(uid.String()), claimTTL)
	c.JSON(http.StatusOK, gin.H{"ok": true, "session_id": sessionID.String(), "claimed_by": uid.String()})
}

func (h *ClaimHandler) release(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return
	}

	_ = h.stores.Redis.Del(ctx, claimKey(sessionID))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *ClaimHandler) getClaim(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return
	}

	val, _ := h.stores.Redis.Get(ctx, claimKey(sessionID))
	if val == nil {
		c.JSON(http.StatusOK, gin.H{"claimed": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"claimed": true, "claimed_by": string(val)})
}
