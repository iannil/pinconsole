// Package antiscrape：行为分析（服务端启发式标记）。
package antiscrape

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/iannil/marketing-monitor/internal/proto"
	"github.com/redis/go-redis/v9"
)

// BehaviorTracker 在 visitorWS read loop 中统计事件模式。
// 满足启发式条件时在 Redis 中标记 session。
type BehaviorTracker struct {
	rdb      *redis.Client
	logger   *slog.Logger
	sessionID string

	mu              sync.Mutex
	mouseEventCount int
	clickPositions  map[[2]int]int // (x,y) → 次数
	eventTypes      map[string]int // type → 次数
	lastEventAt     time.Time
	firstEventAt    time.Time
	maxInterval     time.Duration
	minInterval     time.Duration
	totalEvents     int
}

// NewBehaviorTracker 创建一个 session 的行为追踪器。
func NewBehaviorTracker(rdb *redis.Client, logger *slog.Logger, sessionID string) *BehaviorTracker {
	return &BehaviorTracker{
		rdb:           rdb,
		logger:        logger,
		sessionID:     sessionID,
		clickPositions: make(map[[2]int]int),
		eventTypes:    make(map[string]int),
	}
}

// Observe 处理一个事件 payload，更新统计。
func (bt *BehaviorTracker) Observe(payload proto.EventPayload) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	now := time.Now()
	if bt.firstEventAt.IsZero() {
		bt.firstEventAt = now
	}
	interval := now.Sub(bt.lastEventAt)
	if bt.totalEvents > 0 {
		if interval > bt.maxInterval {
			bt.maxInterval = interval
		}
		if bt.minInterval == 0 || interval < bt.minInterval {
			bt.minInterval = interval
		}
	}
	bt.lastEventAt = now
	bt.totalEvents++

	// 按 rrweb 事件类型统计
	if payload.Type == proto.EvRRWeb && payload.RRWeb != nil {
		switch payload.RRWeb.Type {
		case 3: // IncrementalSnapshot
			data := payload.RRWeb.Data
			if data == nil {
				return
			}
			source, _ := data["source"].(float64)
			switch int(source) {
			case 1: // MouseMove
				bt.mouseEventCount++
			case 2: // MouseInteraction (click)
				if x, y := extractXY(data); x >= 0 {
					bt.clickPositions[[2]int{x, y}]++
				}
			}
		}
	}
}

// CheckAndFlag 检查启发式条件，满足则在 Redis 标记。
// 在每 100 个事件后调用一次。
func (bt *BehaviorTracker) CheckAndFlag(ctx context.Context) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	var reasons []string

	// 启发式 1：零鼠标事件（纯脚本无鼠标交互）
	if bt.totalEvents > 50 && bt.mouseEventCount == 0 {
		reasons = append(reasons, "no_mouse_events")
	}

	// 启发式 2：点击位置高度重复（机器人模式）
	if len(bt.clickPositions) > 0 {
		maxRepeat := 0
		for _, count := range bt.clickPositions {
			if count > maxRepeat {
				maxRepeat = count
			}
		}
		if maxRepeat > 20 {
			reasons = append(reasons, "repetitive_clicks")
		}
	}

	// 启发式 3：事件间隔过于均匀（机器生成）
	if bt.totalEvents > 100 && bt.maxInterval > 0 && bt.minInterval > 0 {
		ratio := bt.maxInterval.Seconds() / bt.minInterval.Seconds()
		if ratio < 2.0 {
			reasons = append(reasons, "uniform_intervals")
		}
	}

	if len(reasons) > 0 {
		reason := "behavior:" + joinReasons(reasons)
		bt.logger.Warn("behavior flag triggered",
			"session_id", bt.sessionID,
			"reasons", reasons,
			"total_events", bt.totalEvents,
			"mouse_events", bt.mouseEventCount,
		)
		_ = FlagSession(ctx, bt.rdb, bt.sessionID, reason)
	}
}

func extractXY(data map[string]any) (int, int) {
	xVal, ok1 := data["x"].(float64)
	yVal, ok2 := data["y"].(float64)
	if !ok1 || !ok2 {
		return -1, -1
	}
	return int(xVal), int(yVal)
}

func joinReasons(reasons []string) string {
	result := ""
	for i, r := range reasons {
		if i > 0 {
			result += ","
		}
		result += r
	}
	return result
}
