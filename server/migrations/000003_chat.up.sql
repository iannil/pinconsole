-- 切片 1g：聊天消息表
CREATE TABLE chat_messages (
    id           BIGSERIAL    PRIMARY KEY,
    tenant_id    UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    session_id   UUID         NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    sender       TEXT         NOT NULL CHECK (sender IN ('operator', 'visitor')),
    content      TEXT         NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX chat_messages_session_idx ON chat_messages (session_id, id);
CREATE INDEX chat_messages_tenant_created_idx ON chat_messages (tenant_id, created_at DESC);
