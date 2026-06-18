package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/iannil/marketing-monitor/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres 封装 PG 连接池。
type Postgres struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

// ConnectPostgres 建立 PG 连接池并验证。
//
// 1z:从 cfg.MaxConns 应用连接池上限(默认 25),
// 取代 pgxpool 默认的 max(4, NumCPU)。
func ConnectPostgres(ctx context.Context, cfg config.PostgresConfig, logger *slog.Logger) (*Postgres, error) {
	pgCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.MaxConns > 0 {
		pgCfg.MaxConns = int32(cfg.MaxConns)
	}
	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	logger.Info("postgres 已连接",
		"host", cfg.Host, "port", cfg.Port, "db", cfg.Database, "max_conns", pgCfg.MaxConns)
	return &Postgres{Pool: pool, logger: logger}, nil
}

// Ping 验证连接。
func (p *Postgres) Ping(ctx context.Context) error {
	return p.Pool.Ping(ctx)
}

// Close 关闭连接池。
func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
