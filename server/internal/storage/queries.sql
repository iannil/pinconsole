-- sqlc queries for v1 切片 1b
-- 详见 docs/progress/2026-06-17-slice-1b-spec.md

-- name: GetVisitorByFingerprint :one
SELECT * FROM visitors
WHERE tenant_id = $1 AND fingerprint = $2;

-- name: CreateVisitor :one
INSERT INTO visitors (tenant_id, fingerprint, ua, ip_first_seen)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, fingerprint)
DO UPDATE SET
    last_seen_at = NOW(),
    ua = COALESCE(EXCLUDED.ua, visitors.ua)
RETURNING *;

-- name: TouchVisitor :exec
UPDATE visitors
SET last_seen_at = NOW()
WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (tenant_id, visitor_id, ua, ip)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions WHERE id = $1;

-- name: TouchSessionEvent :exec
UPDATE sessions
SET last_event_at = NOW(),
    event_count = event_count + $2
WHERE id = $1;

-- name: EndSession :exec
UPDATE sessions
SET ended_at = NOW(),
    status = $2
WHERE id = $1;

-- name: ListActiveSessionsByTenant :many
SELECT s.*, v.fingerprint AS visitor_fingerprint
FROM sessions s
JOIN visitors v ON v.id = s.visitor_id
WHERE s.tenant_id = $1
  AND s.status = 'active'
ORDER BY s.last_event_at DESC NULLS LAST
LIMIT $2;

-- name: CreateEventBlob :one
INSERT INTO event_blobs (
    session_id, tenant_id, blob_index,
    started_at, ended_at, event_count,
    minio_object_key, size_bytes, checksum_sha256
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListEventBlobsBySession :many
SELECT * FROM event_blobs
WHERE session_id = $1
ORDER BY blob_index ASC;
