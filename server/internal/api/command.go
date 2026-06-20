// Package api：co-browsing 命令端点（切片 1e）。
//
// 运营通过 POST /api/sessions/:id/command 发送命令。服务端：
//  1. 写 PG co_browsing_commands（审计）
//  2. 包装为 envelope（type=command）
//  3. hub.SendCommandToVisitor 下发到访客
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/logging"
	"github.com/iannil/marketing-monitor/internal/observability"
	"github.com/iannil/marketing-monitor/internal/proto"
	"github.com/iannil/marketing-monitor/internal/storage"
)

// CommandHub 是 hub 的最小接口约束（避免循环 import + 测试更易）。
type CommandHub interface {
	SendCommandToVisitor(sessionID uuid.UUID, msg []byte) bool
}

// CommandHandler 处理 co-browsing 命令端点。
type CommandHandler struct {
	// 1ai-f:用接口替代具体 *storage.Stores,解锁 mock 注入。
	// stores 保留(requireClaimOwnership helper 仍接受 *storage.Stores,1ai-g 再重构)。
	stores         *storage.Stores
	sessionRepo    claimSessionRepo // 复用 1ai-e 接口
	redis          claimRedisStore  // 复用 1ai-e 接口
	commandRepo    commandRepo
	hub            CommandHub
	logger         *slog.Logger
	allowedDomains []string // 1f：额外允许的域名
}

// NewCommandHandler 创建 command handler。
//
// 1ai-f:签名不变,内部抽取 PG/Redis 适配接口。
func NewCommandHandler(stores *storage.Stores, h CommandHub, allowedDomains string, logger *slog.Logger) *CommandHandler {
	domains := []string{}
	for _, d := range strings.Split(allowedDomains, ",") {
		d = strings.TrimSpace(d)
		if d != "" {
			domains = append(domains, d)
		}
	}
	return &CommandHandler{
		stores:         stores, // 留给 requireClaimOwnership(1ai-g 再删)
		sessionRepo:    stores.PG,
		redis:          stores.Redis,
		commandRepo:    stores.PG,
		hub:            h,
		logger:         logger,
		allowedDomains: domains,
	}
}

// Register 注册路由。
func (h *CommandHandler) Register(r gin.IRoutes) {
	r.POST("/api/sessions/:id/command", h.postCommand)
}

