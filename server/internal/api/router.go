// Package api 注册 HTTP 路由与 handler。
//
// 切片 1a 路由：
//
//	GET /healthz         liveness 探针
//	GET /readyz          readiness 探针（检查 PG/Redis/MinIO）
//	GET /                访客落地页（landing/demo）
//	GET /sdk.js          访客 SDK bundle
//	GET /admin           运营 SPA 入口
//	GET /admin/assets/*  运营 SPA 静态资源（js/css/图片）
//	GET /admin/<path>    SPA fallback，返回 index.html（前端路由由 Vue Router 处理）
package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iannil/marketing-monitor/internal/logging"
	"github.com/iannil/marketing-monitor/internal/storage"
)

// NewRouter 注册全部路由并返回 gin.Engine。
func NewRouter(logger *slog.Logger, stores *storage.Stores, embedded fs.FS, release bool) *gin.Engine {
	if release {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(logging.TraceMiddleware(logger))

	// 健康检查
	r.GET("/healthz", healthLive)
	r.GET("/readyz", healthReady(stores))

	// 静态资源
	handler := newStaticHandler(embedded, release)
	handler.Register(r)

	// NoRoute：dev 模式与 SPA fallback 都走这里
	r.NoRoute(handler.NoRoute)

	return r
}

// staticHandler 封装 dev/prod 两套静态资源逻辑。
type staticHandler struct {
	embedded    fs.FS
	release     bool
	adminFS     fs.FS
	sdkFS       fs.FS
	landingFS   fs.FS
	adminIndex  []byte
	landingRoot []byte
}

func newStaticHandler(embedded fs.FS, release bool) *staticHandler {
	h := &staticHandler{embedded: embedded, release: release}
	if !release {
		return h
	}

	// release: 准备各子 FS 与入口 HTML 字节
	if sub, err := fs.Sub(embedded, "embedded/admin"); err == nil {
		h.adminFS = sub
		if b, err := fs.ReadFile(sub, "index.html"); err == nil {
			h.adminIndex = b
		}
	}
	if sub, err := fs.Sub(embedded, "embedded/sdk"); err == nil {
		h.sdkFS = sub
	}
	if sub, err := fs.Sub(embedded, "embedded/landing"); err == nil {
		h.landingFS = sub
		if b, err := fs.ReadFile(sub, "demo/index.html"); err == nil {
			h.landingRoot = b
		}
	}
	return h
}

// Register 注册显式路由（catch-all 之外的）。
func (h *staticHandler) Register(r *gin.Engine) {
	if !h.release {
		r.GET("/", h.devHint("/", "访客落地页（dev 由 Vite playground 提供，见 visitor-sdk/playground/）"))
		r.GET("/sdk.js", h.devHint("/sdk.js", "http://localhost:5174/sdk.js"))
		return
	}

	// SDK
	if h.sdkFS != nil {
		r.GET("/sdk.js", func(c *gin.Context) {
			c.Header("Cache-Control", "public, max-age=300")
			c.FileFromFS("sdk.js", http.FS(h.sdkFS))
		})
	}

	// Admin 静态资源（必须用具体前缀，避免 catch-all 冲突）
	if h.adminFS != nil {
		r.StaticFS("/admin/assets", http.FS(h.adminFS))
	}

	// Admin SPA 入口
	if h.adminIndex != nil {
		r.GET("/admin", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", h.adminIndex)
		})
	}

	// Landing demo 入口
	if h.landingRoot != nil {
		r.GET("/", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", h.landingRoot)
		})
	}
}

// NoRoute 处理 SPA fallback（如 /admin/visitors）与 dev 提示。
func (h *staticHandler) NoRoute(c *gin.Context) {
	path := c.Request.URL.Path

	if !h.release {
		h.devHint(path, "dev mode: Vite dev server")(c)
		return
	}

	// /admin/assets/* 是显式静态资源；缺失时不应走 SPA fallback，直接 404
	if strings.HasPrefix(path, "/admin/assets/") {
		c.JSON(http.StatusNotFound, gin.H{
			"error":    "asset_not_found",
			"path":     path,
			"trace_id": logging.TraceID(c.Request.Context()),
		})
		return
	}

	// Admin SPA fallback：/admin/* 返回 index.html（前端路由由 Vue Router 处理）
	if strings.HasPrefix(path, "/admin") && h.adminIndex != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", h.adminIndex)
		return
	}

	// Landing demo 子路径
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
