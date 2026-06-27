-- cd-1 自定义域名表
-- 每 tenant 可绑定多个自定义域名，certmagic 自动签发 HTTPS 证书。
CREATE TABLE IF NOT EXISTS custom_domains (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    domain      VARCHAR(255) NOT NULL,
    -- cert_status: pending / active / failed
    cert_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    -- cert_error: 签发失败时的错误信息（成功时为 ''）
    cert_error  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, domain)
);

CREATE INDEX IF NOT EXISTS idx_custom_domains_tenant ON custom_domains(tenant_id);
CREATE INDEX IF NOT EXISTS idx_custom_domains_domain ON custom_domains(domain);
