// 1k 测试：config fail-secure 校验。
package config

import (
	"os"
	"strings"
	"testing"
)

// setEnv批量设置环境变量，测试结束后清理。
func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

// clearEnv 清空可能影响测试的 env。
func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
}

// configEnvKeys 是 Load 涉及的所有 env key，测试间需清理。
var configEnvKeys = []string{
	"SERVER_PORT", "SERVER_ENV", "LOG_LEVEL", "NAVIGATE_ALLOWED_DOMAINS",
	"ADMIN_EMAIL", "ADMIN_PASSWORD", "BCRYPT_COST",
	"RATE_LIMIT_PER_MIN", "BANNED_UAS",
	"PG_HOST", "PG_PORT", "PG_USER", "PG_PASSWORD", "PG_DB", "PG_SSLMODE",
	"REDIS_ADDR", "REDIS_PASSWORD",
	"MINIO_ENDPOINT", "MINIO_ACCESS_KEY", "MINIO_SECRET_KEY", "MINIO_BUCKET", "MINIO_USE_SSL",
	// 1ab P1-5
	"TRUSTED_PROXIES", "BEHIND_REVERSE_PROXY",
}

func TestLoad_DefaultEnvIsProd(t *testing.T) {
	clearEnv(t, configEnvKeys...)
	// 强密码 + prod-safe 凭证
	t.Setenv("ADMIN_PASSWORD", "strong-prod-password-123!")
	t.Setenv("PG_PASSWORD", "prod-pg-secret")
	t.Setenv("MINIO_ACCESS_KEY", "prod-access-key")
	t.Setenv("MINIO_SECRET_KEY", "prod-minio-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Env != "prod" {
		t.Errorf("Env default = %q, want 'prod' (1k fail-secure)", cfg.Env)
	}
}

func TestLoad_AdminPasswordRequired(t *testing.T) {
	clearEnv(t, configEnvKeys...)
	// 不设 ADMIN_PASSWORD
	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail when ADMIN_PASSWORD is empty (1k fail-secure)")
	}
	if !strings.Contains(err.Error(), "ADMIN_PASSWORD") {
		t.Errorf("error should mention ADMIN_PASSWORD, got: %v", err)
	}
}

func TestLoad_AdminPasswordChangemeRejectedInProd(t *testing.T) {
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "changeme123") // 历史弱密码
	t.Setenv("PG_PASSWORD", "prod-pg-secret")
	t.Setenv("MINIO_ACCESS_KEY", "prod-access-key")
	t.Setenv("MINIO_SECRET_KEY", "prod-minio-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should reject changeme123 in prod mode")
	}
	if !strings.Contains(err.Error(), "changeme123") {
		t.Errorf("error should mention changeme123, got: %v", err)
	}
}

func TestLoad_BCryptCostMinimum(t *testing.T) {
	clearEnv(t, configEnvKeys...)
	t.Setenv("ADMIN_PASSWORD", "strong-password-123!")
	t.Setenv("BCRYPT_COST", "10") // < 12

	_, err := Load()
	if err == nil {
		t.Fatal("Load should reject BCryptCost < 12")
	}
	if !strings.Contains(err.Error(), "BCRYPT_COST") {
		t.Errorf("error should mention BCRYPT_COST, got: %v", err)
	}
}

func TestLoad_ProdModeRejectsWeakCreds(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "weak PG password",
			env: map[string]string{
				"SERVER_ENV":      "prod",
				"ADMIN_PASSWORD":  "strong-admin-123!",
				"PG_PASSWORD":     "mm_dev", // 弱
				"MINIO_ACCESS_KEY": "prod-access-key",
				"MINIO_SECRET_KEY": "prod-minio-secret",
			},
		},
		{
			name: "empty PG password",
			env: map[string]string{
				"SERVER_ENV":      "prod",
				"ADMIN_PASSWORD":  "strong-admin-123!",
				"PG_PASSWORD":     "", // 空
				"MINIO_ACCESS_KEY": "prod-access-key",
				"MINIO_SECRET_KEY": "prod-minio-secret",
			},
		},
		{
			name: "weak MinIO access key",
			env: map[string]string{
				"SERVER_ENV":      "prod",
				"ADMIN_PASSWORD":  "strong-admin-123!",
				"PG_PASSWORD":     "prod-pg-secret",
				"MINIO_ACCESS_KEY": "mm_dev", // 弱
				"MINIO_SECRET_KEY": "prod-minio-secret",
			},
		},
		{
			name: "weak MinIO secret key",
			env: map[string]string{
				"SERVER_ENV":      "prod",
				"ADMIN_PASSWORD":  "strong-admin-123!",
				"PG_PASSWORD":     "prod-pg-secret",
				"MINIO_ACCESS_KEY": "prod-access-key",
				"MINIO_SECRET_KEY": "mm_dev_secret", // 弱
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t, configEnvKeys...)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			_, err := Load()
			if err == nil {
				t.Fatalf("Load should reject prod mode with %s", tt.name)
			}
		})
	}
}

