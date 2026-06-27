// cd-1 测试:custom_domains repo PG 集成测试(需 PG)。
package storage

import (
	"context"
	"testing"
)

// TestCustomDomainRepo_CRUD 验证 Create/Get/List/Update/Delete round-trip(需 PG)。
func TestCustomDomainRepo_CRUD(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()

	// 确保表存在
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS custom_domains (
			id          BIGSERIAL PRIMARY KEY,
			tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
			domain      VARCHAR(255) NOT NULL,
			cert_status VARCHAR(16) NOT NULL DEFAULT 'pending',
			cert_error  TEXT NOT NULL DEFAULT '',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(tenant_id, domain)
		)
	`)
	if err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}

	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID

	// 清除旧测试数据
	_, err = pool.Exec(ctx, `DELETE FROM custom_domains WHERE tenant_id = $1`, tenantID)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	// 初始应为空
	all, err := pg.ListCustomDomains(ctx, tenantID)
	if err != nil {
		t.Fatalf("ListCustomDomains (empty): %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("ListCustomDomains: len=%d, want 0", len(all))
	}

	// Create
	created, err := pg.CreateCustomDomain(ctx, tenantID, "example.com")
	if err != nil {
		t.Fatalf("CreateCustomDomain: %v", err)
	}
	if created.Domain != "example.com" {
		t.Errorf("Create.Domain = %q, want example.com", created.Domain)
	}
	if created.CertStatus != "pending" {
		t.Errorf("Create.CertStatus = %q, want pending", created.CertStatus)
	}

	// Get
	got, err := pg.GetCustomDomain(ctx, tenantID, "example.com")
	if err != nil {
		t.Fatalf("GetCustomDomain: %v", err)
	}
	if got == nil {
		t.Fatal("GetCustomDomain: got nil, want non-nil")
	}
	if got.Domain != "example.com" {
		t.Errorf("GetCustomDomain.Domain = %q, want example.com", got.Domain)
	}

	// Get by domain (cross-tenant lookup)
	gotByDomain, err := pg.GetCustomDomainByDomain(ctx, "example.com")
	if err != nil {
		t.Fatalf("GetCustomDomainByDomain: %v", err)
	}
	if gotByDomain == nil {
		t.Fatal("GetCustomDomainByDomain: got nil, want non-nil")
	}

	// List (should have 1)
	all, err = pg.ListCustomDomains(ctx, tenantID)
	if err != nil {
		t.Fatalf("ListCustomDomains: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("ListCustomDomains: len=%d, want 1", len(all))
	}

	// Update status
	if err := pg.UpdateCustomDomainStatus(ctx, created.ID, "active", ""); err != nil {
		t.Fatalf("UpdateCustomDomainStatus: %v", err)
	}
	got, err = pg.GetCustomDomain(ctx, tenantID, "example.com")
	if err != nil {
		t.Fatalf("GetCustomDomain after update: %v", err)
	}
	if got.CertStatus != "active" {
		t.Errorf("CertStatus after update = %q, want active", got.CertStatus)
	}

	// ListActiveCustomDomains
	active, err := pg.ListActiveCustomDomains(ctx)
	if err != nil {
		t.Fatalf("ListActiveCustomDomains: %v", err)
	}
	found := false
	for _, d := range active {
		if d.Domain == "example.com" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ListActiveCustomDomains: example.com not found in active list")
	}

	// Delete
	if err := pg.DeleteCustomDomain(ctx, tenantID, created.ID); err != nil {
		t.Fatalf("DeleteCustomDomain: %v", err)
	}
	got, err = pg.GetCustomDomain(ctx, tenantID, "example.com")
	if err != nil {
		t.Fatalf("GetCustomDomain after delete: %v", err)
	}
	if got != nil {
		t.Errorf("GetCustomDomain after delete: got %v, want nil", got)
	}
}

// TestCustomDomainRepo_CreateDuplicate 验证重复创建返回 ErrConflict(需 PG)。
func TestCustomDomainRepo_CreateDuplicate(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS custom_domains (
			id          BIGSERIAL PRIMARY KEY,
			tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
			domain      VARCHAR(255) NOT NULL,
			cert_status VARCHAR(16) NOT NULL DEFAULT 'pending',
			cert_error  TEXT NOT NULL DEFAULT '',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(tenant_id, domain)
		)
	`)
	if err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}

	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID

	// 清理
	_, _ = pool.Exec(ctx, `DELETE FROM custom_domains WHERE tenant_id = $1`, tenantID)

	// 首次创建
	_, err = pg.CreateCustomDomain(ctx, tenantID, "duplicate.test")
	if err != nil {
		t.Fatalf("First CreateCustomDomain: %v", err)
	}

	// 重复创建应返回 ErrConflict
	_, err = pg.CreateCustomDomain(ctx, tenantID, "duplicate.test")
	if err != ErrConflict {
		t.Errorf("Duplicate CreateCustomDomain: got %v, want ErrConflict", err)
	}
}
