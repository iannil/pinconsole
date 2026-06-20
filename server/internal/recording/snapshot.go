// Package recording：session 最近 full snapshot 的 Redis 缓存。
//
// 设计（详见 docs/progress/2026-06-17-slice-1c-spec.md §订阅初始状态）：
//
//   - 每个 session 在 Redis 中缓存最近一次 full snapshot（msgpack 序列化的 envelope）
//   - TTL 5 分钟，每次新 full 来时刷新
//   - admin subscribe 时，服务端先发该 snapshot 给 admin，再开始广播增量
//   - 让 admin 不需要"等下一次周期性 full"就能立即看到访客当前页面
package recording

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
)

// snapshotTTL 是 Redis 中 snapshot 的过期时间。
const snapshotTTL = 5 * time.Minute

// SnapshotKey 返回 Redis 中存储 session full snapshot 的 key。
func SnapshotKey(sessionID uuid.UUID) string {
	return fmt.Sprintf("snapshot:session:%s", sessionID)
}

// SnapshotCache 封装 snapshot 的读写。
type SnapshotCache struct {
	redis *storage.Redis
}

// NewSnapshotCache 创建 snapshot 缓存。
func NewSnapshotCache(rdb *storage.Redis) *SnapshotCache {
	return &SnapshotCache{redis: rdb}
}

// Set 缓存 session 的最近 full snapshot（envelope bytes）。
func (c *SnapshotCache) Set(ctx context.Context, sessionID uuid.UUID, envelopeBytes []byte) error {
	return c.redis.Set(ctx, SnapshotKey(sessionID), envelopeBytes, snapshotTTL)
}

// Get 取 session 的最近 full snapshot。
// 不存在返回 (nil, nil)。
func (c *SnapshotCache) Get(ctx context.Context, sessionID uuid.UUID) ([]byte, error) {
	return c.redis.Get(ctx, SnapshotKey(sessionID))
}

// Delete 清除 snapshot（session 结束时调用）。
func (c *SnapshotCache) Delete(ctx context.Context, sessionID uuid.UUID) error {
	return c.redis.Del(ctx, SnapshotKey(sessionID))
}