func TestLoad_DevModeAllowsWeakCreds(t *testing.T) {
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "dev")
	t.Setenv("ADMIN_PASSWORD", "dev-only-password")
	// 故意用弱 PG/MinIO 凭证,dev 模式应允许
	t.Setenv("PG_PASSWORD", "mm_dev")
	t.Setenv("MINIO_ACCESS_KEY", "mm_dev")
	t.Setenv("MINIO_SECRET_KEY", "mm_dev_secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load in dev mode should allow weak creds, got: %v", err)
	}
	if cfg.Env != "dev" {
		t.Errorf("Env = %q, want 'dev'", cfg.Env)
	}
}

func TestPostgresConfig_DSN_IncludesSSLMode(t *testing.T) {
	cfg := PostgresConfig{
		Host: "localhost", Port: "7032",
		User: "mm", Password: "secret", Database: "test",
		SSLMode: "require",
	}
	dsn := cfg.DSN()
	if !strings.Contains(dsn, "sslmode=require") {
		t.Errorf("DSN should contain sslmode=require, got: %s", dsn)
	}
	// 默认 prefer
	cfg.SSLMode = ""
	cfg.SSLMode = "prefer"
	dsn = cfg.DSN()
	if !strings.Contains(dsn, "sslmode=prefer") {
		t.Errorf("DSN should contain sslmode=prefer, got: %s", dsn)
	}
}

// 1z:Env 白名单 + prod sslmode/useSSL 缝隙防御测试。

func TestLoad_EnvWhitelist_RejectsTypo(t *testing.T) {
	tests := []string{
		"production", // 常见 typo
		"staging",
		"prod-uat",
		"PROD", // 大小写 normalize 后是 prod,合法;此处测真不合法的
		"live",
	}
	// "PROD" 在 ToLower 后是 "prod",合法;移出 typo 测试
	tests = []string{"production", "staging", "prod-uat", "live"}
	for _, env := range tests {
		t.Run("env="+env, func(t *testing.T) {
			clearEnv(t, configEnvKeys...)
			t.Setenv("SERVER_ENV", env)
			t.Setenv("ADMIN_PASSWORD", "strong-password-123!")
			_, err := Load()
			if err == nil {
				t.Fatalf("Load should reject unknown SERVER_ENV=%q (防 typo 触发 dev bypass)", env)
			}
			if !strings.Contains(err.Error(), "SERVER_ENV") {
				t.Errorf("error should mention SERVER_ENV, got: %v", err)
			}
		})
	}
}

func TestLoad_EnvWhitelist_AllowsCanonical(t *testing.T) {
	for _, env := range []string{"prod", "dev", "test"} {
		t.Run("env="+env, func(t *testing.T) {
			clearEnv(t, configEnvKeys...)
			t.Setenv("SERVER_ENV", env)
			t.Setenv("ADMIN_PASSWORD", "strong-password-123!")
			if env == "prod" {
				t.Setenv("PG_PASSWORD", "prod-pg-secret")
				t.Setenv("MINIO_ACCESS_KEY", "prod-access-key")
				t.Setenv("MINIO_SECRET_KEY", "prod-minio-secret")
			}
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load should accept SERVER_ENV=%q, got: %v", env, err)
			}
			if cfg.Env != env {
				t.Errorf("Env = %q, want %q", cfg.Env, env)
			}
		})
	}
}

func TestLoad_EnvWhitelist_NormalizesCase(t *testing.T) {
	// "PROD" / "DEV" 应被 ToLower 接受
	for _, env := range []string{"PROD", "Dev", "TEST"} {
		t.Run("env="+env, func(t *testing.T) {
			clearEnv(t, configEnvKeys...)
			t.Setenv("SERVER_ENV", env)
			t.Setenv("ADMIN_PASSWORD", "strong-password-123!")
			if strings.EqualFold(env, "prod") {
				t.Setenv("PG_PASSWORD", "prod-pg-secret")
				t.Setenv("MINIO_ACCESS_KEY", "prod-access-key")
				t.Setenv("MINIO_SECRET_KEY", "prod-minio-secret")
			}
			_, err := Load()
			if err != nil {
				t.Fatalf("Load should normalize case for SERVER_ENV=%q, got: %v", env, err)
			}
		})
	}
}

