package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/iannil/pinconsole/internal/config"
	"github.com/redis/go-redis/v9"
)

// Redis 封装 Redis 客户端。
type Redis struct {
	Client *redis.Client
	logger *slog.Logger
}

// ConnectRedis 建立 Redis 客户端并验证。
//
// 1z:从 cfg.PoolSize 应用连接池上限(默认 50),
// 取代 go-redis 默认的 10*NumCPU。
func ConnectRedis(ctx context.Context, cfg config.RedisConfig, logger *slog.Logger) (*Redis, error) {
	opts := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
	}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	logger.Info("redis 已连接", "addr", cfg.Addr, "pool_size", opts.PoolSize)
	return &Redis{Client: client, logger: logger}, nil
}

// Ping 验证连接。
func (r *Redis) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Close 关闭连接。
func (r *Redis) Close() {
	if r.Client != nil {
		_ = r.Client.Close()
	}
}

// Set 设置 KV，支持 TTL（0 = 永久）。
func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.Client.Set(ctx, key, value, ttl).Err()
}

// SetNX 原子设置 KV 仅当 key 不存在，支持 TTL。
// 返回 true 表示成功（之前无 key），false 表示 key 已存在（未写入）。
// 1k P0-4：claim 原子锁使用。
func (r *Redis) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	ok, err := r.Client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("setnx %s: %w", key, err)
	}
	return ok, nil
}

// EvalLua 执行 Lua 脚本。
// 1k P0-4：release claim 原子对比 owner 再 DEL。
func (r *Redis) EvalLua(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return r.Client.Eval(ctx, script, keys, args...).Result()
}

// Get 取 KV。key 不存在返回 nil + nil error。
func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get %s: %w", key, err)
	}
	return val, nil
}

// Del 删除 key。
func (r *Redis) Del(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}

// TTL 返回 key 的剩余生存时间(秒级精度)。
// key 不存在返回 -1(无 TTL)或 -2(key 不存在),与 go-redis 一致。
// 1ai-c:加此方法替代 auth.go 直接调 Client.TTL,让接口化重构可行。
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.Client.TTL(ctx, key).Result()
}
