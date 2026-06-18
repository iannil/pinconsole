// Package storage:GC worker 用的清理方法(1l + 1u 拆自 queries.go)。
//
// 注意:chat_messages / co_browsing_commands / event_blobs 的 GC 方法
// 分别在 chat_repo.go / command_repo.go / event_blob_repo.go,
// 本文件只放跨表的 sessions / visitors 清理。
package storage

import (
	"context"
	"time"
)

// DeleteSessionsEndedBefore 1l GC:删除已结束且超过保留期的 sessions。
// 必须先删 event_blobs / chat_messages / co_browsing_commands(否则 FK 阻塞)。
func (s *Postgres) DeleteSessionsEndedBefore(ctx context.Context, threshold time.Time) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE ended_at IS NOT NULL AND ended_at < $1`, threshold)
	return err
}

// DeleteVisitorsLastSeenBefore 1l GC:删除超过保留期未活动的 visitors(孤立 visitor)。
// 必须在 sessions 已清后调用。
func (s *Postgres) DeleteVisitorsLastSeenBefore(ctx context.Context, threshold time.Time) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM visitors WHERE last_seen_at < $1`, threshold)
	return err
}
