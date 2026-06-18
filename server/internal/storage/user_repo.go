// Package storage:users 表方法(1u 拆自 queries.go)。
package storage

import (
	"context"

	"github.com/google/uuid"
)

// GetUserByEmail 1h:按 email 查用户。
func (s *Postgres) GetUserByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*User, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, display_name, role, created_at, updated_at
		FROM users WHERE tenant_id = $1 AND email = $2
	`, tenantID, email)
	return scanUser(row)
}

// GetUserByID 1h:按 ID 查用户。
func (s *Postgres) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, display_name, role, created_at, updated_at
		FROM users WHERE id = $1
	`, id)
	return scanUser(row)
}

// CreateUser 1h:创建用户。
func (s *Postgres) CreateUser(ctx context.Context, tenantID uuid.UUID, email, passwordHash, displayName, role string) (*User, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, email) DO NOTHING
		RETURNING id, tenant_id, email, password_hash, display_name, role, created_at, updated_at
	`, tenantID, email, passwordHash, displayName, role)
	return scanUser(row)
}

// CountUsers 1h:统计用户数。
func (s *Postgres) CountUsers(ctx context.Context) (int64, error) {
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}
