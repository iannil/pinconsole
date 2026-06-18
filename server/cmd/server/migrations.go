// Package main：migrations 启动自动应用（1k P0-14）。
//
// 设计：手写最小 migrator（不引入 golang-migrate 依赖）。
//   - migrations/*.up.sql 嵌入二进制（见 server/migrations/embed.go）
//   - schema_migrations 表跟踪已应用版本
//   - pg_advisory_lock 防多实例并发
//   - 启动失败 panic 退出（fail-fast）
//
// 文件命名约定：000001_init.up.sql、000001_init.down.sql（version 为前 6 位）。
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/iannil/marketing-monitor/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationAdvisoryLockID 是 pg_advisory_lock 的固定 key（任意 int64）。
// 用项目开始日期 2026-06-18 → 20260618 防止与其他服务冲突。
const migrationAdvisoryLockID = 20260618

// runMigrations 应用所有未执行的 up migration。
//
// 流程：
//  1. pg_advisory_lock 防并发（同库多实例场景）
//  2. CREATE TABLE IF NOT EXISTS schema_migrations
//  3. 按 version 顺序遍历 embedded up.sql，未应用的在事务里执行 + 记录版本
//  4. pg_advisory_unlock（defer）
func runMigrations(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	// 1. advisory lock
	if _, err := pool.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationAdvisoryLockID); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, "SELECT pg_advisory_unlock($1)", migrationAdvisoryLockID); err != nil {
			logger.WarnContext(ctx, "advisory unlock failed", "error", err)
		}
	}()

	// 2. schema_migrations 表
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// 3. 列出 up 文件
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	var ups []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		ups = append(ups, name)
	}
	sort.Strings(ups)
	if len(ups) == 0 {
		return fmt.Errorf("no migrations found embedded")
	}

	// 4. 顺序应用
	applied := 0
	for _, name := range ups {
		version, err := parseMigrationVersion(name)
		if err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}

		var count int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count); err != nil {
			return fmt.Errorf("check version %d: %w", version, err)
		}
		if count > 0 {
			continue
		}

		sqlBytes, err := migrations.Files.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}

		applied++
		logger.InfoContext(ctx, "migration applied", "file", name, "version", version)
	}

	if applied == 0 {
		logger.InfoContext(ctx, "migrations up to date")
	} else {
		logger.InfoContext(ctx, "migrations applied", "count", applied)
	}
	return nil
}

// parseMigrationVersion 从文件名提取 version 数字。
// 000001_init.up.sql → 1。
func parseMigrationVersion(name string) (int, error) {
	var version int
	if _, err := fmt.Sscanf(name, "%06d_", &version); err != nil {
		return 0, fmt.Errorf("sscanf version: %w", err)
	}
	return version, nil
}
