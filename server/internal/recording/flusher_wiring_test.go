// 1ad 续集测试:1d Flusher + R2 上传接线源码契约(审计 T1-1d-1/3)。
// 1ae R3d 扩展:加真 flushSession 行为级测试(用 docker Redis/MinIO/PG 集成)。
//
// T1-1d-1: flushSession 必须调 MinIO.PutObject 上传 blob
// T1-1d-3: Flusher 必须 Register/Unregister 完整生命周期(ws.go 调用)
//
// T1-1d-2(MinIO RemoveObject 补偿)已在 observability_wiring_test.go cover。
// T1-1d-4/5 已在 1ac cover(GC + erasure MinIO)。
package recording

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

// minioCredentialsType 是 *credentials.Credentials 的别名(简化签名)。
type minioCredentialsType = credentials.Credentials

// newMinioCredentialsStatic 用 credentials.NewStaticCredentials 创建静态凭证。
func newMinioCredentialsStatic(accessKey, secretKey string) *minioCredentialsType {
	return credentials.NewStatic(accessKey, secretKey, "", credentials.SignatureDefault)
}

// Test1d_Flusher_HasRegisterUnregister — T1-1d-3:
// Flusher 必须有 Register + Unregister 方法,且 Unregister 同步 flush 最后一批。
func Test1d_Flusher_HasRegisterUnregister(t *testing.T) {
	src := mustReadFile(t, "stream.go")

	for _, must := range []string{
		"func (f *Flusher) Register(",
		"func (f *Flusher) Unregister(",
		"func (f *Flusher) Start(",
		"func (f *Flusher) Stop(",
		"func (f *Flusher) tick(",
		"func (f *Flusher) flushSession(",
	} {
		if !strings.Contains(src, must) {
			t.Errorf("stream.go 缺失 %q — Flusher API 破坏", must)
		}
	}
}

// Test1d_FlushSession_WiresMinIOPut — T1-1d-1:
// flushSession 必须调 MinIO PutBytes(或等价 PutObject)上传 blob。
func Test1d_FlushSession_WiresMinIOPut(t *testing.T) {
	src := mustReadFile(t, "stream.go")
	idx := strings.Index(src, "func (f *Flusher) flushSession")
	if idx < 0 {
		t.Fatal("找不到 flushSession")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	if !strings.Contains(fnBody, "MinIO.PutBytes(") {
		t.Errorf("flushSession 缺失 MinIO.PutBytes 调用 — R2 上传破坏")
	}
}

// Test1d_FlushSession_WiresPGInsertAndRedisXTRIM — T1-1d-1 副验:
// flushSession 完整流程:MinIO Put → PG INSERT event_blobs → Redis XTRIM。
func Test1d_FlushSession_WiresPGInsertAndRedisXTRIM(t *testing.T) {
	src := mustReadFile(t, "stream.go")
	idx := strings.Index(src, "func (f *Flusher) flushSession")
	if idx < 0 {
		t.Fatal("找不到 flushSession")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	for _, must := range []string{
		"CreateEventBlob", // PG INSERT
		"XTRIM",           // Redis stream trim
	} {
		if !strings.Contains(fnBody, must) {
			t.Errorf("flushSession 缺失 %q — flush 流程破坏(MinIO Put → PG INSERT → Redis XTRIM)", must)
		}
	}
}

// Test1d_GC_WiresChatAndCommandsCleanup — T1-1d-4 副验(部分已在 1ac cover):
// gc.go runOnce 必须清 chat_messages + co_browsing_commands(不只 event_blobs)。
func Test1d_GC_WiresChatAndCommandsCleanup(t *testing.T) {
	src := mustReadFile(t, "gc.go")
	idx := strings.Index(src, "func (g *GC) runOnce")
	if idx < 0 {
		t.Fatal("找不到 runOnce")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	for _, must := range []string{
		"ListChatMessagesOlderThan",
		"DeleteChatMessagesByID",
		"DeleteCoBrowsingCommandsOlderThan",
		"DeleteSessionsEndedBefore",
		"DeleteVisitorsLastSeenBefore",
	} {
		if !strings.Contains(fnBody, must) {
			t.Errorf("runOnce 缺失 %q — GC 5 表清理破坏", must)
		}
	}
}

// helperRedisClient 返回真 Redis client(连 docker),不可用时 skip。
func helperRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("需要 Redis")
	}
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:7079", DB: 0})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		t.Skipf("Redis 不可用(%v),跳过行为级测试", err)
	}
	return rdb
}

