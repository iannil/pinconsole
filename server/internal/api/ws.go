// Package api：WebSocket 端点 /ws/visitor 与 /ws/operator。
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/antiscrape"
	"github.com/iannil/pinconsole/internal/hub"
	"github.com/iannil/pinconsole/internal/logging"
	"github.com/iannil/pinconsole/internal/proto"
	"github.com/iannil/pinconsole/internal/recording"
	"github.com/iannil/pinconsole/internal/storage"
	"github.com/redis/go-redis/v9"
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
	// devMode 控制 operatorWS 的 dev bypass(与 AuthMiddleware 一致)。
	// release build 下 tryDevBypass 恒 false,即使 devMode=true 也不绕过。
	devMode bool
}

// 1y P1-4:visitor WS rate limit 常量。
// 阈值估算:正常 SDK 流量 ~10 env/sec + ~1KB/env = 100 env/10s + 100KB/10s。
// 阈值 500 env/10s + 50 MiB/10s 是 5x/500x 余量,只抓真攻击。
const (
	wsRateLimitWindow   = 10 * time.Second
	wsRateLimitMaxMsgs  = 500
	wsRateLimitMaxBytes = 50 * 1024 * 1024 // 50 MiB
)

// wsRateLimitLua 原子 INCR count + INCR bytes + 首次失败时 EXPIRE(2 keys)。
// 返回 {count, bytes},调用方比对阈值。
const wsRateLimitLua = `
local c = redis.call('INCR', KEYS[1])
local b = redis.call('INCRBY', KEYS[2], ARGV[1])
if c == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
end
if b == tonumber(ARGV[1]) then
  redis.call('EXPIRE', KEYS[2], ARGV[2])
end
return {c, b}
`

// checkWSRateLimit 检查 + 记录一次 envelope。
// 返回 (allowed, reason)。allowed=false 时调用方应 FlagSession + close conn。
// Redis 故障时返回 (true, "", err) 让调用方 fail-open。
func checkWSRateLimit(ctx context.Context, rdb *redis.Client, sessionID string, msgSize int) (bool, string, error) {
	countKey := fmt.Sprintf("ws:rate:count:%s", sessionID)
	bytesKey := fmt.Sprintf("ws:rate:bytes:%s", sessionID)
	res, err := rdb.Eval(ctx, wsRateLimitLua,
		[]string{countKey, bytesKey},
		msgSize, int(wsRateLimitWindow.Seconds())).Result()
	if err != nil {
		return true, "", err // fail-open
	}
	// res 是 []interface{}{count(int64), bytes(int64)}
	vals, ok := res.([]interface{})
	if !ok || len(vals) != 2 {
		return true, "", fmt.Errorf("unexpected lua result type %T", res)
	}
	count, _ := vals[0].(int64)
	bytesVal, _ := vals[1].(int64)
	if count > int64(wsRateLimitMaxMsgs) {
		return false, fmt.Sprintf("ws_rate_exceeded_msgs:%d", count), nil
	}
	if bytesVal > int64(wsRateLimitMaxBytes) {
		return false, fmt.Sprintf("ws_rate_exceeded_bytes:%d", bytesVal), nil
	}
	return true, "", nil
}

