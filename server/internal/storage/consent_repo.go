// Package storage:visitor_consents 表方法(1l + 1u 拆自 queries.go)。
package storage

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetLatestConsent 取 fingerprint 在指定 scope + version 下的最新同意状态。
// 返回 (consent, found);未找到时 found=false(调用方应按默认策略处理)。
func (s *Postgres) GetLatestConsent(ctx context.Context, tenantID uuid.UUID, fingerprint, scope, version string) (*VisitorConsent, bool, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, fingerprint, scope, version, accepted, consented_at, expires_at
		FROM visitor_consents
		WHERE tenant_id = $1 AND fingerprint = $2 AND scope = $3 AND version = $4
		ORDER BY consented_at DESC
		LIMIT 1
	`, tenantID, fingerprint, scope, version)
	var c VisitorConsent
	if err := row.Scan(&c.ID, &c.TenantID, &c.Fingerprint, &c.Scope, &c.Version,
		&c.Accepted, &c.ConsentedAt, &c.ExpiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &c, true, nil
}

// UpsertConsent 写入或更新同意状态。
// 同 (fingerprint, scope, version) 只保留最新;旧记录被替换。
func (s *Postgres) UpsertConsent(ctx context.Context, tenantID uuid.UUID, fingerprint, scope, version string, accepted bool) (*VisitorConsent, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO visitor_consents (tenant_id, fingerprint, scope, version, accepted)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (fingerprint, scope, version)
		DO UPDATE SET accepted = EXCLUDED.accepted, consented_at = NOW()
		RETURNING id, tenant_id, fingerprint, scope, version, accepted, consented_at, expires_at
	`, tenantID, fingerprint, scope, version, accepted)
	var c VisitorConsent
	if err := row.Scan(&c.ID, &c.TenantID, &c.Fingerprint, &c.Scope, &c.Version,
		&c.Accepted, &c.ConsentedAt, &c.ExpiresAt); err != nil {
		return nil, err
	}
	return &c, nil
}
