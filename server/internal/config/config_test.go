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
		Host: "localhost", Port: "5432",
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
