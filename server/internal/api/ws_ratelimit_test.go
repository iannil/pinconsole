// Package api:visitor WS rate limit 单测(切片 1y P1-4)。
//
// 5 测试覆盖 checkWSRateLimit 函数:
// 1. 正常流量(<阈值)允许
// 2. 超过 envelope 数 → 拒绝
// 3. 超过 bytes → 拒绝
// 4. 不同 session 独立
// 5. Redis 故障 fail-open
//
// 测试用真 Redis(skipIfNoRedis 模式),与 antiscrape/ratelimit_test.go 一致。

package api

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// skipIfNoRedis 跳过测试如果本地 Redis 不可用。
// 与 antiscrape/ratelimit_test.go 同模式。
func skipIfNoRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DialTimeout: 500 * time.Millisecond})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	return rdb
}

// flushRateLimitKeys 清掉指定 session 的 rate limit key。
func flushRateLimitKeys(t *testing.T, rdb *redis.Client, sessionID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	rdb.Del(ctx, "ws:rate:count:"+sessionID, "ws:rate:bytes:"+sessionID)
}

// TestCheckWSRateLimit_NormalTrafficAllows 验证正常流量(<阈值)放行。
func TestCheckWSRateLimit_NormalTrafficAllows(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()
	const sid = "test-normal-1y"
	flushRateLimitKeys(t, rdb, sid)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 发 100 个小 envelope(< 500 阈值)
	for i := 0; i < 100; i++ {
		allowed, _, err := checkWSRateLimit(ctx, rdb, sid, 1024)
		if err != nil {
			t.Fatalf("attempt %d: unexpected error: %v", i, err)
		}
		if !allowed {
			t.Fatalf("attempt %d: should be allowed (under threshold)", i)
		}
	}
}

// TestCheckWSRateLimit_OverMsgCount 验证超过 envelope 数被拒。
func TestCheckWSRateLimit_OverMsgCount(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()
	const sid = "test-over-msgs-1y"
	flushRateLimitKeys(t, rdb, sid)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 发 wsRateLimitMaxMsgs(500) 个 — 第 501 个应被拒
	for i := 0; i < wsRateLimitMaxMsgs; i++ {
		allowed, _, err := checkWSRateLimit(ctx, rdb, sid, 100)
		if err != nil || !allowed {
			t.Fatalf("attempt %d: should be allowed, err=%v allowed=%v", i, err, allowed)
		}
	}
	// 第 501 个超阈值
	allowed, reason, err := checkWSRateLimit(ctx, rdb, sid, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("attempt 501: should be rejected (over msg threshold)")
	}
	if reason == "" {
		t.Fatal("reason should be non-empty when rejected")
	}
}

// TestCheckWSRateLimit_OverBytes 验证超过 bytes 被拒。
func TestCheckWSRateLimit_OverBytes(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()
	const sid = "test-over-bytes-1y"
	flushRateLimitKeys(t, rdb, sid)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 发 5 个 10MB envelope = 50MiB(达到阈值边缘,还允许)
	for i := 0; i < 5; i++ {
		allowed, _, err := checkWSRateLimit(ctx, rdb, sid, 10*1024*1024)
		if err != nil {
			t.Fatalf("attempt %d: unexpected error: %v", i, err)
		}
		if !allowed {
			t.Fatalf("attempt %d: should be allowed at 5*10MiB = 50MiB (at threshold)", i)
		}
	}
	// 第 6 个 10MB envelope 总和超 50MiB
	allowed, reason, err := checkWSRateLimit(ctx, rdb, sid, 10*1024*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("should be rejected (over byte threshold)")
	}
	if reason == "" {
		t.Fatal("reason should be non-empty")
	}
}

// TestCheckWSRateLimit_SessionIsolation 验证不同 session 独立计数。
func TestCheckWSRateLimit_SessionIsolation(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()
	const sid1 = "test-iso-1y-a"
	const sid2 = "test-iso-1y-b"
	flushRateLimitKeys(t, rdb, sid1)
	flushRateLimitKeys(t, rdb, sid2)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// sid1 发 200,sid2 发 200,各自都未到阈值
	for i := 0; i < 200; i++ {
		if allowed, _, err := checkWSRateLimit(ctx, rdb, sid1, 100); err != nil || !allowed {
			t.Fatalf("sid1 attempt %d: err=%v allowed=%v", i, err, allowed)
		}
		if allowed, _, err := checkWSRateLimit(ctx, rdb, sid2, 100); err != nil || !allowed {
			t.Fatalf("sid2 attempt %d: err=%v allowed=%v", i, err, allowed)
		}
	}

	// sid1 继续发到 500(应仍允许),sid2 仍是 200
	for i := 0; i < 300; i++ {
		if allowed, _, err := checkWSRateLimit(ctx, rdb, sid1, 100); err != nil || !allowed {
			t.Fatalf("sid1 attempt %d: should be allowed up to 500", 200+i)
		}
	}

	// sid1 第 501 应被拒
	allowed, _, _ := checkWSRateLimit(ctx, rdb, sid1, 100)
	if allowed {
		t.Fatal("sid1 should be rejected at 501")
	}

	// sid2 仍允许(独立计数)
	allowed2, _, err := checkWSRateLimit(ctx, rdb, sid2, 100)
	if err != nil || !allowed2 {
		t.Fatalf("sid2 should still be allowed (independent counter), err=%v allowed=%v", err, allowed2)
	}
}

// TestCheckWSRateLimit_RedisFailureFailOpen 验证 Redis 故障时 fail-open。
// 关闭的 client 模拟 Redis 故障。
func TestCheckWSRateLimit_RedisFailureFailOpen(t *testing.T) {
	// 创建 client 立即关闭,模拟故障
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:1", DialTimeout: 100 * time.Millisecond})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	allowed, reason, err := checkWSRateLimit(ctx, rdb, "test-failopen-1y", 1024)
	if err == nil {
		t.Fatal("expected error from closed redis, got nil")
	}
	if !allowed {
		t.Fatal("fail-open: should be allowed even with redis error")
	}
	if reason != "" {
		t.Fatal("fail-open: reason should be empty when allowed")
	}
}
