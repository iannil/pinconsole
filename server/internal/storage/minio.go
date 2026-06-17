package storage

import (
	"bytes"
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

// PutBytes 上传字节数据到指定 object key。
// contentType 默认 application/octet-stream。
func (m *MinIO) PutBytes(ctx context.Context, objectKey string, data []byte) error {
	_, err := m.Client.PutObject(ctx, m.Bucket, objectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("put object %s: %w", objectKey, err)
	}
	return nil
}

// GetBytes 下载指定 object key 的全部内容。
func (m *MinIO) GetBytes(ctx context.Context, objectKey string) ([]byte, error) {
	obj, err := m.Client.GetObject(ctx, m.Bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", objectKey, err)
	}
	defer obj.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(obj); err != nil {
		return nil, fmt.Errorf("read object %s: %w", objectKey, err)
	}
	return buf.Bytes(), nil
}

// Close 占位（minio-go 无显式 close）。
func (m *MinIO) Close() {}
