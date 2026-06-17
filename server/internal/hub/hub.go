// Package hub 管理 WebSocket 连接、房间与订阅。
//
// 设计（详见 docs/progress/2026-06-17-slice-1b-spec.md §Hub 路由）：
//
//   - 每个 visitor 连接 = 一个 session 通道（session:<id>）
//   - 每个 operator 连接默认加入租户房间（room:tenant:<id>）
//   - operator 显式 subscribe session:<id> 后，开始接收该 visitor 的高频事件
//   - visitor 上线/下线广播到 room:tenant:<id>（presence）
//   - 高频事件只发给已订阅该 session 的 operator
//
// hub 是单例，进程内。所有 client 与 room 都注册到 Hub。
package hub

import (
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// Hub 是连接与房间的中心注册表。
type Hub struct {
	mu       sync.RWMutex
	clients  map[uuid.UUID]*Client      // 连接 ID → client
	sessions map[uuid.UUID]*SessionChan // session_id → 该 session 的事件广播通道
	tenants  map[uuid.UUID]*TenantRoom  // tenant_id → 该租户的 presence 房间
	// 1e：sessionID → visitor client ID（用于反向下行命令）
	visitorClients map[uuid.UUID]uuid.UUID
	logger         *slog.Logger
}

// SessionChan 是一个访客 session 的事件广播通道。
// 订阅者通过 Subscribe 获取接收 channel；发布者通过 Publish 广播。
type SessionChan struct {
	SessionID uuid.UUID
	mu        sync.RWMutex
	subs      map[uuid.UUID]chan []byte // subID → chan
}

// TenantRoom 是租户级 presence 房间。
// 用于广播 visitor 上线/下线事件给该租户的所有 operator。
type TenantRoom struct {
	TenantID uuid.UUID
	mu       sync.RWMutex
	subs     map[uuid.UUID]chan []byte
}

// New 创建 Hub 实例。
func New(logger *slog.Logger) *Hub {
	return &Hub{
		clients:        make(map[uuid.UUID]*Client),
		sessions:       make(map[uuid.UUID]*SessionChan),
		tenants:        make(map[uuid.UUID]*TenantRoom),
		visitorClients: make(map[uuid.UUID]uuid.UUID),
		logger:         logger,
	}
}

// RegisterClient 注册一个新连接。
func (h *Hub) RegisterClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.ID] = c
	h.logger.Info("client registered",
		"client_id", c.ID, "role", c.Role,
	)
}

// UnregisterClient 注销连接并清理订阅。
func (h *Hub) UnregisterClient(c *Client) {
	h.mu.Lock()
	delete(h.clients, c.ID)
	h.mu.Unlock()

	// 清理该 client 在所有 session 与 tenant room 中的订阅
	c.mu.Lock()
	sessions := make([]uuid.UUID, 0, len(c.subscribedSessions))
	for sid := range c.subscribedSessions {
		sessions = append(sessions, sid)
	}
	tenants := make([]uuid.UUID, 0, len(c.joinedTenants))
	for tid := range c.joinedTenants {
		tenants = append(tenants, tid)
	}
	c.subscribedSessions = make(map[uuid.UUID]struct{})
	c.joinedTenants = make(map[uuid.UUID]struct{})
	c.mu.Unlock()

	for _, sid := range sessions {
		if sc := h.session(sid); sc != nil {
			sc.unsubscribeAll(c.ID)
		}
	}
	for _, tid := range tenants {
		if tr := h.tenant(tid); tr != nil {
			tr.unsubscribeAll(c.ID)
		}
	}
	h.logger.Info("client unregistered", "client_id", c.ID)
}

