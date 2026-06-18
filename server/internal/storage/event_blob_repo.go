// Package storage:event_blobs 表方法(1u 拆自 queries.go)。
package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CreateEventBlob 记录一个 MinIO blob。
func (s *Postgres) CreateEventBlob(ctx context.Context, b EventBlob) (*EventBlob, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO event_blobs (
			session_id, tenant_id, blob_index,
			started_at, ended_at, event_count,
			minio_object_key, size_bytes, checksum_sha256
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, session_id, tenant_id, blob_index,
		          started_at, ended_at, event_count,
		          minio_object_key, size_bytes, checksum_sha256, created_at
	`,
		b.SessionID, b.TenantID, b.BlobIndex,
		b.StartedAt, b.EndedAt, b.EventCount,
		b.MinIOObjectKey, b.SizeBytes, b.ChecksumSHA256,
	)
	var out EventBlob
	err := row.Scan(
		&out.ID, &out.SessionID, &out.TenantID, &out.BlobIndex,
		&out.StartedAt, &out.EndedAt, &out.EventCount,
		&out.MinIOObjectKey, &out.SizeBytes, &out.ChecksumSHA256, &out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEventBlobsBySession 列出某会话的全部 blob。
func (s *Postgres) ListEventBlobsBySession(ctx context.Context, sessionID uuid.UUID) ([]EventBlob, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, session_id, tenant_id, blob_index,
		       started_at, ended_at, event_count,
		       minio_object_key, size_bytes, checksum_sha256, created_at
		FROM event_blobs
		WHERE session_id = $1
		ORDER BY blob_index ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EventBlob
	for rows.Next() {
		var b EventBlob
		if err := rows.Scan(
			&b.ID, &b.SessionID, &b.TenantID, &b.BlobIndex,
			&b.StartedAt, &b.EndedAt, &b.EventCount,
			&b.MinIOObjectKey, &b.SizeBytes, &b.ChecksumSHA256, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ListEventBlobsOlderThan 列出 created_at 早于 threshold 的 blob(GC 用)。
func (s *Postgres) ListEventBlobsOlderThan(ctx context.Context, threshold time.Time, limit int32) ([]EventBlob, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, session_id, tenant_id, blob_index,
		       started_at, ended_at, event_count,
		       minio_object_key, size_bytes, checksum_sha256, created_at
		FROM event_blobs
		WHERE created_at < $1
		ORDER BY created_at ASC
		LIMIT $2
	`, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EventBlob
	for rows.Next() {
		var b EventBlob
		if err := rows.Scan(
			&b.ID, &b.SessionID, &b.TenantID, &b.BlobIndex,
			&b.StartedAt, &b.EndedAt, &b.EventCount,
			&b.MinIOObjectKey, &b.SizeBytes, &b.ChecksumSHA256, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// DeleteEventBlobByID 按 ID 删除 blob(GC 用,配合 MinIO 删除)。
func (s *Postgres) DeleteEventBlobByID(ctx context.Context, id uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM event_blobs WHERE id = $1`, id)
	return err
}
