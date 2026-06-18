// Package api 注册 HTTP 路由与 handler。
//
// 切片 1a + 1b 路由：
//
//	GET  /healthz              liveness 探针
//	GET  /readyz               readiness 探针（PG / Redis / MinIO）
//	POST /api/session/init     SDK 签发 session_id（1b 新增）
//	GET  /api/sessions         admin 拉取活跃会话列表（1b 新增）
//	GET  /ws/visitor           访客 SDK WebSocket（1b 新增）
//	GET  /ws/operator          运营 admin WebSocket（1b 新增）
//	GET  /                     访客落地页（landing/demo）
//	GET  /sdk.js               访客 SDK bundle
//	GET  /admin                运营 SPA 入口
//	GET  /admin/assets/*       运营 SPA 静态资源
//	GET  /admin/<spa-route>    SPA fallback
package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iannil/marketing-monitor/internal/antiscrape"
	"github.com/iannil/marketing-monitor/internal/hub"
	"github.com/iannil/marketing-monitor/internal/logging"
	"github.com/iannil/marketing-monitor/internal/recording"
	"github.com/iannil/marketing-monitor/internal/storage"
)

// Options 是 NewRouter 的可选参数集合。
type Options struct {
	Logger                 *slog.Logger
	Stores                 *storage.Stores
	Hub                    *hub.Hub
	Stream                 *recording.Stream
	Flusher                *recording.Flusher
	Snapshots              *recording.SnapshotCache
	NavigateAllowedDomains string
	Embedded               fs.FS
	Release                bool
	Env                    string
	// 1i
	RateLimitPerMin int
	BannedUAs       []string
	// 1o P1-5:TrustedProxies 列表(nil/empty = 不信任任何反代)
	TrustedProxies []string
}

// NewRouterWithOpts 注册全部路由并返回 gin.Engine。
func NewRouterWithOpts(opts Options) *gin.Engine {
	if opts.Release {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(logging.TraceMiddleware(opts.Logger))

	// 1o P1-5:TrustedProxies 配置
	// 不信任任何反代 → ClientIP() 返回 RemoteAddr(防 X-Forwarded-For 伪造绕过 rate limit)
	// 部署在 nginx/caddy 后时通过 TRUSTED_PROXIES env 显式配置(逗号分隔 CIDR)
	if err := r.SetTrustedProxies(opts.TrustedProxies); err != nil {
		opts.Logger.Warn("SetTrustedProxies failed, defaulting to no trust", "error", err)
	}

	// 1i：反爬虫中间件（dev 模式仅 UA 黑名单，跳过 rate limit 便于 e2e）
	devMode := opts.Env != "prod"
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/healthz" || path == "/readyz" {
			c.Next()
			return
		}
		// UA 黑名单（始终生效）
		uaMW := antiscrape.UABlockMiddleware(opts.BannedUAs)
		uaMW(c)
		if c.IsAborted() {
			return
		}
		// Rate limit（仅 prod 模式生效）
		if !devMode {
			rlMW := antiscrape.RateLimitMiddleware(opts.Stores.Redis.Client, antiscrape.RateLimitConfig{
				RequestsPerMin: opts.RateLimitPerMin,
				Window:         time.Minute,
			})
			rlMW(c)
		}
	})

	// 健康检查（公开）
	r.GET("/healthz", healthLive)
	r.GET("/readyz", healthReady(opts.Stores))

	// 1h：认证（公开）
	authH := NewAuthHandler(opts.Stores, opts.Logger, opts.Env == "prod")
	authH.Register(r)

	// 1b：访客 REST session API（公开，SDK 用）
	sessionH := NewSessionHandler(opts.Stores, opts.Hub, opts.Logger)
	sessionH.Register(r)

	// 1l：隐私合规 consent 端点(公开,SDK 用)
	privacyH := NewPrivacyHandler(opts.Stores, opts.Logger)
	privacyH.RegisterPublic(r)

	// 1b/1c/1d：访客 WebSocket（公开）
	wsH := NewWSHandler(opts.Hub, opts.Stores, opts.Stream, opts.Flusher, opts.Snapshots, opts.Logger)
	wsH.Register(r)

	// 1h：运营受保护端点（SERVER_ENV=dev 时自动绕过，便于 e2e）
	authMW := AuthMiddleware(opts.Stores.Redis.Get, opts.Env != "prod")
	protected := r.Group("/", authMW)
	{
		// 1d：replay REST API
		replayH := NewReplayHandler(opts.Stores, opts.Logger)
		replayH.Register(protected)

		// 1e/1f：co-browsing command REST API
		commandH := NewCommandHandler(opts.Stores, opts.Hub, opts.NavigateAllowedDomains, opts.Logger)
		commandH.Register(protected)

		// 1g：聊天消息 REST API
		chatH := NewChatHandler(opts.Stores, opts.Hub, opts.Logger)
		chatH.Register(protected)

		// 1h：claim/release
		claimH := NewClaimHandler(opts.Stores, opts.Logger)
		claimH.Register(protected)

		// 1l:erasure(GDPR Art.17 被遗忘权,admin only)
		protected.DELETE("/api/privacy/visitor/:fingerprint", privacyH.DeleteVisitor)
	}

	// 静态资源（landing / sdk / admin）
	handler := newStaticHandler(opts.Embedded, opts.Release)
	handler.Register(r)

	// NoRoute：dev 模式与 SPA fallback 都走这里
	r.NoRoute(handler.NoRoute)

	return r
}

