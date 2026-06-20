// Package api：录像归档与历史回放相关 REST 端点（切片 1d）。
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/antiscrape"
	"github.com/iannil/marketing-monitor/internal/proto"
	"github.com/iannil/marketing-monitor/internal/storage"
	"github.com/vmihailenco/msgpack/v5"
)

// ReplayHandler 处理历史会话列表与事件流端点。
type ReplayHandler struct {
	stores *storage.Stores
	logger *slog.Logger
}

// NewReplayHandler 创建 replay handler。
func NewReplayHandler(stores *storage.Stores, logger *slog.Logger) *ReplayHandler {
	return &ReplayHandler{stores: stores, logger: logger}
}

// Register 注册路由。
func (h *ReplayHandler) Register(r gin.IRoutes) {
	// 1d：扩展 GET /api/sessions 支持 status=ended&since=（在 session.go 已注册，这里加 ended 列表 handler 需要在 session.go 中区分；为了减少改动，我们用单独路径）
	r.GET("/api/sessions/ended", h.listEndedSessions)
	r.GET("/api/sessions/:id/replay", h.getSessionReplay)
}

// listEndedSessionsResponse 是 GET /api/sessions/ended 响应。
type listEndedSessionsResponse struct {
	Sessions []endedSessionItem `json:"sessions"`
	Total    int                `json:"total"`
}

type endedSessionItem struct {
	SessionID   string `json:"session_id"`
	VisitorID   string `json:"visitor_id"`
	Fingerprint string `json:"fingerprint"`
	StartedAt   int64  `json:"started_at"`
	EndedAt     int64  `json:"ended_at"`
	DurationMS  int64  `json:"duration_ms"`
	EventCount  int32  `json:"event_count"`
	UA          string `json:"ua"`
}

