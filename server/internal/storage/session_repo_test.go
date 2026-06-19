// 1ai 测试:sessions 表 repo PG 集成测试。
//
// 补 1b/1d 依赖的 CreateSession/GetSession/TouchSessionEvent/EndSession/
// ListActiveSessionsByTenant 路径。沿用 1ac erasure_test.go 既定模式。
//
// 数据隔离:每次用 uuid.New() 后缀的 visitor fingerprint,
// defer cleanup(sessions FK CASCADE,删 visitor 自动清 sessions)。
package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedVisitor 在 visitors 表插入一行,返回 (visitorID, fingerprint)。
// 调用方应 defer cleanup:pool.Exec(DELETE FROM visitors WHERE fingerprint = $1, fp)。
func seedVisitor(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, string) {
	t.Helper()
	ctx := context.Background()
	fp := "1ai-session-" + uuid.New().String()[:8]
	visitorID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO visitors (id, tenant_id, fingerprint, ua, ip_first_seen, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, 'test-ua', '10.0.0.1', NOW(), NOW())
	`, visitorID, DefaultTenantID, fp); err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	return visitorID, fp
}

// TestCreateSession_AndRetrieve — CreateSession → GetSession 字段一致。
func TestCreateSession_AndRetrieve(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	visitorID, fp := seedVisitor(t, pool)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	created, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "Mozilla/5.0", "10.0.0.5")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if created == nil || created.ID == uuid.Nil {
		t.Fatal("CreateSession returned nil/zero-ID session")
	}
	if created.VisitorID != visitorID {
		t.Errorf("VisitorID = %s, want %s", created.VisitorID, visitorID)
	}
	if created.Status != "active" {
		t.Errorf("Status = %q, want active (default)", created.Status)
	}
	if created.UA == nil || *created.UA != "Mozilla/5.0" {
		t.Errorf("UA = %v, want 'Mozilla/5.0'", created.UA)
	}

	fetched, err := pg.GetSession(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("GetSession.ID = %s, want %s", fetched.ID, created.ID)
	}
	if fetched.VisitorID != visitorID {
		t.Errorf("GetSession.VisitorID mismatch")
	}
}

// TestCreateSession_NullUA_NullIP — 空字符串 ua/ip 必须存为 NULL,而非 ''。
//
// 防回归:CreateSession 用 `if ua != "" { uaArg = ua }` 模式,
// 误改成直接传 "" 会破坏 admin UI 显示。
func TestCreateSession_NullUA_NullIP(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	visitorID, fp := seedVisitor(t, pool)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	created, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "", "")
	if err != nil {
		t.Fatalf("CreateSession with empty ua/ip: %v", err)
	}
	if created.UA != nil {
		t.Errorf("UA = %v, want nil (empty → NULL)", created.UA)
	}
	if created.IP != nil {
		t.Errorf("IP = %v, want nil (empty → NULL)", created.IP)
	}
}

// TestTouchSessionEvent_IncrementsCount — Touch 后 event_count 增量正确。
func TestTouchSessionEvent_IncrementsCount(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	visitorID, fp := seedVisitor(t, pool)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	created, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if created.EventCount != 0 {
		t.Errorf("initial EventCount = %d, want 0", created.EventCount)
	}

	// Touch 5 + 10 + 3 = 18 个事件
	for _, delta := range []int32{5, 10, 3} {
		if err := pg.TouchSessionEvent(ctx, created.ID, delta); err != nil {
			t.Fatalf("TouchSessionEvent(%d): %v", delta, err)
		}
	}
	fetched, err := pg.GetSession(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetSession after touch: %v", err)
	}
	if fetched.EventCount != 18 {
		t.Errorf("EventCount = %d, want 18 (5+10+3)", fetched.EventCount)
	}
	if !fetched.LastEventAt.Valid {
		t.Error("LastEventAt should be set after TouchSessionEvent")
	}
}

// TestEndSession_SetsEndedAt — EndSession 后 ended_at 非 null + status 更新。
func TestEndSession_SetsEndedAt(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	visitorID, fp := seedVisitor(t, pool)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	created, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if created.EndedAt.Valid {
		t.Error("initial EndedAt should be NULL")
	}

	if err := pg.EndSession(ctx, created.ID, "ended"); err != nil {
		t.Fatalf("EndSession: %v", err)
	}
	fetched, err := pg.GetSession(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetSession after end: %v", err)
	}
	if !fetched.EndedAt.Valid {
		t.Error("EndedAt should be set after EndSession")
	}
	if fetched.Status != "ended" {
		t.Errorf("Status = %q, want ended", fetched.Status)
	}
}

// TestListActiveSessionsByTenant_FiltersByStatus — 只返 active 状态的会话。
func TestListActiveSessionsByTenant_FiltersByStatus(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	visitorID, fp := seedVisitor(t, pool)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	// 创建 1 active + 1 ended
	active, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "", "")
	if err != nil {
		t.Fatalf("CreateSession active: %v", err)
	}
	ended, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "", "")
	if err != nil {
		t.Fatalf("CreateSession ended: %v", err)
	}
	if err := pg.EndSession(ctx, ended.ID, "ended"); err != nil {
		t.Fatalf("EndSession: %v", err)
	}

	sessions, err := pg.ListActiveSessionsByTenant(ctx, DefaultTenantID, 100)
	if err != nil {
		t.Fatalf("ListActiveSessionsByTenant: %v", err)
	}
	foundActive := false
	for _, s := range sessions {
		if s.ID == active.ID {
			foundActive = true
		}
		if s.ID == ended.ID {
			t.Error("ended session 不应出现在 active 列表")
		}
		if s.Status != "active" {
			t.Errorf("session %s status = %q, want active", s.ID, s.Status)
		}
	}
	if !foundActive {
		t.Error("active session 未出现在列表(可能 filter 反向了)")
	}
}

// TestListEndedSessionsByTenant_FiltersByWindow — ended session 在窗口内被列出。
func TestListEndedSessionsByTenant_FiltersByWindow(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	visitorID, fp := seedVisitor(t, pool)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	ended, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := pg.EndSession(ctx, ended.ID, "ended"); err != nil {
		t.Fatalf("EndSession: %v", err)
	}

	// 窗口 24h,刚结束的必在内
	sessions, err := pg.ListEndedSessionsByTenant(ctx, DefaultTenantID, 24*time.Hour, 100)
	if err != nil {
		t.Fatalf("ListEndedSessionsByTenant: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s.ID == ended.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("刚 EndSession 的 session 应在 24h 窗口的 ended 列表内")
	}

	// 窗口 1ms(几乎必为空,除非测试超快)
	time.Sleep(2 * time.Millisecond)
	sessions, err = pg.ListEndedSessionsByTenant(ctx, DefaultTenantID, 1*time.Millisecond, 100)
	if err != nil {
		t.Fatalf("ListEndedSessionsByTenant(1ms): %v", err)
	}
	for _, s := range sessions {
		if s.ID == ended.ID {
			t.Error("session 不应在 1ms 窗口内(刚结束 = 已超过 1ms)")
			break
		}
	}
}
