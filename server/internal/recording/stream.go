// Package recording 实现 Redis Stream 写入与 MinIO 快照 flush。
//
// 设计（详见 docs/progress/2026-06-17-slice-1b-spec.md §Stream 快照）：
//
//   - 每个访客 session 一个 Redis Stream（key: stream:session:<uuid>）
//   - 每个 visitor 事件 XADD 到对应 stream
//   - 后台 flusher：达到 1000 events 或 30s 间隔时，
//     XREAD 消费该 stream、msgpack 聚合 + zstd 压缩、PutObject 到 MinIO、INSERT event_blobs
//   - flush 后 XTRIM stream，保留最近 N 事件供新订阅者从尾部拉取（默认 200）
package recording

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

// StreamKey 返回 Redis Stream key。
func StreamKey(sessionID uuid.UUID) string {
	return fmt.Sprintf("stream:session:%s", sessionID)
}

// Stream 封装 Redis Stream 写入。
type Stream struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// NewStream 创建 Stream 写入器。
func NewStream(rdb *redis.Client, logger *slog.Logger) *Stream {
	return &Stream{rdb: rdb, logger: logger}
}

// Append 把一条 msgpack envelope 字节流 XADD 到对应 session stream。
func (s *Stream) Append(ctx context.Context, sessionID uuid.UUID, msg []byte) error {
	key := StreamKey(sessionID)
	if err := s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		Values: map[string]interface{}{
			"data": msg,
		},
		MaxLen: 0, // flusher 负责 trim，这里不限制
	}).Err(); err != nil {
		return fmt.Errorf("xadd: %w", err)
	}
	return nil
}

// Len 返回 stream 当前长度。
func (s *Stream) Len(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	key := StreamKey(sessionID)
	n, err := s.rdb.XLen(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}

// Range 返回 stream 中 [start, end) 范围的 entry（含 ID 与 data 字节）。
// 用于 flusher 读取一批数据。
func (s *Stream) Range(ctx context.Context, sessionID uuid.UUID, start, stop string) ([]StreamEntry, error) {
	key := StreamKey(sessionID)
	msgs, err := s.rdb.XRange(ctx, key, start, stop).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]StreamEntry, 0, len(msgs))
	for _, m := range msgs {
		var data []byte
		if v, ok := m.Values["data"].(string); ok {
			data = []byte(v)
		}
		out = append(out, StreamEntry{ID: m.ID, Data: data})
	}
	return out, nil
}

// Trim 删除 stream 中给定 ID 之前的所有 entry。
// 使用 XTRIM MAXLEN ~ n 保留最近 n 条（go-redis v9 用 XTrimMaxLenApprox）。
func (s *Stream) Trim(ctx context.Context, sessionID uuid.UUID, keepApprox int64) error {
	key := StreamKey(sessionID)
	// limit=0 表示无显式 limit（让 Redis 自行决定）
	return s.rdb.XTrimMaxLenApprox(ctx, key, keepApprox, 0).Err()
}

// Delete 从 Redis 删除整个 stream（session 结束时调用）。
func (s *Stream) Delete(ctx context.Context, sessionID uuid.UUID) error {
	key := StreamKey(sessionID)
	return s.rdb.Del(ctx, key).Err()
}

// StreamEntry 是 stream 中的一条记录。
type StreamEntry struct {
	ID   string
	Data []byte
}

// Config 是 flusher 的配置。
type Config struct {
	EventThreshold int           // 触发 flush 的累计事件数
	Interval       time.Duration // 触发 flush 的固定间隔
	TrimKeep       int64         // flush 后保留多少事件
}

// DefaultConfig 是规格中锁定的默认值。
func DefaultConfig() Config {
	return Config{
		EventThreshold: 1000,
		Interval:       30 * time.Second,
		TrimKeep:       200,
	}
}

// Flusher 是后台 worker，扫描活跃 session、按阈值或间隔 flush 到 MinIO。
//
// v1 切片 1b：简化实现——hub 维护活跃 session 注册表，
// flusher 定期遍历注册表，检查每个 stream 长度，超阈值则 flush。
// 真实生产可能需要按 session 维护独立 ticker；1b 用全局 ticker 足够。
type Flusher struct {
	cfg       Config
	stream    *Stream
	stores    *storage.Stores
	logger    *slog.Logger

	mu       sync.RWMutex
	active   map[uuid.UUID]*activeSession
	stopCh   chan struct{}
	stopped  bool
}

type activeSession struct {
	sessionID uuid.UUID
	tenantID  uuid.UUID
	lastFlushedEventCount int64  // 上次 flush 时 stream 长度
	blobIndex int32               // 下一个 blob 的 index
}

// NewFlusher 创建 flusher。
func NewFlusher(cfg Config, stream *Stream, stores *storage.Stores, logger *slog.Logger) *Flusher {
	return &Flusher{
		cfg:     cfg,
		stream:  stream,
		stores:  stores,
		logger:  logger,
		active:  make(map[uuid.UUID]*activeSession),
		stopCh:  make(chan struct{}),
	}
}

// Register 把一个 session 加入 flusher 监控。
func (f *Flusher) Register(sessionID, tenantID uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.active[sessionID]; !exists {
		f.active[sessionID] = &activeSession{
			sessionID: sessionID,
			tenantID:  tenantID,
			blobIndex: 0,
		}
	}
}

