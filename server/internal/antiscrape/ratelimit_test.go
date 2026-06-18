package antiscrape

import (
	"context"
	cryptoRand "crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// skipIfNoRedis 跳过测试如果本地 Redis 不可用(允许 CI 在无 infra 时跳过)。
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

// TestRateLimitMiddleware_Triggers429 验证超过配额后真触发 429。
// 这是 1i 的关键负向测试:rate limit 必须真生效,不是只"注册了 middleware"。
//
// 1n P0-9 修复:
//   - unique IP 用 crypto/rand 生成完整 /8 前缀(原 %200+1 在同包其他测试后撞同 bucket)
//   - 测试 setup 加 FlushDB 清理,确保每次跑测试时 Redis 状态干净
func TestRateLimitMiddleware_Triggers429(t *testing.T) {
	rdb := skipIfNoRedis(t)
	defer rdb.Close()

	// 1n:FlushDB 确保干净状态(防 flaky)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("FlushDB failed: %v", err)
	}

	// 1n:用 crypto/rand 生成真随机 IP(避免 %200 范围太小撞 bucket)
	b := make([]byte, 2)
	if _, err := cryptoRand.Read(b); err != nil {
		t.Fatalf("crypto/rand failed: %v", err)
	}
	octet3 := int(b[0]) | 1 // 1-255
	octet4 := int(b[1]) | 1 // 1-255
	uniqueIP := fmt.Sprintf("10.99.%d.%d", octet3, octet4)

	cfg := RateLimitConfig{RequestsPerMin: 5, Window: time.Minute}
	mw := RateLimitMiddleware(rdb, cfg)

	gin.SetMode(gin.TestMode)
	hitCount := 0
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request.RemoteAddr = uniqueIP + ":12345"
		mw(c)
	})
	router.GET("/test", func(c *gin.Context) {
		hitCount++
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 5 次应通过
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 Test")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
	if hitCount != 5 {
		t.Fatalf("expected 5 successful hits, got %d", hitCount)
	}

	// 第 6 次应被限流
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Test")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("request 6: expected 429, got %d (body=%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "rate_limited") {
		t.Fatalf("expected body to contain 'rate_limited', got %s", w.Body.String())
	}
}

// TestUABlockMiddleware_BansListedUA 验证 UA 黑名单对各种 bot UA 生效。
func TestUABlockMiddleware_BansListedUA(t *testing.T) {
	banned := []string{"curl/", "python-requests/", "scrapy", "HeadlessChrome"}
	mw := UABlockMiddleware(banned)

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		ua        string
		expect403 bool
	}{
		{"curl/8.0", true},
		{"python-requests/2.31", true},
		{"scrapy-2.0", true},
		{"Mozilla/5.0 HeadlessChrome/130", true},
		{"Mozilla/5.0 (Windows NT 10.0) Chrome/149.0 Safari/537.36", false}, // 现代桌面 Chrome
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0) Mobile Safari", false},   // 移动 Safari
		{"", true}, // 空 UA 也拦截
	}

	for _, tc := range testCases {
		t.Run(tc.ua, func(t *testing.T) {
			router := gin.New()
			router.Use(mw)
			router.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tc.ua)
			router.ServeHTTP(w, req)

			if tc.expect403 && w.Code != http.StatusForbidden {
				t.Errorf("UA=%q: expected 403, got %d", tc.ua, w.Code)
			}
			if !tc.expect403 && w.Code != http.StatusOK {
				t.Errorf("UA=%q: expected 200, got %d", tc.ua, w.Code)
			}
		})
	}
}
