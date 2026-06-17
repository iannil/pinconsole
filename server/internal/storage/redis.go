package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/iannil/marketing-monitor/internal/config"
	"github.com/redis/go-redis/v9"
)

// Redis 封装 Redis 客户端。
type Redis struct {
	Client *redis.Client
	logger *slog.Logger
}

// ConnectRedis 建立 Redis 客户端并验证。
func ConnectRedis(ctx context.Context, cfg config.RedisConfig, logger *slog.Logger) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	logger.Info("redis 已连接", "addr", cfg.Addr)
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
