// Package storage:sessions 表方法(1u 拆自 queries.go)。
package storage

import (
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// CreateSession 创建新会话。
func (s *Postgres) CreateSession(ctx context.Context, tenantID, visitorID uuid.UUID, ua, ip string) (*Session, error) {
	var uaArg any
	if ua != "" {
		uaArg = ua
	}
	var ipArg any
	if ip != "" {
		ipArg = ip
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO sessions (tenant_id, visitor_id, ua, ip)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, visitor_id, started_at, last_event_at,
		          ended_at, status, event_count, ua, ip::text
	`, tenantID, visitorID, uaArg, ipArg)
	return scanSession(row)
}

// GetSession 按 ID 取会话。
func (s *Postgres) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, visitor_id, started_at, last_event_at,
		       ended_at, status, event_count, ua, ip::text
		FROM sessions WHERE id = $1
	`, id)
	return scanSession(row)
}

// TouchSessionEvent 更新会话最近事件时间与计数。
func (s *Postgres) TouchSessionEvent(ctx context.Context, id uuid.UUID, addedEvents int32) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE sessions
		SET last_event_at = NOW(), event_count = event_count + $2
		WHERE id = $1
	`, id, addedEvents)
	return err
}

// EndSession 关闭会话。
func (s *Postgres) EndSession(ctx context.Context, id uuid.UUID, status string) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE sessions SET ended_at = NOW(), status = $2 WHERE id = $1
	`, id, status)
	return err
}

// ListActiveSessionsByTenant 列出租户下所有活跃会话(含 visitor fingerprint)。
func (s *Postgres) ListActiveSessionsByTenant(ctx context.Context, tenantID uuid.UUID, limit int32) ([]Session, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT s.id, s.tenant_id, s.visitor_id, s.started_at, s.last_event_at,
		       s.ended_at, s.status, s.event_count, s.ua, s.ip::text,
		       v.fingerprint
		FROM sessions s
		JOIN visitors v ON v.id = s.visitor_id
		WHERE s.tenant_id = $1 AND s.status = 'active'
		ORDER BY s.last_event_at DESC NULLS LAST
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Session
	for rows.Next() {
		var sess Session
		var fp string
		var uaPtr *string
		var ipStr *string
		if err := rows.Scan(
			&sess.ID, &sess.TenantID, &sess.VisitorID, &sess.StartedAt, &sess.LastEventAt,
			&sess.EndedAt, &sess.Status, &sess.EventCount, &uaPtr, &ipStr,
			&fp,
		); err != nil {
			return nil, err
		}
		sess.UA = uaPtr
		if ipStr != nil {
			if addr, err := netip.ParseAddr(*ipStr); err == nil {
				sess.IP = &addr
			}
		}
		sess.VisitorFingerprint = &fp
		out = append(out, sess)
	}
	return out, rows.Err()
}

// ListEndedSessionsByTenant 列出租户下指定时间窗口内已结束的会话。
// since 参数:24h / 7d / 30d 等(Go duration 字符串)。
func (s *Postgres) ListEndedSessionsByTenant(ctx context.Context, tenantID uuid.UUID, since time.Duration, limit int32) ([]Session, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT s.id, s.tenant_id, s.visitor_id, s.started_at, s.last_event_at,
		       s.ended_at, s.status, s.event_count, s.ua, s.ip::text,
		       v.fingerprint
		FROM sessions s
		JOIN visitors v ON v.id = s.visitor_id
		WHERE s.tenant_id = $1
		  AND s.status IN ('ended', 'timed_out')
		  AND s.ended_at >= NOW() - $2::interval
		ORDER BY s.ended_at DESC
		LIMIT $3
	`, tenantID, since.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Session
	for rows.Next() {
		var sess Session
		var fp string
		var uaPtr *string
		var ipStr *string
		if err := rows.Scan(
			&sess.ID, &sess.TenantID, &sess.VisitorID, &sess.StartedAt, &sess.LastEventAt,
			&sess.EndedAt, &sess.Status, &sess.EventCount, &uaPtr, &ipStr,
			&fp,
		); err != nil {
			return nil, err
		}
		sess.UA = uaPtr
		if ipStr != nil {
			if addr, err := netip.ParseAddr(*ipStr); err == nil {
				sess.IP = &addr
			}
		}
		sess.VisitorFingerprint = &fp
		out = append(out, sess)
	}
	return out, rows.Err()
}
