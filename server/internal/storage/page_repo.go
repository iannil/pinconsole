// Package storage:pages 表方法（page-editor pe-1）。
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreatePage 创建新页面。slug 为空时由调用方填充。
func (s *Postgres) CreatePage(ctx context.Context, tenantID uuid.UUID, slug, title string) (*Page, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO pages (tenant_id, slug, title)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, slug, title, status, schema, created_at, updated_at
	`, tenantID, slug, title)

	var p Page
	if err := row.Scan(&p.ID, &p.TenantID, &p.Slug, &p.Title, &p.Status, &p.Schema, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	return &p, nil
}

// ListPages 返回指定 tenant 的所有页面（不包含 schema 完整内容）。
func (s *Postgres) ListPages(ctx context.Context, tenantID uuid.UUID) ([]*Page, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, tenant_id, slug, title, status, schema, created_at, updated_at
		FROM pages
		WHERE tenant_id = $1
		ORDER BY updated_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list pages: %w", err)
	}
	defer rows.Close()

	var out []*Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Slug, &p.Title, &p.Status, &p.Schema, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan page: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// GetPageBySlug 取指定 slug 的页面。不存在返回 (nil, nil)。
func (s *Postgres) GetPageBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*Page, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, slug, title, status, schema, created_at, updated_at
		FROM pages
		WHERE tenant_id = $1 AND slug = $2
	`, tenantID, slug)

	var p Page
	err := row.Scan(&p.ID, &p.TenantID, &p.Slug, &p.Title, &p.Status, &p.Schema, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get page by slug: %w", err)
	}
	return &p, nil
}

// UpdatePage 更新页面 schema 和/或 title。只更新非零字段。
func (s *Postgres) UpdatePage(ctx context.Context, tenantID uuid.UUID, slug string, title *string, schema []byte, status *string) (*Page, error) {
	// 构建动态 UPDATE — 仅更新提供的字段
	setClauses := ""
	args := []any{}
	argIdx := 1

	if title != nil {
		setClauses += fmt.Sprintf("title = $%d, ", argIdx)
		args = append(args, *title)
		argIdx++
	}
	if schema != nil {
		setClauses += fmt.Sprintf("schema = $%d, ", argIdx)
		args = append(args, schema)
		argIdx++
	}
	if status != nil {
		setClauses += fmt.Sprintf("status = $%d, ", argIdx)
		args = append(args, *status)
		argIdx++
	}
	setClauses += fmt.Sprintf("updated_at = $%d", argIdx)
	args = append(args, time.Now())
	argIdx++

	args = append(args, tenantID, slug)

	query := fmt.Sprintf(`
		UPDATE pages
		SET %s
		WHERE tenant_id = $%d AND slug = $%d
		RETURNING id, tenant_id, slug, title, status, schema, created_at, updated_at
	`, setClauses, argIdx, argIdx+1)

	row := s.Pool.QueryRow(ctx, query, args...)

	var p Page
	if err := row.Scan(&p.ID, &p.TenantID, &p.Slug, &p.Title, &p.Status, &p.Schema, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("update page: %w", err)
	}
	return &p, nil
}

// DeletePage 删除指定页面。
func (s *Postgres) DeletePage(ctx context.Context, tenantID uuid.UUID, slug string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM pages WHERE tenant_id = $1 AND slug = $2`, tenantID, slug)
	if err != nil {
		return fmt.Errorf("delete page: %w", err)
	}
	return nil
}

// DeletePageByID 按 ID 删除（用于测试清理）。
func (s *Postgres) DeletePageByID(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM pages WHERE id = $1`, id)
	return err
}
