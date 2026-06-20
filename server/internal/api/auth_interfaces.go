// 1ai-c:AuthHandler 用接口替代具体 *storage.Stores,解锁 mock 注入。
//
// 设计:
//   - 接口定义在 api 包(消费者侧),遵循 "accept interfaces, return structs"
//   - 接口仅声明 AuthHandler 实际使用的方法(ISP,避免暴露 100+ 方法)
//   - *storage.Postgres / *storage.Redis 自动满足接口
//   - 测试用手写 mock(简单可控,见 auth_happy_path_test.go)
package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
)

// authUserRepo 是 AuthHandler 需要的 user 查询接口。
// *storage.Postgres 自动满足。
type authUserRepo interface {
	GetUserByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*storage.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*storage.User, error)
}

// authRedisStore 是 AuthHandler 需要的 Redis 操作接口。
// *storage.Redis 自动满足(1ai-c 加了 TTL 方法)。
type authRedisStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	EvalLua(ctx context.Context, script string, keys []string, args ...any) (any, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
}
