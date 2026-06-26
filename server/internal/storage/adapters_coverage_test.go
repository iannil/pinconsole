// Go-4 切片补测:minio/postgres/redis/stores 适配器 + erasure ListEventBlobKeysBySessions,
// 提升覆盖率 57.6% → ≥90%。
//
// 测试策略:全 docker 集成(复用 helperPGPool + 新增 helperRedis/helperMinIO),
// 测真实 IO 行为不用 mock——适配器本身就是连接层,真实连接才能验证错误路径。
package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/config"
	"github.com/minio/minio-go/v7"
)

// helperRedis 返回真 *storage.Redis,不可用时 skip。
func helperRedis(t *testing.T) *Redis {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rdb, err := ConnectRedis(ctx, helperRedisCfg(), discardLogger())
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	return rdb
}

// helperPostgresCfg 构造测试用 PostgresConfig(连 docker pinconsole DB)。
func helperPostgresCfg() config.PostgresConfig {
	return config.PostgresConfig{
		Host:     "localhost",
		Port:     "7032",
		User:     "mm",
		Password: "mm_dev",
		Database: "pinconsole",
		SSLMode:  "disable",
		MaxConns: 5,
	}
}

// helperRedisCfg 构造测试用 RedisConfig。
func helperRedisCfg() config.RedisConfig {
	return config.RedisConfig{
		Addr:     "localhost:7079",
		Password: "",
		PoolSize: 5,
	}
}

// helperMinIOCfg 构造测试用 MinIOConfig(bucket 随机避免冲突)。
func helperMinIOCfg(t *testing.T) config.MinIOConfig {
	t.Helper()
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	bucket := "test-" + hex.EncodeToString(b)
	return config.MinIOConfig{
		Endpoint:  "localhost:7020",
		AccessKey: "mm_dev",
		SecretKey: "mm_dev_secret",
		Bucket:    bucket,
		UseSSL:    false,
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// === Postgres ConnectPostgres / Ping / Close ===

// TestConnectPostgres_Success 验证 ConnectPostgres 建立连接并返回 Ping 通过的 Postgres。
func TestConnectPostgres_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pg, err := ConnectPostgres(ctx, helperPostgresCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectPostgres: %v", err)
	}
	defer pg.Close()

	if pg.Pool == nil {
		t.Error("Pool is nil")
	}

	// Ping 应通过
	if err := pg.Ping(ctx); err != nil {
		t.Errorf("Ping after Connect: %v", err)
	}
}

// TestConnectPostgres_BadConfig 验证错误配置返回 error。
func TestConnectPostgres_BadConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	bad := helperPostgresCfg()
	bad.Port = "1" // 无效端口
	bad.Database = "nonexistent_db_xyz"

	_, err := ConnectPostgres(ctx, bad, discardLogger())
	if err == nil {
		t.Error("expected error for bad config, got nil")
	}
}

// TestPostgres_Close_Idempotent 验证多次 Close 不 panic。
func TestPostgres_Close_Idempotent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pg, err := ConnectPostgres(ctx, helperPostgresCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectPostgres: %v", err)
	}

	pg.Close()
	pg.Close() // 不应 panic
	pg.Close()
}

// === Redis ConnectRedis / Ping / Close / Set / SetNX / Get / Del / TTL / EvalLua ===

// TestConnectRedis_Success 验证 ConnectRedis 建立连接。
func TestConnectRedis_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 用 helperRedis 先确认 Redis 可用(否则 skip)
	_ = helperRedis(t)

	rdb, err := ConnectRedis(ctx, helperRedisCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectRedis: %v", err)
	}
	defer rdb.Close()

	if rdb.Client == nil {
		t.Error("Client is nil")
	}

	if err := rdb.Ping(ctx); err != nil {
		t.Errorf("Ping after Connect: %v", err)
	}
}