// helperMinIOClient 返回真 MinIO client(连 docker),不可用时 skip。
func helperMinIOClient(t *testing.T) (*minio.Client, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("需要 MinIO")
	}
	// 与 docker-compose.yml 默认值一致
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "mm_dev"
	}
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "mm_dev_secret"
	}
	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "pinconsole"
	}
	mio, err := minio.New("localhost:7020", &minio.Options{
		Creds:  minioCredentialsStatic(accessKey, secretKey),
		Secure: false,
	})
	if err != nil {
		t.Skipf("MinIO client 创建失败(%v),跳过", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := mio.ListBuckets(ctx); err != nil {
		t.Skipf("MinIO 不可用(%v),跳过", err)
	}
	// 确保 bucket 存在
	if err := mio.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		// 已存在忽略
	}
	return mio, bucket
}

// minioCredentialsStatic 用 credentials.NewStaticCredentials 创建静态凭证。
// 单独抽出便于 1ae R3d 测试在不依赖全局 env 的情况下访问 MinIO。
func minioCredentialsStatic(accessKey, secretKey string) *minioCredentialsType {
	return newMinioCredentialsStatic(accessKey, secretKey)
}

// TestFlushSession_Behavioral_WiresMinioAndRedisAndPG — 1ae R3d 升级:
// 真调 FlushSession + 真 Redis/MinIO/PG,验证:
//   1. MinIO 上传 blob 成功(ListObjects 能查到)
//   2. Redis stream 被 XTRIM(trim 后长度 ≤ TrimKeep)
//   3. PG event_blobs 表新增一行
//
// 此前的源码契约测试只 grep "MinIO.PutBytes(" 字符串,不能捕获:
// - 调用顺序错(MinIO 写后忘了 PG INSERT)
// - 参数错(传错 sessionID 导致 object key 错位)
// - 错误处理 broken(MinIO 失败仍假装成功)
//
// 行为级测试通过真集成验证端到端。
//
// 实现:用 docker Redis/MinIO/PG + 真 Stream + 真 Flusher,seed entries → flushSession → 验证副作用。
func TestFlushSession_Behavioral_WiresMinioAndRedisAndPG(t *testing.T) {
	rdb := helperRedisClient(t)
	defer rdb.Close()

	mio, bucket := helperMinIOClient(t)
	_ = mio
	_ = bucket

	// PG pool
	pgPool := helperPGPoolForRecording(t)
	defer pgPool.Close()

	ctx := context.Background()

	// 准备 unique session ID(避免与其他测试冲突)
	sessionID := uuid.New()
	tenantID := storage.DefaultTenantID

	// 清理残留
	rdb.Del(ctx, fmt.Sprintf("stream:session:%s", sessionID))
	_, _ = pgPool.Exec(ctx, `DELETE FROM event_blobs WHERE session_id = $1`, sessionID)
	_, _ = pgPool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	_, _ = pgPool.Exec(ctx, `DELETE FROM visitors WHERE id = $1`, sessionID)
	defer func() {
		rdb.Del(ctx, fmt.Sprintf("stream:session:%s", sessionID))
		_, _ = pgPool.Exec(ctx, `DELETE FROM event_blobs WHERE session_id = $1`, sessionID)
		_, _ = pgPool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
		_, _ = pgPool.Exec(ctx, `DELETE FROM visitors WHERE id = $1`, sessionID)
	}()

	// Seed PG visitor + session(FK 要求)
	visitorID := uuid.New()
	// 1ae R4 fix: 用 unique fingerprint 防止跨 test run 的 unique constraint 冲突
	uniqueFP := "1ae-flush-test-" + uuid.New().String()[:8]
	_, err := pgPool.Exec(ctx, `
		INSERT INTO visitors (id, tenant_id, fingerprint, ua, ip_first_seen, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, 'test-ua', '10.0.0.1', NOW(), NOW())
	`, visitorID, tenantID, uniqueFP)
	if err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	_, err = pgPool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sessionID, tenantID, visitorID)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}

	// Stream + Flusher 构造
	logger := slogNewNop()
	stream := NewStream(rdb, logger)
	stores := &storage.Stores{
		PG:    &storage.Postgres{Pool: pgPool},
		Redis: &storage.Redis{Client: rdb},
		MinIO: &storage.MinIO{Client: mio, Bucket: bucket},
	}
	flusher := NewFlusher(Config{
		EventThreshold: 1,
		Interval:       time.Hour, // 不让 tick 触发
		TrimKeep:       0,         // flush 后清空(测试用)
	}, stream, stores, logger)

	// Seed 3 entries 到 stream
	for i := 0; i < 3; i++ {
		msg := []byte(fmt.Sprintf("test-envelope-%d", i))
		if err := stream.Append(ctx, sessionID, msg); err != nil {
			t.Fatalf("append entry %d: %v", i, err)
		}
	}

	// 验证 seed 成功
	streamLen, err := stream.Len(ctx, sessionID)
	if err != nil {
		t.Fatalf("stream len: %v", err)
	}
	if streamLen != 3 {
		t.Fatalf("stream len after seed = %d, want 3", streamLen)
	}

	// 直接调 flushSession(私有,通过 Register 暴露)
	as := &activeSession{
		sessionID:             sessionID,
		tenantID:              tenantID,
		lastFlushedEventCount: 0,
		blobIndex:             0,
	}
	// flushSession 是私有方法;通过 tick 间接调用
	// 简化:直接用反射或导出 helper
	// 这里改用更简单方式:用 flusher.tick() 触发
	flusher.active = map[uuid.UUID]*activeSession{sessionID: as}
	flusher.tick(ctx)

	// 验证 1:MinIO 上有新 object
	objectKey := fmt.Sprintf("sessions/%s/0.msgpack", sessionID)
	_, err = mio.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		t.Errorf("MinIO StatObject 失败 — flushSession 没上传 blob: %v", err)
	}

	// 验证 2:Redis stream 被 trim(len <= TrimKeep)
	streamLenAfter, _ := stream.Len(ctx, sessionID)
	if streamLenAfter > 0 {
		t.Errorf("stream len after flush = %d, want 0(XTRIM should clear)", streamLenAfter)
	}

	// 验证 3:PG event_blobs 新增一行
	var pgCount int
	err = pgPool.QueryRow(ctx, `SELECT COUNT(*) FROM event_blobs WHERE session_id = $1`, sessionID).Scan(&pgCount)
	if err != nil {
		t.Fatalf("count event_blobs: %v", err)
	}
	if pgCount != 1 {
		t.Errorf("event_blobs count after flush = %d, want 1", pgCount)
	}

	// 清理 MinIO object
	_ = mio.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{})
}

// helperPGPoolForRecording 返回 PG pool(给 recording 测试用)。
func helperPGPoolForRecording(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("需要 PG")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, "postgres://mm:mm_dev@localhost:7032/pinconsole?sslmode=disable")
	if err != nil {
		t.Skipf("PG 不可用(%v),跳过", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("PG ping 失败(%v),跳过", err)
	}
	return pool
}

// slogNewNop 返回不输出日志的 slog.Logger。
func slogNewNop() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: (slog.Level)(100)}))
}
