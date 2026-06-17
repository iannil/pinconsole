package antiscrape

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/iannil/marketing-monitor/internal/proto"
)

// testLogger 返回一个丢弃所有输出的 slog logger 用于测试。
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{Level: slog.LevelError}))
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// TestBehaviorTracker_NoMouseEvents 验证 50+ 事件且零鼠标事件时触发 FlagSession。
func TestBehaviorTracker_NoMouseEvents(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()

	sessionID := fmt.Sprintf("test-no-mouse-%d", time.Now().UnixNano())
	bt := NewBehaviorTracker(rdb, testLogger(), sessionID)

	// 发送 60 个 rrweb IncrementalSnapshot 事件,无 mouse move source
	for i := 0; i < 60; i++ {
		bt.Observe(proto.EventPayload{
			Type: proto.EvRRWeb,
			TS:   int64(i),
			RRWeb: &proto.RRWebEvent{
				Type: 3, // IncrementalSnapshot
				Data: map[string]any{
					"source": float64(2), // MouseInteraction (click),不是 MouseMove
					"x":      float64(10),
					"y":      float64(10),
				},
			},
		})
	}

	bt.CheckAndFlag(context.Background())

	// 验证 Redis 中有 flagged:session:{sessionID}
	val, err := rdb.Get(context.Background(), "flagged:session:"+sessionID).Result()
	if err != nil {
		t.Fatalf("expected flag in redis, got error: %v", err)
	}
	if val == "" {
		t.Fatalf("expected non-empty flag reason, got empty")
	}
}

// TestBehaviorTracker_RepetitiveClicks 验证 20+ 同位置点击触发 FlagSession。
func TestBehaviorTracker_RepetitiveClicks(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()

	sessionID := fmt.Sprintf("test-repetitive-%d", time.Now().UnixNano())
	bt := NewBehaviorTracker(rdb, testLogger(), sessionID)

	// 在同一位置点击 25 次,加 10 个鼠标移动避免触发"零鼠标"
	for i := 0; i < 25; i++ {
		bt.Observe(proto.EventPayload{
			Type: proto.EvRRWeb,
			TS:   int64(i),
			RRWeb: &proto.RRWebEvent{
				Type: 3,
				Data: map[string]any{
					"source": float64(2), // click
					"x":      float64(100),
					"y":      float64(200),
				},
			},
		})
	}
	for i := 0; i < 30; i++ {
		bt.Observe(proto.EventPayload{
			Type: proto.EvRRWeb,
			TS:   int64(i),
			RRWeb: &proto.RRWebEvent{
				Type: 3,
				Data: map[string]any{
					"source": float64(1), // MouseMove
				},
			},
		})
	}

	bt.CheckAndFlag(context.Background())

	val, err := rdb.Get(context.Background(), "flagged:session:"+sessionID).Result()
	if err != nil {
		t.Fatalf("expected flag in redis, got error: %v", err)
	}
	if val == "" {
		t.Fatalf("expected non-empty flag reason, got empty")
	}
}

// TestBehaviorTracker_NoFlagForNormalTraffic 验证正常事件模式不触发 FlagSession。
func TestBehaviorTracker_NoFlagForNormalTraffic(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()

	sessionID := fmt.Sprintf("test-normal-%d", time.Now().UnixNano())
	bt := NewBehaviorTracker(rdb, testLogger(), sessionID)

	// 模拟正常用户:多变的位置 + 自然间隔
	for i := 0; i < 100; i++ {
		bt.Observe(proto.EventPayload{
			Type: proto.EvRRWeb,
			TS:   int64(i * (100 + i%50)), // 间隔不均匀
			RRWeb: &proto.RRWebEvent{
				Type: 3,
				Data: map[string]any{
					"source": float64(1), // MouseMove
					"x":      float64(i * 3 % 800),
					"y":      float64(i * 5 % 600),
				},
			},
		})
	}

	bt.CheckAndFlag(context.Background())

	// 正常流量不应触发 flag
	exists, err := rdb.Exists(context.Background(), "flagged:session:"+sessionID).Result()
	if err != nil {
		t.Fatalf("redis error: %v", err)
	}
	if exists != 0 {
		val, _ := rdb.Get(context.Background(), "flagged:session:"+sessionID).Result()
		t.Fatalf("expected NO flag for normal traffic, got reason: %s", val)
	}
}
