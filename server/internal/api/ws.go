// Package api：WebSocket 端点 /ws/visitor 与 /ws/operator。
package api

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/antiscrape"
	"github.com/iannil/marketing-monitor/internal/hub"
	"github.com/iannil/marketing-monitor/internal/logging"
	"github.com/iannil/marketing-monitor/internal/proto"
	"github.com/iannil/marketing-monitor/internal/recording"
	"github.com/iannil/marketing-monitor/internal/storage"
)

// WSHandler 处理 /ws/* 端点。
type WSHandler struct {
	hub       *hub.Hub
	stores    *storage.Stores
	stream    *recording.Stream
	flusher   *recording.Flusher
	snapshots *recording.SnapshotCache
	logger    *slog.Logger
	maxMsg    int64
	pingEvery time.Duration
}

// NewWSHandler 创建 WS handler。
func NewWSHandler(h *hub.Hub, stores *storage.Stores, stream *recording.Stream, flusher *recording.Flusher, snapshots *recording.SnapshotCache, logger *slog.Logger) *WSHandler {
	return &WSHandler{
		hub:       h,
		stores:    stores,
		stream:    stream,
		flusher:   flusher,
		snapshots: snapshots,
		logger:    logger,
		maxMsg:    1 << 20, // 1 MiB
		pingEvery: 15 * time.Second,
	}
}

// Register 注册 /ws/* 路由。
func (h *WSHandler) Register(r gin.IRoutes) {
	r.GET("/ws/visitor", h.visitorWS)
	r.GET("/ws/operator", h.operatorWS)
}