// VisitorOnline 注册一个 visitor 上线，创建对应的 SessionChan，
// 并向该租户的 TenantRoom 广播 presence.online。
// 1e：同时记录 sessionID → visitorClientID 映射，供 SendCommandToVisitor 用。
func (h *Hub) VisitorOnline(tenantID, sessionID uuid.UUID, visitorClientID uuid.UUID, presenceMsg []byte) {
	h.mu.Lock()
	if _, exists := h.sessions[sessionID]; !exists {
		h.sessions[sessionID] = &SessionChan{
			SessionID: sessionID,
			subs:      make(map[uuid.UUID]chan []byte),
		}
	}
	h.visitorClients[sessionID] = visitorClientID
	h.mu.Unlock()

	if tr := h.tenant(tenantID); tr != nil {
		tr.publish(presenceMsg)
	}
}

// VisitorOffline 移除 visitor 的 SessionChan，
// 并向 TenantRoom 广播 presence.offline。
// 1e：同时清理 visitorClients 映射。
func (h *Hub) VisitorOffline(tenantID, sessionID uuid.UUID, presenceMsg []byte) {
	h.mu.Lock()
	if sc, ok := h.sessions[sessionID]; ok {
		sc.unsubscribeAllLocking()
		delete(h.sessions, sessionID)
	}
	delete(h.visitorClients, sessionID)
	h.mu.Unlock()

	if tr := h.tenant(tenantID); tr != nil {
		tr.publish(presenceMsg)
	}
}

// SendCommandToVisitor 1e：把命令字节流下发给指定 session 的 visitor client。
// 若 visitor 不在线，返回 false（调用方应放弃或缓存）。
func (h *Hub) SendCommandToVisitor(sessionID uuid.UUID, msg []byte) bool {
	h.mu.RLock()
	visitorID, ok := h.visitorClients[sessionID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	h.mu.RLock()
	client := h.clients[visitorID]
	h.mu.RUnlock()
	if client == nil {
		return false
	}
	return client.Send(msg)
}

// PublishEvent 把一个事件广播给所有订阅了 sessionID 的 client。
func (h *Hub) PublishEvent(sessionID uuid.UUID, msg []byte) {
	h.mu.RLock()
	sc := h.sessions[sessionID]
	h.mu.RUnlock()
	if sc == nil {
		return
	}
	sc.publish(msg)
}

// SubscribeSession 让 client 订阅一个 session 的事件流。
// 返回接收 channel；client 关闭时必须 UnsubscribeSession。
func (h *Hub) SubscribeSession(c *Client, sessionID uuid.UUID) <-chan []byte {
	h.mu.Lock()
	sc, ok := h.sessions[sessionID]
	if !ok {
		sc = &SessionChan{
			SessionID: sessionID,
			subs:      make(map[uuid.UUID]chan []byte),
		}
		h.sessions[sessionID] = sc
	}
	h.mu.Unlock()

	ch := sc.subscribe(c.ID)
	c.mu.Lock()
	c.subscribedSessions[sessionID] = struct{}{}
	c.mu.Unlock()
	return ch
}

// UnsubscribeSession 让 client 退订一个 session。
func (h *Hub) UnsubscribeSession(c *Client, sessionID uuid.UUID) {
	h.mu.RLock()
	sc := h.sessions[sessionID]
	h.mu.RUnlock()
	if sc == nil {
		return
	}
	sc.unsubscribe(c.ID)
	c.mu.Lock()
	delete(c.subscribedSessions, sessionID)
	c.mu.Unlock()
}

// JoinTenantRoom 让 client 加入租户 presence 房间。
func (h *Hub) JoinTenantRoom(c *Client, tenantID uuid.UUID) <-chan []byte {
	h.mu.Lock()
	tr, ok := h.tenants[tenantID]
	if !ok {
		tr = &TenantRoom{
			TenantID: tenantID,
			subs:     make(map[uuid.UUID]chan []byte),
		}
		h.tenants[tenantID] = tr
	}
	h.mu.Unlock()

	ch := tr.subscribe(c.ID)
	c.mu.Lock()
	c.joinedTenants[tenantID] = struct{}{}
	c.mu.Unlock()
	return ch
}

// 内部辅助：取 session / tenant channel（线程安全）。
func (h *Hub) session(id uuid.UUID) *SessionChan {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sessions[id]
}

func (h *Hub) tenant(id uuid.UUID) *TenantRoom {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.tenants[id]
}
