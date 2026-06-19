// Package api：认证 handler（切片 1h）。
// POST /api/auth/login + POST /api/auth/logout + GET /api/auth/me
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

	// 1x P1-3:登录暴力破解防护阈值。
	// 同一 email+IP 在 loginLockoutWindow 内失败 loginMaxAttempts 次后,
	// 锁定 loginLockoutWindow(计数器 TTL 自然过期)。
	loginMaxAttempts   = 5
	loginLockoutWindow = 15 * time.Minute
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
	// /api/auth/me 走 protected 组(由 router.go 挂 AuthMiddleware),
	// 否则 user_id 不会注入 context,handler 永远返回 401 not_authenticated。
	// 见 router.go: authH.RegisterMe(protected)
}

// RegisterMe 在 protected 路由组上注册 /api/auth/me。
// 调用方应传入已挂 AuthMiddleware 的 group。
func (h *AuthHandler) RegisterMe(protected gin.IRoutes) {
	protected.GET("/api/auth/me", h.me)
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

	// 1x P1-3:登录暴力破解防护 — 锁定检查(email+IP 双 key)
	clientIP := c.ClientIP()
	throttleKey := loginThrottleKey(req.Email, clientIP)
	if locked, retryAfter, err := h.checkLoginThrottle(ctx, throttleKey); err != nil {
		// Redis 故障 fail-open(与 1i rate limit 一致),仅 warn
		h.logger.WarnContext(ctx, "login throttle check failed (fail-open)", "error", err)
	} else if locked {
		h.logger.WarnContext(ctx, "login rejected: throttle locked",
			"email", req.Email, "client_ip", clientIP, "retry_after_s", retryAfter)
		c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "too_many_attempts",
			"retry_after": retryAfter,
		})
		return
	}

	// 查 user
	user, err := h.stores.PG.GetUserByEmail(ctx, storage.DefaultTenantID, req.Email)
	if err != nil {
		h.logger.WarnContext(ctx, "login: user not found", "email", req.Email, "error", err)
		h.recordLoginFailure(ctx, throttleKey)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.recordLoginFailure(ctx, throttleKey)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	// 1x P1-3:登录成功清零计数器(防偶然失败锁定正常用户)
	if err := h.stores.Redis.Del(ctx, throttleKey); err != nil {
		h.logger.WarnContext(ctx, "login throttle clear failed", "error", err)
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
	h.setSessionCookie(c, sessionID, int(sessionTTL.Seconds()))

	h.logger.InfoContext(ctx, "login success", "user_id", user.ID, "email", user.Email)

	c.JSON(http.StatusOK, meResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	})
}

// checkLoginThrottle 检查 email+IP 是否被锁定。
// 返回 (locked, retryAfter秒, err)。Redis 故障时返回 (false, 0, err) 让调用方 fail-open。
func (h *AuthHandler) checkLoginThrottle(ctx context.Context, key string) (bool, int64, error) {
	val, err := h.stores.Redis.Get(ctx, key)
	if err != nil {
		return false, 0, err
	}
	if val == nil {
		return false, 0, nil // 未记录
	}
	// val 是失败计数 ASCII
	countStr := string(val)
	var count int64
	if _, err := fmt.Sscanf(countStr, "%d", &count); err != nil {
		return false, 0, fmt.Errorf("parse throttle count %q: %w", countStr, err)
	}
	if count < int64(loginMaxAttempts) {
		return false, 0, nil
	}
	// 已锁定;查 TTL 算 retry-after
	ttl, err := h.stores.Redis.Client.TTL(ctx, key).Result()
	if err != nil {
		return true, int64(loginLockoutWindow.Seconds()), nil // 兜底
	}
	if ttl < 0 {
		return true, int64(loginLockoutWindow.Seconds()), nil
	}
	return true, int64(ttl.Seconds()), nil
}

// recordLoginFailure 记录一次登录失败 — INCR 计数器,首次失败时设 TTL。
// Redis 故障静默(fail-open),仅 warn。
func (h *AuthHandler) recordLoginFailure(ctx context.Context, key string) {
	// 用 Lua 原子:INCR + 首次(返回值=1)时 EXPIRE
	const luaScript = `local c = redis.call('INCR', KEYS[1])
if c == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return c`
	if _, err := h.stores.Redis.EvalLua(ctx, luaScript,
		[]string{key}, int(loginLockoutWindow.Seconds())); err != nil {
		h.logger.WarnContext(ctx, "record login failure failed (fail-open)", "error", err)
	}
}

// loginThrottleKey 构造 Redis key:auth:throttle:<email>:<ip>。
func loginThrottleKey(email, ip string) string {
	return fmt.Sprintf("auth:throttle:%s:%s", email, ip)
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

// setSessionCookie 抽出 login 的 cookie 设置逻辑(1ae R3b)。
// 让测试可真调此方法 + 断言 Set-Cookie header 属性,而非 grep 源码字符串。
//
// 行为:
//   - SameSite=Lax(防 CSRF)
//   - Secure=h.secureCookie(prod 模式 true)
//   - HttpOnly=true(防 XSS 偷 cookie)
//   - MaxAge=sessionTTL(秒)
func (h *AuthHandler) setSessionCookie(c *gin.Context, sessionID string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, sessionID, maxAge, "/", "", h.secureCookie, true)
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
