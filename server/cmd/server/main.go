// Package main 是 pinconsole 后端 server 的入口。
//
// v1 切片 1a：启动 HTTP server，注册 health/sdk/admin/landing 路由。
// v1 切片 1b：增加 hub、WS、recording、session REST 端点。
// 业务逻辑（认证、co-browsing）从切片 1e/1h 起加入。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/iannil/pinconsole/internal/api"
	"github.com/iannil/pinconsole/internal/config"
	"github.com/iannil/pinconsole/internal/hub"
	"github.com/iannil/pinconsole/internal/logging"
	"github.com/iannil/pinconsole/internal/recording"
	"github.com/iannil/pinconsole/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// 版本信息，构建时通过 -ldflags 注入。
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	logger := logging.NewLogger(cfg.LogLevel, cfg.Env)
	logger.Info("启动 pinconsole",
		"version", version,
		"commit", commit,
		"env", cfg.Env,
		"port", cfg.ServerPort,
	)

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// 初始化存储
	stores, err := storage.Connect(rootCtx, cfg, logger)
	if err != nil {
		logger.Error("存储连接失败", "error", err)
		os.Exit(1)
	}
	defer stores.Close()

	// 1k：启动时自动应用 migrations（嵌入二进制，无需外部 CLI）
	if err := runMigrations(rootCtx, stores.PG.Pool, logger); err != nil {
		logger.Error("migrations 应用失败，启动中止（fail-fast）", "error", err)
		os.Exit(1)
	}

	// 1h：启动时初始化默认 admin 用户（如果 users 表为空）
	if err := seedAdminUser(rootCtx, stores, cfg, logger); err != nil {
		logger.Warn("seed admin user failed", "error", err)
	}

	// 初始化 hub、stream、flusher、snapshot cache、GC worker
	h := hub.New(logger)
	stream := recording.NewStream(stores.Redis.Client, logger)
	snapshots := recording.NewSnapshotCache(stores.Redis)
	flusher := recording.NewFlusher(recording.DefaultConfig(), stream, stores, logger)
	go flusher.Start(rootCtx)
	defer flusher.Stop()

	// 1d：GC worker（每小时清理 > 30 天的 blob）
	gc := recording.NewGC(recording.DefaultGCConfig(), stores, logger)
	gc.Start(rootCtx)
	defer gc.Stop()

	// 路由
	// 1i:解析 BannedUAs
	bannedUAs := []string{}
	for _, ua := range strings.Split(cfg.BannedUAs, ",") {
		ua = strings.TrimSpace(ua)
		if ua != "" {
			bannedUAs = append(bannedUAs, ua)
		}
	}

	// 1o P1-5:解析 TrustedProxies
	trustedProxies := []string{}
	for _, p := range strings.Split(cfg.TrustedProxies, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			trustedProxies = append(trustedProxies, p)
		}
	}

	router := api.NewRouterWithOpts(api.Options{
		Logger:                 logger,
		Stores:                 stores,
		Hub:                    h,
		Stream:                 stream,
		Flusher:                flusher,
		Snapshots:              snapshots,
		NavigateAllowedDomains: cfg.NavigateAllowedDomains,
		Embedded:               embeddedAssets,
		Release:                isRelease(),
		Env:                    cfg.Env,
		RateLimitPerMin:        cfg.RateLimitPerMin,
		BannedUAs:              bannedUAs,
		TrustedProxies:         trustedProxies,
	})

	// 1o P1-6:WriteTimeout=0(coder/websocket 文档明确要求,否则所有 WS 每 30s 被踢)
	// ReadTimeout 也设 0,WS 长连接的读由 conn.Read(ctx) 控制
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server 异常退出", "error", err)
			os.Exit(1)
		}
	}()
	logger.Info("HTTP server 已监听", "addr", srv.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("收到退出信号，开始优雅关闭")

	rootCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("优雅关闭失败", "error", err)
	}
	logger.Info("已退出")
}

// seedAdminUser 在 users 表为空时创建默认 admin。
func seedAdminUser(ctx context.Context, stores *storage.Stores, cfg *config.Config, logger *slog.Logger) error {
	count, err := stores.PG.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	// 1k：bcrypt cost 从配置读（CLAUDE.md 要求 ≥ 12）
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), cfg.BCryptCost)
	if err != nil {
		return err
	}
	_, err = stores.PG.CreateUser(ctx, storage.DefaultTenantID, cfg.AdminEmail, string(hash), "Admin", "admin")
	if err != nil {
		return err
	}
	logger.Info("默认 admin 用户已创建", "email", cfg.AdminEmail, "bcrypt_cost", cfg.BCryptCost)
	return nil
}
