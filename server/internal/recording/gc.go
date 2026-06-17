// Package recording：blob GC worker。
//
// 设计（详见 docs/progress/2026-06-17-slice-1d-spec.md §Blob GC）：
//
//   - 每小时扫描 PG event_blobs.created_at < NOW() - retention_days
//   - 对每条记录：
//       1. 删 MinIO 对象（key 来自 minio_object_key）
//       2. 删 PG event_blobs 行
//   - 默认 retention_days=30（与 PLAN.md "默认 30 天" 一致）
//   - 单次扫描最多删 1000 条（避免长事务）
//   - 失败不阻塞下次扫描
package recording

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/iannil/marketing-monitor/internal/storage"
)

// GCConfig 是 GC worker 的配置。
type GCConfig struct {
	Retention     time.Duration // 默认 30*24h
	ScanInterval  time.Duration // 默认 1h
	BatchSize     int32         // 默认 1000
}

// DefaultGCConfig 默认配置。
func DefaultGCConfig() GCConfig {
	return GCConfig{
		Retention:    30 * 24 * time.Hour,
		ScanInterval: time.Hour,
		BatchSize:    1000,
	}
}

// GC 是后台 GC worker。
type GC struct {
	cfg    GCConfig
	stores *storage.Stores
	logger *slog.Logger

	stopCh  chan struct{}
	stopped bool
	mu      sync.Mutex
}

// NewGC 创建 GC worker。
func NewGC(cfg GCConfig, stores *storage.Stores, logger *slog.Logger) *GC {
	return &GC{
		cfg:    cfg,
		stores: stores,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Start 启动后台 ticker。
func (g *GC) Start(ctx context.Context) {
	// 启动后立即跑一次，然后按 ticker
	go func() {
		g.runOnce(ctx)
		ticker := time.NewTicker(g.cfg.ScanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-g.stopCh:
				return
			case <-ticker.C:
				g.runOnce(ctx)
			}
		}
	}()
}

// Stop 停止 GC。
func (g *GC) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.stopped {
		g.stopped = true
		close(g.stopCh)
	}
}

// runOnce 执行一次 GC 扫描。
func (g *GC) runOnce(ctx context.Context) {
	threshold := time.Now().Add(-g.cfg.Retention)
	blobs, err := g.stores.PG.ListEventBlobsOlderThan(ctx, threshold, g.cfg.BatchSize)
	if err != nil {
		g.logger.Warn("gc list failed", "error", err)
		return
	}
	if len(blobs) == 0 {
		return
	}

	var deleted, failed int
	for _, b := range blobs {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// 删 MinIO（RemoveObject 幂等，对象不存在不报错）
		if err := g.stores.MinIO.Client.RemoveObject(ctx, g.stores.MinIO.Bucket, b.MinIOObjectKey, minioRemoveObjectOpts); err != nil {
			g.logger.Warn("gc minio remove failed", "key", b.MinIOObjectKey, "error", err)
			failed++
			continue
		}
		// 删 PG
		if err := g.stores.PG.DeleteEventBlobByID(ctx, b.ID); err != nil {
			g.logger.Warn("gc pg delete failed", "id", b.ID, "error", err)
			failed++
			continue
		}
		deleted++
	}

	g.logger.Info("gc completed",
		"scanned", len(blobs),
		"deleted", deleted,
		"failed", failed,
		"retention", g.cfg.Retention.String(),
	)
}
