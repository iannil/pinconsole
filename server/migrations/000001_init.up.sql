-- v1 切片 1b 初始 schema
-- 详见 docs/progress/2026-06-17-slice-1b-spec.md §DB Schema

-- 启用扩展
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 默认 tenant_id（v1 不启用多租户，预留）
-- 所有表用此值作为占位符
-- 未来激活多租户时，迁移将添加真实租户并 UPDATE 关联表

-- ===== visitors =====
CREATE TABLE visitors (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    fingerprint   TEXT         NOT NULL,
    ua            TEXT,
    ip_first_seen INET,
    first_seen_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_seen_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    meta          JSONB        NOT NULL DEFAULT '{}'::jsonb
);

-- 同租户下 fingerprint 唯一（v1 tenant_id 全部相同，等同于全局唯一）
CREATE UNIQUE INDEX visitors_tenant_fingerprint_uniq
    ON visitors (tenant_id, fingerprint);

CREATE INDEX visitors_last_seen_at_idx
    ON visitors (last_seen_at DESC);

-- ===== sessions =====
CREATE TABLE sessions (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    visitor_id    UUID         NOT NULL REFERENCES visitors(id) ON DELETE CASCADE,
    started_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_event_at TIMESTAMPTZ,
    ended_at      TIMESTAMPTZ,
    status        TEXT         NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'ended', 'timed_out')),
    event_count   INTEGER      NOT NULL DEFAULT 0,
    ua            TEXT,
    ip            INET
);

-- 列表查询：按租户、状态、最近活跃
CREATE INDEX sessions_tenant_status_last_event_idx
    ON sessions (tenant_id, status, last_event_at DESC);

CREATE INDEX sessions_visitor_idx
    ON sessions (visitor_id);

-- ===== event_blobs =====
-- 每个 blob 是从 Redis Stream flush 出的 msgpack 文件，存到 MinIO
CREATE TABLE event_blobs (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id        UUID         NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    tenant_id         UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    blob_index        INTEGER      NOT NULL,
    started_at        TIMESTAMPTZ  NOT NULL,
    ended_at          TIMESTAMPTZ  NOT NULL,
    event_count       INTEGER      NOT NULL,
    minio_object_key  TEXT         NOT NULL,
    size_bytes        BIGINT       NOT NULL,
    checksum_sha256   TEXT         NOT NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX event_blobs_session_blob_idx
    ON event_blobs (session_id, blob_index);

CREATE INDEX event_blobs_session_idx
    ON event_blobs (session_id, blob_index);

CREATE INDEX event_blobs_created_at_idx
    ON event_blobs (created_at DESC);