func TestLoad_ProdModeRejectsRemotePGSSLDisable(t *testing.T) {
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "strong-password-123!")
	t.Setenv("PG_HOST", "db.example.com") // 远程
	t.Setenv("PG_PASSWORD", "prod-pg-secret")
	t.Setenv("MINIO_ACCESS_KEY", "prod-access-key")
	t.Setenv("MINIO_SECRET_KEY", "prod-minio-secret")

	// caarlos0/env 在 env var 设为空串时填 envDefault("prefer"),
	// 故 "" 无法触发;仅 disable 是真实风险。
	t.Setenv("PG_SSLMODE", "disable")
	_, err := Load()
	if err == nil {
		t.Fatal("Load should reject PG_SSLMODE=disable in prod + remote PG")
	}
	if !strings.Contains(err.Error(), "PG_SSLMODE") {
		t.Errorf("error should mention PG_SSLMODE, got: %v", err)
	}
}

func TestLoad_ProdModeAllowsLocalPGSSLDisable(t *testing.T) {
	// 本地 PG(docker-compose 内部网络)允许 disable —— 容器间不开 TLS 是合理部署。
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "strong-password-123!")
	t.Setenv("PG_HOST", "localhost")
	t.Setenv("PG_SSLMODE", "disable")
	t.Setenv("PG_PASSWORD", "prod-pg-secret")
	t.Setenv("MINIO_ACCESS_KEY", "prod-access-key")
	t.Setenv("MINIO_SECRET_KEY", "prod-minio-secret")
	t.Setenv("MINIO_ENDPOINT", "localhost:7020")

	_, err := Load()
	if err != nil {
		t.Fatalf("Load should allow PG_SSLMODE=disable when PG_HOST is local, got: %v", err)
	}
}

func TestLoad_ProdModeRejectsRemoteMinIOWithoutTLS(t *testing.T) {
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "strong-password-123!")
	t.Setenv("PG_HOST", "localhost")
	t.Setenv("PG_SSLMODE", "prefer")
	t.Setenv("PG_PASSWORD", "prod-pg-secret")
	t.Setenv("MINIO_ENDPOINT", "minio.example.com:9000") // 远程
	t.Setenv("MINIO_USE_SSL", "false")
	t.Setenv("MINIO_ACCESS_KEY", "prod-access-key")
	t.Setenv("MINIO_SECRET_KEY", "prod-minio-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should reject MINIO_USE_SSL=false in prod + remote MinIO")
	}
	if !strings.Contains(err.Error(), "MINIO_USE_SSL") {
		t.Errorf("error should mention MINIO_USE_SSL, got: %v", err)
	}
}

func TestLoad_ProdModeAllowsLocalMinIOWithoutTLS(t *testing.T) {
	// docker-compose 内部网络:容器名 "minio" 视为本地,允许 USE_SSL=false。
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "strong-password-123!")
	t.Setenv("PG_HOST", "postgres")
	t.Setenv("PG_SSLMODE", "disable")
	t.Setenv("PG_PASSWORD", "prod-pg-secret")
	t.Setenv("MINIO_ENDPOINT", "minio:9000")
	t.Setenv("MINIO_USE_SSL", "false")
	t.Setenv("MINIO_ACCESS_KEY", "prod-access-key")
	t.Setenv("MINIO_SECRET_KEY", "prod-minio-secret")

	_, err := Load()
	if err != nil {
		t.Fatalf("Load should allow MINIO_USE_SSL=false when endpoint is local (docker-compose internal), got: %v", err)
	}
}

func TestLoad_DevModeSkipsSSLValidations(t *testing.T) {
	// dev 模式不校验 sslmode/useSSL,允许任意组合方便本地开发。
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "dev")
	t.Setenv("ADMIN_PASSWORD", "dev-password")
	t.Setenv("PG_HOST", "db.example.com")
	t.Setenv("PG_SSLMODE", "disable")
	t.Setenv("PG_PASSWORD", "any")
	t.Setenv("MINIO_ENDPOINT", "minio.example.com:9000")
	t.Setenv("MINIO_USE_SSL", "false")

	_, err := Load()
	if err != nil {
		t.Fatalf("Load in dev mode should skip SSL validations, got: %v", err)
	}
}