// Unregister 移除 flusher 监控（session 结束时调用，会触发最后一次 flush）。
func (f *Flusher) Unregister(ctx context.Context, sessionID uuid.UUID) {
	f.mu.Lock()
	as, exists := f.active[sessionID]
	if exists {
		delete(f.active, sessionID)
	}
	f.mu.Unlock()
	if exists {
		_ = f.flushSession(ctx, as)
	}
}

// FlushSessionNow 立即同步 flush 某 session 的剩余事件。
// 用于 visitor WS 断开时确保最后一批事件归档（spec §End flush 时机）。
// 若该 session 不在 flusher 监控中（未注册），无操作。
func (f *Flusher) FlushSessionNow(ctx context.Context, sessionID uuid.UUID) error {
	f.mu.RLock()
	as, exists := f.active[sessionID]
	f.mu.RUnlock()
	if !exists {
		return nil
	}
	return f.flushSession(ctx, as)
}

// Start 启动后台 ticker。
func (f *Flusher) Start(ctx context.Context) {
	ticker := time.NewTicker(f.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-f.stopCh:
			return
		case <-ticker.C:
			f.tick(ctx)
		}
	}
}

// Stop 停止 flusher。
func (f *Flusher) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.stopped {
		f.stopped = true
		close(f.stopCh)
	}
}

// tick 扫描所有活跃 session，按阈值或时间触发 flush。
func (f *Flusher) tick(ctx context.Context) {
	f.mu.RLock()
	sessions := make([]*activeSession, 0, len(f.active))
	for _, as := range f.active {
		sessions = append(sessions, as)
	}
	f.mu.RUnlock()

	for _, as := range sessions {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := f.stream.Len(ctx, as.sessionID)
		if err != nil {
			f.logger.Warn("stream len failed", "error", err, "session_id", as.sessionID)
			continue
		}
		flushed := n - as.lastFlushedEventCount
		if flushed >= int64(f.cfg.EventThreshold) {
			if err := f.flushSession(ctx, as); err != nil {
				f.logger.Warn("flush failed", "error", err, "session_id", as.sessionID)
			}
		}
	}
}

// flushSession 读 stream 全量（自上次 flush）、写 MinIO、写 PG、XTRIM。
func (f *Flusher) flushSession(ctx context.Context, as *activeSession) error {
	// 读自上次 flush 后的所有 entry
	// 简化：XRange 全部，flush 完 XTRIM 整条 stream（保留 TrimKeep）
	entries, err := f.stream.Range(ctx, as.sessionID, "-", "+")
	if err != nil {
		return fmt.Errorf("range: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}

	// 聚合：msgpack array of bytes
	// 注：每条 entry 的 Data 已经是 msgpack envelope，直接拼接成 array
	// 为减少 MinIO 对象数量，flush 出的 blob 是 "msgpack array of envelope bytes"
	// 回放时逐条 decode
	blob, startedAt, endedAt, checksum, err := encodeBlob(entries)
	if err != nil {
		return fmt.Errorf("encode blob: %w", err)
	}

	// 1o P1-7:补偿事务模式
	// MinIO PutObject → PG INSERT → Redis XTRIM
	// PG INSERT 失败时,RemoveObject 补偿删 MinIO 对象避免孤儿
	// Redis XTRIM 失败不影响一致性(stream 多保留些 entry,下次 flush 再 trim)
	objectKey := fmt.Sprintf("sessions/%s/%d.msgpack", as.sessionID, as.blobIndex)
	if err := f.stores.MinIO.PutBytes(ctx, objectKey, blob); err != nil {
		return fmt.Errorf("minio put: %w", err)
	}

	// 写 PG event_blobs
	_, err = f.stores.PG.CreateEventBlob(ctx, storage.EventBlob{
		SessionID:      as.sessionID,
		TenantID:       as.tenantID,
		BlobIndex:      as.blobIndex,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		EventCount:     int32(len(entries)),
		MinIOObjectKey: objectKey,
		SizeBytes:      int64(len(blob)),
		ChecksumSHA256: checksum,
	})
	if err != nil {
		// 补偿:删 MinIO 对象,避免孤儿
		compensateErr := f.stores.MinIO.Client.RemoveObject(ctx, f.stores.MinIO.Bucket, objectKey, minio.RemoveObjectOptions{})
		if compensateErr != nil {
			f.logger.Error("compensate minio remove failed (orphan risk)",
				"key", objectKey, "pg_error", err, "minio_error", compensateErr)
		} else {
			f.logger.Warn("compensate minio remove ok after PG insert failed",
				"key", objectKey, "pg_error", err)
		}
		return fmt.Errorf("create event_blob: %w", err)
	}

	// XTRIM(保留 TrimKeep) — PG 写入成功后才 trim,失败仅 warn(下次 flush 再试)
	if err := f.stream.Trim(ctx, as.sessionID, f.cfg.TrimKeep); err != nil {
		f.logger.Warn("xtrim failed (non-fatal, will retry next flush)", "error", err)
	}

	as.lastFlushedEventCount = 0
	as.blobIndex++

	f.logger.Info("flushed blob",
		"session_id", as.sessionID,
		"blob_index", as.blobIndex-1,
		"event_count", len(entries),
		"size_bytes", len(blob),
	)
	return nil
}
