// Package api：聊天消息 REST 端点（切片 1g）。
package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/logging"
	"github.com/iannil/pinconsole/internal/observability"
	"github.com/iannil/pinconsole/internal/proto"
	"github.com/iannil/pinconsole/internal/storage"
)

type ChatHandler struct {
	// 1ai-g:postMessage 走接口(requireClaimOwnership 接口化),
	// stores 保留(暂未用,1ai-h 删除)。
	stores      *storage.Stores
	sessionRepo claimSessionRepo
	redis       claimRedisStore
	messageRepo chatMessageRepo // 1ai-e:listMessages 用接口
	createMsg   chatMessageCreator
	hub         CommandHub
	logger      *slog.Logger
}

// chatMessageCreator 扩展 chatMessageRepo,加 CreateChatMessage(postMessage 用)。
// *storage.Postgres 自动满足。
type chatMessageCreator interface {
	ListChatMessagesBySession(ctx context.Context, sessionID uuid.UUID, sinceID int64, limit int32) ([]storage.ChatMessage, error)
	CreateChatMessage(ctx context.Context, tenantID uuid.UUID, sessionID uuid.UUID, sender, content string) (*storage.ChatMessage, error)
}

func NewChatHandler(stores *storage.Stores, h CommandHub, logger *slog.Logger) *ChatHandler {
	return &ChatHandler{
		stores:      stores,
		sessionRepo: stores.PG,
		redis:       stores.Redis,
		messageRepo: stores.PG,
		createMsg:   stores.PG,
		hub:         h,
		logger:      logger,
	}
}

func (h *ChatHandler) Register(r gin.IRoutes) {
	r.GET("/api/sessions/:id/messages", h.listMessages)
	r.POST("/api/sessions/:id/messages", h.postMessage)
}

// RegisterVisitorPublic 注册访客侧公开路由(不挂 admin AuthMiddleware)。
// 与 PrivacyHandler.RegisterPublic 同模式:visitor 无 admin cookie,
// 但需写 chat_messages 让 admin 轮询拿到。
func (h *ChatHandler) RegisterVisitorPublic(r gin.IRoutes) {
	r.POST("/api/sessions/:id/visitor-message", h.postVisitorMessage)
}

// listMessagesResponse
type listMessagesResponse struct {
	Messages []chatMessageItem `json:"messages"`
}

type chatMessageItem struct {
	ID        int64  `json:"id"`
	Sender    string `json:"sender"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
}

func (h *ChatHandler) listMessages(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 只读历史消息:任何已认证 admin 都能看(不要求 claim)。
	// claim 锁定只用于"写"操作(postMessage / postCommand),保护 1:1 控制语义。
	// 原 1k P0-3 实现错误地对 listMessages 也 requireClaimOwnership,导致 admin
	// 选中访客后无法看历史聊天(没 claim 就 403)。
	//
	// 仅校验 session_id 格式 + (可选)session 存在性,不查 Redis claim。
	idStr := c.Param("id")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return
	}

	sinceID := int64(0)
	if s := c.Query("since_id"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			sinceID = n
		}
	}
	limit := int32(200)

	msgs, err := h.messageRepo.ListChatMessagesBySession(ctx, sessionID, sinceID, limit)
	if err != nil {
		h.logger.ErrorContext(ctx, "list messages failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	items := make([]chatMessageItem, 0, len(msgs))
	for _, m := range msgs {
		items = append(items, chatMessageItem{
			ID:        m.ID,
			Sender:    m.Sender,
			Content:   m.Content,
			CreatedAt: m.CreatedAt.UnixMilli(),
		})
	}
	c.JSON(http.StatusOK, listMessagesResponse{Messages: items})
}

// postMessageRequest 1k P0-3：移除 client-controllable sender 字段。
// REST POST 永远是 operator 发起；visitor → admin 走 WS 上行。
type postMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// visitorMessageRequest 不带 binding:required — 空内容返回更友好的 empty_content,
// 而不是 binding 的 invalid_json。
type visitorMessageRequest struct {
	Content string `json:"content"`
}

func (h *ChatHandler) postMessage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 1s:Lifecycle
	logger := logging.FromContext(ctx, h.logger)
	defer observability.Lifecycle(ctx, "PostMessage", logger)()

	// 1k P0-3:校验调用方拥有 session claim
	sessionID, _, ok := requireClaimOwnership(c, h.sessionRepo, h.redis, h.logger, false)
	if !ok {
		return
	}

	var req postMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	// 1k P0-3：sender 固定为 "operator"（防止审计污染/伪造访客发言）
	sender := "operator"

	// 写 PG
	msg, err := h.createMsg.CreateChatMessage(ctx, storage.DefaultTenantID, sessionID, sender, req.Content)
	if err != nil {
		h.logger.ErrorContext(ctx, "create message failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	// admin → visitor：下行到访客 WS
	envBytes, _ := proto.Encode(proto.Envelope{
		V:         proto.ProtocolVersion,
		Type:      proto.MsgCommand,
		SessionID: sessionID.String(),
		TS:        time.Now().UnixMilli(),
		Payload: proto.CommandPayload{
			Type: "chat_message",
			TS:   time.Now().UnixMilli(),
			Chat: &proto.CommandChatMessage{
				MessageID: msg.ID,
				Content:   msg.Content,
			},
		},
	})
	h.hub.SendCommandToVisitor(sessionID, envBytes)
	// visitor → admin 的消息通过 admin 的订阅 channel 自动到达

	c.JSON(http.StatusOK, chatMessageItem{
		ID:        msg.ID,
		Sender:    msg.Sender,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt.UnixMilli(),
	})
}

// postVisitorMessage 访客侧公开端点:visitor 发消息进 DB。
// admin ChatPanel 轮询 GET /api/sessions/:id/messages 自动取走(无需 WS 推送)。
// 安全:sender 固定 "visitor",session 必须存在 + 未结束,内容长度上限 2000。
// 全局 rate limit(antiscrape.RateLimitMiddleware,prod 启用)按 IP 限频。
func (h *ChatHandler) postVisitorMessage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	logger := logging.FromContext(ctx, h.logger)
	defer observability.Lifecycle(ctx, "PostVisitorMessage", logger)()

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return
	}

	// session 必须存在 + 未结束(防止向历史会话灌水)
	sess, err := h.sessionRepo.GetSession(ctx, sessionID)
	if err != nil || sess == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session_not_found"})
		return
	}
	if sess.EndedAt.Valid {
		c.JSON(http.StatusConflict, gin.H{"error": "session_ended"})
		return
	}

	var req visitorMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	content := truncate(strings.TrimSpace(req.Content), 2000)
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty_content"})
		return
	}

	msg, err := h.createMsg.CreateChatMessage(ctx, storage.DefaultTenantID, sessionID, "visitor", content)
	if err != nil {
		h.logger.ErrorContext(ctx, "create visitor message failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	// 无下行:visitor chatWidget.sendCurrent 已做本地回声;
	// admin 侧靠 2s 轮询 GET /messages 拉取(sender=visitor)。

	c.JSON(http.StatusOK, chatMessageItem{
		ID:        msg.ID,
		Sender:    msg.Sender,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt.UnixMilli(),
	})
}

// truncate 按 rune 截断字符串到 maxRunes,防止超长内容压垮 PG/网络。
func truncate(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}
