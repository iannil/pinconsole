// Go-7 切片补测:config.Load prod fail-secure 路径 buffer。
package config

import (
	"os"
	"strings"
	"testing"
)

// setEnvDefaults 设置 dev 默认环境(满足 validate 基本要求)。
func setEnvDefaults(t *testing.T) {
	t.Helper()
	t.Setenv("SERVER_ENV", "dev")
	t.Setenv("ADMIN_PASSWORD", "test-only-pinconsole-strong-pwd-2026")
	t.Setenv("BCRYPT_COST", "12")
	t.Setenv("PG_HOST", "localhost")
	t.Setenv("PG_PASSWORD", "mm_dev")
	t.Setenv("MINIO_ACCESS_KEY", "mm_dev")
	t.Setenv("MINIO_SECRET_KEY", "mm_dev_secret")
	t.Setenv("MINIO_USE_SSL", "false")
}

// TestLoad_DevModeAllDefaultsOK 验证 dev 模式默认值通过 validate。
func TestLoad_DevModeAllDefaultsOK(t *testing.T) {
	setEnvDefaults(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load dev: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg is nil")
	}
	if cfg.Env != "dev" {
		t.Errorf("Env: got %q, want dev", cfg.Env)
	}
}

// TestLoad_ProdChangemeFails 验证 prod 模式 changeme123 密码被拒绝。
func TestLoad_ProdChangemeFails(t *testing.T) {
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "changeme123")
	t.Setenv("BCRYPT_COST", "12")
	t.Setenv("PG_PASSWORD", "strong-prod-pwd")
	t.Setenv("MINIO_ACCESS_KEY", "prod-key")
	t.Setenv("MINIO_SECRET_KEY", "prod-secret")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("PG_SSLMODE", "require")
	t.Setenv("PG_HOST", "prod-pg.internal")
	t.Setenv("MINIO_ENDPOINT", "prod-minio.internal:9000")

	_, err := Load()
	if err == nil {
		t.Error("prod + changeme123: expected error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "changeme123") {
		t.Errorf("error should mention changeme123, got: %v", err)
	}
}

// TestLoad_ProdDefaultPGPasswordFails 验证 prod 模式 PG_PASSWORD=mm_dev 被拒绝。
func TestLoad_ProdDefaultPGPasswordFails(t *testing.T) {
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "strong-prod-admin-pwd")
	t.Setenv("BCRYPT_COST", "12")
	t.Setenv("PG_PASSWORD", "mm_dev") // dev 默认值,prod 应拒绝
	t.Setenv("MINIO_ACCESS_KEY", "prod-key")
	t.Setenv("MINIO_SECRET_KEY", "prod-secret")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("PG_SSLMODE", "require")
	t.Setenv("PG_HOST", "prod-pg.internal")
	t.Setenv("MINIO_ENDPOINT", "prod-minio.internal:9000")

	_, err := Load()
	if err == nil {
		t.Error("prod + PG_PASSWORD=mm_dev: expected error, got nil")
	}
}

// TestLoad_ProdMinIOWeakKeyFails 验证 prod 模式 MINIO_ACCESS_KEY=mm_dev 被拒绝。
func TestLoad_ProdMinIOWeakKeyFails(t *testing.T) {
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "strong-prod-admin-pwd")
	t.Setenv("BCRYPT_COST", "12")
	t.Setenv("PG_PASSWORD", "strong-pg-pwd")
	t.Setenv("MINIO_ACCESS_KEY", "mm_dev") // dev 默认值
	t.Setenv("MINIO_SECRET_KEY", "prod-secret")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("PG_SSLMODE", "require")
	t.Setenv("PG_HOST", "prod-pg.internal")
	t.Setenv("MINIO_ENDPOINT", "prod-minio.internal:9000")

	_, err := Load()
	if err == nil {
		t.Error("prod + MINIO_ACCESS_KEY=mm_dev: expected error, got nil")
	}
}

