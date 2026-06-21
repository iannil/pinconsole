// Go-1 切片补测:DefaultRateLimitConfig + IsSessionFlagged + extractXY 边界,
// 提升覆盖率 86.7% → 100%。
package antiscrape

import (
	"context"
	"testing"
	"time"

	"github.com/iannil/pinconsole/internal/proto"
)

// TestDefaultRateLimitConfig 验证默认配置值。
func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	if cfg.RequestsPerMin != 60 {
		t.Errorf("RequestsPerMin: got %d, want 60", cfg.RequestsPerMin)
	}
	if cfg.Window != time.Minute {
		t.Errorf("Window: got %v, want %v", cfg.Window, time.Minute)
	}
}

// TestIsSessionFlagged_NotFlagged 验证未标记 session 返回 false。
func TestIsSessionFlagged_NotFlagged(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sessionID := "test-not-flagged-" + time.Now().Format("150405.000000000")
	rdb.Del(ctx, "flagged:session:"+sessionID)

	flagged, reason, err := IsSessionFlagged(ctx, rdb, sessionID)
	if err != nil {
		t.Fatalf("IsSessionFlagged: %v", err)
	}
	if flagged {
		t.Errorf("flagged: got true, want false")
	}
	if reason != "" {
		t.Errorf("reason: got %q, want empty", reason)
	}
}

// TestIsSessionFlagged_Flagged 验证 FlagSession 后能查到。
func TestIsSessionFlagged_Flagged(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sessionID := "test-flagged-" + time.Now().Format("150405.000000000")
	reason := "behavior:repetitive_clicks"

	rdb.Del(ctx, "flagged:session:"+sessionID)
	defer rdb.Del(ctx, "flagged:session:"+sessionID)

	if err := FlagSession(ctx, rdb, sessionID, reason); err != nil {
		t.Fatalf("FlagSession: %v", err)
	}

	flagged, gotReason, err := IsSessionFlagged(ctx, rdb, sessionID)
	if err != nil {
		t.Fatalf("IsSessionFlagged: %v", err)
	}
	if !flagged {
		t.Errorf("flagged: got false, want true")
	}
	if gotReason != reason {
		t.Errorf("reason: got %q, want %q", gotReason, reason)
	}
}

// TestExtractXY_InvalidData 验证无效 x/y 返回 -1,-1。
func TestExtractXY_InvalidData(t *testing.T) {
	cases := []struct {
		name string
		data map[string]any
	}{
		{"missing x", map[string]any{"y": float64(10)}},
		{"missing y", map[string]any{"x": float64(5)}},
		{"x not float64", map[string]any{"x": "10", "y": float64(10)}},
		{"y not float64", map[string]any{"x": float64(5), "y": "10"}},
		{"both missing", map[string]any{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x, y := extractXY(tc.data)
			if x != -1 || y != -1 {
				t.Errorf("extractXY(%v): got (%d,%d), want (-1,-1)", tc.data, x, y)
			}
		})
	}
}

// TestExtractXY_Valid 验证有效 x/y 正确返回(覆盖 happy path 已有,这里补 type assert 成功路径)。
func TestExtractXY_Valid(t *testing.T) {
	x, y := extractXY(map[string]any{"x": float64(100), "y": float64(200)})
	if x != 100 || y != 200 {
		t.Errorf("extractXY: got (%d,%d), want (100,200)", x, y)
	}
}

// TestObserve_NonRRWebEvent 验证非 rrweb 事件被忽略但仍计入 totalEvents。
func TestObserve_NonRRWebEvent(t *testing.T) {
	bt := NewBehaviorTracker(nil, nil, "s-test")

	bt.Observe(proto.EventPayload{Type: proto.EvClick, TS: 1})

	bt.mu.Lock()
	defer bt.mu.Unlock()
	if bt.totalEvents != 1 {
		t.Errorf("totalEvents: got %d, want 1", bt.totalEvents)
	}
	if bt.mouseEventCount != 0 {
		t.Errorf("mouseEventCount: got %d, want 0(非 rrweb 不应计入)", bt.mouseEventCount)
	}
}

// TestObserve_RRWebNonIncrementalSnapshot 验证 rrweb 非 IncrementalSnapshot 类型被忽略。
func TestObserve_RRWebNonIncrementalSnapshot(t *testing.T) {
	bt := NewBehaviorTracker(nil, nil, "s-test")

	bt.Observe(proto.EventPayload{
		Type: proto.EvRRWeb,
		TS:   1,
		RRWeb: &proto.RRWebEvent{
			Type: 2, // FullSnapshot,不是 IncrementalSnapshot(3)
			Data: map[string]any{"source": float64(1)},
		},
	})

	bt.mu.Lock()
	defer bt.mu.Unlock()
	if bt.totalEvents != 1 {
		t.Errorf("totalEvents: got %d, want 1", bt.totalEvents)
	}
	if bt.mouseEventCount != 0 {
		t.Errorf("mouseEventCount: got %d, want 0(非 IncrementalSnapshot)", bt.mouseEventCount)
	}
}

// TestObserve_NilRRWebData 验证 IncrementalSnapshot 但 data 为 nil 不 panic。
func TestObserve_NilRRWebData(t *testing.T) {
	bt := NewBehaviorTracker(nil, nil, "s-test")

	// IncrementalSnapshot(3) 但 Data=nil → Observe 内部 data == nil 检查应 return
	bt.Observe(proto.EventPayload{
		Type: proto.EvRRWeb,
		TS:   1,
		RRWeb: &proto.RRWebEvent{
			Type: 3,
			Data: nil,
		},
	})

	bt.mu.Lock()
	defer bt.mu.Unlock()
	if bt.totalEvents != 1 {
		t.Errorf("totalEvents: got %d, want 1", bt.totalEvents)
	}
}

// TestObserve_MouseInteractionWithoutXY 验证点击但 x/y 无效不污染 clickPositions。
func TestObserve_MouseInteractionWithoutXY(t *testing.T) {
	bt := NewBehaviorTracker(nil, nil, "s-test")

	// IncrementalSnapshot(3) + source=2(MouseInteraction) 但 x/y 缺失
	bt.Observe(proto.EventPayload{
		Type: proto.EvRRWeb,
		TS:   1,
		RRWeb: &proto.RRWebEvent{
			Type: 3,
			Data: map[string]any{"source": float64(2)}, // 无 x/y
		},
	})

	bt.mu.Lock()
	defer bt.mu.Unlock()
	if len(bt.clickPositions) != 0 {
		t.Errorf("clickPositions: got %d entries, want 0(无效 x/y 不应记录)", len(bt.clickPositions))
	}
}

// TestCheckAndFlag_NoReasons 验证无满足启发式条件时不调用 FlagSession。
func TestCheckAndFlag_NoReasons(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	bt := NewBehaviorTracker(rdb, nil, "test-no-reasons-"+time.Now().Format("150405"))

	bt.CheckAndFlag(ctx)

	flagged, _, err := IsSessionFlagged(ctx, rdb, bt.sessionID)
	if err != nil {
		t.Fatalf("IsSessionFlagged: %v", err)
	}
	if flagged {
		t.Errorf("session was flagged without reasons; expected unflagged")
	}
}