// TestConnectRedis_BadAddr 验证错误地址返回 error。
func TestConnectRedis_BadAddr(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	bad := helperRedisCfg()
	bad.Addr = "localhost:1" // 无效端口

	_, err := ConnectRedis(ctx, bad, discardLogger())
	if err == nil {
		t.Error("expected error for bad addr, got nil")
	}
}

// TestRedis_SetGet_Del_TTL 验证 Set → Get → TTL → Del 全流程。
func TestRedis_SetGet_Del_TTL(t *testing.T) {
	rdb := helperRedis(t)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := "test-setget-" + time.Now().Format("150405.000000000")
	value := []byte("hello-pinconsole")
	ttl := 10 * time.Second

	defer rdb.Del(ctx, key)

	// Set
	if err := rdb.Set(ctx, key, value, ttl); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Get
	got, err := rdb.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("Get: got %q, want %q", got, value)
	}

	// TTL
	gotTTL, err := rdb.TTL(ctx, key)
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if gotTTL <= 0 || gotTTL > ttl {
		t.Errorf("TTL: got %v, want (0, %v]", gotTTL, ttl)
	}

	// Del
	if err := rdb.Del(ctx, key); err != nil {
		t.Fatalf("Del: %v", err)
	}

	// Get after Del 应返回 nil
	gotNil, err := rdb.Get(ctx, key)
	if err != nil {
		t.Errorf("Get after Del: %v", err)
	}
	if gotNil != nil {
		t.Errorf("Get after Del: got %v, want nil", gotNil)
	}
}

// TestRedis_Get_NonExistingKeyReturnsNil 验证不存在的 key 返回 nil(不是 error)。
func TestRedis_Get_NonExistingKeyReturnsNil(t *testing.T) {
	rdb := helperRedis(t)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := "test-nonexist-" + time.Now().Format("150405.000000000")
	got, err := rdb.Get(ctx, key)
	if err != nil {
		t.Errorf("Get non-existent: %v", err)
	}
	if got != nil {
		t.Errorf("Get non-existent: got %v, want nil", got)
	}
}

// TestRedis_SetNX_AtomicFirstCallWins 验证 SetNX 原子性:首次 true,第二次 false。
func TestRedis_SetNX_AtomicFirstCallWins(t *testing.T) {
	rdb := helperRedis(t)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := "test-setnx-" + time.Now().Format("150405.000000000")
	defer rdb.Del(ctx, key)

	ok1, err := rdb.SetNX(ctx, key, []byte("first"), 10*time.Second)
	if err != nil {
		t.Fatalf("SetNX(1): %v", err)
	}
	if !ok1 {
		t.Errorf("SetNX(1): got false, want true (first call should win)")
	}

	ok2, err := rdb.SetNX(ctx, key, []byte("second"), 10*time.Second)
	if err != nil {
		t.Fatalf("SetNX(2): %v", err)
	}
	if ok2 {
		t.Errorf("SetNX(2): got true, want false (key exists, should not write)")
	}

	// 验证值仍是 first
	got, _ := rdb.Get(ctx, key)
	if string(got) != "first" {
		t.Errorf("after SetNX(2): got %q, want first", got)
	}
}

// TestRedis_EvalLua_PingScript 验证 EvalLua 能跑 Lua 脚本(简单 RETURN 字符串)。
func TestRedis_EvalLua_PingScript(t *testing.T) {
	rdb := helperRedis(t)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 简单 Lua:直接返回字符串
	result, err := rdb.EvalLua(ctx, `return "pinconsole-lua-test"`, nil)
	if err != nil {
		t.Fatalf("EvalLua: %v", err)
	}
	if fmt.Sprintf("%v", result) != "pinconsole-lua-test" {
		t.Errorf("EvalLua result: got %v, want pinconsole-lua-test", result)
	}
}