// NewWSHandler 创建 WS handler。
func NewWSHandler(h *hub.Hub, stores *storage.Stores, stream *recording.Stream, flusher *recording.Flusher, snapshots *recording.SnapshotCache, logger *slog.Logger, devMode bool) *WSHandler {
	return &WSHandler{
		hub:       h,
		stores:    stores,
		stream:    stream,
		flusher:   flusher,
		snapshots: snapshots,
		logger:    logger,
		maxMsg:    1 << 20, // 1 MiB
		devMode:   devMode,
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
//  1. accept WS
//  2. 等待 hello 消息（含 visitor_id + session_id）
//  3. 验证 session_id 存在
//  4. hub.VisitorOnline（注册 session chan + 广播 presence.online）
//  5. 启动 read loop：每个 MsgEvent 写 Redis Stream + PublishEvent
//  6. 连接断开：hub.VisitorOffline + DB EndSession
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
		// 1m:SDK 发的 trace_id 写回 ctx,使后续日志能关联 SDK 端
		if env.TraceID != "" {
			ctx = logging.WithTraceID(ctx, env.TraceID)
			logger = logging.FromContext(ctx, h.logger)
		}
		if env.Type != proto.MsgEvent {
			continue
		}

		// 1y P1-4:per-session 滑动窗口 rate limit(10s/500 env/50MiB)。
		// 在 stream.Append 之前检查,避免被限流的 envelope 也写到 Redis Stream/MinIO。
		// 超限:FlagSession(让 admin 看到)+ force close(PolicyViolation)。
		// Redis 故障 fail-open(只 warn),与 1i 行为一致。
		allowed, reason, rlErr := checkWSRateLimit(ctx, h.stores.Redis.Client, sessionID.String(), len(msg))
		if rlErr != nil {
			logger.Warn("ws rate limit check failed (fail-open)", "error", rlErr)
		} else if !allowed {
			logger.Warn("visitor ws rate limit exceeded, closing",
				"session_id", sessionID, "reason", reason, "msg_size", len(msg))
			if flagErr := antiscrape.FlagSession(ctx, h.stores.Redis.Client, sessionID.String(), reason); flagErr != nil {
				logger.Warn("FlagSession failed (already closing)", "error", flagErr)
			}
			conn.Close(websocket.StatusPolicyViolation, "rate limit exceeded")
			return
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
//  1. accept WS
//  2. 等待 hello（可选，1b 不强制 capabilities）
//  3. 默认加入 room:tenant:<id> 接收 presence
//  4. read loop：subscribe / unsubscribe 命令
//  5. 同时 select tenant room chan 与 session chan（已订阅的）
func (h *WSHandler) operatorWS(c *gin.Context) {
	// 1ac-final T0-1h-2 修复:operatorWS 必须校验 cookie session。
	// 此前完全无认证,任意匿名客户端可连 /ws/operator 接收 visitor 事件流。
	_, authOK := authenticateOperatorWS(c, h.stores.Redis, h.devMode)
	if !authOK {
		// authenticateOperatorWS 已写 401 响应
		return
	}

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
		action    string // subscribe / unsubscribe
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
	// 1o P1-8 修复:每个 sub 启动一个 goroutine,但用 per-sub cancel context 防止泄漏
	forwardCh := make(chan []byte, 256)
	defer close(forwardCh)

	// subCancels 跟踪每个 session sub 的 cancel 函数,unsubscribe 时调用
	subCancels := make(map[uuid.UUID]context.CancelFunc, 8)
	defer func() {
		// 退出时 cancel 所有 sub goroutine
		for _, cancel := range subCancels {
			cancel()
		}
	}()

	restartSubs := func() {
		// 启动新 sub 的 goroutine(已在 subs 中但 ch 未 listen)
		// 1o P1-8:每个 sub 用独立 cancel ctx,unsubscribe 时调用 cancel 让 goroutine 退出
		for sid, s := range subs {
			if s.ch == nil {
				continue
			}
			ch := s.ch
			sid := sid
			subCtx, subCancel := context.WithCancel(ctx)
			subCancels[sid] = subCancel
			go func() {
				defer subCancel()
				for {
					select {
					case <-subCtx.Done():
						return
					case msg, ok := <-ch:
						if !ok {
							return
						}
						select {
						case forwardCh <- msg:
						case <-subCtx.Done():
							return
						}
					}
				}
			}()
			// 标记已 listen,避免重复(同一 session 多次 subscribe 时只 listen 一次)
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
				// 1w P1-29:订阅 flagged session 时 warn(不阻断,留给运营决策)
				if h.stores != nil && h.stores.Redis != nil && h.stores.Redis.Client != nil {
					if flagged, reason, err := antiscrape.IsSessionFlagged(ctx, h.stores.Redis.Client, cmd.sessionID.String()); err != nil {
						h.logger.WarnContext(ctx, "is_session_flagged check failed on subscribe",
							"session_id", cmd.sessionID, "error", err)
					} else if flagged {
						h.logger.WarnContext(ctx, "operator subscribing to flagged session",
							"session_id", cmd.sessionID, "flag_reason", reason,
							"note", "behavior tracker marked this session as suspicious")
					}
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
				// 1o P1-8:cancel 该 sub 的 forwarder goroutine
				if cancel, ok := subCancels[cmd.sessionID]; ok {
					cancel()
					delete(subCancels, cmd.sessionID)
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

// extractFullSnapshotEnvelope 从 envelope(single 或 batch)中提取含 full snapshot 的子 envelope。
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
