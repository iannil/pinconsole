-- 切片 1e：co-browsing 命令审计表
-- 详见 docs/progress/2026-06-17-slice-1e-spec.md §审计日志

CREATE TABLE co_browsing_commands (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    session_id      UUID         NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    -- v1 不做认证，operator_id 暂用 client IP 或 client_id 占位
    operator_id     TEXT         NOT NULL DEFAULT 'unknown',
    command_type    TEXT         NOT NULL CHECK (command_type IN (
        'cursor_highlight', 'click', 'scroll', 'fill_input', 'navigate', 'release_control'
    )),
    target_node_id  INTEGER,
    payload         JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX cobrowsing_commands_session_idx
    ON co_browsing_commands (session_id, created_at DESC);

CREATE INDEX cobrowsing_commands_tenant_created_idx
    ON co_browsing_commands (tenant_id, created_at DESC);