// TestLoad_ProdMinIOWeakSecretFails 验证 prod 模式 MINIO_SECRET_KEY=mm_dev_secret 被拒绝。
func TestLoad_ProdMinIOWeakSecretFails(t *testing.T) {
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "strong-prod-admin-pwd")
	t.Setenv("BCRYPT_COST", "12")
	t.Setenv("PG_PASSWORD", "strong-pg-pwd")
	t.Setenv("MINIO_ACCESS_KEY", "prod-key")
	t.Setenv("MINIO_SECRET_KEY", "mm_dev_secret") // dev 默认值
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("PG_SSLMODE", "require")
	t.Setenv("PG_HOST", "prod-pg.internal")
	t.Setenv("MINIO_ENDPOINT", "prod-minio.internal:9000")

	_, err := Load()
	if err == nil {
		t.Error("prod + MINIO_SECRET_KEY=mm_dev_secret: expected error, got nil")
	}
}

// TestLoad_ProdRemotePGDisallowedSSLMode 验证 prod + 远程 PG + disallowed SSL mode 被拒绝。
func TestLoad_ProdRemotePGDisallowedSSLMode(t *testing.T) {
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "strong-prod-admin-pwd")
	t.Setenv("BCRYPT_COST", "12")
	t.Setenv("PG_PASSWORD", "strong-pg-pwd")
	t.Setenv("MINIO_ACCESS_KEY", "prod-key")
	t.Setenv("MINIO_SECRET_KEY", "prod-secret")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("PG_SSLMODE", "disable") // 远程 PG 不允许
	t.Setenv("PG_HOST", "prod-pg.internal")
	t.Setenv("MINIO_ENDPOINT", "prod-minio.internal:9000")

	_, err := Load()
	if err == nil {
		t.Error("prod + remote PG + SSL=disable: expected error, got nil")
	}
}

// TestLoad_ProdRemoteMinIONoSSLFails 验证 prod + 远程 MinIO + UseSSL=false 被拒绝。
func TestLoad_ProdRemoteMinIONoSSLFails(t *testing.T) {
	t.Setenv("SERVER_ENV", "prod")
	t.Setenv("ADMIN_PASSWORD", "strong-prod-admin-pwd")
	t.Setenv("BCRYPT_COST", "12")
	t.Setenv("PG_PASSWORD", "strong-pg-pwd")
	t.Setenv("MINIO_ACCESS_KEY", "prod-key")
	t.Setenv("MINIO_SECRET_KEY", "prod-secret")
	t.Setenv("MINIO_USE_SSL", "false") // 远程 MinIO 不允许
	t.Setenv("PG_SSLMODE", "require")
	t.Setenv("PG_HOST", "prod-pg.internal")
	t.Setenv("MINIO_ENDPOINT", "prod-minio.internal:9000")

	_, err := Load()
	if err == nil {
		t.Error("prod + remote MinIO + UseSSL=false: expected error, got nil")
	}
}

// TestLoad_BCryptCostBelow12Fails 验证 BCRYPT_COST < 12 被拒绝。
func TestLoad_BCryptCostBelow12Fails(t *testing.T) {
	setEnvDefaults(t)
	t.Setenv("BCRYPT_COST", "10") // < 12

	_, err := Load()
	if err == nil {
		t.Error("BCRYPT_COST=10: expected error, got nil")
	}
}

// TestLoad_BehindReverseProxyNoTrustedProxiesFails 验证 BehindReverseProxy=true 但 TrustedProxies 未设被拒。
func TestLoad_BehindReverseProxyNoTrustedProxiesFails(t *testing.T) {
	setEnvDefaults(t)
	t.Setenv("BEHIND_REVERSE_PROXY", "true")
	t.Setenv("TRUSTED_PROXIES", "")

	_, err := Load()
	if err == nil {
		t.Error("BehindReverseProxy=true + empty TrustedProxies: expected error, got nil")
	}
}

// TestLoad_UnknownEnvFails 验证 SERVER_ENV 非 {prod,dev,test} 被拒(1z typo 防御)。
func TestLoad_UnknownEnvFails(t *testing.T) {
	setEnvDefaults(t)
	t.Setenv("SERVER_ENV", "production") // typo,应该用 prod

	_, err := Load()
	if err == nil {
		t.Error("SERVER_ENV=production: expected error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "SERVER_ENV") {
		t.Errorf("error should mention SERVER_ENV, got: %v", err)
	}
}

// 确保未使用 os import(用 t.Setenv 代替)。
var _ = os.Setenv
