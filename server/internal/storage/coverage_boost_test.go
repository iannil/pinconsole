// CV-1 切片补测:storage 包覆盖率从 86.5% → ≥90%。
//
// 策略:
// 1. mockScanner 触发 scanSession/scanVisitor 的 invalid IP 分支(netip.ParseAddr err)
// 2. errPool 注入 Query/QueryRow/Exec error 触发各 repo 函数 error 分支
// 3. 关闭的 MinIO client 触发 PutBytes/GetBytes error
// 4. 直接调 Close() 提升 minio.Close 覆盖率(no-op 函数仅 0% 因未被调)
package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// === mockScanner: 触发 scanSession/scanVisitor 的 invalid IP 分支 ===

type mockScanner struct {
	values []any
	err    error
}

func (m mockScanner) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}
	for i, v := range m.values {
		if i >= len(dest) {
			break
		}
		switch d := dest[i].(type) {
		case *uuid.UUID:
			*d = v.(uuid.UUID)
		case *string:
			*d = v.(string)
		case **string:
			if vs, ok := v.(string); ok {
				*d = &vs
			}
		case *time.Time:
			*d = v.(time.Time)
		case *[]byte:
			*d = v.([]byte)
		case *int32:
			*d = v.(int32)
		case *int64:
			*d = v.(int64)
		case *bool:
			*d = v.(bool)
		}
	}
	return nil
}

// TestScanVisitor_InvalidIPBranch 验证 scanVisitor 的 netip.ParseAddr 失败分支。
func TestScanVisitor_InvalidIPBranch(t *testing.T) {
	badIP := "not-an-ip"
	now := time.Now()
	v, err := scanVisitor(mockScanner{values: []any{
		uuid.New(), uuid.Nil, "fp", (*string)(nil), &badIP,
		now, now, []byte(nil),
	}})
	if err != nil {
		t.Fatalf("scanVisitor with invalid IP: %v", err)
	}
	if v == nil {
		t.Fatal("scanVisitor returned nil")
	}
	if v.IPFirstSeen != nil {
		t.Errorf("IPFirstSeen should be nil for invalid IP, got %v", v.IPFirstSeen)
	}
}

// TestScanSession_InvalidIPBranch 验证 scanSession 的 netip.ParseAddr 失败分支。
func TestScanSession_InvalidIPBranch(t *testing.T) {
	badIP := "not-an-ip-either"
	now := time.Now()
	s, err := scanSession(mockScanner{values: []any{
		uuid.New(), uuid.Nil, uuid.New(), now, pgtypeTimestamptz(now),
		pgtypeTimestamptz(now), "active", int32(0), (*string)(nil), &badIP,
	}})
	if err != nil {
		t.Fatalf("scanSession with invalid IP: %v", err)
	}
	if s == nil {
		t.Fatal("scanSession returned nil")
	}
	if s.IP != nil {
		t.Errorf("IP should be nil for invalid IP, got %v", s.IP)
	}
}

// pgtypeTimestamptz 是测试 helper 返回 pgtype.Timestamptz(避免引入 pgtype 包污染其他文件)。
// 实际值由 scan 直接写入 dest 字段。
func pgtypeTimestamptz(t time.Time) time.Time {
	return t
}

// === errPool: 注入 error 的 PgxPool mock ===

// errPool 实现 PgxPool 接口,所有方法返回预设 error。
// 用于覆盖 repo 函数的 error 分支(Query/QueryRow/Exec 失败)。
type errPool struct {
	queryErr error
	execErr  error
}

func (e errPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, e.execErr
}
func (e errPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return errRows{err: e.queryErr}, e.queryErr
}
func (e errPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return errRow{err: e.queryErr}
}
func (e errPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, e.execErr
}
func (e errPool) Ping(ctx context.Context) error { return e.execErr }
func (e errPool) Close()                         {}

// errRow 实现 pgx.Row,Scan 返回预设 error。
type errRow struct {
	err error
}

func (r errRow) Scan(dest ...any) error { return r.err }

// errRows 实现 pgx.Rows,Next() 返回 false,Err() 返回预设 error。
type errRows struct {
	err error
}

