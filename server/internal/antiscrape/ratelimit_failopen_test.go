// 1ac 续集测试:Redis fail-open 行为(审计 T0-1i-1)。
//
// 1i 的 rate limit 中间件必须在 Redis 不可用时 fail-open(c.Next() 放行),
// 否则 Redis 故障 = 全站不可用。
//
// 注:已有 ratelimit_test.go 覆盖正常路径(阈值/ban UA/BehaviorTracker),
// 本测试专门覆盖 fail-open 接线。
package antiscrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// TestRateLimitMiddleware_RedisUnavailable_FailOpen — T0-1i-1:
// Redis 连接失败/超时时,请求应放行(fail-open),不返回 5xx/429。
func TestRateLimitMiddleware_RedisUnavailable_FailOpen(t *testing.T) {
	if testing.Short() {
		t.Skip("需要可关闭的 redis 连接")
	}
	// 连到不存在的 port,模拟 Redis 故障
	rdb := redis.NewClient(&redis.Options{
		Addr:        "localhost:1", // 不可达
		DialTimeout: 100 * time.Millisecond,
	})
	defer rdb.Close()

	// 验证 redis 真不可达
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err == nil {
		t.Skip("localhost:1 居然可达,跳过")
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	called := false
	r.Use(RateLimitMiddleware(rdb, RateLimitConfig{
		RequestsPerMin: 60,
		Window:         time.Minute,
	}))
	r.GET("/", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Errorf("next handler NOT called — fail-open 破坏(Redis 故障应放行)")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status=%d want 200 (fail-open should pass-through), body=%s",
			w.Code, w.Body.String())
	}
	// 不应是 429(超限)/5xx(故障)
	if w.Code == http.StatusTooManyRequests || w.Code >= 500 {
		t.Errorf("fail-open 不应返回 429/5xx, got %d", w.Code)
	}
}

// TestRateLimitMiddleware_RedisUnavailable_NoPanic — T0-1i-1 边界:
// Redis 故障时中间件不应 panic(否则 Gin 会 500)。
func TestRateLimitMiddleware_RedisUnavailable_NoPanic(t *testing.T) {
	if testing.Short() {
		t.Skip("需要可关闭的 redis 连接")
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:        "localhost:1",
		DialTimeout: 100 * time.Millisecond,
	})
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitMiddleware(rdb, RateLimitConfig{
		RequestsPerMin: 1,
		Window:         time.Minute,
	}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	// 跑 10 次,确保无 panic
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		// 不应 panic
		r.ServeHTTP(w, req)
		if w.Code == 0 {
			t.Errorf("iter %d: status=0 (handler panic)", i)
		}
	}
}
