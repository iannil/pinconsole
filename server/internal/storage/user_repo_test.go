// 1ai 测试:users 表 repo PG 集成测试。
//
// 补 1h auth.go login 依赖的 GetUserByEmail/GetUserByID/CreateUser/CountUsers 路径。
// 沿用 1ac erasure_test.go 既定模式:真 PG + helperPGPool(不可用 skip)。
//
// 数据隔离:每次用 uuid.New() 后缀的 email,defer cleanup。
package storage

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestCreateUser_AndRetrieve — CreateUser 返回的 user 应被
// GetUserByEmail 与 GetUserByID 双路径取回且字段一致。
func TestCreateUser_AndRetrieve(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	tenantID := DefaultTenantID
	email := "1ai-user-" + uuid.New().String()[:8] + "@example.com"
	hash := "$2a$10$dummyhashfor1aitestxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email)
	}()

	created, err := pg.CreateUser(ctx, tenantID, email, hash, "1ai-test", "admin")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created == nil || created.ID == uuid.Nil {
		t.Fatal("CreateUser returned nil or zero-ID user")
	}
	if created.Email != email {
		t.Errorf("created.Email = %q, want %q", created.Email, email)
	}
	if created.PasswordHash != hash {
		t.Errorf("created.PasswordHash mismatch")
	}
	if created.Role != "admin" {
		t.Errorf("created.Role = %q, want admin", created.Role)
	}

	byEmail, err := pg.GetUserByEmail(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if byEmail.ID != created.ID {
		t.Errorf("GetUserByEmail.ID = %s, want %s", byEmail.ID, created.ID)
	}

	byID, err := pg.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if byID.Email != email {
		t.Errorf("GetUserByID.Email = %q, want %q", byID.Email, email)
	}
	if byID.ID != created.ID {
		t.Errorf("GetUserByID.ID mismatch")
	}
}

// TestCreateUser_OnConflict_NoOp — 同 (tenant, email) 二次插入应 NO ACTION。
//
// PG 行为:`ON CONFLICT DO NOTHING` + `RETURNING` 在冲突时**不返回行**,
// pgx 的 QueryRow.Scan 返回 ErrNoRows。本测试验证此行为:
//  1. 二次 CreateUser 返回 error
//  2. 原行未被覆盖(GetUserByEmail 仍返回 first 的字段)
func TestCreateUser_OnConflict_NoOp(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	email := "1ai-conflict-" + uuid.New().String()[:8] + "@example.com"
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email)
	}()

	first, err := pg.CreateUser(ctx, DefaultTenantID, email, "hash1", "first", "operator")
	if err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}

	// 二次插入:ON CONFLICT DO NOTHING → 不返行 → ErrNoRows
	_, err = pg.CreateUser(ctx, DefaultTenantID, email, "hash2", "second", "admin")
	if err == nil {
		t.Fatal("second CreateUser: err = nil, want ErrNoRows (ON CONFLICT DO NOTHING)")
	}

	// 原行未被覆盖
	stored, err := pg.GetUserByEmail(ctx, DefaultTenantID, email)
	if err != nil {
		t.Fatalf("GetUserByEmail after conflict: %v", err)
	}
	if stored.ID != first.ID {
		t.Errorf("原行 ID 被覆盖:stored=%s, want first=%s", stored.ID, first.ID)
	}
	if stored.PasswordHash != "hash1" {
		t.Errorf("原 hash 被覆盖:got %q, want hash1", stored.PasswordHash)
	}
	if stored.DisplayName != "first" {
		t.Errorf("原 display_name 被覆盖:got %q, want first", stored.DisplayName)
	}
	if stored.Role != "operator" {
		t.Errorf("原 role 被覆盖:got %q, want operator", stored.Role)
	}
}

// TestGetUserByEmail_NotFound_ReturnsError — 不存在 email 必返 error。
func TestGetUserByEmail_NotFound_ReturnsError(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	missingEmail := "1ai-missing-" + uuid.New().String()[:8] + "@nowhere.test"
	_, err := pg.GetUserByEmail(ctx, DefaultTenantID, missingEmail)
	if err == nil {
		t.Error("GetUserByEmail on missing email: err = nil, want non-nil (pgx.ErrNoRows)")
	}
}

// TestGetUserByID_NotFound_ReturnsError — 不存在 ID 必返 error。
func TestGetUserByID_NotFound_ReturnsError(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	_, err := pg.GetUserByID(ctx, uuid.New())
	if err == nil {
		t.Error("GetUserByID on random UUID: err = nil, want non-nil")
	}
}

// TestCountUsers_ReturnsCount — 创建后计数应 ≥ 1。
func TestCountUsers_ReturnsCount(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	email := "1ai-count-" + uuid.New().String()[:8] + "@example.com"
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email)
	}()

	before, err := pg.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers before: %v", err)
	}

	if _, err := pg.CreateUser(ctx, DefaultTenantID, email, "hash", "1ai-count", "operator"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	after, err := pg.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers after: %v", err)
	}
	if after != before+1 {
		t.Errorf("CountUsers after creating one user: %d → %d, want %d → %d", before, after, before, before+1)
	}
}
