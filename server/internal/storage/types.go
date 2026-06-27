// Package storage:类型定义 + scanner helpers(1u 拆自 queries.go)。
package storage

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Visitor 对应 visitors 表。
type Visitor struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Fingerprint string
	UA          *string
	IPFirstSeen *netip.Addr
	FirstSeenAt time.Time
	LastSeenAt  time.Time
	Meta        []byte
}

// Session 对应 sessions 表。
type Session struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	VisitorID   uuid.UUID
	StartedAt   time.Time
	LastEventAt pgtype.Timestamptz // 可空
	EndedAt     pgtype.Timestamptz // 可空
	Status      string
	EventCount  int32
	UA          *string
	IP          *netip.Addr

	// JOIN 字段(仅 ListActiveSessionsByTenant 填充)
	VisitorFingerprint *string
}

// EventBlob 对应 event_blobs 表。
type EventBlob struct {
	ID             uuid.UUID
	SessionID      uuid.UUID
	TenantID       uuid.UUID
	BlobIndex      int32
	StartedAt      time.Time
	EndedAt        time.Time
	EventCount     int32
	MinIOObjectKey string
	SizeBytes      int64
	ChecksumSHA256 string
	CreatedAt      time.Time
}

// User 对应 users 表(1h)。
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

// ChatMessage 对应 chat_messages 表(1g)。
type ChatMessage struct {
	ID        int64
	TenantID  uuid.UUID
	SessionID uuid.UUID
	Sender    string // operator / visitor
	Content   string
	CreatedAt time.Time
}

// CoBrowsingCommand 对应 co_browsing_commands 表(1e)。
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

// VisitorConsent 对应 visitor_consents 表(1l)。
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

// DefaultTenantID 是 v1 占位 tenant_id(全 0)。
// 多租户未激活,所有记录均归属此 tenant。
var DefaultTenantID = uuid.Nil

// CustomDomain 是自定义域名的 DB 模型（cd-1）。
type CustomDomain struct {
	ID         int64     `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Domain     string    `json:"domain"`
	CertStatus string    `json:"cert_status"`  // pending / active / failed
	CertError  string    `json:"cert_error"`   // 失败原因（成功时 ""）
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WidgetConfig 是 widget 配置的 DB 模型（page-editor pe-1）。
type WidgetConfig struct {
	ID         int64     `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	WidgetType string    `json:"widget_type"`
	Config     []byte    `json:"config"` // JSONB
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
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

// Page 是落地页的 DB 模型（page-editor pe-1）。
type Page struct {
	ID        int64     `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Schema    []byte    `json:"schema"` // JSONB
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PageLead 是落地页表单提交的 DB 模型（page-editor pe-1）。
type PageLead struct {
	ID        int64     `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	PageSlug  string    `json:"page_slug"`
	Fields    []byte    `json:"fields"` // JSONB
	CreatedAt time.Time `json:"created_at"`
}
