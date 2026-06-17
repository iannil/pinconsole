-- 切片 1h：运营用户表
CREATE TABLE users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    email         TEXT         NOT NULL,
    password_hash TEXT         NOT NULL,
    display_name  TEXT         NOT NULL DEFAULT '',
    role          TEXT         NOT NULL DEFAULT 'operator' CHECK (role IN ('admin', 'operator')),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX users_tenant_email_uniq ON users (tenant_id, email);
