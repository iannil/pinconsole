// Package storage:co_browsing_commands 表方法(1u 拆自 queries.go)。
package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CreateCoBrowsingCommand 记录一条 co-browsing 命令(1e)。
func (s *Postgres) CreateCoBrowsingCommand(ctx context.Context, cmd CoBrowsingCommand) (*CoBrowsingCommand, error) {
	var nodeIDArg any
	if cmd.TargetNodeID != nil {
		nodeIDArg = *cmd.TargetNodeID
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO co_browsing_commands (
			tenant_id, session_id, operator_id,
			command_type, target_node_id, payload
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, session_id, operator_id,
		          command_type, target_node_id, payload, created_at
	`,
		cmd.TenantID, cmd.SessionID, cmd.OperatorID,
		cmd.CommandType, nodeIDArg, cmd.Payload,
	)
	var out CoBrowsingCommand
	var nodeIDPtr *int32
	if err := row.Scan(
		&out.ID, &out.TenantID, &out.SessionID, &out.OperatorID,
		&out.CommandType, &nodeIDPtr, &out.Payload, &out.CreatedAt,
	); err != nil {
		return nil, err
	}
	out.TargetNodeID = nodeIDPtr
	return &out, nil
}

// ListCoBrowsingCommandsBySession 列出某会话的全部 co-browsing 命令(按时间正序)。
func (s *Postgres) ListCoBrowsingCommandsBySession(ctx context.Context, sessionID uuid.UUID, limit int32) ([]CoBrowsingCommand, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, tenant_id, session_id, operator_id,
		       command_type, target_node_id, payload, created_at
		FROM co_browsing_commands
		WHERE session_id = $1
		ORDER BY created_at ASC
		LIMIT $2
	`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CoBrowsingCommand
	for rows.Next() {
		var c CoBrowsingCommand
		var nodeIDPtr *int32
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.SessionID, &c.OperatorID,
			&c.CommandType, &nodeIDPtr, &c.Payload, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		c.TargetNodeID = nodeIDPtr
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteCoBrowsingCommandsOlderThan 1l GC:删除超期 co_browsing_commands。
func (s *Postgres) DeleteCoBrowsingCommandsOlderThan(ctx context.Context, threshold time.Time) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM co_browsing_commands WHERE created_at < $1`, threshold)
	return err
}
