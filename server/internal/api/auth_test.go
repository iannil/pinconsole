// 1x 测试:登录暴力破解防护(修审计 P1-3)。
//
// 覆盖 checkLoginThrottle + recordLoginFailure 的纯函数行为(Redis Lua + Get/TTL
// 通过 miniredis 或 docker redis;此处用真 Redis,失败时 skip)。
package api

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iannil/marketing-monitor/internal/storage"
	"github.com/redis/go-redis/v9"
)

// TestLoginThrottleKey_Format 验证 key 格式 — 防止格式变更导致 Redis 数据隔离失效。
func TestLoginThrottleKey_Format(t *testing.T) {
	got := loginThrottleKey("admin@x.com", "10.0.0.1")
	want := "auth:throttle:admin@x.com:10.0.0.1"
	if got != want {
		t.Errorf("loginThrottleKey = %q, want %q", got, want)
	}
}

// TestLoginThrottle_Constants 验证阈值常量未误改。
func TestLoginThrottle_Constants(t *testing.T) {
	if loginMaxAttempts != 5 {
		t.Errorf("loginMaxAttempts = %d, want 5(审计 P1-3 默认)", loginMaxAttempts)
	}
	if loginLockoutWindow != 15*time.Minute {
		t.Errorf("loginLockoutWindow = %v, want 15m", loginLockoutWindow)
	}
}

// TestLoginThrottle_LockAfter5Failures 端到端验证 5 次失败后第 6 次锁定。
// 需要真 Redis,不可用时 skip。
func TestLoginThrottle_LockAfter5Failures(t *testing.T) {
	if testing.Short() {
		t.Skip("需要 Redis")
	}
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis 不可用(%v),跳过", err)
	}

	email := "throttle-test-1x@example.com"
	ip := "10.99.99.1"
	key := loginThrottleKey(email, ip)
	defer rdb.Del(ctx, key)

	// 构造 AuthHandler(只需 stores.Redis.Client)
	h := &AuthHandler{
		stores: &storage.Stores{Redis: &storage.Redis{Client: rdb}},
	}

	// 失败 5 次后应被锁
	for i := 1; i <= loginMaxAttempts; i++ {
		h.recordLoginFailure(ctx, key)
		locked, _, err := h.checkLoginThrottle(ctx, key)
		if err != nil {
			t.Fatalf("iter %d checkLoginThrottle: %v", i, err)
		}
		if i < loginMaxAttempts {
			if locked {
				t.Errorf("iter %d: locked=true, want false (锁阈值 %d)", i, loginMaxAttempts)
			}
		} else {
			if !locked {
				t.Errorf("iter %d: locked=false, want true (已达阈值)", i)
			}
		}
	}

	// 验证 TTL 在窗口内
	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 0 || ttl > loginLockoutWindow {
		t.Errorf("TTL = %v, want (0, %v]", ttl, loginLockoutWindow)
	}
}

// TestLoginThrottle_DelOnSuccess 验证 recordLoginFailure 后清零成功。
// 即正常用户偶然失败几次后登录成功 → 计数清零。
func TestLoginThrottle_DelOnSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("需要 Redis")
	}
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis 不可用(%v),跳过", err)
	}

	key := loginThrottleKey("success-test-1x@example.com", "10.99.99.2")
	defer rdb.Del(ctx, key)

	h := &AuthHandler{stores: &storage.Stores{Redis: &storage.Redis{Client: rdb}}}

	// 失败 3 次
	for i := 0; i < 3; i++ {
		h.recordLoginFailure(ctx, key)
	}

	// Del 模拟 login 成功路径
	if err := rdb.Del(ctx, key).Err(); err != nil {
		t.Fatalf("Del: %v", err)
	}

	// 验证清零
	locked, _, err := h.checkLoginThrottle(ctx, key)
	if err != nil {
		t.Fatalf("checkLoginThrottle: %v", err)
	}
	if locked {
		t.Errorf("after Del: locked=true, want false")
	}
}

