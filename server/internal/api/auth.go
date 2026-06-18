// Package api：认证 handler（切片 1h）。
// POST /api/auth/login + POST /api/auth/logout + GET /api/auth/me
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "mm_session"
	sessionTTL         = 24 * time.Hour
)

type AuthHandler struct {
	stores       *storage.Stores
	logger       *slog.Logger
	secureCookie bool // 1k：prod 模式下 cookie Secure=true
}

// NewAuthHandler 创建 auth handler。
// secureCookie 控制是否在 SetCookie 时启用 Secure flag（prod 模式下必须 true）。
func NewAuthHandler(stores *storage.Stores, logger *slog.Logger, secureCookie bool) *AuthHandler {
	return &AuthHandler{stores: stores, logger: logger, secureCookie: secureCookie}
}

func (h *AuthHandler) Register(r gin.IRoutes) {
	r.POST("/api/auth/login", h.login)
	r.POST("/api/auth/logout", h.logout)
	r.GET("/api/auth/me", h.me)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type meResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

func (h *AuthHandler) login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	// 查 user
	user, err := h.stores.PG.GetUserByEmail(ctx, storage.DefaultTenantID, req.Email)
	if err != nil {
		h.logger.WarnContext(ctx, "login: user not found", "email", req.Email, "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	// 创建 session
	sessionID := generateSessionID()
	sessionKey := sessionRedisKey(sessionID)
	sessionVal := user.ID.String()
	if err := h.stores.Redis.Set(ctx, sessionKey, []byte(sessionVal), sessionTTL); err != nil {
		h.logger.ErrorContext(ctx, "login: redis set failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session_error"})
		return
	}

	// 设置 cookie
	// 1k：prod 模式 Secure=true（HTTPS only）
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, sessionID, int(sessionTTL.Seconds()), "/", "", h.secureCookie, true)

	h.logger.InfoContext(ctx, "login success", "user_id", user.ID, "email", user.Email)

	c.JSON(http.StatusOK, meResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	})
}

func (h *AuthHandler) logout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	sessionID, err := c.Cookie(sessionCookieName)
	if err == nil && sessionID != "" {
		_ = h.stores.Redis.Del(ctx, sessionRedisKey(sessionID))
	}
	c.SetCookie(sessionCookieName, "", -1, "/", "", h.secureCookie, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not_authenticated"})
		return
	}
	uid := userID.(uuid.UUID)
	user, err := h.stores.PG.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_not_found"})
		return
	}
	c.JSON(http.StatusOK, meResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	})
}

// generateSessionID 生成 32 字节 hex session ID。
func generateSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func sessionRedisKey(sessionID string) string {
	return "auth:session:" + sessionID
}
