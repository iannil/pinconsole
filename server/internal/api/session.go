// Package api：REST 端点 /api/session/* 与 /api/sessions。
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iannil/pinconsole/internal/antiscrape"
	"github.com/iannil/pinconsole/internal/hub"
	"github.com/iannil/pinconsole/internal/logging"
	"github.com/iannil/pinconsole/internal/privacy"
	"github.com/iannil/pinconsole/internal/storage"
)

// SessionHandler 处理 session REST 端点。
type SessionHandler struct {
	// 1ai-h:用接口替代具体 *storage.Stores,解锁 mock 注入。
	// stores/redis 保留(listSessions 用 antiscrape.IsSessionFlagged 需要 Redis.Client)。
	stores      *storage.Stores
	sessionRepo sessionInitRepo
	logger      *slog.Logger
	hub         *hub.Hub
}

// NewSessionHandler 创建 session handler。
//
// 1ai-h:签名不变,内部抽取 PG 适配接口。
func NewSessionHandler(stores *storage.Stores, h *hub.Hub, logger *slog.Logger) *SessionHandler {
	return &SessionHandler{
		stores:      stores,
		sessionRepo: stores.PG,
		hub:         h,
		logger:      logger,
	}
}

// Register 在 router 上注册路由。
func (h *SessionHandler) Register(r gin.IRoutes) {
	r.POST("/api/session/init", h.initSession)
	r.GET("/api/sessions", h.listSessions)
}

// initSessionPayload 是 POST /api/session/init 的请求体。
type initSessionPayload struct {
	VisitorID string `json:"visitor_id"` // SDK 持久化在 localStorage 的 fingerprint
	UA        string `json:"ua"`
	IP        string `json:"ip"` // 信息性；服务端用 c.ClientIP() 不信任
}

// initSessionResponse 是 POST /api/session/init 的响应。
type initSessionResponse struct {
	SessionID string `json:"session_id"`
	VisitorID string `json:"visitor_id"`
	TenantID  string `json:"tenant_id"`
}

// InitSession 处理 SDK 的握手前置请求：签发 session_id。
//
// 流程（详见 docs/progress/2026-06-17-slice-1b-spec.md §消息流）：
//  1. SDK 持 visitor_id（localStorage）调本端点
//  2. 后端按 tenant+fingerprint upsert visitor
//  3. 后端创建 session（status=active）
//  4. 返回 session_id 给 SDK
//  5. SDK 用 visitor_id + session_id 发起 WS 连接
//
// visitor 端 IP 取自 gin.Context.ClientIP()（忽略 payload.IP，防伪造）。
func (h *SessionHandler) initSession(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req initSessionPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json", "detail": err.Error()})
		return
	}
	if req.VisitorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_visitor_id"})
		return
	}

	tenantID := storage.DefaultTenantID
	// 1l: GDPR 数据最小化 — IP 截断 IPv4 /24 + IPv6 /64,使 IP 不再是个人数据
	truncatedIP := privacy.TruncateIP(c.ClientIP())

	// upsert visitor
	visitor, err := h.sessionRepo.CreateVisitor(ctx, tenantID, req.VisitorID, req.UA, truncatedIP)
	if err != nil {
		h.logger.ErrorContext(ctx, "create visitor failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	// create session
	sess, err := h.sessionRepo.CreateSession(ctx, tenantID, visitor.ID, req.UA, truncatedIP)
	if err != nil {
		h.logger.ErrorContext(ctx, "create session failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	logging.FromContext(ctx, h.logger).Info("session init",
		"session_id", sess.ID, "visitor_id", visitor.ID,
	)

	c.JSON(http.StatusOK, initSessionResponse{
		SessionID: sess.ID.String(),
		VisitorID: visitor.ID.String(),
		TenantID:  tenantID.String(),
	})
}

// listSessionsResponse 是 GET /api/sessions 的响应。
type listSessionsResponse struct {
	Sessions []sessionListItem `json:"sessions"`
}

type sessionListItem struct {
	SessionID   string `json:"session_id"`
	VisitorID   string `json:"visitor_id"`
	Fingerprint string `json:"fingerprint"`
	StartedAt   int64  `json:"started_at"` // 毫秒时间戳
	LastEventAt *int64 `json:"last_event_at,omitempty"`
	EventCount  int32  `json:"event_count"`
	UA          string `json:"ua"`
	// 1w P1-29:BehaviorTracker 标记的可疑 session(is_flagged=true 时 admin 应警惕)。
	IsFlagged  bool   `json:"is_flagged"`
	FlagReason string `json:"flag_reason,omitempty"`
}

// ListSessions 返回当前租户的全部活跃会话（初始列表，admin 用）。
// 后续增量通过 WS presence 推送。
func (h *SessionHandler) listSessions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	tenantID := storage.DefaultTenantID
	limit := int32(200)
	if l := c.Query("limit"); l != "" {
		var n int32
		if _, err := fmt.Sscanf(l, "%d", &n); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	sessions, err := h.stores.PG.ListActiveSessionsByTenant(ctx, tenantID, limit)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		h.logger.ErrorContext(ctx, "list sessions failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	items := make([]sessionListItem, 0, len(sessions))
	for _, s := range sessions {
		var fp string
		if s.VisitorFingerprint != nil {
			fp = *s.VisitorFingerprint
		}
		var lastEventAt *int64
		if s.LastEventAt.Valid {
			ms := s.LastEventAt.Time.UnixMilli()
			lastEventAt = &ms
		}
		var ua string
		if s.UA != nil {
			ua = *s.UA
		}
		// 1w P1-29:查 Redis flagged:session:{id};失败不阻断列表(Redis 故障 ≠ 业务故障)。
		flagged, reason, err := antiscrape.IsSessionFlagged(ctx, h.stores.Redis.Client, s.ID.String())
		if err != nil {
			h.logger.WarnContext(ctx, "is_session_flagged failed",
				"session_id", s.ID, "error", err)
		}
		items = append(items, sessionListItem{
			SessionID:   s.ID.String(),
			VisitorID:   s.VisitorID.String(),
			Fingerprint: fp,
			StartedAt:   s.StartedAt.UnixMilli(),
			LastEventAt: lastEventAt,
			EventCount:  s.EventCount,
			UA:          ua,
			IsFlagged:   flagged,
			FlagReason:  reason,
		})
	}

	c.JSON(http.StatusOK, listSessionsResponse{Sessions: items})
}
