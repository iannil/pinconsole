-- Page Leads 表（page-editor pe-1）
-- 存储落地页表单提交数据。
CREATE TABLE IF NOT EXISTS page_leads (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    page_slug   VARCHAR(64) NOT NULL,
    fields      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_page_leads_tenant_slug ON page_leads(tenant_id, page_slug);