// staticHandler 封装 dev/prod 两套静态资源逻辑。
type staticHandler struct {
	embedded      fs.FS
	release       bool
	adminFS       fs.FS
	adminAssetsFS fs.FS
	sdkFS         fs.FS
	sdkBytes      []byte
	landingFS     fs.FS
	adminIndex    []byte
	landingRoot   []byte
}

func newStaticHandler(embedded fs.FS, release bool) *staticHandler {
	h := &staticHandler{embedded: embedded, release: release}
	if !release {
		return h
	}

	if sub, err := fs.Sub(embedded, "embedded/admin"); err == nil {
		h.adminFS = sub
		if assetsSub, err := fs.Sub(sub, "assets"); err == nil {
			h.adminAssetsFS = assetsSub
		} else {
			slog.Warn("admin/assets sub failed", "error", err)
		}
		if b, err := fs.ReadFile(sub, "index.html"); err == nil {
			h.adminIndex = b
		} else {
			slog.Warn("admin/index.html read failed", "error", err)
		}
	} else {
		slog.Warn("admin sub failed", "error", err)
	}

	// SDK：直接读取 bytes（不走 fs.Sub，避免奇怪问题）
	if b, err := fs.ReadFile(embedded, "embedded/sdk/sdk.js"); err == nil {
		h.sdkBytes = b
		slog.Info("sdk sdk.js read ok", "size", len(b))
	} else {
		slog.Warn("sdk sdk.js read failed", "error", err)
	}

	if sub, err := fs.Sub(embedded, "embedded/landing"); err == nil {
		h.landingFS = sub
		if b, err := fs.ReadFile(sub, "demo/index.html"); err == nil {
			h.landingRoot = b
			slog.Info("landing demo/index.html read ok", "size", len(b))
		} else {
			slog.Warn("landing demo/index.html read failed", "error", err)
		}
	} else {
		slog.Warn("landing sub failed", "error", err)
	}
	return h
}

// Register 注册显式路由（catch-all 之外的）。
func (h *staticHandler) Register(r *gin.Engine) {
	if !h.release {
		r.GET("/", h.devHint("/", "访客落地页（dev 由 Vite playground 提供）"))
		r.GET("/sdk.js", h.devHint("/sdk.js", "http://localhost:5174/sdk.js"))
		return
	}

	// SDK：直接用预读的 bytes
	if len(h.sdkBytes) > 0 {
		sdkBytes := h.sdkBytes
		r.GET("/sdk.js", func(c *gin.Context) {
			c.Header("Cache-Control", "public, max-age=300")
			c.Data(http.StatusOK, "application/javascript; charset=utf-8", sdkBytes)
		})
		slog.Info("registered /sdk.js", "size", len(sdkBytes))
	} else {
		slog.Warn("sdkBytes empty, /sdk.js not registered")
	}

	if h.adminAssetsFS != nil {
		r.StaticFS("/admin/assets", http.FS(h.adminAssetsFS))
	}

	if h.adminIndex != nil {
		idx := h.adminIndex
		r.GET("/admin", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", idx)
		})
	}

	if h.landingRoot != nil {
		landing := h.landingRoot
		r.GET("/", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", landing)
		})
		slog.Info("registered /", "size", len(landing))
	} else {
		slog.Warn("landingRoot nil, / not registered")
	}
}

// NoRoute 处理 SPA fallback（如 /admin/visitors）与 dev 提示。
func (h *staticHandler) NoRoute(c *gin.Context) {
	path := c.Request.URL.Path

	if !h.release {
		h.devHint(path, "dev mode: Vite dev server")(c)
		return
	}

	if strings.HasPrefix(path, "/admin/assets/") {
		c.JSON(http.StatusNotFound, gin.H{
			"error":    "asset_not_found",
			"path":     path,
			"trace_id": logging.TraceID(c.Request.Context()),
		})
		return
	}

	if strings.HasPrefix(path, "/admin") && h.adminIndex != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", h.adminIndex)
		return
	}

	if strings.HasPrefix(path, "/landing") && h.landingFS != nil {
		c.FileFromFS(strings.TrimPrefix(path, "/landing"), http.FS(h.landingFS))
		return
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error":    "not_found",
		"path":     path,
		"trace_id": logging.TraceID(c.Request.Context()),
	})
}

func (h *staticHandler) devHint(path, msg string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"hint":     msg,
			"path":     path,
			"trace_id": logging.TraceID(c.Request.Context()),
		})
	}
}
