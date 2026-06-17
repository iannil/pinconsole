// Package hub：单个 WS 连接的封装。
package hub

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// Role 区分 visitor 与 operator。
type Role string

const (
	RoleVisitor  Role = "visitor"
	RoleOperator Role = "operator"
)

// Client 是单个 WebSocket 连接的封装。
type Client struct {
	ID   uuid.UUID
	Role Role
	Conn *websocket.Conn

	logger *slog.Logger

	// 订阅状态（hub 维护）
	mu                  sync.RWMutex
	subscribedSessions  map[uuid.UUID]struct{}
	joinedTenants       map[uuid.UUID]struct{}

	// 写入：每条消息通过 writeCh 串行化发到 Conn
	writeCh   chan []byte
	closeCh   chan struct{}
	closeOnce sync.Once
}

// NewClient 创建 client 并启动写循环。
func NewClient(id uuid.UUID, role Role, conn *websocket.Conn, logger *slog.Logger) *Client {
	c := &Client{
		ID:                 id,
		Role:               role,
		Conn:               conn,
		logger:             logger,
		subscribedSessions: make(map[uuid.UUID]struct{}),
		joinedTenants:      make(map[uuid.UUID]struct{}),
		writeCh:            make(chan []byte, 256),
		closeCh:            make(chan struct{}),
	}
	go c.writeLoop()
	return c
}

// Send 把消息加入写队列。队列满时返回 false（调用方应关闭此 client）。
func (c *Client) Send(b []byte) bool {
	select {
	case c.writeCh <- b:
		return true
	case <-c.closeCh:
		return false
	default:
		// 队列满，丢弃并记录；防止慢消费方拖垮 hub
		c.logger.Warn("client write queue full, dropping",
			"client_id", c.ID, "role", c.Role)
		return false
	}
}

// Close 关闭 client（幂等）。
func (c *Client) Close(ctx context.Context) {
	c.closeOnce.Do(func() {
		close(c.closeCh)
		_ = c.Conn.Close(websocket.StatusNormalClosure, "")
	})
}

// writeLoop 串行化写入底层 Conn。
func (c *Client) writeLoop() {
	for {
		select {
		case msg := <-c.writeCh:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := c.Conn.Write(ctx, websocket.MessageBinary, msg)
			cancel()
			if err != nil {
				c.logger.Debug("send failed, closing client",
					"client_id", c.ID, "error", err)
				c.Close(context.Background())
				return
			}
		case <-c.closeCh:
			return
		}
	}
}
