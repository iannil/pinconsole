-- pe-1 page-editor: Widget 配置表
-- 存储 4 类 visitor UI widget 的 JSON 配置（popup/chat/cobrowse_banner/consent_banner）
-- 每 tenant 每 widget_type 一条记录，admin 编辑后 SDK 在 start() 时拉取。
CREATE TABLE widget_configs (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    widget_type VARCHAR(32) NOT NULL,
    -- 配置 JSONB，结构与 proto WidgetConfigMap 中各 widget 的 TS 类型对应
    config      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, widget_type)
);

CREATE INDEX idx_widget_configs_tenant ON widget_configs(tenant_id);