// visitorWS 处理 SDK 的 WebSocket 连接。
//
// 流程：
//   1. accept WS
//   2. 等待 hello 消息（含 visitor_id + session_id）
//   3. 验证 session_id 存在
//   4. hub.VisitorOnline（注册 session chan + 广播 presence.online）
//   5. 启动 read loop：每个 MsgEvent 写 Redis Stream + PublishEvent
//   6. 连接断开：hub.VisitorOffline + DB EndSession
func (h *WSHandler) visitorWS(c *gin.Context) {
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		// v1 同源（admin 也走同源），不留跨域口子（与 PLAN.md 一致）
		InsecureSkipVerify: false,
	})
	if err != nil {
		h.logger.Warn("ws accept failed", "error", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "closing")

	conn.SetReadLimit(h.maxMsg)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// 等待 hello（10s 超时）
	helloCtx, helloCancel := context.WithTimeout(ctx, 10*time.Second)
	_, helloBytes, err := conn.Read(helloCtx)
	helloCancel()
	if err != nil {
		h.logger.Debug("read hello failed", "error", err)
		return
	}

	env, err := proto.Decode(helloBytes)
	if err != nil || env.Type != proto.MsgHello {
		h.sendError(ctx, conn, "invalid_hello", "expected hello message")
		return
	}

	var hello proto.HelloPayload
	if err := proto.DecodePayload(env.Payload, &hello); err != nil {
		h.sendError(ctx, conn, "invalid_hello_payload", err.Error())
		return
	}

	visitorID, err := uuid.Parse(hello.VisitorID)
	if err != nil {
		h.sendError(ctx, conn, "invalid_visitor_id", err.Error())
		return
	}
	sessionID, err := uuid.Parse(hello.SessionID)
	if err != nil {
		h.sendError(ctx, conn, "invalid_session_id", err.Error())
		return
	}

	// 验证 session 存在
	sess, err := h.stores.PG.GetSession(ctx, sessionID)
	if err != nil {
		h.sendError(ctx, conn, "session_not_found", err.Error())
		return
	}
	if sess.VisitorID != visitorID {
		h.sendError(ctx, conn, "visitor_session_mismatch", "")
		return
	}

	// ack
	ack, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgAck,
		TS:   time.Now().UnixMilli(),
		Payload: proto.AckPayload{
			OK: true,
		},
	})
	if err := conn.Write(ctx, websocket.MessageBinary, ack); err != nil {
		return
	}

	// 注册到 hub
	clientID := uuid.New()
	client := hub.NewClient(clientID, hub.RoleVisitor, conn, h.logger)
	h.hub.RegisterClient(client)
	defer h.hub.UnregisterClient(client)

	tenantID := storage.DefaultTenantID

	// presence online
	presenceOnline, _ := proto.Encode(proto.Envelope{
		V:         proto.ProtocolVersion,
		Type:      proto.MsgPresence,
		TS:        time.Now().UnixMilli(),
		SessionID: sessionID.String(),
		Payload: proto.PresencePayload{
			Event:       "online",
			SessionID:   sessionID.String(),
			VisitorID:   visitorID.String(),
			Fingerprint: hello.VisitorID, // visitor_id 即 fingerprint
			StartedAt:   time.Now().UnixMilli(),
		},
	})
	h.hub.VisitorOnline(tenantID, sessionID, clientID, presenceOnline)

	// 1d：注册到 flusher，让周期性 + end-time flush 能归档事件
	if h.flusher != nil {
		h.flusher.Register(sessionID, tenantID)
	}
	defer func() {
		presenceOffline, _ := proto.Encode(proto.Envelope{
			V:         proto.ProtocolVersion,
			Type:      proto.MsgPresence,
			TS:        time.Now().UnixMilli(),
			SessionID: sessionID.String(),
			Payload: proto.PresencePayload{
				Event:     "offline",
				SessionID: sessionID.String(),
				VisitorID: visitorID.String(),
			},
		})
		// 1d：同步 flush 剩余事件（确保 replay 立即看到完整会话）+ Unregister
		if h.flusher != nil {
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			h.flusher.Unregister(flushCtx, sessionID) // Unregister 内部触发最后一次 flush
			cancel()
		}
		// 1d：清 snapshot 缓存
		if h.snapshots != nil {
			delCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = h.snapshots.Delete(delCtx, sessionID)
			cancel()
		}
		h.hub.VisitorOffline(tenantID, sessionID, presenceOffline)
		_ = h.stores.PG.EndSession(ctx, sessionID, "ended")
	}()

	logger := logging.FromContext(ctx, h.logger).With("session_id", sessionID, "visitor_id", visitorID)
	logger.Info("visitor ws connected")

	// 1i：行为分析 tracker，每 100 事件检查启发式并在 Redis 标记可疑 session
	bt := antiscrape.NewBehaviorTracker(h.stores.Redis.Client, logger, sessionID.String())
	var behaviorCounter int

	// read loop
	for {
		msgType, msg, err := conn.Read(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || websocket.CloseStatus(err) != -1 {
				logger.Debug("visitor ws closed")
			} else {
				logger.Warn("visitor ws read error", "error", err)
			}
			return
		}
		if msgType != websocket.MessageBinary {
			continue
		}

		env, err := proto.Decode(msg)
		if err != nil {
			logger.Warn("invalid envelope", "error", err)
			continue
		}
		if env.Type != proto.MsgEvent {
			continue
		}

		// 1) 写 Redis Stream：batch envelope 作为单条 stream entry（保留批量结构）
		if err := h.stream.Append(ctx, sessionID, msg); err != nil {
			logger.Warn("stream append failed", "error", err)
		}

		// 1c 新增：如果 envelope 含 rrweb full snapshot，缓存到 Redis（5 分钟 TTL）。
		// batch 模式下从 array 中提取 full snapshot 后单独包装缓存。
		if h.snapshots != nil {
			if snapEnv := extractFullSnapshotEnvelope(ctx, env); snapEnv != nil {
				if err := h.snapshots.Set(ctx, sessionID, snapEnv); err != nil {
					logger.Warn("snapshot cache set failed", "error", err)
				}
			}
		}

		// 2) 广播给已订阅该 session 的 operator（envelope 整体广播，admin 自行拆 array）
		h.hub.PublishEvent(sessionID, msg)

		// 3) 更新 PG session 元数据
		eventCount := eventCountOf(env)
		_ = h.stores.PG.TouchSessionEvent(ctx, sessionID, int32(eventCount))

		// 4) 1i：行为分析 — 逐事件 Observe，每 100 事件 CheckAndFlag
		forEachEventPayload(env, func(ep proto.EventPayload) {
			bt.Observe(ep)
			behaviorCounter++
		})
		if behaviorCounter >= 100 {
			bt.CheckAndFlag(ctx)
			behaviorCounter = 0
		}
	}
}

