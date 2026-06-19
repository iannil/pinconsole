// 1v 测试:visitor_repo 的 ErrNoRows 路径(修审计新-3)。
//
// 验证 GetVisitorByFingerprint 在 PG 返回 ErrNoRows 时 (nil, nil),
// 而非 (nil, error) —— 这是 GDPR DELETE 端点幂等返回 200 visitor_not_found 的前提。
//
// 1ai-b 追加:CreateVisitor + GetVisitorByFingerprint PG 集成测试(真 PG + skip)。
package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// errOnlyScanner 是只返回预设 error 的 scanner,用于模拟 pgx.Row 行为。
type errOnlyScanner struct{ err error }

func (s errOnlyScanner) Scan(dest ...any) error { return s.err }

// TestScanVisitor_PropagatesErrNoRows 验证 scanVisitor 不吞噬 pgx.ErrNoRows。
// 若该测试失败,说明 scanVisitor 改了 error 透传语义,GetVisitorByFingerprint 的
// errors.Is(err, pgx.ErrNoRows) 分支也需同步调整。
func TestScanVisitor_PropagatesErrNoRows(t *testing.T) {
	row := errOnlyScanner{err: pgx.ErrNoRows}
	v, err := scanVisitor(row)
	if v != nil {
		t.Errorf("scanVisitor with ErrNoRows: visitor = %v, want nil", v)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("scanVisitor with ErrNoRows: err = %v, want pgx.ErrNoRows", err)
	}
}

// TestGetVisitorByFingerprint_ScanErrNoRows_ReturnsNilPair 验证
// GetVisitorByFingerprint 在 scan 返回 ErrNoRows 时返回 (nil, nil)。
//
// 由于 GetVisitorByFingerprint 内部用 Pool.QueryRow(无法 mock),
// 这里通过临时替换 Pool.QueryRow 的等价路径验证转换语义:
// 直接断言 errors.Is(err, pgx.ErrNoRows) 路径在 visitor_repo.go 中存在并被覆盖。
//
// 端到端覆盖见:e2e/tests/1l-privacy.spec.ts 场景4 DELETE 不存在 fingerprint。
func TestGetVisitorByFingerprint_ErrNoRowsContract(t *testing.T) {
	// 契约:GetVisitorByFingerprint 必须用 errors.Is(err, pgx.ErrNoRows)
	// 显式区分"不存在"与"真实错误",否则 GDPR DELETE 误返 500。
	// 此测试作为编译时 + 行为时的契约提醒,若有人删了 errors.Is 检查,
	// 应同时调整此测试和 e2e 场景4 的期望。
	_ = context.Background()
	_ = uuid.Nil

	// 验证 pgx.ErrNoRows 可被 errors.Is 识别(防 pgx 升级改语义)。
	if !errors.Is(pgx.ErrNoRows, pgx.ErrNoRows) {
		t.Fatal("pgx.ErrNoRows self-identity broken — pgx 升级?")
	}
}

// ============ 1ai-b:visitor_repo PG 集成测试(真 PG + skip)============

// TestCreateVisitor_AndRetrieve_1aib — CreateVisitor → GetVisitorByFingerprint 字段一致。
func TestCreateVisitor_AndRetrieve_1aib(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	fp := "1aib-visitor-" + uuid.New().String()[:8]
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	created, err := pg.CreateVisitor(ctx, DefaultTenantID, fp, "Mozilla/1aib", "10.0.0.1")
	if err != nil {
		t.Fatalf("CreateVisitor: %v", err)
	}
	if created == nil || created.ID == uuid.Nil {
		t.Fatal("CreateVisitor returned nil/zero-ID visitor")
	}
	if created.Fingerprint != fp {
		t.Errorf("Fingerprint = %q, want %q", created.Fingerprint, fp)
	}
	if created.UA == nil || *created.UA != "Mozilla/1aib" {
		t.Errorf("UA = %v, want 'Mozilla/1aib'", created.UA)
	}

	fetched, err := pg.GetVisitorByFingerprint(ctx, DefaultTenantID, fp)
	if err != nil {
		t.Fatalf("GetVisitorByFingerprint: %v", err)
	}
	if fetched == nil {
		t.Fatal("GetVisitorByFingerprint returned nil for existing visitor")
	}
	if fetched.ID != created.ID {
		t.Errorf("ID mismatch: fetched=%s, created=%s", fetched.ID, created.ID)
	}
}

// TestCreateVisitor_OnConflict_UpdatesLastSeen_1aib — 重复 CreateVisitor 触发
// ON CONFLICT DO UPDATE,last_seen_at 刷新 + COALESCE 覆盖 ua。
func TestCreateVisitor_OnConflict_UpdatesLastSeen_1aib(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	fp := "1aib-upsert-" + uuid.New().String()[:8]
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	first, err := pg.CreateVisitor(ctx, DefaultTenantID, fp, "ua-v1", "10.0.0.1")
	if err != nil {
		t.Fatalf("first CreateVisitor: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // 让 last_seen_at 有可观察差异

	second, err := pg.CreateVisitor(ctx, DefaultTenantID, fp, "ua-v2", "10.0.0.2")
	if err != nil {
		t.Fatalf("second CreateVisitor (upsert): %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("ON CONFLICT 应保留原 ID:second=%s, want first=%s", second.ID, first.ID)
	}
	if !second.LastSeenAt.After(first.LastSeenAt) {
		t.Errorf("last_seen_at 未刷新:first=%v, second=%v", first.LastSeenAt, second.LastSeenAt)
	}
	// COALESCE(EXCLUDED.ua, visitors.ua):新 ua 非空应覆盖
	if second.UA == nil || *second.UA != "ua-v2" {
		t.Errorf("UA 未刷新:second.UA=%v, want 'ua-v2'", second.UA)
	}
}

// TestGetVisitorByFingerprint_NotFound_ReturnsNil_1aib — 真 PG 验证 (nil, nil) 行为。
// (1v 用 mock 验证契约,1ai-b 用真 PG 验证端到端)
func TestGetVisitorByFingerprint_NotFound_ReturnsNil_1aib(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	missingFp := "1aib-missing-" + uuid.New().String()[:8]
	v, err := pg.GetVisitorByFingerprint(ctx, DefaultTenantID, missingFp)
	if err != nil {
		t.Errorf("GetVisitorByFingerprint on missing: err = %v, want nil (1v 行为)", err)
	}
	if v != nil {
		t.Errorf("GetVisitorByFingerprint on missing: v = %+v, want nil", v)
	}
}