// TestLoginThrottle_IsolationByIpAndEmail 验证 email 和 IP 维度独立计数。
func TestLoginThrottle_IsolationByIpAndEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("需要 Redis")
	}
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis 不可用(%v),跳过", err)
	}

	email := "iso@example.com"
	key1 := loginThrottleKey(email, "10.99.99.10")
	key2 := loginThrottleKey(email, "10.99.99.11") // 不同 IP
	key3 := loginThrottleKey("other@example.com", "10.99.99.10") // 不同 email
	defer rdb.Del(ctx, key1, key2, key3)

	h := &AuthHandler{stores: &storage.Stores{Redis: &storage.Redis{Client: rdb}}}

	// key1 锁定,key2/key3 不应受影响
	for i := 0; i < loginMaxAttempts; i++ {
		h.recordLoginFailure(ctx, key1)
	}
	h.recordLoginFailure(ctx, key2)
	h.recordLoginFailure(ctx, key3)

	locked1, _, _ := h.checkLoginThrottle(ctx, key1)
	locked2, _, _ := h.checkLoginThrottle(ctx, key2)
	locked3, _, _ := h.checkLoginThrottle(ctx, key3)

	if !locked1 {
		t.Errorf("key1 should be locked after %d failures", loginMaxAttempts)
	}
	if locked2 {
		t.Errorf("key2 (different IP) should NOT be locked after 1 failure")
	}
	if locked3 {
		t.Errorf("key3 (different email) should NOT be locked after 1 failure")
	}
}

// 编译时契约:防 storage.Stores 字段变更破坏 1x 接入。
func TestStores_RedisFieldContract(t *testing.T) {
	var s storage.Stores
	_ = s.Redis
}

// 编译时契约:防 EvalLua 签名变更。
func TestRedis_EvalLuaContract(t *testing.T) {
	var r storage.Redis
	_ = r.EvalLua
	_ = fmt.Sprintf // 防 fmt import 在编译期被裁剪
}

// TestLoginThrottle_RecordFailureUsesLuaAtomic — 1ac T0-1x-1 回归测试。
//
// recordLoginFailure 必须用 Lua 脚本原子 INCR + EXPIRE,
// 防并发场景下首次失败 race(两个并发请求同时 INCR 都看到 1 → 都不 EXPIRE → 永久 lockout)。
//
// 源码契约:auth.go 中 recordLoginFailure 必须含 EvalLua + 完整 Lua 脚本。
func TestLoginThrottle_RecordFailureUsesLuaAtomic(t *testing.T) {
	src, err := os.ReadFile("auth.go")
	if err != nil {
		t.Fatalf("read auth.go: %v", err)
	}
	body := string(src)

	// 定位 recordLoginFailure 函数
	idx := strings.Index(body, "func (h *AuthHandler) recordLoginFailure")
	if idx < 0 {
		t.Fatal("找不到 recordLoginFailure 函数")
	}
	// 截取该函数体(到下一个 func 或文件末)
	end := strings.Index(body[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(body) - idx - 1
	}
	fnBody := body[idx : idx+1+end]

	for _, must := range []string{
		"EvalLua",                       // 用 Lua 而非裸 INCR + EXPIRE
		"redis.call('INCR'",             // Lua 内含 INCR
		"redis.call('EXPIRE'",           // Lua 内含 EXPIRE
		"if c == 1 then",                // 仅首次(返回 1)才 EXPIRE
		"loginLockoutWindow.Seconds()", // TTL 参数
	} {
		if !strings.Contains(fnBody, must) {
			t.Errorf("recordLoginFailure 缺失 %q — Lua 原子性破坏:\n%s", must, fnBody)
		}
	}

	// 反模式检测:裸 INCR/EXPIRE 拆分调用会破坏原子性
	if strings.Contains(fnBody, "rdb.Incr(") ||
		strings.Contains(fnBody, "Client.Incr(") ||
		strings.Contains(fnBody, "Redis.Incr(") {
		t.Errorf("recordLoginFailure 用了裸 Incr — 应改为 Lua 原子 INCR+EXPIRE")
	}
}
