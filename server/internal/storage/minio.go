package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/iannil/marketing-monitor/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIO 封装 MinIO 客户端。
type MinIO struct {
	Client *minio.Client
	Bucket string
	logger *slog.Logger
}

// ConnectMinIO 建立 MinIO 客户端、验证、确保 bucket 存在。
func ConnectMinIO(ctx context.Context, cfg config.MinIOConfig, logger *slog.Logger) (*MinIO, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("new client: %w", err)
	}

	// 确保 bucket 存在（不存在则创建）
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket exists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
		logger.Info("minio bucket 已创建", "bucket", cfg.Bucket)
	}

	logger.Info("minio 已连接", "endpoint", cfg.Endpoint, "bucket", cfg.Bucket)
	return &MinIO{Client: client, Bucket: cfg.Bucket, logger: logger}, nil
}

// Ping 验证连接（通过列出 buckets）。
func (m *MinIO) Ping(ctx context.Context) error {
	_, err := m.Client.ListBuckets(ctx)
	return err
}

// Close 占位（minio-go 无显式 close）。
func (m *MinIO) Close() {}
