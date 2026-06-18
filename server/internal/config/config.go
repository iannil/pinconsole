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

// 默认值集中常量，便于查阅与修改。
const (
	defaultEnv           = "prod" // 1k fail-secure：默认 prod，dev 必须显式 opt-in
	defaultBCryptCost    = 12     // 1k：CLAUDE.md 要求 ≥ 12
	defaultPGSSLMode     = "prefer"
	defaultAdminPassword = "" // 1k：无默认值，缺失时 Load 校验失败
)

// Config 是 server 的全部配置。
type Config struct {
	ServerPort string `env:"SERVER_PORT" envDefault:"8080"`
	Env        string `env:"SERVER_ENV" envDefault:"prod"`
	LogLevel   string `env:"LOG_LEVEL" envDefault:"info"`
	// 1f：navigate 命令允许的额外域名（逗号分隔）
	NavigateAllowedDomains string `env:"NAVIGATE_ALLOWED_DOMAINS" envDefault:""`
	// 1h：初始 admin 用户（1k：无默认密码，启动校验）
	AdminEmail    string `env:"ADMIN_EMAIL" envDefault:"admin@marketing-monitor.local"`
	AdminPassword string `env:"ADMIN_PASSWORD"`
	// 1k：bcrypt cost（项目要求 ≥ 12）
	BCryptCost int `env:"BCRYPT_COST" envDefault:"12"`
	// 1i：反爬虫
	RateLimitPerMin int    `env:"RATE_LIMIT_PER_MIN" envDefault:"60"`
	BannedUAs       string `env:"BANNED_UAS" envDefault:""`

	Postgres PostgresConfig
	Redis    RedisConfig
	MinIO    MinIOConfig
}

// PostgresConfig 是 PostgreSQL 连接配置。
type PostgresConfig struct {
	Host     string `env:"PG_HOST" envDefault:"localhost"`
	Port     string `env:"PG_PORT" envDefault:"5432"`
	User     string `env:"PG_USER" envDefault:"mm"`
	Password string `env:"PG_PASSWORD" envDefault:""`
	Database string `env:"PG_DB" envDefault:"marketing_monitor"`
	SSLMode  string `env:"PG_SSLMODE" envDefault:"prefer"`
}

// DSN 返回 PostgreSQL 连接串。
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, p.SSLMode,
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
	AccessKey string `env:"MINIO_ACCESS_KEY" envDefault:""`
	SecretKey string `env:"MINIO_SECRET_KEY" envDefault:""`
	Bucket    string `env:"MINIO_BUCKET" envDefault:"marketing-monitor"`
	UseSSL    bool   `env:"MINIO_USE_SSL" envDefault:"false"`
}

// Load 从环境变量加载 Config 并做 fail-secure 校验。
//
// 校验规则（1k silent defaults 修复）：
//   - AdminPassword 必须非空（无默认值）
//   - BCryptCost 必须 ≥ 12（CLAUDE.md 要求）
//   - prod 模式下 AdminPassword 不能等于历史默认 "changeme123"
//   - PG.Password / MinIO.AccessKey / MinIO.SecretKey 在 prod 模式下必须非默认弱凭证
func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("解析环境变量失败: %w", err)
	}
	cfg.Env = strings.ToLower(cfg.Env)
	cfg.LogLevel = strings.ToLower(cfg.LogLevel)

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// validate 执行 fail-secure 校验。
func (cfg *Config) validate() error {
	if cfg.AdminPassword == "" {
		return fmt.Errorf("ADMIN_PASSWORD 未设置：必须显式提供（无默认值，1k fail-secure）")
	}
	if cfg.AdminPassword == "changeme123" && cfg.Env == "prod" {
		return fmt.Errorf("ADMIN_PASSWORD=changeme123 在 prod 模式下不允许：请改为强密码（1k fail-secure）")
	}
	if cfg.BCryptCost < 12 {
		return fmt.Errorf("BCRYPT_COST=%d 不合规：必须 ≥ 12（CLAUDE.md 要求，1k 修复）", cfg.BCryptCost)
	}
	if cfg.Env == "prod" {
		if cfg.Postgres.Password == "" || cfg.Postgres.Password == "mm_dev" {
			return fmt.Errorf("PG_PASSWORD 在 prod 模式下必须为非空且非 dev 弱凭证（1k fail-secure）")
		}
		if cfg.MinIO.AccessKey == "" || cfg.MinIO.AccessKey == "mm_dev" {
			return fmt.Errorf("MINIO_ACCESS_KEY 在 prod 模式下必须为非空且非 dev 弱凭证（1k fail-secure）")
		}
		if cfg.MinIO.SecretKey == "" || cfg.MinIO.SecretKey == "mm_dev_secret" {
			return fmt.Errorf("MINIO_SECRET_KEY 在 prod 模式下必须为非空且非 dev 弱凭证（1k fail-secure）")
		}
	}
	return nil
}
