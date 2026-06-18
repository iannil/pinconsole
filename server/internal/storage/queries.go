// Package storage：v1 切片 1b 的 queries 实现（手写 pgx）。
//
// 设计偏差：规格锁定 "pgx + sqlc"，但 sqlc 安装因网络问题失败。
// 此文件手写等价查询；当 sqlc 可用时，可删除此文件改用 sqlc 生成的代码。
// 详见 docs/progress/2026-06-17-slice-1b-implementation.md §与规格的偏差。
package storage

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Visitor 对应 visitors 表。
type Visitor struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Fingerprint  string
	UA           *string
	IPFirstSeen  *netip.Addr
	FirstSeenAt  time.Time
	LastSeenAt   time.Time
	Meta         []byte
}

// Session 对应 sessions 表。
type Session struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	VisitorID    uuid.UUID
	StartedAt    time.Time
	LastEventAt  pgtype.Timestamptz // 可空
	EndedAt      pgtype.Timestamptz // 可空
	Status       string
	EventCount   int32
	UA           *string
	IP           *netip.Addr

	// JOIN 字段（仅 ListActiveSessionsByTenant 填充）
	VisitorFingerprint *string
}

// EventBlob 对应 event_blobs 表。
type EventBlob struct {
	ID              uuid.UUID
	SessionID       uuid.UUID
	TenantID        uuid.UUID
	BlobIndex       int32
	StartedAt       time.Time
	EndedAt         time.Time
	EventCount      int32
	MinIOObjectKey  string
	SizeBytes       int64
	ChecksumSHA256  string
	CreatedAt       time.Time
}

// User 对应 users 表（1h）。
type User struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Email        string
	PasswordHash string
	DisplayName  string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ChatMessage 对应 chat_messages 表（1g）。
type ChatMessage struct {
	ID        int64
	TenantID  uuid.UUID
	SessionID uuid.UUID
	Sender    string // operator / visitor
	Content   string
	CreatedAt time.Time
}

// CoBrowsingCommand 对应 co_browsing_commands 表（1e）。
type CoBrowsingCommand struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	SessionID    uuid.UUID
	OperatorID   string
	CommandType  string
	TargetNodeID *int32
	Payload      []byte
	CreatedAt    time.Time
}

// DefaultTenantID 是 v1 占位 tenant_id（全 0）。
// 多租户未激活，所有记录均归属此 tenant。
var DefaultTenantID = uuid.Nil

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

// ListActiveSessionsByTenant 列出租户下所有活跃会话（含 visitor fingerprint）。
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

