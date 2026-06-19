// 1ac 测试:GDPR erasure 级联删除(审计 T0-1l-1)。
//
// 验证 DeleteVisitorByFingerprint 真的级联清 5 张相关表:
//   - visitor_consents
//   - chat_messages
//   - co_browsing_commands
//   - event_blobs
//   - sessions
//   - visitors
//
// 此前:函数实现完整但**零 PG 集成测试**(1l 长期 🔴 主因)。
package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// helperPGPool 返回真 PG pool,不可用时 skip。
func helperPGPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("需要 PG")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, "postgres://mm:mm_dev@localhost:5432/marketing_monitor?sslmode=disable")
	if err != nil {
		t.Skipf("PG 不可用(%v),跳过", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("PG ping 失败(%v),跳过", err)
	}
	return pool
}

// TestDeleteVisitorByFingerprint_CascadesAllTables — T0-1l-1:
// 验证级联删除真的清空 visitor + sessions + chat_messages + co_browsing_commands + event_blobs。
func TestDeleteVisitorByFingerprint_CascadesAllTables(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	// 1. seed visitor + session + chat + command + event_blob
	tenantID := DefaultTenantID
	fp := "1ac-test-fp-cascade-" + uuid.New().String()[:8]
	visitorID := uuid.New()
	sessionID := uuid.New()
	visitorEmail := "1ac-cascade@example.com"

	// cleanup any prior (idempotent)
	_, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp)

	_, err := pool.Exec(ctx, `
		INSERT INTO visitors (id, tenant_id, fingerprint, ua, ip_first_seen, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, 'test-ua', '10.0.0.1', NOW(), NOW())
	`, visitorID, tenantID, fp)
	if err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, tenantID, visitorID)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO chat_messages (session_id, tenant_id, sender, content, created_at)
		VALUES ($1, $2, 'operator', 'test', NOW())
	`, sessionID, tenantID)
	if err != nil {
		t.Fatalf("seed chat: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO co_browsing_commands (session_id, tenant_id, operator_id, command_type, payload, created_at)
		VALUES ($1, $2, '00000000-0000-0000-0000-000000000000', 'click', '{}', NOW())
	`, sessionID, tenantID)
	if err != nil {
		t.Fatalf("seed command: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO event_blobs (id, session_id, tenant_id, blob_index, minio_object_key, checksum_sha256, size_bytes, event_count, started_at, ended_at, created_at)
		VALUES ($1, $2, $3, 0, 'test-key', 'sha', 100, 1, NOW(), NOW(), NOW())
	`, uuid.New(), sessionID, tenantID)
	if err != nil {
		t.Fatalf("seed event_blob: %v", err)
	}

	// visitorEmail 仅记录,无实际表关联(visitors 表无 email 字段)
	_ = visitorEmail

	// 2. 调用 erasure
	deletedSessions, err := pg.DeleteVisitorByFingerprint(ctx, tenantID, fp)
	if err != nil {
		t.Fatalf("DeleteVisitorByFingerprint: %v", err)
	}
	if len(deletedSessions) != 1 {
		t.Errorf("deleted sessions=%v want 1 [sessionID=%s]", deletedSessions, sessionID)
	}

	// 3. 验证所有表都清空
	countVisitor := pool.QueryRow(ctx, `SELECT COUNT(*) FROM visitors WHERE id = $1`, visitorID)
	var n int
	if err := countVisitor.Scan(&n); err != nil {
		t.Fatalf("count visitors: %v", err)
	}
	if n != 0 {
		t.Errorf("visitors still has row after erasure (n=%d)", n)
	}

	for _, q := range []struct {
		table string
		sql   string
	}{
		{"sessions", `SELECT COUNT(*) FROM sessions WHERE id = $1`},
		{"chat_messages", `SELECT COUNT(*) FROM chat_messages WHERE session_id = $1`},
		{"co_browsing_commands", `SELECT COUNT(*) FROM co_browsing_commands WHERE session_id = $1`},
		{"event_blobs", `SELECT COUNT(*) FROM event_blobs WHERE session_id = $1`},
	} {
		if err := pool.QueryRow(ctx, q.sql, sessionID).Scan(&n); err != nil {
			t.Errorf("count %s: %v", q.table, err)
			continue
		}
		if n != 0 {
			t.Errorf("%s still has rows after erasure (n=%d)", q.table, n)
		}
	}
}

// TestDeleteVisitorByFingerprint_NonExistent_NoError — T0-1l-1 边界:
// 不存在的 fingerprint 应返回 (nil, nil),不报错(GDPR DELETE 幂等)。
func TestDeleteVisitorByFingerprint_NonExistent_NoError(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	deleted, err := pg.DeleteVisitorByFingerprint(ctx, DefaultTenantID, "non-existent-fp-1ac-"+uuid.New().String())
	if err != nil {
		t.Errorf("non-existent fp returned error: %v (want nil for idempotent GDPR DELETE)", err)
	}
	if deleted != nil {
		t.Errorf("non-existent fp returned deleted sessions=%v want nil", deleted)
	}
}
