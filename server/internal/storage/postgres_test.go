// 1t 测试:storage 包的纯函数 + 边界(不依赖真实 PG)。
package storage

import (
	"strings"
	"testing"

	"github.com/iannil/marketing-monitor/internal/config"
)

// TestPostgresConfig_DSN 验证 DSN 拼接正确,特别是 sslmode 参数化。
func TestPostgresConfig_DSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.PostgresConfig
		want []string // DSN 中应包含的子串
	}{
		{
			name: "default",
			cfg: config.PostgresConfig{
				Host: "localhost", Port: "5432",
				User: "mm", Password: "secret",
				Database: "test", SSLMode: "prefer",
			},
			want: []string{
				"postgres://mm:secret@localhost:5432/test",
				"sslmode=prefer",
			},
		},
		{
			name: "sslmode require",
			cfg: config.PostgresConfig{
				Host: "db.example.com", Port: "5432",
				User: "app", Password: "p@ssw0rd",
				Database: "prod", SSLMode: "require",
			},
			want: []string{
				"postgres://app:p@ssw0rd@db.example.com:5432/prod",
				"sslmode=require",
			},
		},
		{
			name: "empty password",
			cfg: config.PostgresConfig{
				Host: "h", Port: "p",
				User: "u", Password: "",
				Database: "d", SSLMode: "disable",
			},
			want: []string{
				"postgres://u:@h:p/d",
				"sslmode=disable",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.DSN()
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("DSN() = %q, missing substring %q", got, w)
				}
			}
		})
	}
}

// TestDefaultTenantID 验证默认 tenant_id 是 nil UUID(全零)。
// 业务约定:所有数据归属 tenant_id = uuid.Nil 直到多租户启用。
func TestDefaultTenantID(t *testing.T) {
	if DefaultTenantID.String() != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("DefaultTenantID = %s, want nil UUID", DefaultTenantID)
	}
}
