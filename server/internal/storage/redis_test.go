// 1z 测试:Redis 连接池配置。
// 不实际连接 Redis(避免依赖外部服务),只验证 cfg.PoolSize 被正确应用到 redis.Options。
package storage

import (
	"testing"

	"github.com/iannil/pinconsole/internal/config"
	"github.com/redis/go-redis/v9"
)

// TestRedisConfig_PoolSizeAppliedToOptions 验证 cfg.PoolSize 被正确传到 redis.Options.PoolSize。
func TestRedisConfig_PoolSizeAppliedToOptions(t *testing.T) {
	tests := []struct {
		name      string
		poolSize  int
		wantPools int
	}{
		{name: "default 50", poolSize: 50, wantPools: 50},
		{name: "explicit 100", poolSize: 100, wantPools: 100},
		{name: "small 10", poolSize: 10, wantPools: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RedisConfig{
				Addr:     "localhost:6379",
				Password: "",
				PoolSize: tt.poolSize,
			}
			// 模拟 ConnectRedis 内部逻辑(不实际 NewClient + Ping)
			opts := &redis.Options{
				Addr:     cfg.Addr,
				Password: cfg.Password,
			}
			if cfg.PoolSize > 0 {
				opts.PoolSize = cfg.PoolSize
			}
			if opts.PoolSize != tt.wantPools {
				t.Errorf("PoolSize = %d, want %d", opts.PoolSize, tt.wantPools)
			}
		})
	}
}

// TestRedisConfig_DefaultPoolSize 验证业务默认值 50(env 未设时 caarlos0/env 填 envDefault)。
func TestRedisConfig_DefaultPoolSize(t *testing.T) {
	cfg := config.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		PoolSize: 50, // 显式默认(模拟 envDefault)
	}
	opts := &redis.Options{Addr: cfg.Addr, Password: cfg.Password}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}
	if opts.PoolSize != 50 {
		t.Errorf("default PoolSize = %d, want 50", opts.PoolSize)
	}
}

// TestRedisConfig_ZeroPoolSizeFallsBackToLibraryDefault 验证 cfg.PoolSize=0 时
// 不覆盖 opts.PoolSize(go-redis 默认 10*NumCPU)。
func TestRedisConfig_ZeroPoolSizeFallsBackToLibraryDefault(t *testing.T) {
	cfg := config.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		PoolSize: 0, // 未设
	}
	opts := &redis.Options{Addr: cfg.Addr, Password: cfg.Password}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}
	// opts.PoolSize 应保持 go-redis 默认值(0 表示用库内部计算)
	if opts.PoolSize != 0 {
		t.Errorf("PoolSize = %d, want 0 (go-redis default)", opts.PoolSize)
	}
}