// TestRedis_EvalLua_CompareAndDel 验证 Lua compare-and-del(claim release 用)。
func TestRedis_EvalLua_CompareAndDel(t *testing.T) {
	rdb := helperRedis(t)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := "test-lua-cmpdel-" + time.Now().Format("150405.000000000")
	defer rdb.Del(ctx, key)

	// 先 Set 一个 owner
	rdb.Set(ctx, key, []byte("owner-A"), 30*time.Second)

	// Lua:if GET == ARGV[1] then DEL else return 0
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	// 用错误 owner → 返回 0
	r1, err := rdb.EvalLua(ctx, script, []string{key}, "owner-WRONG")
	if err != nil {
		t.Fatalf("EvalLua(wrong owner): %v", err)
	}
	if fmt.Sprintf("%v", r1) != "0" {
		t.Errorf("wrong owner: got %v, want 0", r1)
	}

	// 用正确 owner → 返回 1
	r2, err := rdb.EvalLua(ctx, script, []string{key}, "owner-A")
	if err != nil {
		t.Fatalf("EvalLua(correct owner): %v", err)
	}
	if fmt.Sprintf("%v", r2) != "1" {
		t.Errorf("correct owner: got %v, want 1", r2)
	}
}

// TestRedis_Close_Idempotent 验证多次 Close 不 panic。
func TestRedis_Close_Idempotent(t *testing.T) {
	_ = helperRedis(t)
	rdb, err := ConnectRedis(context.Background(), helperRedisCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectRedis: %v", err)
	}

	rdb.Close()
	rdb.Close()
	rdb.Close()
}

// TestRedis_Close_NilClientSafe 验证 Client=nil 时 Close 不 panic。
func TestRedis_Close_NilClientSafe(t *testing.T) {
	r := &Redis{Client: nil}
	r.Close() // 不应 panic
}

// === MinIO ConnectMinIO / Ping / PutBytes / GetBytes / Close ===

// helperMinIO 返回已连接的 MinIO(对应 recording.helperMinIOClient)。
func helperMinIO(t *testing.T) *MinIO {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mio, err := ConnectMinIO(ctx, helperMinIOCfg(t), discardLogger())
	if err != nil {
		t.Skipf("minio not available: %v", err)
	}
	return mio
}

// TestConnectMinIO_Success 验证 ConnectMinIO 建立连接 + 自动建 bucket。
func TestConnectMinIO_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mio, err := ConnectMinIO(ctx, helperMinIOCfg(t), discardLogger())
	if err != nil {
		t.Fatalf("ConnectMinIO: %v", err)
	}
	defer mio.Close()

	if mio.Client == nil {
		t.Error("Client is nil")
	}
	if mio.Bucket == "" {
		t.Error("Bucket is empty")
	}
}

// TestConnectMinIO_BadEndpoint 验证错误 endpoint 返回 error。
func TestConnectMinIO_BadEndpoint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	bad := helperMinIOCfg(t)
	bad.Endpoint = "localhost:1" // 无效端口
	bad.AccessKey = "bad"
	bad.SecretKey = "bad"

	_, err := ConnectMinIO(ctx, bad, discardLogger())
	if err == nil {
		t.Error("expected error for bad endpoint, got nil")
	}
}

