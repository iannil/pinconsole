// Package antiscrape 实现反爬虫：rate limit + UA 黑名单 + 行为分析 + fingerprint。
//
// 详见 docs/progress/2026-06-17-slice-1i-spec.md
package antiscrape

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitConfig 是固定窗口 rate limit 配置。
type RateLimitConfig struct {
	RequestsPerMin int           // 每分钟允许请求数
	Window         time.Duration // 窗口大小（默认 1 分钟）
}

// DefaultRateLimitConfig 默认配置。
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMin: 60,
		Window:         time.Minute,
	}
}

// RateLimitMiddleware 创建 Gin 中间件：per-IP 固定窗口 rate limit。
// 超限返回 429。
func RateLimitMiddleware(rdb *redis.Client, cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		windowKey := fmt.Sprintf("ratelimit:%s:%d", ip, time.Now().Unix()/int64(cfg.Window.Seconds()))

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		// INCR
		count, err := rdb.Incr(ctx, windowKey).Result()
		if err != nil {
			// Redis 不可用不阻塞请求
			c.Next()
			return
		}
		if count == 1 {
			// 首次请求设置 TTL
			_ = rdb.Expire(ctx, windowKey, cfg.Window+time.Second).Err()
		}

		// 设置响应头
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RequestsPerMin))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, cfg.RequestsPerMin-int(count))))

		if int(count) > cfg.RequestsPerMin {
			c.Header("Retry-After", fmt.Sprintf("%d", int(cfg.Window.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":     "rate_limited",
				"retry_after": int(cfg.Window.Seconds()),
			})
			return
		}
		c.Next()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// 内置爬虫/自动化 UA 片段（substring 匹配，case-insensitive）
//
// 注意:UA 黑名单是浅层防御,只能挡低水平爬虫。
// 现代自动化工具(Puppeteer/Playwright + headless=new)UA 与真实浏览器
// 几乎一致,无法仅靠 UA 识别。真正的纵深防御在 SDK hello fingerprint
// (canvas/WebGL) + BehaviorTracker 启发式 + rate limit。
var builtinBannedUAs = []string{
	// 命令行 HTTP 工具
	"curl/",
	"wget/",
	// 编程语言 HTTP 库
	"python-requests/",
	"aiohttp/",
	"httpx/",
	"Go-http-client/",
	"okhttp/",
	"Java/",
	"Apache-HttpClient/",
	"node-fetch/",
	"axios/",
	"got/",
	// 爬虫框架
	"scrapy",
	// 老式 headless 浏览器(UA 明确标记)
	"HeadlessChrome", // 仅对老式 headless Chrome 有效;现代 Playwright 用 Chrome/... 绕过
	"PhantomJS",
	// 通用爬虫标识
	"bot",
	"crawler",
	"spider",
}

// UABlockMiddleware 检查 User-Agent 是否匹配黑名单。
// 匹配返回 403。
func UABlockMiddleware(extraBanned []string) gin.HandlerFunc {
	all := append(builtinBannedUAs, extraBanned...)
	return func(c *gin.Context) {
		ua := c.Request.UserAgent()
		if ua == "" {
			// 空 UA 也拦截
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "empty_user_agent"})
			return
		}
		uaLower := strings.ToLower(ua)
		for _, pattern := range all {
			if strings.Contains(uaLower, strings.ToLower(pattern)) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":  "banned_user_agent",
					"detail": "automated requests are not allowed",
				})
				return
			}
		}
		c.Next()
	}
}

// FlagSession 在 Redis 中标记一个 session 为可疑（行为分析触发）。
func FlagSession(ctx context.Context, rdb *redis.Client, sessionID string, reason string) error {
	key := fmt.Sprintf("flagged:session:%s", sessionID)
	return rdb.Set(ctx, key, reason, 10*time.Minute).Err()
}

// IsSessionFlagged 检查 session 是否被标记。
func IsSessionFlagged(ctx context.Context, rdb *redis.Client, sessionID string) (bool, string, error) {
	key := fmt.Sprintf("flagged:session:%s", sessionID)
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, val, nil
}
