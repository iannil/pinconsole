// Package storage 包含 PG / Redis / MinIO 连接适配器。
//
// 切片 1a 仅建立连接 + Ping 验证，业务方法从 1b 起加入。
package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/iannil/marketing-monitor/internal/config"
)

// Stores 聚合三个后端依赖。
type Stores struct {
	PG    *Postgres
	Redis *Redis
	MinIO *MinIO
}

// Connect 同时建立 PG / Redis / MinIO 连接。
func Connect(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Stores, error) {
	pg, err := ConnectPostgres(ctx, cfg.Postgres, logger)
	if err != nil {
		return nil, fmt.Errorf("pg: %w", err)
	}
	rdb, err := ConnectRedis(ctx, cfg.Redis, logger)
	if err != nil {
		pg.Close()
		return nil, fmt.Errorf("redis: %w", err)
	}
	mio, err := ConnectMinIO(ctx, cfg.MinIO, logger)
	if err != nil {
		pg.Close()
		rdb.Close()
		return nil, fmt.Errorf("minio: %w", err)
	}
	return &Stores{PG: pg, Redis: rdb, MinIO: mio}, nil
}

// Close 关闭全部连接。
func (s *Stores) Close() {
	if s.PG != nil {
		s.PG.Close()
	}
	if s.Redis != nil {
		s.Redis.Close()
	}
	if s.MinIO != nil {
		s.MinIO.Close()
	}
}

// PingAll 并发 ping 全部存储，返回首个错误。
func (s *Stores) PingAll(ctx context.Context) error {
	if err := s.PG.Ping(ctx); err != nil {
		return fmt.Errorf("pg: %w", err)
	}
	if err := s.Redis.Ping(ctx); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	if err := s.MinIO.Ping(ctx); err != nil {
		return fmt.Errorf("minio: %w", err)
	}
	return nil
}
