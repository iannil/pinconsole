// Package recording：session 最近 full snapshot + meta event 的 Redis 缓存。
//
// 设计（详见 docs/progress/2026-06-17-slice-1c-spec.md §订阅初始状态）：
//
//   - 每个 session 在 Redis 中缓存最近一次 full snapshot（msgpack 序列化的 envelope）
//   - 同时缓存首个 meta event(rrweb type=4,含访客 viewport 尺寸)
//   - snapshot TTL 5 分钟，每次新 full 来时刷新
//   - meta TTL 30 分钟(只在 session 开始时发一次,需覆盖典型 session 时长)
//   - admin subscribe 时,服务端先发 meta 再发 snapshot,然后开始广播增量
//   - 让 admin 不需要"等下一次周期性 full"就能立即看到访客当前页面,
//     且 rrweb-player 收到 meta 后能正确触发 handleResize 让 iframe 显示
package recording

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
)

// snapshotTTL 是 Redis 中 snapshot 的过期时间。
//
// 2026-06-21:从 5min 提到 30min,与 metaTTL 对齐。
// 原因:visitor SDK 周期性 full snapshot 续期间隔 10s(原 30s),理论上不会过期,
// 但实际边界:visitor 长时间静默(>5min 无交互)/ 短暂断网重连 / SDK reload,
// 都可能让 admin subscribe 时错过续期窗口,snapshot 过期,只拿到 meta,
// rrweb Replayer 缺 FullSnapshot 不能重建初始 DOM,iframe 空白。
// 30min 覆盖典型 session 时长,Redis 内存压力可控(每 session 一份 envelope bytes,
// 单 session 几 KB ~ 几十 KB)。
const snapshotTTL = 30 * time.Minute

// metaTTL 是 Redis 中 meta event 的过期时间。
// meta 只在 session 开始时由 SDK 发一次,后面不会刷新。
// TTL 需要覆盖典型 session 时长(访客在页面停留的时间)。
// 30 分钟平衡:足够大多数 session,Redis 内存压力可控。
const metaTTL = 30 * time.Minute

// SnapshotKey 返回 Redis 中存储 session full snapshot 的 key。
func SnapshotKey(sessionID uuid.UUID) string {
	return fmt.Sprintf("snapshot:session:%s", sessionID)
}

// MetaKey 返回 Redis 中存储 session meta event 的 key。
func MetaKey(sessionID uuid.UUID) string {
	return fmt.Sprintf("meta:session:%s", sessionID)
}

// SnapshotCache 封装 snapshot + meta 的读写。
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

// SetMeta 缓存 session 的 meta event(envelope bytes)。
// meta 只在 session 开始时设置一次,后续 full snapshot 刷新时不覆盖。
// 用 NX 语义:仅在 key 不存在时设置(避免周期性 full 把已有 meta 覆盖成 nil)。
// 但 Redis 单独 SET NX 不带 TTL 会留下永久 key,所以这里用 SET 直接覆盖 + 长 TTL,
// 因为 meta event 内容在 session 内不变,覆盖等价于刷新 TTL。
func (c *SnapshotCache) SetMeta(ctx context.Context, sessionID uuid.UUID, envelopeBytes []byte) error {
	return c.redis.Set(ctx, MetaKey(sessionID), envelopeBytes, metaTTL)
}

// GetMeta 取 session 的 meta event。
// 不存在返回 (nil, nil)。
func (c *SnapshotCache) GetMeta(ctx context.Context, sessionID uuid.UUID) ([]byte, error) {
	return c.redis.Get(ctx, MetaKey(sessionID))
}

// DeleteMeta 清除 meta(session 结束时调用)。
func (c *SnapshotCache) DeleteMeta(ctx context.Context, sessionID uuid.UUID) error {
	return c.redis.Del(ctx, MetaKey(sessionID))
}