// CreateEventBlob 记录一个 MinIO blob。
func (s *Postgres) CreateEventBlob(ctx context.Context, b EventBlob) (*EventBlob, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO event_blobs (
			session_id, tenant_id, blob_index,
			started_at, ended_at, event_count,
			minio_object_key, size_bytes, checksum_sha256
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, session_id, tenant_id, blob_index,
		          started_at, ended_at, event_count,
		          minio_object_key, size_bytes, checksum_sha256, created_at
	`,
		b.SessionID, b.TenantID, b.BlobIndex,
		b.StartedAt, b.EndedAt, b.EventCount,
		b.MinIOObjectKey, b.SizeBytes, b.ChecksumSHA256,
	)
	var out EventBlob
	err := row.Scan(
		&out.ID, &out.SessionID, &out.TenantID, &out.BlobIndex,
		&out.StartedAt, &out.EndedAt, &out.EventCount,
		&out.MinIOObjectKey, &out.SizeBytes, &out.ChecksumSHA256, &out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEventBlobsBySession 列出某会话的全部 blob。
func (s *Postgres) ListEventBlobsBySession(ctx context.Context, sessionID uuid.UUID) ([]EventBlob, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, session_id, tenant_id, blob_index,
		       started_at, ended_at, event_count,
		       minio_object_key, size_bytes, checksum_sha256, created_at
		FROM event_blobs
		WHERE session_id = $1
		ORDER BY blob_index ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EventBlob
	for rows.Next() {
		var b EventBlob
		if err := rows.Scan(
			&b.ID, &b.SessionID, &b.TenantID, &b.BlobIndex,
			&b.StartedAt, &b.EndedAt, &b.EventCount,
			&b.MinIOObjectKey, &b.SizeBytes, &b.ChecksumSHA256, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ListEndedSessionsByTenant 列出租户下指定时间窗口内已结束的会话。
// since 参数：24h / 7d / 30d 等（Go duration 字符串）。
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

// ListEventBlobsOlderThan 列出 created_at 早于 threshold 的 blob（GC 用）。
func (s *Postgres) ListEventBlobsOlderThan(ctx context.Context, threshold time.Time, limit int32) ([]EventBlob, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, session_id, tenant_id, blob_index,
		       started_at, ended_at, event_count,
		       minio_object_key, size_bytes, checksum_sha256, created_at
		FROM event_blobs
		WHERE created_at < $1
		ORDER BY created_at ASC
		LIMIT $2
	`, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EventBlob
	for rows.Next() {
		var b EventBlob
		if err := rows.Scan(
			&b.ID, &b.SessionID, &b.TenantID, &b.BlobIndex,
			&b.StartedAt, &b.EndedAt, &b.EventCount,
			&b.MinIOObjectKey, &b.SizeBytes, &b.ChecksumSHA256, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// DeleteEventBlobByID 按 ID 删除 blob（GC 用，配合 MinIO 删除）。
func (s *Postgres) DeleteEventBlobByID(ctx context.Context, id uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM event_blobs WHERE id = $1`, id)
	return err
}

// scanner 兼容 *pgx.Row 与 *pgx.Rows 的 Scan 接口。
type scanner interface {
	Scan(dest ...any) error
}

func scanVisitor(row scanner) (*Visitor, error) {
	var v Visitor
	var ua *string
	var ipStr *string
	if err := row.Scan(
		&v.ID, &v.TenantID, &v.Fingerprint, &ua, &ipStr,
		&v.FirstSeenAt, &v.LastSeenAt, &v.Meta,
	); err != nil {
		return nil, err
	}
	v.UA = ua
	if ipStr != nil {
		if addr, err := netip.ParseAddr(*ipStr); err == nil {
			v.IPFirstSeen = &addr
		}
	}
	return &v, nil
}

func scanSession(row scanner) (*Session, error) {
	var s Session
	var ua *string
	var ipStr *string
	if err := row.Scan(
		&s.ID, &s.TenantID, &s.VisitorID, &s.StartedAt, &s.LastEventAt,
		&s.EndedAt, &s.Status, &s.EventCount, &ua, &ipStr,
	); err != nil {
		return nil, err
	}
	s.UA = ua
	if ipStr != nil {
		if addr, err := netip.ParseAddr(*ipStr); err == nil {
			s.IP = &addr
		}
	}
	return &s, nil
}

// CreateCoBrowsingCommand 记录一条 co-browsing 命令（1e）。
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

// ListCoBrowsingCommandsBySession 列出某会话的全部 co-browsing 命令（按时间正序）。
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

// CreateChatMessage 1g：写入聊天消息。
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

// ListChatMessagesBySession 1g：列出某 session 的聊天消息（sinceID 之后，按 id 升序）。
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

// GetUserByEmail 1h：按 email 查用户。
func (s *Postgres) GetUserByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*User, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, display_name, role, created_at, updated_at
		FROM users WHERE tenant_id = $1 AND email = $2
	`, tenantID, email)
	return scanUser(row)
}

// GetUserByID 1h：按 ID 查用户。
func (s *Postgres) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, display_name, role, created_at, updated_at
		FROM users WHERE id = $1
	`, id)
	return scanUser(row)
}

// CreateUser 1h：创建用户。
func (s *Postgres) CreateUser(ctx context.Context, tenantID uuid.UUID, email, passwordHash, displayName, role string) (*User, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, email) DO NOTHING
		RETURNING id, tenant_id, email, password_hash, display_name, role, created_at, updated_at
	`, tenantID, email, passwordHash, displayName, role)
	return scanUser(row)
}

// CountUsers 1h：统计用户数。
func (s *Postgres) CountUsers(ctx context.Context) (int64, error) {
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func scanUser(row scanner) (*User, error) {
	var u User
	if err := row.Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash,
		&u.DisplayName, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

// ===== 1l-privacy-gdpr =====

// VisitorConsent 对应 visitor_consents 表。
type VisitorConsent struct {
	ID          int64
	TenantID    uuid.UUID
	Fingerprint string
	Scope       string
	Version     string
	Accepted    bool
	ConsentedAt time.Time
	ExpiresAt   pgtype.Timestamptz // 可空
}

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

// DeleteCoBrowsingCommandsOlderThan 1l GC:删除超期 co_browsing_commands。
func (s *Postgres) DeleteCoBrowsingCommandsOlderThan(ctx context.Context, threshold time.Time) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM co_browsing_commands WHERE created_at < $1`, threshold)
	return err
}

// DeleteSessionsEndedBefore 1l GC:删除已结束且超过保留期的 sessions。
// 必须先删 event_blobs / chat_messages / co_browsing_commands(否则 FK 阻塞)。
func (s *Postgres) DeleteSessionsEndedBefore(ctx context.Context, threshold time.Time) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE ended_at IS NOT NULL AND ended_at < $1`, threshold)
	return err
}

// DeleteVisitorsLastSeenBefore 1l GC:删除超过保留期未活动的 visitors(孤立 visitor)。
// 必须在 sessions 已清后调用。
func (s *Postgres) DeleteVisitorsLastSeenBefore(ctx context.Context, threshold time.Time) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM visitors WHERE last_seen_at < $1`, threshold)
	return err
}
