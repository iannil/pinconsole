// pe-1 测试:widget_configs CRUD 集成测试(需 PG)。
package storage

import (
	"context"
	"encoding/json"
	"testing"
)

// jsonBytesEqual compares two JSON byte slices by value (ignores key ordering).
func jsonBytesEqual(a, b []byte) bool {
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		return false
	}
	ae, _ := json.Marshal(va)
	be, _ := json.Marshal(vb)
	return string(ae) == string(be)
}

// TestWidgetConfigRepo_CRUD 验证 Get/Upsert/List/Delete round-trip(需 PG)。
func TestWidgetConfigRepo_CRUD(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()

	// 确保表存在（手动 CREATE IF NOT EXISTS，迁移由 docker-compose 启动时执行）
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS widget_configs (
			id          BIGSERIAL PRIMARY KEY,
			tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
			widget_type VARCHAR(32) NOT NULL,
			config      JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(tenant_id, widget_type)
		)
	`)
	if err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}

	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID

	// 初始应为空
	got, err := pg.GetWidgetConfig(ctx, tenantID, "popup")
	if err != nil {
		t.Fatalf("GetWidgetConfig (empty): %v", err)
	}
	if got != nil {
		t.Fatalf("GetWidgetConfig: got %v, want nil", got)
	}

	// Upsert
	cfg := []byte(`{"title":"Hello","body":"World","action_label":"OK","dismissible":true}`)
	created, err := pg.UpsertWidgetConfig(ctx, tenantID, "popup", cfg)
	if err != nil {
		t.Fatalf("UpsertWidgetConfig: %v", err)
	}
	if created.WidgetType != "popup" {
		t.Errorf("Upsert.WidgetType = %q, want popup", created.WidgetType)
	}

	// Get
	got, err = pg.GetWidgetConfig(ctx, tenantID, "popup")
	if err != nil {
		t.Fatalf("GetWidgetConfig: %v", err)
	}
	if got == nil {
		t.Fatal("GetWidgetConfig: got nil, want non-nil")
	}
	if !jsonBytesEqual(got.Config, cfg) {
		t.Errorf("GetWidgetConfig.Config: got %s, want %s", got.Config, cfg)
	}

	// List (should have 1)
	all, err := pg.ListWidgetConfigs(ctx, tenantID)
	if err != nil {
		t.Fatalf("ListWidgetConfigs: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("ListWidgetConfigs: len=%d, want 1", len(all))
	}

	// Upsert update
	cfg2 := []byte(`{"title":"Updated","body":"Updated","action_label":"Go","dismissible":false}`)
	updated, err := pg.UpsertWidgetConfig(ctx, tenantID, "popup", cfg2)
	if err != nil {
		t.Fatalf("UpsertWidgetConfig (update): %v", err)
	}
	if !jsonBytesEqual(updated.Config, cfg2) {
		t.Errorf("UpsertWidgetConfig update: got %s, want %s", updated.Config, cfg2)
	}

	// Delete
	if err := pg.DeleteWidgetConfig(ctx, tenantID, "popup"); err != nil {
		t.Fatalf("DeleteWidgetConfig: %v", err)
	}
	got, err = pg.GetWidgetConfig(ctx, tenantID, "popup")
	if err != nil {
		t.Fatalf("GetWidgetConfig after delete: %v", err)
	}
	if got != nil {
		t.Errorf("GetWidgetConfig after delete: got %v, want nil", got)
	}
}