// 1ab P1-5:TrustedProxies 配置一致性校验测试

func TestLoad_DirectExposure_Default_NoTrustedProxies(t *testing.T) {
	// 默认 BEHIND_REVERSE_PROXY=false + 空 TrustedProxies → OK(裸暴露部署)
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "dev")
	t.Setenv("ADMIN_PASSWORD", "dev-password")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should succeed for direct exposure default, got: %v", err)
	}
	if cfg.BehindReverseProxy != false {
		t.Errorf("BehindReverseProxy default = true, want false")
	}
}

func TestLoad_BehindProxy_RequiresTrustedProxies(t *testing.T) {
	// BEHIND_REVERSE_PROXY=true 但 TrustedProxies 空 → REJECT
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "dev")
	t.Setenv("ADMIN_PASSWORD", "dev-password")
	t.Setenv("BEHIND_REVERSE_PROXY", "true")
	// 不设 TRUSTED_PROXIES

	_, err := Load()
	if err == nil {
		t.Fatal("Load should reject BEHIND_REVERSE_PROXY=true with empty TRUSTED_PROXIES")
	}
	if !strings.Contains(err.Error(), "TRUSTED_PROXIES") {
		t.Errorf("error should mention TRUSTED_PROXIES, got: %v", err)
	}
}

func TestLoad_BehindProxy_ValidCIDRs_OK(t *testing.T) {
	// BEHIND_REVERSE_PROXY=true + 有效 CIDR 列表 → OK
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "dev")
	t.Setenv("ADMIN_PASSWORD", "dev-password")
	t.Setenv("BEHIND_REVERSE_PROXY", "true")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8,192.168.0.0/16,172.16.0.1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should accept valid CIDR list, got: %v", err)
	}
	if cfg.TrustedProxies != "10.0.0.0/8,192.168.0.0/16,172.16.0.1" {
		t.Errorf("TrustedProxies = %q, want list", cfg.TrustedProxies)
	}
}

func TestLoad_BehindProxy_InvalidCIDR_Rejects(t *testing.T) {
	// BEHIND_REVERSE_PROXY=true + 无效 CIDR → REJECT
	tests := []struct {
		name  string
		value string
	}{
		{"not enough octets", "10.0.0/8"},
		{"bad prefix", "10.0.0.0/99"},
		{"non-IP string", "not-an-ip"},
		{"trailing garbage", "10.0.0.0/8,garbage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t, configEnvKeys...)
			t.Setenv("SERVER_ENV", "dev")
			t.Setenv("ADMIN_PASSWORD", "dev-password")
			t.Setenv("BEHIND_REVERSE_PROXY", "true")
			t.Setenv("TRUSTED_PROXIES", tt.value)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load should reject invalid CIDR %q", tt.value)
			}
			if !strings.Contains(err.Error(), "TRUSTED_PROXIES") {
				t.Errorf("error should mention TRUSTED_PROXIES, got: %v", err)
			}
		})
	}
}

func TestLoad_DirectExposure_TrustedProxiesSet_SilentAllow(t *testing.T) {
	// BEHIND_REVERSE_PROXY=false + TrustedProxies 设了 → silent OK
	// (用户防御性配置,无害 — XFF 被忽略)
	clearEnv(t, configEnvKeys...)
	t.Setenv("SERVER_ENV", "dev")
	t.Setenv("ADMIN_PASSWORD", "dev-password")
	t.Setenv("BEHIND_REVERSE_PROXY", "false")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8")

	_, err := Load()
	if err != nil {
		t.Fatalf("Load should silently allow TrustedProxies in direct mode (no harm), got: %v", err)
	}
}

func TestValidateProxyCIDRList_Formats(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", false},
		{"single CIDR", "10.0.0.0/8", false},
		{"multiple CIDRs", "10.0.0.0/8,192.168.0.0/16", false},
		{"bare IP", "10.0.0.1", false},
		{"IPv6 CIDR", "2001:db8::/32", false},
		{"mixed CIDR + IP", "10.0.0.0/8,192.168.1.1", false},
		{"whitespace tolerated", "  10.0.0.0/8  ,  192.168.0.0/16  ", false},
		{"bad CIDR prefix", "10.0.0.0/33", true},
		{"not IP", "hello", true},
		{"trailing comma", "10.0.0.0/8,", false}, // empty parts skipped
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProxyCIDRList(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("validateProxyCIDRList(%q) should error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateProxyCIDRList(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}
