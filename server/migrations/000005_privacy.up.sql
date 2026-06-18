-- 1l-privacy-gdpr: 访客同意记录表
-- 存储 fingerprint 级的 consent 状态(GDPR Art.7 合规证据)。
CREATE TABLE visitor_consents (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    fingerprint  VARCHAR(64) NOT NULL,
    -- scope 标识同意覆盖的范围:v1 用 'all'(全采集);未来可分 'recording'/'keyboard'/'screenshot'
    scope        VARCHAR(32) NOT NULL DEFAULT 'all',
    -- version 是同意书版本;条款变更时升级,旧版本同意自动失效
    version      VARCHAR(16) NOT NULL DEFAULT 'v1',
    accepted     BOOLEAN NOT NULL,
    consented_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- expires_at 可选;NULL 表示不过期
    expires_at   TIMESTAMPTZ,
    UNIQUE(fingerprint, scope, version)
);
CREATE INDEX idx_visitor_consents_fp ON visitor_consents(fingerprint);
CREATE INDEX idx_visitor_consents_tenant_fp ON visitor_consents(tenant_id, fingerprint);
