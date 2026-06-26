// Package storage:widget_configs 表方法（page-editor pe-1）。
package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetWidgetConfig 取指定 tenant + widget_type 的配置。
// 不存在返回 (nil, nil)。
func (s *Postgres) GetWidgetConfig(ctx context.Context, tenantID uuid.UUID, widgetType string) (*WidgetConfig, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, tenant_id, widget_type, config, created_at, updated_at
		FROM widget_configs
		WHERE tenant_id = $1 AND widget_type = $2
	`, tenantID, widgetType)

	var c WidgetConfig
	err := row.Scan(&c.ID, &c.TenantID, &c.WidgetType, &c.Config, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get widget_config: %w", err)
	}
	return &c, nil
}

// UpsertWidgetConfig 创建或更新 widget 配置（INSERT ON CONFLICT UPDATE）。
// 返回写入后的完整行。
func (s *Postgres) UpsertWidgetConfig(ctx context.Context, tenantID uuid.UUID, widgetType string, config []byte) (*WidgetConfig, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO widget_configs (tenant_id, widget_type, config)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, widget_type)
		DO UPDATE SET config = $3, updated_at = NOW()
		RETURNING id, tenant_id, widget_type, config, created_at, updated_at
	`, tenantID, widgetType, config)

	var c WidgetConfig
	if err := row.Scan(&c.ID, &c.TenantID, &c.WidgetType, &c.Config, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, fmt.Errorf("upsert widget_config: %w", err)
	}
	return &c, nil
}

// ListWidgetConfigs 返回指定 tenant 的所有 widget 配置。
func (s *Postgres) ListWidgetConfigs(ctx context.Context, tenantID uuid.UUID) ([]*WidgetConfig, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, tenant_id, widget_type, config, created_at, updated_at
		FROM widget_configs
		WHERE tenant_id = $1
		ORDER BY widget_type
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list widget_configs: %w", err)
	}
	defer rows.Close()

	var out []*WidgetConfig
	for rows.Next() {
		var c WidgetConfig
		if err := rows.Scan(&c.ID, &c.TenantID, &c.WidgetType, &c.Config, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan widget_config: %w", err)
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// DeleteWidgetConfig 删除指定 widget 配置。
func (s *Postgres) DeleteWidgetConfig(ctx context.Context, tenantID uuid.UUID, widgetType string) error {
	_, err := s.Pool.Exec(ctx, `
		DELETE FROM widget_configs WHERE tenant_id = $1 AND widget_type = $2
	`, tenantID, widgetType)
	if err != nil {
		return fmt.Errorf("delete widget_config: %w", err)
	}
	return nil
}

// DeleteWidgetConfigByID 按 ID 删除（用于测试清理）。
func (s *Postgres) DeleteWidgetConfigByID(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM widget_configs WHERE id = $1`, id)
	return err
}