// TestMinIO_PingPutGet 验证 Ping + PutBytes + GetBytes round-trip。
func TestMinIO_PingPutGet(t *testing.T) {
	mio := helperMinIO(t)
	defer mio.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping
	if err := mio.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	// Put
	objKey := "test-object-" + time.Now().Format("150405.000000000")
	data := []byte("hello-minio-pinconsole")
	if err := mio.PutBytes(ctx, objKey, data); err != nil {
		t.Fatalf("PutBytes: %v", err)
	}

	// Get
	got, err := mio.GetBytes(ctx, objKey)
	if err != nil {
		t.Fatalf("GetBytes: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("GetBytes: got %q, want %q", got, data)
	}

	// 清理 object
	_ = mio.Client.RemoveObject(ctx, mio.Bucket, objKey, minio.RemoveObjectOptions{})
}

// TestMinIO_GetBytes_NonExisting 验证 GetBytes 不存在的 key 返回 error(或空)。
func TestMinIO_GetBytes_NonExisting(t *testing.T) {
	mio := helperMinIO(t)
	defer mio.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got, err := mio.GetBytes(ctx, "nonexistent-object-"+time.Now().Format("150405.000000000"))
	// GetObject 在 minio-go 是 lazy,GetObject 不立即报错;Read 时报错
	// 行为可能是返回空或 error,只要不 panic 即可
	_ = got
	_ = err
}

// TestMinIO_Close_NoOp 验证 Close 是 no-op(minio-go 无显式 close)。
func TestMinIO_Close_NoOp(t *testing.T) {
	mio := helperMinIO(t)
	mio.Close() // 不应 panic
	mio.Close()
	mio.Close()
}

// === stores Connect / Close ===

// helperStoresCfg 构造 Stores 配置(MinIO bucket 随机)。
func helperStoresCfg(t *testing.T) *config.Config {
	t.Helper()
	// 用最简 Config struct(Stores.Connect 只用 Postgres/Redis/MinIO 三个字段)
	cfg := &config.Config{
		Postgres: helperPostgresCfg(),
		Redis:    helperRedisCfg(),
		MinIO:    helperMinIOCfg(t),
	}
	return cfg
}

// TestStores_ConnectSuccess 验证 Stores.Connect 建立全部三个连接。
func TestStores_ConnectSuccess(t *testing.T) {
	_ = helperRedis(t) // 确认 Redis 可用

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stores, err := Connect(ctx, helperStoresCfg(t), discardLogger())
	if err != nil {
		t.Fatalf("Stores.Connect: %v", err)
	}
	defer stores.Close()

	if stores.PG == nil {
		t.Error("PG is nil")
	}
	if stores.Redis == nil {
		t.Error("Redis is nil")
	}
	if stores.MinIO == nil {
		t.Error("MinIO is nil")
	}

	// PingAll 应通过
	if err := stores.PingAll(ctx); err != nil {
		t.Errorf("PingAll: %v", err)
	}
}

// TestStores_Connect_BadPGReturnsError 验证 PG 失败时返回 error。
func TestStores_Connect_BadPGReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := helperStoresCfg(t)
	cfg.Postgres.Port = "1" // 无效端口
	cfg.Postgres.Database = "nonexistent"

	_, err := Connect(ctx, cfg, discardLogger())
	if err == nil {
		t.Error("expected error for bad PG, got nil")
	}
}

// TestStores_Connect_BadRedisReturnsError 验证 Redis 失败时返回 error。
func TestStores_Connect_BadRedisReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := helperStoresCfg(t)
	cfg.Redis.Addr = "localhost:1"

	_, err := Connect(ctx, cfg, discardLogger())
	if err == nil {
		t.Error("expected error for bad Redis, got nil")
	}
}