// listEndedSessions 列出已结束会话。
//
// 查询参数：
//
//	since (24h / 7d / 30d，默认 24h)
//	limit (最大 1000，默认 200)
func (h *ReplayHandler) listEndedSessions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	since, err := parseSince(c.Query("since"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_since", "detail": err.Error()})
		return
	}
	limit := int32(200)
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = int32(n)
		}
	}

	tenantID := storage.DefaultTenantID
	sessions, err := h.stores.PG.ListEndedSessionsByTenant(ctx, tenantID, since, limit)
	if err != nil {
		h.logger.ErrorContext(ctx, "list ended sessions failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	items := make([]endedSessionItem, 0, len(sessions))
	for _, s := range sessions {
		var fp string
		if s.VisitorFingerprint != nil {
			fp = *s.VisitorFingerprint
		}
		var ua string
		if s.UA != nil {
			ua = *s.UA
		}
		var endedAt int64
		var durationMS int64
		if s.EndedAt.Valid {
			endedAt = s.EndedAt.Time.UnixMilli()
			durationMS = s.EndedAt.Time.Sub(s.StartedAt).Milliseconds()
		}
		items = append(items, endedSessionItem{
			SessionID:   s.ID.String(),
			VisitorID:   s.VisitorID.String(),
			Fingerprint: fp,
			StartedAt:   s.StartedAt.UnixMilli(),
			EndedAt:     endedAt,
			DurationMS:  durationMS,
			EventCount:  s.EventCount,
			UA:          ua,
		})
	}

	c.JSON(http.StatusOK, listEndedSessionsResponse{
		Sessions: items,
		Total:    len(items),
	})
}

// replayEventsResponse 是 GET /api/sessions/:id/replay 响应。
type replayEventsResponse struct {
	SessionID string           `json:"session_id"`
	Events    []map[string]any `json:"events"`
	Total     int64            `json:"total"`
	Offset    int              `json:"offset"`
	Limit     int              `json:"limit"`
	HasMore   bool             `json:"has_more"`
}

// getSessionReplay 返回某 session 的完整事件流（分页）。
//
// 服务端从 PG event_blobs 拉所有 blob 的 MinIO key，逐个下载 + msgpack 解码 + 提取 rrweb 事件。
// 注：1d 简化实现，全 blob 拉到内存再分页。1e+ 可加流式响应优化。
func (h *ReplayHandler) getSessionReplay(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_session_id"})
		return
	}

	// 1w P1-29:replay 请求查 flagged;日志 warn,不阻断回放(管理员可能需要分析可疑 session)。
	if h.stores.Redis != nil && h.stores.Redis.Client != nil {
		if flagged, reason, err := antiscrape.IsSessionFlagged(ctx, h.stores.Redis.Client, sessionIDStr); err != nil {
			h.logger.WarnContext(ctx, "is_session_flagged check failed on replay",
				"session_id", sessionID, "error", err)
		} else if flagged {
			h.logger.WarnContext(ctx, "replay requested for flagged session",
				"session_id", sessionID, "flag_reason", reason,
				"note", "behavior tracker marked this session as suspicious")
		}
	}

	offset := 0
	limit := 10000
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 50000 {
			limit = n
		}
	}

	// 拉 blob 列表
	blobs, err := h.stores.PG.ListEventBlobsBySession(ctx, sessionID)
	if err != nil {
		h.logger.ErrorContext(ctx, "list blobs failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	// 全部 blob 合并为 rrweb 事件流
	// 初始化为空 slice 避免 JSON 序列化为 null（admin 期望 []）
	allEvents := []map[string]any{}
	for _, b := range blobs {
		data, err := h.stores.MinIO.GetBytes(ctx, b.MinIOObjectKey)
		if err != nil {
			h.logger.WarnContext(ctx, "minio get failed", "key", b.MinIOObjectKey, "error", err)
			continue
		}
		// blob 是 msgpack array of envelope bytes
		events, err := decodeRRWebEventsFromBlob(data)
		if err != nil {
			h.logger.WarnContext(ctx, "decode blob failed", "key", b.MinIOObjectKey, "error", err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	// 计算分页
	total := int64(len(allEvents))
	end := offset + limit
	if end > len(allEvents) {
		end = len(allEvents)
	}
	if offset > len(allEvents) {
		offset = len(allEvents)
	}
	page := allEvents[offset:end]
	hasMore := end < len(allEvents)

	c.JSON(http.StatusOK, replayEventsResponse{
		SessionID: sessionIDStr,
		Events:    page,
		Total:     total,
		Offset:    offset,
		Limit:     limit,
		HasMore:   hasMore,
	})
}

// decodeRRWebEventsFromBlob 解 msgpack array of envelope bytes，提取所有 rrweb 事件。
// 返回原生 rrweb 事件结构（type/timestamp/data）。
func decodeRRWebEventsFromBlob(data []byte) ([]map[string]any, error) {
	// 先解 array of bytes
	var rawEnvelopes [][]byte
	if err := msgpack.Unmarshal(data, &rawEnvelopes); err != nil {
		return nil, fmt.Errorf("unmarshal array: %w", err)
	}

	out := make([]map[string]any, 0, len(rawEnvelopes))
	for _, envBytes := range rawEnvelopes {
		env, err := proto.Decode(envBytes)
		if err != nil {
			continue
		}
		if env.Type != proto.MsgEvent {
			continue
		}
		events := extractRRWebEventsFromPayload(env.Payload)
		out = append(out, events...)
	}
	return out, nil
}

// extractRRWebEventsFromPayload 从 envelope.Payload 提取所有 rrweb 事件（map 形式）。
func extractRRWebEventsFromPayload(payload any) []map[string]any {
	out := []map[string]any{}

	// 尝试 single
	if single, err := decodePayloadAsEvent(payload); err == nil && single != nil {
		out = append(out, single)
		return out
	}

	// 尝试 array
	var arr []proto.EventPayload
	if err := proto.DecodePayload(payload, &arr); err != nil {
		return out
	}
	for _, ep := range arr {
		if m := eventPayloadToMap(ep); m != nil {
			out = append(out, m)
		}
	}
	return out
}

func decodePayloadAsEvent(payload any) (map[string]any, error) {
	var ep proto.EventPayload
	if err := proto.DecodePayload(payload, &ep); err != nil {
		return nil, err
	}
	return eventPayloadToMap(ep), nil
}

func eventPayloadToMap(ep proto.EventPayload) map[string]any {
	if ep.Type != proto.EvRRWeb || ep.RRWeb == nil {
		return nil
	}
	// 转换为 admin 可直接用的 rrweb 原生结构
	return map[string]any{
		"type":      ep.RRWeb.Type,
		"timestamp": ep.RRWeb.Timestamp,
		"data":      ep.RRWeb.Data,
	}
}

// parseSince 解析 "24h" / "7d" / "30d" 为 time.Duration。
func parseSince(s string) (time.Duration, error) {
	if s == "" {
		return 24 * time.Hour, nil
	}
	// 简单解析：支持 h / d
	if len(s) < 2 {
		return 0, errors.New("invalid since format")
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, err
	}
	// 1aj:拒绝非正数 — Atoi 接受 "-1",会产生负 duration,语义无意义
	// (查询"结束于 -24h 内"无结果但 SQL 不报错,可能被滥用)。
	if num <= 0 {
		return 0, fmt.Errorf("duration must be positive: %d", num)
	}
	switch unit {
	case 'h':
		return time.Duration(num) * time.Hour, nil
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported unit: %c", unit)
	}
}
