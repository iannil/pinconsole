// Package storage:chat_messages 表方法(1u 拆自 queries.go)。
package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CreateChatMessage 1g:写入聊天消息。
func (s *Postgres) CreateChatMessage(ctx context.Context, tenantID, sessionID uuid.UUID, sender, content string) (*ChatMessage, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO chat_messages (tenant_id, session_id, sender, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, session_id, sender, content, created_at
	`, tenantID, sessionID, sender, content)
	var m ChatMessage
	if err := row.Scan(&m.ID, &m.TenantID, &m.SessionID, &m.Sender, &m.Content, &m.CreatedAt); err != nil {
		return nil, err
	}
	return &m, nil
}

// ListChatMessagesBySession 1g:列出某 session 的聊天消息(sinceID 之后,按 id 升序)。
func (s *Postgres) ListChatMessagesBySession(ctx context.Context, sessionID uuid.UUID, sinceID int64, limit int32) ([]ChatMessage, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, tenant_id, session_id, sender, content, created_at
		FROM chat_messages
		WHERE session_id = $1 AND id > $2
		ORDER BY id ASC
		LIMIT $3
	`, sessionID, sinceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ChatMessage{}
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.TenantID, &m.SessionID, &m.Sender, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListChatMessagesOlderThan 1l GC 扩展:列出超过保留期的 chat_messages。
func (s *Postgres) ListChatMessagesOlderThan(ctx context.Context, threshold time.Time, limit int32) ([]int64, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id FROM chat_messages WHERE created_at < $1 LIMIT $2
	`, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DeleteChatMessagesByID 1l GC:批量删除 chat_messages。
func (s *Postgres) DeleteChatMessagesByID(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `DELETE FROM chat_messages WHERE id = ANY($1)`, ids)
	return err
}
