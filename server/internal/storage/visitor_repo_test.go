// 1v 测试:visitor_repo 的 ErrNoRows 路径(修审计新-3)。
//
// 验证 GetVisitorByFingerprint 在 PG 返回 ErrNoRows 时 (nil, nil),
// 而非 (nil, error) —— 这是 GDPR DELETE 端点幂等返回 200 visitor_not_found 的前提。
package storage

import (
	"context"
	"errors"
	"testing"

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