// commandRequest 是 POST /api/sessions/:id/command 的请求体。
type commandRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// PostCommand 处理运营命令。
//
// 校验：
//   - session 存在
//   - command_type 在白名单
//   - navigate 命令的 URL 同源或白名单域名
//
// 副作用：
//   - 写 PG co_browsing_commands
//   - 发 envelope 到 visitor WS
func (h *CommandHandler) postCommand(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 1s:LifecycleTracker 全链路埋点
	logger := logging.FromContext(ctx, h.logger)
	defer observability.Lifecycle(ctx, "PostCommand", logger)()

	// 1k P0-3:校验调用方拥有 session claim(不要求 alive,因命令可能针对刚结束的 session)
	sessionID, callerUID, ok := requireClaimOwnership(c, h.sessionRepo, h.redis, h.logger, false)
	if !ok {
		observability.LogPoint(ctx, logger, observability.EventBranch, "PostCommand",
			"claim_check", "failed")
		return
	}
	observability.LogPoint(ctx, logger, observability.EventBranch, "PostCommand",
		"claim_check", "ok", "command_type", "")

	var req commandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json", "detail": err.Error()})
		return
	}
	// 1s 安全修复:command_type 可能是用户输入,限制长度 + 用白名单
	observability.LogPoint(ctx, logger, observability.EventBranch, "PostCommand",
		"command_type", sanitizeCommandType(req.Type))

	// 构造 CommandPayload
	cp, err := buildCommandPayload(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_command", "detail": err.Error()})
		return
	}

	// navigate 安全:同源 + localhost + 额外白名单(env var)
	if cp.Navigate != nil {
		if !h.isURLAllowed(cp.Navigate.URL, c.Request.Host) {
			// 1s 安全修复:日志只记 URL 主体(去掉 query string 防泄露 PII)
			observability.LogPoint(ctx, logger, observability.EventBranch, "PostCommand",
				"navigate_check", "rejected", "url_safe", sanitizeURLForLog(cp.Navigate.URL))
			c.JSON(http.StatusForbidden, gin.H{"error": "url_not_allowed", "url": cp.Navigate.URL})
			return
		}
	}
	// 1k P0-8:show_popup action_url scheme 白名单(防 javascript:/data: 注入)
	if cp.Popup != nil && cp.Popup.ActionURL != "" {
		if !isURLSchemeAllowed(cp.Popup.ActionURL) {
			// 1s 安全修复:同样去掉 query string
			observability.LogPoint(ctx, logger, observability.EventBranch, "PostCommand",
				"popup_url_check", "rejected", "url_safe", sanitizeURLForLog(cp.Popup.ActionURL))
			c.JSON(http.StatusBadRequest, gin.H{"error": "popup_url_scheme_not_allowed", "url": cp.Popup.ActionURL})
			return
		}
	}

	// 计算 target_node_id（审计用）
	var nodeID *int32
	if cp.Click != nil {
		v := int32(cp.Click.NodeID)
		nodeID = &v
	} else if cp.FillInput != nil {
		v := int32(cp.FillInput.NodeID)
		nodeID = &v
	}

	// 写 PG 审计(1k P0-3:OperatorID 用 user_id 而非 ClientIP,修复审计污染)
	payloadBytes, _ := json.Marshal(req.Payload)
	_, err = h.commandRepo.CreateCoBrowsingCommand(ctx, storage.CoBrowsingCommand{
		TenantID:     storage.DefaultTenantID,
		SessionID:    sessionID,
		OperatorID:   callerUID.String(),
		CommandType:  req.Type,
		TargetNodeID: nodeID,
		Payload:      payloadBytes,
	})
	if err != nil {
		observability.LogExternalCall(ctx, logger, "pg.CreateCoBrowsingCommand", "error", "session_id", sessionID)
		h.logger.ErrorContext(ctx, "create command audit failed", "error", err)
		// 不阻塞下行;继续
	} else {
		observability.LogExternalCall(ctx, logger, "pg.CreateCoBrowsingCommand", "ok", "session_id", sessionID)
	}

	// 包装 envelope,下行到 visitor(1m:透传 ctx trace_id)
	envBytes, err := proto.Encode(proto.Envelope{
		V:         proto.ProtocolVersion,
		Type:      proto.MsgCommand,
		SessionID: sessionID.String(),
		TraceID:   logging.TraceID(c.Request.Context()),
		TS:        time.Now().UnixMilli(),
		Payload:   cp,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encode_failed"})
		return
	}

	delivered := h.hub.SendCommandToVisitor(sessionID, envBytes)
	if !delivered {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "visitor_offline"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "type": req.Type})
}

// sanitizeCommandType 1s 安全修复:限制 command_type 长度,防止用户输入溢出日志。
// 已知 8 种合法命令,白名单优先;未知截断到 50 字符。
var allowedCommandTypes = map[string]bool{
	"cursor_highlight": true, "click": true, "scroll": true,
	"fill_input": true, "navigate": true, "release_control": true,
	"show_popup": true, "chat_message": true,
}

func sanitizeCommandType(t string) string {
	if allowedCommandTypes[t] {
		return t
	}
	if len(t) > 50 {
		return t[:50] + "...(truncated)"
	}
	return t
}

// sanitizeURLForLog 1s 安全修复:URL 进入日志前剥离 query string(可能含 PII / token)
// 并限制总长度。返回 "scheme://host/path" 形式(最多 200 字符)。
func sanitizeURLForLog(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	clean := rawURL
	if idx := strings.Index(clean, "?"); idx >= 0 {
		clean = clean[:idx]
	}
	if idx := strings.Index(clean, "#"); idx >= 0 {
		clean = clean[:idx]
	}
	const maxLen = 200
	if len(clean) > maxLen {
		clean = clean[:maxLen] + "...(truncated)"
	}
	return clean
}

