package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/iannil/marketing-monitor/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxPool 是 Postgres.Pool 的最小接口(1ae R2 引入)。
// 仅声明 storage 包用到的 5 个方法 + Begin(migrations 用),让测试可注入事务包装器
// (用 ALTER FK NO ACTION 验证 erasure 显式 DELETE 真生效)。
// 生产代码用 *pgxpool.Pool 自动满足此接口。
type PgxPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	Ping(ctx context.Context) error
	Close()
}

// Postgres 封装 PG 连接池。
type Postgres struct {
	Pool   PgxPool
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
