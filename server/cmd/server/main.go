// Package main 是 marketing-monitor 后端 server 的入口。
//
// v1 切片 1a：仅启动 HTTP server，注册 health/sdk/admin/landing 路由。
// 业务逻辑（WebSocket hub、认证、录制）从切片 1b 起逐步加入。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iannil/marketing-monitor/internal/api"
	"github.com/iannil/marketing-monitor/internal/config"
	"github.com/iannil/marketing-monitor/internal/logging"
	"github.com/iannil/marketing-monitor/internal/storage"
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
	logger.Info("启动 marketing-monitor",
		"version", version,
		"commit", commit,
		"env", cfg.Env,
		"port", cfg.ServerPort,
	)

	// 初始化存储（连接占位，1b 起真正使用）
	stores, err := storage.Connect(context.Background(), cfg, logger)
	if err != nil {
		logger.Error("存储连接失败", "error", err)
		os.Exit(1)
	}
	defer stores.Close()

	// 路由注册
	router := api.NewRouter(logger, stores, embeddedAssets, isRelease())

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 优雅退出
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("优雅关闭失败", "error", err)
	}
	logger.Info("已退出")
}