// operatorWS 处理 admin 的 WebSocket 连接。
//
// 流程：
//   1. accept WS
//   2. 等待 hello（可选，1b 不强制 capabilities）
//   3. 默认加入 room:tenant:<id> 接收 presence
//   4. read loop：subscribe / unsubscribe 命令
//   5. 同时 select tenant room chan 与 session chan（已订阅的）
func (h *WSHandler) operatorWS(c *gin.Context) {
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		InsecureSkipVerify: false,
	})
	if err != nil {
		h.logger.Warn("ws accept failed", "error", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "closing")

	conn.SetReadLimit(h.maxMsg)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// hello 不强制；如果没有，直接当作 operator 上线
	clientID := uuid.New()
	client := hub.NewClient(clientID, hub.RoleOperator, conn, h.logger)
	h.hub.RegisterClient(client)
	defer h.hub.UnregisterClient(client)

	tenantID := storage.DefaultTenantID
	presenceCh := h.hub.JoinTenantRoom(client, tenantID)

	// 启动 read loop（独立 goroutine）
	type subMsg struct {
		action   string // subscribe / unsubscribe
		sessionID uuid.UUID
	}
	cmdCh := make(chan subMsg, 16)

	go func() {
		defer close(cmdCh)
		for {
			_, msg, err := conn.Read(ctx)
			if err != nil {
				return
			}
			env, err := proto.Decode(msg)
			if err != nil {
				continue
			}
			switch env.Type {
			case proto.MsgSubscribe, proto.MsgUnsubscribe:
				var p proto.SubscribePayload
				if err := proto.DecodePayload(env.Payload, &p); err != nil {
					continue
				}
				sid, err := uuid.Parse(p.SessionID)
				if err != nil {
					continue
				}
				action := "subscribe"
				if env.Type == proto.MsgUnsubscribe {
					action = "unsubscribe"
				}
				select {
				case cmdCh <- subMsg{action: action, sessionID: sid}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// 主 loop：select presence / cmd / sub chan / ctx done
	// sub chan 动态变化，用一个专门的 goroutine 管理
	type activeSub struct {
		sessionID uuid.UUID
		ch        <-chan []byte
	}
	subs := make(map[uuid.UUID]activeSub)
	defer func() {
		for sid := range subs {
			h.hub.UnsubscribeSession(client, sid)
		}
	}()

	// 用于把 sub chan 的数据 forward 到 conn.Send
	// 每个 sub 启动一个 goroutine，事件经 chan 推到统一 forwardCh
	forwardCh := make(chan []byte, 256)
	defer close(forwardCh)

	restartSubs := func() {
		// 启动新 sub 的 goroutine（已在 subs 中但 ch 未 listen）
		// 此处简单实现：每次启动一个 forwarder
		// 真实实现需要仔细管理 goroutine 生命周期（1b 简化）
		for sid, s := range subs {
			if s.ch == nil {
				continue
			}
			ch := s.ch
			sid := sid
			go func() {
				for msg := range ch {
					select {
					case forwardCh <- msg:
					case <-ctx.Done():
						return
					}
				}
			}()
			// 标记已 listen，避免重复（同一 session 多次 subscribe 时只 listen 一次）
			subs[sid] = activeSub{sessionID: sid, ch: nil}
		}
	}

	// 初始无 sub，restartSubs 是 no-op
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-presenceCh:
			if !ok {
				return
			}
			if err := conn.Write(ctx, websocket.MessageBinary, msg); err != nil {
				return
			}
		case msg, ok := <-forwardCh:
			if !ok {
				return
			}
			if err := conn.Write(ctx, websocket.MessageBinary, msg); err != nil {
				return
			}
		case cmd, ok := <-cmdCh:
			if !ok {
				return
			}
			switch cmd.action {
			case "subscribe":
				if _, exists := subs[cmd.sessionID]; exists {
					continue
				}
				// 1c 新增：订阅时先发缓存的最近 full snapshot（如果有）
				if h.snapshots != nil {
					if snap, _ := h.snapshots.Get(ctx, cmd.sessionID); snap != nil {
						if err := conn.Write(ctx, websocket.MessageBinary, snap); err != nil {
							h.logger.Debug("send snapshot on subscribe failed", "error", err)
						}
					}
				}
				ch := h.hub.SubscribeSession(client, cmd.sessionID)
				subs[cmd.sessionID] = activeSub{sessionID: cmd.sessionID, ch: ch}
				restartSubs()
			case "unsubscribe":
				if _, exists := subs[cmd.sessionID]; !exists {
					continue
				}
				h.hub.UnsubscribeSession(client, cmd.sessionID)
				delete(subs, cmd.sessionID)
			}
		}
	}
}

func (h *WSHandler) sendError(ctx context.Context, conn *websocket.Conn, code, msg string) {
	env, _ := proto.Encode(proto.Envelope{
		V:    proto.ProtocolVersion,
		Type: proto.MsgError,
		TS:   time.Now().UnixMilli(),
		Payload: proto.ErrorPayload{
			Code:    code,
			Message: msg,
		},
	})
	_ = conn.Write(ctx, websocket.MessageBinary, env)
}

// isFullSnapshotEnvelope 检测 envelope bytes 是否含 rrweb FullSnapshot 事件。
// 已废弃：用 extractFullSnapshotEnvelope 替代（支持 batch 模式）。
//
// 保留：测试代码可能引用。
func isFullSnapshotEnvelope(msg []byte) bool {
	env, err := proto.Decode(msg)
	if err != nil || env.Type != proto.MsgEvent {
		return false
	}
	var ep proto.EventPayload
	if err := proto.DecodePayload(env.Payload, &ep); err != nil {
		return false
	}
	return ep.Type == proto.EvRRWeb && ep.RRWeb != nil && ep.RRWeb.Type == 2
}

// extractFullSnapshotEnvelope 从 envelope（single 或 batch）中提取含 full snapshot 的子 envelope。
// 返回 nil 表示未找到。
//
// batch 模式：payload 是 array of EventPayload；找到 type=rrweb && rrweb.type=2 的项，
// 重新包装为单事件 envelope 返回。
// single 模式：payload 是 EventPayload；若满足 full snapshot，返回原 envelope bytes。
func extractFullSnapshotEnvelope(_ context.Context, env proto.Envelope) []byte {
	if env.Type != proto.MsgEvent {
		return nil
	}

	// 尝试解析为 single EventPayload
	var single proto.EventPayload
	if err := proto.DecodePayload(env.Payload, &single); err == nil {
		if single.Type == proto.EvRRWeb && single.RRWeb != nil && single.RRWeb.Type == 2 {
			// 原样返回 envelope（重新编码确保完整）
			if bytes, err := proto.Encode(env); err == nil {
				return bytes
			}
		}
		return nil
	}

	// 尝试解析为 array
	var arr []proto.EventPayload
	if err := proto.DecodePayload(env.Payload, &arr); err != nil {
		return nil
	}
	for _, ep := range arr {
		if ep.Type == proto.EvRRWeb && ep.RRWeb != nil && ep.RRWeb.Type == 2 {
			// 重新包装为单事件 envelope
			out := proto.Envelope{
				V:         env.V,
				Type:      proto.MsgEvent,
				SessionID: env.SessionID,
				TraceID:   env.TraceID,
				TS:        ep.TS,
				Payload:   ep,
			}
			if bytes, err := proto.Encode(out); err == nil {
				return bytes
			}
		}
	}
	return nil
}

// eventCountOf 计算 envelope 含的事件数（batch 时为 array 长度，single 时为 1）。
func eventCountOf(env proto.Envelope) int {
	if env.Type != proto.MsgEvent {
		return 0
	}
	// 尝试 array
	var arr []proto.EventPayload
	if err := proto.DecodePayload(env.Payload, &arr); err == nil {
		return len(arr)
	}
	// 尝试 single
	var single proto.EventPayload
	if err := proto.DecodePayload(env.Payload, &single); err == nil {
		return 1
	}
	return 0
}

// forEachEventPayload 解码 envelope（支持 single 和 batch 模式）并对每个 EventPayload 调用 fn。
// 用于 1i 行为分析等需要逐事件处理的场景。
func forEachEventPayload(env proto.Envelope, fn func(proto.EventPayload)) {
	if env.Type != proto.MsgEvent {
		return
	}
	var arr []proto.EventPayload
	if err := proto.DecodePayload(env.Payload, &arr); err == nil {
		for _, ep := range arr {
			fn(ep)
		}
		return
	}
	var single proto.EventPayload
	if err := proto.DecodePayload(env.Payload, &single); err == nil {
		fn(single)
	}
}