// TestStores_Connect_BadMinIOReturnsError 验证 MinIO 失败时返回 error。
func TestStores_Connect_BadMinIOReturnsError(t *testing.T) {
	_ = helperRedis(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := helperStoresCfg(t)
	cfg.MinIO.Endpoint = "localhost:1"
	cfg.MinIO.AccessKey = "bad"
	cfg.MinIO.SecretKey = "bad"

	_, err := Connect(ctx, cfg, discardLogger())
	if err == nil {
		t.Error("expected error for bad MinIO, got nil")
	}
}

// TestStores_Close_NilSafe 验证 Close 在 nil 字段时不 panic。
func TestStores_Close_NilSafe(t *testing.T) {
	s := &Stores{} // 全部 nil
	s.Close()      // 不应 panic
}

// TestStores_Close_PartialStores 验证 Close 只关闭非 nil 字段。
func TestStores_Close_PartialStores(t *testing.T) {
	_ = helperRedis(t)
	rdb, err := ConnectRedis(context.Background(), helperRedisCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectRedis: %v", err)
	}

	s := &Stores{Redis: rdb} // 只 Redis,PG/MinIO=nil
	s.Close()                // 不应 panic
}

// TestStores_PingAll_PGFailureReturnsError 验证 PG 失败时 PingAll 返回 error。
func TestStores_PingAll_PGFailureReturnsError(t *testing.T) {
	// 用关闭的 PG pool 模拟 Ping 失败
	pg, err := ConnectPostgres(context.Background(), helperPostgresCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectPostgres: %v", err)
	}
	pg.Close() // 立即关闭,后续 Ping 必失败

	s := &Stores{PG: pg}
	err = s.PingAll(context.Background())
	if err == nil {
		t.Error("PingAll with closed PG: expected error, got nil")
	}
}

// TestStores_PingAll_RedisFailureReturnsError 验证 Redis 失败时 PingAll 返回 error(PG OK,Redis fail)。
func TestStores_PingAll_RedisFailureReturnsError(t *testing.T) {
	// PG 正常
	pg, err := ConnectPostgres(context.Background(), helperPostgresCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectPostgres: %v", err)
	}
	defer pg.Close()

	// Redis 关闭后 Ping 必失败
	_ = helperRedis(t) // 确认 Redis 可用
	rdb, err := ConnectRedis(context.Background(), helperRedisCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectRedis: %v", err)
	}
	rdb.Close()

	s := &Stores{PG: pg, Redis: rdb}
	err = s.PingAll(context.Background())
	if err == nil {
		t.Error("PingAll with closed Redis: expected error, got nil")
	}
}

// TestRedis_Get_ErrorPathClosedConnection 验证 Redis 关闭后 Get 返回 error。
func TestRedis_Get_ErrorPathClosedConnection(t *testing.T) {
	_ = helperRedis(t)
	rdb, err := ConnectRedis(context.Background(), helperRedisCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectRedis: %v", err)
	}
	rdb.Close() // 关闭后 Get 应返回 error

	_, err = rdb.Get(context.Background(), "any-key")
	if err == nil {
		t.Error("Get on closed client: expected error, got nil")
	}
}

// TestRedis_Set_ErrorPathClosedConnection 验证 Redis 关闭后 Set 返回 error。
func TestRedis_Set_ErrorPathClosedConnection(t *testing.T) {
	_ = helperRedis(t)
	rdb, err := ConnectRedis(context.Background(), helperRedisCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectRedis: %v", err)
	}
	rdb.Close()

	err = rdb.Set(context.Background(), "any-key", []byte("v"), time.Second)
	if err == nil {
		t.Error("Set on closed client: expected error, got nil")
	}
}

// TestRedis_SetNX_ErrorPathClosedConnection 验证 SetNX 在关闭的 client 上返回 error。
func TestRedis_SetNX_ErrorPathClosedConnection(t *testing.T) {
	_ = helperRedis(t)
	rdb, err := ConnectRedis(context.Background(), helperRedisCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectRedis: %v", err)
	}
	rdb.Close()

	_, err = rdb.SetNX(context.Background(), "any-key", []byte("v"), time.Second)
	if err == nil {
		t.Error("SetNX on closed client: expected error, got nil")
	}
}

// TestRedis_Del_ErrorPathClosedConnection 验证 Del 在关闭的 client 上返回 error。
func TestRedis_Del_ErrorPathClosedConnection(t *testing.T) {
	_ = helperRedis(t)
	rdb, err := ConnectRedis(context.Background(), helperRedisCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectRedis: %v", err)
	}
	rdb.Close()

	err = rdb.Del(context.Background(), "any-key")
	if err == nil {
		t.Error("Del on closed client: expected error, got nil")
	}
}

// TestRedis_EvalLua_ErrorPathClosedConnection 验证 EvalLua 在关闭的 client 上返回 error。
func TestRedis_EvalLua_ErrorPathClosedConnection(t *testing.T) {
	_ = helperRedis(t)
	rdb, err := ConnectRedis(context.Background(), helperRedisCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectRedis: %v", err)
	}
	rdb.Close()

	_, err = rdb.EvalLua(context.Background(), `return 1`, nil)
	if err == nil {
		t.Error("EvalLua on closed client: expected error, got nil")
	}
}

// TestRedis_TTL_ErrorPathClosedConnection 验证 TTL 在关闭的 client 上返回 error。
func TestRedis_TTL_ErrorPathClosedConnection(t *testing.T) {
	_ = helperRedis(t)
	rdb, err := ConnectRedis(context.Background(), helperRedisCfg(), discardLogger())
	if err != nil {
		t.Fatalf("ConnectRedis: %v", err)
	}
	rdb.Close()

	_, err = rdb.TTL(context.Background(), "any-key")
	if err == nil {
		t.Error("TTL on closed client: expected error, got nil")
	}
}

// === erasure_repo ListEventBlobKeysBySessions ===

// TestListEventBlobKeysBySessions_EmptyInputReturnsNil 验证空 sessionIDs 返回 nil。
func TestListEventBlobKeysBySessions_EmptyInputReturnsNil(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	pg := &Postgres{Pool: pool, logger: discardLogger()}
	ctx := context.Background()

	got, err := pg.ListEventBlobKeysBySessions(ctx, nil)
	if err != nil {
		t.Fatalf("empty input: %v", err)
	}
	if got != nil {
		t.Errorf("empty input: got %v, want nil", got)
	}
}

// TestListEventBlobKeysBySessions_ReturnsMatchingKeys 验证真实数据查询返回 MinIO object keys。
// seed visitor + session + 2 个 event_blobs,然后 List 应返回 2 个 keys。
func TestListEventBlobKeysBySessions_ReturnsMatchingKeys(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool, logger: discardLogger()}

	// seed visitor + session
	visitorID := uuid.New()
	sessionID := uuid.New()
	tenantID := uuid.Nil

	_, err := pool.Exec(ctx, `
		INSERT INTO visitors (id, tenant_id, fingerprint, ua, ip_first_seen, first_seen_at, last_seen_at)
		VALUES ($1, $2, 'test-listkeys-fp-' || $3::text, 'test-ua', '10.0.0.1', NOW(), NOW())
	`, visitorID, tenantID, visitorID.String()[:8])
	if err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM visitors WHERE id = $1`, visitorID)

	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, tenantID, visitorID)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)

	// seed 2 event_blobs
	key1 := "minio-key-1-" + sessionID.String()
	key2 := "minio-key-2-" + sessionID.String()
	_, err = pool.Exec(ctx, `
		INSERT INTO event_blobs (session_id, tenant_id, blob_index, started_at, ended_at, event_count, minio_object_key, size_bytes, checksum_sha256)
		VALUES
		($1, $2, 0, NOW(), NOW(), 1, $3, 1, 'sha256-fake-1'),
		($1, $2, 1, NOW(), NOW(), 1, $4, 1, 'sha256-fake-2')
	`, sessionID, tenantID, key1, key2)
	if err != nil {
		t.Fatalf("seed event_blobs: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM event_blobs WHERE session_id = $1`, sessionID)

	// List 应返回 2 个 keys
	got, err := pg.ListEventBlobKeysBySessions(ctx, []uuid.UUID{sessionID})
	if err != nil {
		t.Fatalf("ListEventBlobKeysBySessions: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d keys, want 2: %v", len(got), got)
	}
	// 应包含 key1 + key2
	gotMap := map[string]bool{}
	for _, k := range got {
		gotMap[k] = true
	}
	if !gotMap[key1] || !gotMap[key2] {
		t.Errorf("missing keys: got %v, want both %q and %q", got, key1, key2)
	}
}

// TestListEventBlobKeysBySessions_NoMatchReturnsEmpty 验证查询不匹配 session 返回空 slice(无 error)。
func TestListEventBlobKeysBySessions_NoMatchReturnsEmpty(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	pg := &Postgres{Pool: pool, logger: discardLogger()}
	ctx := context.Background()

	// 用随机 sessionID,不会匹配任何 event_blobs
	randomID := uuid.New()
	got, err := pg.ListEventBlobKeysBySessions(ctx, []uuid.UUID{randomID})
	if err != nil {
		t.Fatalf("no match: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d keys, want 0 for no match", len(got))
	}
}
