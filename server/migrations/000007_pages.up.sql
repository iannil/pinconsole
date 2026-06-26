-- Pages 表（page-editor pe-1）
-- 存储拖拽编辑器创建的落地页 schema。
CREATE TABLE IF NOT EXISTS pages (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    slug        VARCHAR(64) NOT NULL,
    title       VARCHAR(255) NOT NULL DEFAULT '',
    status      VARCHAR(16) NOT NULL DEFAULT 'draft',
    schema      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_pages_tenant ON pages(tenant_id);
CREATE INDEX IF NOT EXISTS idx_pages_tenant_status ON pages(tenant_id, status);
