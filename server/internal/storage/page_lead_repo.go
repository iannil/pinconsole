// Package storage:page_leads 表方法（page-editor pe-1）。
package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// InsertPageLead 创建表单提交记录。
func (s *Postgres) InsertPageLead(ctx context.Context, tenantID uuid.UUID, pageSlug string, fields []byte) (*PageLead, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO page_leads (tenant_id, page_slug, fields)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, page_slug, fields, created_at
	`, tenantID, pageSlug, fields)

	var l PageLead
	if err := row.Scan(&l.ID, &l.TenantID, &l.PageSlug, &l.Fields, &l.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert page_lead: %w", err)
	}
	return &l, nil
}

// ListPageLeads 返回指定页面的所有表单提交。
func (s *Postgres) ListPageLeads(ctx context.Context, tenantID uuid.UUID, pageSlug string) ([]*PageLead, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, tenant_id, page_slug, fields, created_at
		FROM page_leads
		WHERE tenant_id = $1 AND page_slug = $2
		ORDER BY created_at DESC
	`, tenantID, pageSlug)
	if err != nil {
		return nil, fmt.Errorf("list page_leads: %w", err)
	}
	defer rows.Close()

	var out []*PageLead
	for rows.Next() {
		var l PageLead
		if err := rows.Scan(&l.ID, &l.TenantID, &l.PageSlug, &l.Fields, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan page_lead: %w", err)
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

// DeletePageLeadByID 按 ID 删除（用于测试清理）。
func (s *Postgres) DeletePageLeadByID(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM page_leads WHERE id = $1`, id)
	return err
}
