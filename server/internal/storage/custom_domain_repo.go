// Package storage:custom_domains 表方法（cd-1 自定义域名）。
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetCustomDomain 取指定 tenant + domain 的域名配置。不存在返回 (nil, nil)。
func (s *Postgres) GetCustomDomain(ctx context.Context, tenantID uuid.UUID, domain string) (*CustomDomain, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, domain, cert_status, cert_error, created_at, updated_at
		FROM custom_domains
		WHERE tenant_id = $1 AND domain = $2
	`, tenantID, domain)

	var c CustomDomain
	err := row.Scan(&c.ID, &c.TenantID, &c.Domain, &c.CertStatus, &c.CertError, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get custom_domain: %w", err)
	}
	return &c, nil
}

// GetCustomDomainByDomain 仅按 domain 查找（用于 host-header 路由）。
func (s *Postgres) GetCustomDomainByDomain(ctx context.Context, domain string) (*CustomDomain, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, domain, cert_status, cert_error, created_at, updated_at
		FROM custom_domains
		WHERE domain = $1
	`, domain)

	var c CustomDomain
	err := row.Scan(&c.ID, &c.TenantID, &c.Domain, &c.CertStatus, &c.CertError, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get custom_domain by domain: %w", err)
	}
	return &c, nil
}

// ListCustomDomains 返回指定 tenant 的所有自定义域名。
func (s *Postgres) ListCustomDomains(ctx context.Context, tenantID uuid.UUID) ([]*CustomDomain, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, tenant_id, domain, cert_status, cert_error, created_at, updated_at
		FROM custom_domains
		WHERE tenant_id = $1
		ORDER BY domain
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list custom_domains: %w", err)
	}
	defer rows.Close()

	var out []*CustomDomain
	for rows.Next() {
		var c CustomDomain
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Domain, &c.CertStatus, &c.CertError, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan custom_domain: %w", err)
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// ListActiveCustomDomains 返回所有 cert_status='active' 的域名（跨 tenant，用于启动时加载）。
func (s *Postgres) ListActiveCustomDomains(ctx context.Context) ([]*CustomDomain, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, tenant_id, domain, cert_status, cert_error, created_at, updated_at
		FROM custom_domains
		WHERE cert_status = 'active'
		ORDER BY domain
	`)
	if err != nil {
		return nil, fmt.Errorf("list active custom_domains: %w", err)
	}
	defer rows.Close()

	var out []*CustomDomain
	for rows.Next() {
		var c CustomDomain
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Domain, &c.CertStatus, &c.CertError, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan custom_domain: %w", err)
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// CreateCustomDomain 创建自定义域名记录。已存在时返回 ErrConflict。
func (s *Postgres) CreateCustomDomain(ctx context.Context, tenantID uuid.UUID, domain string) (*CustomDomain, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO custom_domains (tenant_id, domain, cert_status)
		VALUES ($1, $2, 'pending')
		RETURNING id, tenant_id, domain, cert_status, cert_error, created_at, updated_at
	`, tenantID, domain)

	var c CustomDomain
	err := row.Scan(&c.ID, &c.TenantID, &c.Domain, &c.CertStatus, &c.CertError, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if pgErr := pgErrCode(err); pgErr == "23505" { // unique_violation
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create custom_domain: %w", err)
	}
	return &c, nil
}

// UpdateCustomDomainStatus 更新域名证书状态与错误信息。
func (s *Postgres) UpdateCustomDomainStatus(ctx context.Context, id int64, status, errMsg string) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE custom_domains SET cert_status = $1, cert_error = $2, updated_at = NOW()
		WHERE id = $3
	`, status, errMsg, id)
	if err != nil {
		return fmt.Errorf("update custom_domain status: %w", err)
	}
	return nil
}

// DeleteCustomDomain 删除自定义域名。
func (s *Postgres) DeleteCustomDomain(ctx context.Context, tenantID uuid.UUID, id int64) error {
	_, err := s.Pool.Exec(ctx, `
		DELETE FROM custom_domains WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete custom_domain: %w", err)
	}
	return nil
}

// DeleteCustomDomainByID 按 ID 删除（用于测试清理）。
func (s *Postgres) DeleteCustomDomainByID(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM custom_domains WHERE id = $1`, id)
	return err
}

// NowUTC 返回当前 UTC 时间，用于测试替换。
var NowUTC = func() time.Time { return time.Now().UTC() }
