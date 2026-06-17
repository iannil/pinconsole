// Package api：聊天消息 REST 端点（切片 1g）。
package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/proto"
	"github.com/iannil/marketing-monitor/internal/storage"
)

type ChatHandler struct {
	stores *storage.Stores
	hub    CommandHub
	logger *slog.Logger
}

func NewChatHandler(stores *storage.Stores, h CommandHub, logger *slog.Logger) *ChatHandler {
	return &ChatHandler{stores: stores, hub: h, logger: logger}
}

func (h *ChatHandler) Register(r gin.IRoutes) {
	r.GET("/api/sessions/:id/messages", h.listMessages)
	r.POST("/api/sessions/:id/messages", h.postMessage)
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

	sessionID, err := uuid.Parse(c.Param("id"))
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

	msgs, err := h.stores.PG.ListChatMessagesBySession(ctx, sessionID, sinceID, limit)
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

type postMessageRequest struct {
	Content string `json:"content" binding:"required"`
	Sender  string `json:"sender"` // operator / visitor，默认 operator
}

func (h *ChatHandler) postMessage(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return
	}

	var req postMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	sender := req.Sender
	if sender != "visitor" {
		sender = "operator"
	}

	// 写 PG
	msg, err := h.stores.PG.CreateChatMessage(ctx, storage.DefaultTenantID, sessionID, sender, req.Content)
	if err != nil {
		h.logger.ErrorContext(ctx, "create message failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	// 消息推送到对端
	if sender == "operator" {
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
	}
	// visitor → admin 的消息通过 admin 的订阅 channel 自动到达
	// （admin 端定期 GET messages 或 WS event 接收）

	c.JSON(http.StatusOK, chatMessageItem{
		ID:        msg.ID,
		Sender:    msg.Sender,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt.UnixMilli(),
	})
}