// isURLSchemeAllowed 1k P0-8：popup action_url scheme 白名单。
// 只允许 http/https；拒绝 javascript:/data:/vbscript:/file:/mailto: 等其他 scheme。
// 空字符串、protocol-relative (//host)、相对路径 (/path 或 page.html) 按同源允许。
func isURLSchemeAllowed(rawURL string) bool {
	if rawURL == "" {
		return true
	}
	lower := strings.ToLower(rawURL)

	// 显式拒绝危险 scheme（深度防御,即使下面的 scheme 检测错过这类也先拒）
	for _, bad := range []string{"javascript:", "data:", "vbscript:", "file:", "about:"} {
		if strings.HasPrefix(lower, bad) {
			return false
		}
	}

	// 显式允许 http/https
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return true
	}

	// 允许 protocol-relative (//host/path)
	if strings.HasPrefix(rawURL, "//") {
		return true
	}

	// 检测是否含 scheme：":" 出现在第一个 "/" 之前
	firstColon := strings.Index(rawURL, ":")
	firstSlash := strings.Index(rawURL, "/")
	if firstColon == -1 || (firstSlash != -1 && firstSlash < firstColon) {
		// 无 scheme,视为相对路径,允许
		return true
	}

	// 含非 http/https scheme（mailto:/tel:/ftp:/custom: 等）— 拒绝
	return false
}

// buildCommandPayload 把请求体解析为 CommandPayload。
func buildCommandPayload(req commandRequest) (proto.CommandPayload, error) {
	cp := proto.CommandPayload{
		Type: req.Type,
		TS:   time.Now().UnixMilli(),
	}
	switch req.Type {
	case "cursor_highlight":
		var m struct {
			X    int    `json:"x"`
			Y    int    `json:"y"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Payload, &m); err != nil {
			return cp, fmt.Errorf("cursor_highlight: %w", err)
		}
		cp.Cursor = &proto.CommandCursor{X: m.X, Y: m.Y, Name: m.Name}
	case "click":
		var m struct {
			NodeID int `json:"node_id"`
			X      int `json:"x"`
			Y      int `json:"y"`
		}
		if err := json.Unmarshal(req.Payload, &m); err != nil {
			return cp, fmt.Errorf("click: %w", err)
		}
		cp.Click = &proto.CommandClick{NodeID: m.NodeID, X: m.X, Y: m.Y}
	case "scroll":
		var m struct {
			X int `json:"x"`
			Y int `json:"y"`
		}
		if err := json.Unmarshal(req.Payload, &m); err != nil {
			return cp, fmt.Errorf("scroll: %w", err)
		}
		cp.Scroll = &proto.CommandScroll{X: m.X, Y: m.Y}
	case "fill_input":
		var m struct {
			NodeID int    `json:"node_id"`
			Value  string `json:"value"`
		}
		if err := json.Unmarshal(req.Payload, &m); err != nil {
			return cp, fmt.Errorf("fill_input: %w", err)
		}
		cp.FillInput = &proto.CommandFillInput{NodeID: m.NodeID, Value: m.Value}
	case "navigate":
		var m struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(req.Payload, &m); err != nil {
			return cp, fmt.Errorf("navigate: %w", err)
		}
		cp.Navigate = &proto.CommandNavigate{URL: m.URL}
	case "release_control":
		// 无 payload
	case "show_popup":
		var m struct {
			Title       string `json:"title"`
			Body        string `json:"body"`
			ActionLabel string `json:"action_label"`
			ActionURL   string `json:"action_url"`
			Dismissible bool   `json:"dismissible"`
		}
		if err := json.Unmarshal(req.Payload, &m); err != nil {
			return cp, fmt.Errorf("show_popup: %w", err)
		}
		cp.Popup = &proto.CommandPopup{
			Title:       m.Title,
			Body:        m.Body,
			ActionLabel: m.ActionLabel,
			ActionURL:   m.ActionURL,
			Dismissible: m.Dismissible,
		}
	default:
		return cp, fmt.Errorf("unknown command type: %s", req.Type)
	}
	return cp, nil
}

// isURLAllowed 1f navigate 安全：同源 + localhost + 额外白名单。
func (h *CommandHandler) isURLAllowed(rawURL, requestHost string) bool {
	if rawURL == "" {
		return false
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		return true // 相对路径，同源
	}
	// 同 host
	if host == requestHost || strings.HasPrefix(requestHost, host+":") {
		return true
	}
	// localhost（dev）
	if host == "localhost" || host == "127.0.0.1" {
		return true
	}
	// 1f：额外白名单
	for _, d := range h.allowedDomains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}
