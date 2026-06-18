// Package storage:GDPR 被遗忘权级联删除(1l + 1u 拆自 queries.go)。
package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DeleteVisitorByFingerprint 1l:级联删除访客的所有数据(GDPR Art.17 被遗忘权)。
//
// 删除顺序(依赖关系反向):
//  1. visitor_consents(无依赖)
//  2. chat_messages(依赖 sessions)
//  3. co_browsing_commands(依赖 sessions)
//  4. event_blobs(依赖 sessions) — 仅 PG 行;MinIO 对象由调用方删除
//  5. sessions(依赖 visitors)
//  6. visitors
//
// 返回:删除的 session IDs(供调用方定位 MinIO/Redis 清理)。
//
// 注意:不在事务里,因 PG 事务有大小限制;每步独立提交。
// 失败时已删的数据不回滚(GDPR 偏向"多删而非少删")。
func (s *Postgres) DeleteVisitorByFingerprint(ctx context.Context, tenantID uuid.UUID, fingerprint string) (deletedSessionIDs []uuid.UUID, err error) {
	// 1. 找到 visitor ID + 关联 sessions
	var visitorID uuid.UUID
	err = s.Pool.QueryRow(ctx, `
		SELECT id FROM visitors WHERE tenant_id = $1 AND fingerprint = $2
	`, tenantID, fingerprint).Scan(&visitorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // visitor 不存在,无操作
		}
		return nil, fmt.Errorf("lookup visitor: %w", err)
	}

	// 收集 session IDs
	rows, err := s.Pool.Query(ctx, `
		SELECT id FROM sessions WHERE visitor_id = $1
	`, visitorID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sid uuid.UUID
		if err := rows.Scan(&sid); err != nil {
			return nil, fmt.Errorf("scan session id: %w", err)
		}
		deletedSessionIDs = append(deletedSessionIDs, sid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}

	// 2. visitor_consents
	if _, err := s.Pool.Exec(ctx, `
		DELETE FROM visitor_consents WHERE tenant_id = $1 AND fingerprint = $2
	`, tenantID, fingerprint); err != nil {
		return deletedSessionIDs, fmt.Errorf("delete consents: %w", err)
	}

	// 3-4. 按 session 删除 chat_messages / co_browsing_commands / event_blobs
	if len(deletedSessionIDs) > 0 {
		if _, err := s.Pool.Exec(ctx, `
			DELETE FROM chat_messages WHERE session_id = ANY($1)
		`, deletedSessionIDs); err != nil {
			return deletedSessionIDs, fmt.Errorf("delete chat_messages: %w", err)
		}
		if _, err := s.Pool.Exec(ctx, `
			DELETE FROM co_browsing_commands WHERE session_id = ANY($1)
		`, deletedSessionIDs); err != nil {
			return deletedSessionIDs, fmt.Errorf("delete co_browsing_commands: %w", err)
		}
		if _, err := s.Pool.Exec(ctx, `
			DELETE FROM event_blobs WHERE session_id = ANY($1)
		`, deletedSessionIDs); err != nil {
			return deletedSessionIDs, fmt.Errorf("delete event_blobs: %w", err)
		}

		// 5. sessions
		if _, err := s.Pool.Exec(ctx, `
			DELETE FROM sessions WHERE visitor_id = $1
		`, visitorID); err != nil {
			return deletedSessionIDs, fmt.Errorf("delete sessions: %w", err)
		}
	}

	// 6. visitors
	if _, err := s.Pool.Exec(ctx, `
		DELETE FROM visitors WHERE id = $1
	`, visitorID); err != nil {
		return deletedSessionIDs, fmt.Errorf("delete visitor: %w", err)
	}

	return deletedSessionIDs, nil
}

// ListEventBlobKeysBySessions 列出指定 sessions 的 MinIO object keys。
// 用于 erasure 时调用方批量删 MinIO 对象。
func (s *Postgres) ListEventBlobKeysBySessions(ctx context.Context, sessionIDs []uuid.UUID) ([]string, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT minio_object_key FROM event_blobs WHERE session_id = ANY($1)
	`, sessionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
