// Package config 负责从环境变量加载配置。
//
// 配置约定（见 docs/progress/2026-06-17-slice-1a-spec.md §环境变量）：
//   - 所有配置通过 env var 注入
//   - 开发期通过 .env 文件（不入库），生产通过容器环境
//   - struct tag 声明字段，caarlos0/env 解析
package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config 是 server 的全部配置。
type Config struct {
	ServerPort string `env:"SERVER_PORT" envDefault:"8080"`
	Env        string `env:"SERVER_ENV" envDefault:"dev"`
	LogLevel   string `env:"LOG_LEVEL" envDefault:"info"`

	Postgres  PostgresConfig
	Redis     RedisConfig
	MinIO     MinIOConfig
}

// PostgresConfig 是 PostgreSQL 连接配置。
type PostgresConfig struct {
	Host     string `env:"PG_HOST" envDefault:"localhost"`
	Port     string `env:"PG_PORT" envDefault:"5432"`
	User     string `env:"PG_USER" envDefault:"mm"`
	Password string `env:"PG_PASSWORD" envDefault:"mm_dev"`
	Database string `env:"PG_DB" envDefault:"marketing_monitor"`
}

// DSN 返回 PostgreSQL 连接串。
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.Database,
	)
}

// RedisConfig 是 Redis 连接配置。
type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
}

// MinIOConfig 是 MinIO 连接配置。
type MinIOConfig struct {
	Endpoint  string `env:"MINIO_ENDPOINT" envDefault:"localhost:9000"`
	AccessKey string `env:"MINIO_ACCESS_KEY" envDefault:"mm_dev"`
	SecretKey string `env:"MINIO_SECRET_KEY" envDefault:"mm_dev_secret"`
	Bucket    string `env:"MINIO_BUCKET" envDefault:"marketing-monitor"`
	UseSSL    bool   `env:"MINIO_USE_SSL" envDefault:"false"`
}

// Load 从环境变量加载 Config。
func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("解析环境变量失败: %w", err)
	}
	cfg.Env = strings.ToLower(cfg.Env)
	cfg.LogLevel = strings.ToLower(cfg.LogLevel)
	return &cfg, nil
}
