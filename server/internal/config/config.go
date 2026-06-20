// Package config 负责从环境变量加载配置。
//
// 配置约定（见 docs/progress/2026-06-17-slice-1a-spec.md §环境变量）：
//   - 所有配置通过 env var 注入
//   - 开发期通过 .env 文件（不入库），生产通过容器环境
//   - struct tag 声明字段，caarlos0/env 解析
package config

import (
	"fmt"
	"net"
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

// 1z:允许的 Env 值白名单。typo 防御(如 "production" 会被 router 当 dev mode 触发 bypass)。
var allowedEnvs = map[string]struct{}{
	"prod": {},
	"dev":  {},
	"test": {},
}

// 1z:prod 模式下禁止的 PG sslmode(明文凭证走网络)。
// 注:envDefault="prefer" 保证 caarlos0/env 在 PG_SSLMODE 未设或显式空串时填 "prefer",
// 因此 "" 进不来此 map,只需列 disable。
var disallowedProdSSLMode = map[string]struct{}{
	"disable": {}, // 明文
}

// 1z:判定 endpoint 是否视为本地(不强制 TLS)。
// docker-compose 内部网络中 MinIO/PG 容器间通信不开 TLS 是合理的。
func isLocalEndpoint(endpoint string) bool {
	e := strings.ToLower(strings.TrimSpace(endpoint))
	// 去端口
	if i := strings.LastIndex(e, ":"); i > 0 {
		e = e[:i]
	}
	return e == "localhost" || e == "127.0.0.1" || e == "::1" || e == "postgres" || e == "minio" || e == "redis"
}

// Config 是 server 的全部配置。
type Config struct {
	ServerPort string `env:"SERVER_PORT" envDefault:"8080"`
	Env        string `env:"SERVER_ENV" envDefault:"prod"`
	LogLevel   string `env:"LOG_LEVEL" envDefault:"info"`
	// 1f：navigate 命令允许的额外域名（逗号分隔）
	NavigateAllowedDomains string `env:"NAVIGATE_ALLOWED_DOMAINS" envDefault:""`
	// 1h：初始 admin 用户（1k：无默认密码，启动校验）
	AdminEmail    string `env:"ADMIN_EMAIL" envDefault:"admin@pinconsole.local"`
	AdminPassword string `env:"ADMIN_PASSWORD"`
	// 1k：bcrypt cost（项目要求 ≥ 12）
	BCryptCost int `env:"BCRYPT_COST" envDefault:"12"`
	// 1i：反爬虫
	RateLimitPerMin int    `env:"RATE_LIMIT_PER_MIN" envDefault:"60"`
	BannedUAs       string `env:"BANNED_UAS" envDefault:""`

	Postgres PostgresConfig
	Redis    RedisConfig
	MinIO    MinIOConfig

	// 1o P1-5:TrustedProxies CIDR 列表(逗号分隔);prod 部署在 nginx/caddy 后必填
	TrustedProxies string `env:"TRUSTED_PROXIES" envDefault:""`
	// 1ab P1-5:显式声明部署拓扑。true = 部署在反代后(TrustedProxies 必填)。
	// false/默认 = 直接暴露(TrustedProxies 应为空,XFF 被忽略防伪造)。
	BehindReverseProxy bool `env:"BEHIND_REVERSE_PROXY" envDefault:"false"`
}

// PostgresConfig 是 PostgreSQL 连接配置。
type PostgresConfig struct {
	Host     string `env:"PG_HOST" envDefault:"localhost"`
	Port     string `env:"PG_PORT" envDefault:"5432"`
	User     string `env:"PG_USER" envDefault:"mm"`
	Password string `env:"PG_PASSWORD" envDefault:""`
	Database string `env:"PG_DB" envDefault:"pinconsole"`
	SSLMode  string `env:"PG_SSLMODE" envDefault:"prefer"`
	// 1z:连接池上限。pgxpool 默认 max(4, NumCPU) 对 4 核机器仅 4 条,
	// 多并发访客场景(每访客至少 1 PG 写连接)快速耗尽。默认 25 = ~20 访客 + 5 运营余量。
	MaxConns int `env:"PG_MAX_CONNS" envDefault:"25"`
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
	// 1z:连接池上限。go-redis 默认 10*NumCPU 对 4 核机器仅 40 条,
	// rate limit + claim + stream append + presence 等多 key 类操作很快耗尽。默认 50。
	PoolSize int `env:"REDIS_POOL_SIZE" envDefault:"50"`
}

// MinIOConfig 是 MinIO 连接配置。
type MinIOConfig struct {
	Endpoint  string `env:"MINIO_ENDPOINT" envDefault:"localhost:9000"`
	AccessKey string `env:"MINIO_ACCESS_KEY" envDefault:""`
	SecretKey string `env:"MINIO_SECRET_KEY" envDefault:""`
	Bucket    string `env:"MINIO_BUCKET" envDefault:"pinconsole"`
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
	// 1z:Env 白名单。typo 防御(如 "production" 会被 router 当 dev mode 触发 bypass)。
	if _, ok := allowedEnvs[cfg.Env]; !ok {
		return fmt.Errorf("SERVER_ENV=%q 不合法：仅允许 {prod, dev, test}（1z fail-secure，防 typo 触发 dev bypass）", cfg.Env)
	}
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
		// 1z:PG sslmode 明文防御。disable / 未设 在 prod 不允许(除非 PG_HOST 是本地)。
		if !isLocalEndpoint(cfg.Postgres.Host) {
			if _, bad := disallowedProdSSLMode[strings.ToLower(cfg.Postgres.SSLMode)]; bad {
				return fmt.Errorf("PG_SSLMODE=%q 在 prod 模式 + 远程 PG 下不允许：明文凭证走网络，请用 prefer/require/verify-ca/verify-full（1z fail-secure）", cfg.Postgres.SSLMode)
			}
		}
		// 1z:MinIO 跨网络 TLS 防御。本地/docker 内部网络允许 false。
		if !cfg.MinIO.UseSSL && !isLocalEndpoint(cfg.MinIO.Endpoint) {
			return fmt.Errorf("MINIO_USE_SSL=false 在 prod 模式 + 远程 MinIO 下不允许：录像 blob 明文走网络，请设 MINIO_USE_SSL=true（1z fail-secure）")
		}
	}

	// 1ab P1-5:TrustedProxies 配置一致性校验(fail-secure)。
	// 反代部署忘配 → rate limit 共用单 IP 预算,功能退化且不易发现。
	// 直接暴露 + 配 TrustedProxies 是无害的防御性配置(silent allow)。
	if cfg.BehindReverseProxy {
		if strings.TrimSpace(cfg.TrustedProxies) == "" {
			return fmt.Errorf("BEHIND_REVERSE_PROXY=true 但 TRUSTED_PROXIES 未设置：反代部署必须显式配置信任的反代 CIDR（1ab fail-secure，防 rate limit 共用单 IP 预算）")
		}
		if err := validateProxyCIDRList(cfg.TrustedProxies); err != nil {
			return fmt.Errorf("TRUSTED_PROXIES 格式错误：%v（1ab fail-secure）", err)
		}
	}
	return nil
}

// validateProxyCIDRList 校验逗号分隔的 CIDR/IP 列表格式。
// 接受 "10.0.0.0/8" / "192.168.1.1" 等格式;空字符串视为 OK(由调用方判断是否允许)。
func validateProxyCIDRList(s string) error {
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.Contains(p, "/") {
			if _, _, err := net.ParseCIDR(p); err != nil {
				return fmt.Errorf("invalid CIDR %q: %w", p, err)
			}
		} else {
			if net.ParseIP(p) == nil {
				return fmt.Errorf("invalid IP %q", p)
			}
		}
	}
	return nil
}