func (r errRows) Next() bool                   { return false }
func (r errRows) Scan(dest ...any) error       { return r.err }
func (r errRows) Err() error                   { return r.err }
func (r errRows) Close()                       {}
func (r errRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r errRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (r errRows) Values() ([]any, error) { return nil, r.err }
func (r errRows) RawValues() [][]byte    { return nil }
func (r errRows) Conn() *pgx.Conn        { return nil }

// sentinelErr 用于断言 error 透传
var sentinelErr = errors.New("coverage-boost-sentinel")

// TestRepoErrorPaths 各 repo 函数 error 分支用 errPool 触发。
func TestRepoErrorPaths(t *testing.T) {
	ctx := context.Background()
	pg := &Postgres{Pool: errPool{queryErr: sentinelErr, execErr: sentinelErr}}

	// chat_repo
	if _, err := pg.CreateChatMessage(ctx, uuid.Nil, uuid.Nil, "", ""); err == nil {
		t.Error("CreateChatMessage: expected error from errPool")
	}
	if _, err := pg.ListChatMessagesBySession(ctx, uuid.Nil, 0, 1); err == nil {
		t.Error("ListChatMessagesBySession: expected error")
	}
	if _, err := pg.ListChatMessagesOlderThan(ctx, time.Now(), 1); err == nil {
		t.Error("ListChatMessagesOlderThan: expected error")
	}

	// command_repo
	if _, err := pg.CreateCoBrowsingCommand(ctx, CoBrowsingCommand{}); err == nil {
		t.Error("CreateCoBrowsingCommand: expected error")
	}
	if _, err := pg.ListCoBrowsingCommandsBySession(ctx, uuid.Nil, 1); err == nil {
		t.Error("ListCoBrowsingCommandsBySession: expected error")
	}

	// consent_repo
	if _, _, err := pg.GetLatestConsent(ctx, uuid.Nil, "", "", ""); err == nil {
		t.Error("GetLatestConsent: expected error")
	}
	if _, err := pg.UpsertConsent(ctx, uuid.Nil, "", "", "", false); err == nil {
		t.Error("UpsertConsent: expected error")
	}

	// event_blob_repo
	if _, err := pg.CreateEventBlob(ctx, EventBlob{}); err == nil {
		t.Error("CreateEventBlob: expected error")
	}
	if _, err := pg.ListEventBlobsBySession(ctx, uuid.Nil); err == nil {
		t.Error("ListEventBlobsBySession: expected error")
	}
	if _, err := pg.ListEventBlobsOlderThan(ctx, time.Now(), 1); err == nil {
		t.Error("ListEventBlobsOlderThan: expected error")
	}

	// session_repo
	if _, err := pg.CreateSession(ctx, uuid.Nil, uuid.Nil, "", ""); err == nil {
		t.Error("CreateSession: expected error")
	}
	if _, err := pg.GetSession(ctx, uuid.Nil); err == nil {
		t.Error("GetSession: expected error")
	}
	if err := pg.TouchSessionEvent(ctx, uuid.Nil, 0); err == nil {
		t.Error("TouchSessionEvent: expected error")
	}
	if err := pg.EndSession(ctx, uuid.Nil, ""); err == nil {
		t.Error("EndSession: expected error")
	}
	if _, err := pg.ListActiveSessionsByTenant(ctx, uuid.Nil, 1); err == nil {
		t.Error("ListActiveSessionsByTenant: expected error")
	}
	if _, err := pg.ListEndedSessionsByTenant(ctx, uuid.Nil, time.Second, 1); err == nil {
		t.Error("ListEndedSessionsByTenant: expected error")
	}

	// visitor_repo
	if _, err := pg.CreateVisitor(ctx, uuid.Nil, "", "", ""); err == nil {
		t.Error("CreateVisitor: expected error")
	}
	// GetVisitorByFingerprint 错误透传分支(非 ErrNoRows 的 error)
	// ErrNoRows 时返回 (nil, nil),其他 error 时返回 (nil, err)
	if _, err := pg.GetVisitorByFingerprint(ctx, uuid.Nil, ""); err == nil {
		t.Error("GetVisitorByFingerprint: expected error")
	}

	// erasure_repo
	if _, err := pg.DeleteVisitorByFingerprint(ctx, uuid.Nil, "fp-not-found-for-errpath"); err == nil {
		t.Error("DeleteVisitorByFingerprint: expected error")
	}
	if _, err := pg.ListEventBlobKeysBySessions(ctx, []uuid.UUID{uuid.New()}); err == nil {
		t.Error("ListEventBlobKeysBySessions: expected error")
	}

	// gc_repo
	if err := pg.DeleteSessionsEndedBefore(ctx, time.Now()); err == nil {
		t.Error("DeleteSessionsEndedBefore: expected error")
	}
	if err := pg.DeleteVisitorsLastSeenBefore(ctx, time.Now()); err == nil {
		t.Error("DeleteVisitorsLastSeenBefore: expected error")
	}
}

// TestErrRowsScan_Path 直接调用 errRows.Scan/Values 触发其 error 分支。
func TestErrRowsScan_Path(t *testing.T) {
	r := errRows{err: sentinelErr}
	if err := r.Scan(); err != sentinelErr {
		t.Errorf("errRows.Scan: got %v, want sentinelErr", err)
	}
	if err := r.Err(); err != sentinelErr {
		t.Errorf("errRows.Err: got %v, want sentinelErr", err)
	}
	if _, err := r.Values(); err != sentinelErr {
		t.Errorf("errRows.Values: got %v, want sentinelErr", err)
	}
	if err := r.Scan(); err != sentinelErr {
		t.Errorf("errRows.Scan second: %v", err)
	}
	r.Close() // no-op, just for coverage
	_ = r.CommandTag()
	_ = r.FieldDescriptions()
	_ = r.RawValues()
	_ = r.Conn()
}

// TestMinIO_Close_Coverage 直接调 Close 触发 no-op 函数覆盖。
// minio.Close 是空函数,go tool cover 仅在函数被调用时统计。
func TestMinIO_Close_Coverage(t *testing.T) {
	m := &MinIO{}
	m.Close() // no-op,提升 Close() 覆盖率从 0% → 100%
}

// TestMinIO_PutBytes_ErrorPath 用 nil Client 触发 PutBytes panic-free error path。
// PutObject 在 nil Client 上会 panic,改用关闭的 bucket 名触发 error。
func TestMinIO_PutBytes_ErrorPath(t *testing.T) {
	mio := helperMinIO(t)
	defer mio.Close()

	// 用取消的 ctx 触发 PutObject error
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	err := mio.PutBytes(ctx, "any-key", []byte("data"))
	if err == nil {
		t.Error("PutBytes with canceled ctx: expected error, got nil")
	}
}

// TestMinIO_GetBytes_ErrorPath 用取消的 ctx 触发 GetBytes error。
func TestMinIO_GetBytes_ErrorPath(t *testing.T) {
	mio := helperMinIO(t)
	defer mio.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mio.GetBytes(ctx, "any-key")
	if err == nil {
		t.Error("GetBytes with canceled ctx: expected error, got nil")
	}
}

// TestConnectPostgres_BadDSN 覆盖 ConnectPostgres 的 ParseConfig error 分支。
func TestConnectPostgres_BadDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// helperPostgresCfg + 改 Host 为无效字符让 DSN parse 失败
	bad := helperPostgresCfg()
	bad.Host = ""
	bad.Port = ""
	bad.User = ""
	bad.Database = ""
	// DSN() 返回 "postgres://:@:/?sslmode=disable" 等,parse 应失败
	_, err := ConnectPostgres(ctx, bad, discardLogger())
	if err == nil {
		t.Error("ConnectPostgres with bad DSN: expected error, got nil")
	}
}

// TestConnectPostgres_MaxConnsZero 覆盖 cfg.MaxConns == 0 跳过 MaxConns 设置分支。
func TestConnectPostgres_MaxConnsZero(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := helperPostgresCfg()
	cfg.MaxConns = 0 // 走 if cfg.MaxConns > 0 = false 分支

	pg, err := ConnectPostgres(ctx, cfg, discardLogger())
	if err != nil {
		t.Fatalf("ConnectPostgres(MaxConns=0): %v", err)
	}
	defer pg.Close()
}
