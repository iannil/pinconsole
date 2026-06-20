// 1ai-e:ClaimHandler + ChatHandler 接口化(1ai-c/d 模式扩展)。
//
// 设计同 auth_interfaces.go:
//   - 接口定义在 api 包(消费者侧)
//   - *storage.Postgres / *storage.Redis 自动满足
//   - 接口仅声明 handler 实际使用的方法(ISP)
package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
)

// claimSessionRepo 是 ClaimHandler.claim 需要的 session 查询接口。
// *storage.Postgres 自动满足。
type claimSessionRepo interface {
	GetSession(ctx context.Context, id uuid.UUID) (*storage.Session, error)
}

// claimRedisStore 是 ClaimHandler 需要的 Redis 操作接口。
// 与 authRedisStore 重叠但不完全一样(claim 用 SetNX,auth 用 Set)。
// *storage.Redis 同时满足两者。
type claimRedisStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	EvalLua(ctx context.Context, script string, keys []string, args ...any) (any, error)
}

// chatMessageRepo 是 ChatHandler.listMessages 需要的消息查询接口。
// *storage.Postgres 自动满足。
type chatMessageRepo interface {
	ListChatMessagesBySession(ctx context.Context, sessionID uuid.UUID, sinceID int64, limit int32) ([]storage.ChatMessage, error)
}

// commandRepo 是 CommandHandler.postCommand 需要的命令写入接口(1ai-f)。
// *storage.Postgres 自动满足。
type commandRepo interface {
	CreateCoBrowsingCommand(ctx context.Context, cmd storage.CoBrowsingCommand) (*storage.CoBrowsingCommand, error)
}
