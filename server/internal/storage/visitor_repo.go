// Package storage:visitors 表方法(1u 拆自 queries.go)。
package storage

import (
	"context"

	"github.com/google/uuid"
)

// GetVisitorByFingerprint 按租户与 fingerprint 查找访客。
func (s *Postgres) GetVisitorByFingerprint(ctx context.Context, tenantID uuid.UUID, fingerprint string) (*Visitor, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, fingerprint, ua, ip_first_seen::text,
		       first_seen_at, last_seen_at, meta
		FROM visitors
		WHERE tenant_id = $1 AND fingerprint = $2
	`, tenantID, fingerprint)
	v, err := scanVisitor(row)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// CreateVisitor 创建访客或更新已存在的 last_seen_at。
func (s *Postgres) CreateVisitor(ctx context.Context, tenantID uuid.UUID, fingerprint, ua, ip string) (*Visitor, error) {
	var uaArg any
	if ua != "" {
		uaArg = ua
	}
	var ipArg any
	if ip != "" {
		ipArg = ip
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO visitors (tenant_id, fingerprint, ua, ip_first_seen)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id, fingerprint)
		DO UPDATE SET
			last_seen_at = NOW(),
			ua = COALESCE(EXCLUDED.ua, visitors.ua)
		RETURNING id, tenant_id, fingerprint, ua, ip_first_seen::text,
		          first_seen_at, last_seen_at, meta
	`, tenantID, fingerprint, uaArg, ipArg)
	return scanVisitor(row)
}
