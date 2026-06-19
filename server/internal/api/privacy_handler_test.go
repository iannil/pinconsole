// 1ac 续集测试:GDPR erasure MinIO + Redis 级联(审计 T0-1l-2 + T0-1l-3)。
//
// 验证 privacy.go deleteVisitor handler 的 MinIO RemoveObject + Redis Del 接线:
//   - 调用 DeleteVisitorByFingerprint 前,ListEventBlobKeysBySessions 收集 minio_object_keys
//   - PG 级联删除后,逐个 RemoveObject(失败 best-effort 不阻塞)
//   - 同步 Del Redis claim:session:{sid}
//
// 此前:接线代码完整但无集成测试(1l-2/3 长期 T0)。
package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

// helperMinioIfAvailable 返回真 MinIO client + bucket,不可用时 skip。
func helperMinioIfAvailable(t *testing.T) (*minio.Client, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("需要 MinIO")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("mm_dev", "mm_dev_secret", ""),
		Secure: false,
	})
	if err != nil {
		t.Skipf("minio client: %v", err)
	}
	if _, err := client.ListBuckets(ctx); err != nil {
		t.Skipf("minio ListBuckets: %v", err)
	}
	bucket := "marketing-monitor"
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil || !exists {
		t.Skipf("bucket %s 不存在或不可访问: %v", bucket, err)
	}
	return client, bucket
}

// helperRedisClient 返回真 redis client,不可用时 skip(复用 ws_ratelimit_test.go 的 skipIfNoRedis,
// 但此处需要 *redis.Client 不是 wrapper)。
func helperRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("需要 Redis")
	}
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DialTimeout: 500 * time.Millisecond})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis 不可用(%v),跳过", err)
	}
	return rdb
}

// TestPrivacyDeleteVisitor_MinioCascade — T0-1l-2:
// visitor 关联的 event_blob 在 PG 删除后,对应的 MinIO 对象也被删。
func TestPrivacyDeleteVisitor_MinioCascade(t *testing.T) {
	pool := helperPGIfAvailable(t)
	defer pool.Close()
	mclient, bucket := helperMinioIfAvailable(t)

	ctx := context.Background()
	tenantID := storage.DefaultTenantID
	fp := "1ac-minio-cascade-" + uuid.New().String()[:8]

	// 1. seed MinIO 对象
	minioKey := "1ac-test-blob-" + uuid.New().String()[:8] + ".bin"
	body := []byte("test blob content")
	if _, err := mclient.PutObject(ctx, bucket, minioKey, bytes.NewReader(body), int64(len(body)), minio.PutObjectOptions{}); err != nil {
		t.Fatalf("seed MinIO object: %v", err)
	}
	defer func() {
		// cleanup 残留(测试失败路径)
		mclient.RemoveObject(ctx, bucket, minioKey, minio.RemoveObjectOptions{})
	}()

	// 2. seed visitor + session + event_blob 引用 minioKey
	vid := uuid.New()
	sid := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, vid, tenantID, fp); err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sid, tenantID, vid); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO event_blobs (id, session_id, tenant_id, blob_index, minio_object_key, checksum_sha256, size_bytes, event_count, started_at, ended_at)
		VALUES ($1, $2, $3, 0, $4, 'sha', 100, 1, NOW(), NOW())
	`, uuid.New(), sid, tenantID, minioKey); err != nil {
		t.Fatalf("seed event_blob: %v", err)
	}

	// 3. 构造 admin 调用 deleteVisitor
	adminEmail := "1ac-minio-admin@example.com"
	admin := seedTestUser(t, pool, adminEmail, "admin")
	defer deleteTestUserByEmail(t, pool, adminEmail)

	stores := &storage.Stores{
		PG:    &storage.Postgres{Pool: pool},
		MinIO: &storage.MinIO{Client: mclient, Bucket: bucket},
		// Redis 也需要,因 deleteVisitor 最后会 Del claim keys
		Redis: &storage.Redis{Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"})},
	}
	defer stores.Redis.Close()
	h := &PrivacyHandler{stores: stores, logger: testLogger()}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/privacy/visitor/"+fp, nil)
	c.Params = gin.Params{{Key: "fingerprint", Value: fp}}
	c.Set("user_id", admin.ID)
	// 给 c.Request 一个 context(否则 WithTimeout 会 panic)
	c.Request = c.Request.WithContext(ctx)

	h.deleteVisitor(c)

	if w.Code != http.StatusOK {
		t.Fatalf("deleteVisitor status=%d want 200, body=%s", w.Code, w.Body.String())
	}

	// 4. 验证 MinIO 对象已被删除
	_, err := mclient.StatObject(ctx, bucket, minioKey, minio.StatObjectOptions{})
	if err == nil {
		t.Errorf("MinIO object %q still exists after GDPR delete", minioKey)
	}
}

// TestPrivacyDeleteVisitor_RedisCascade — T0-1l-3:
// visitor 关联的 session claim:session:{sid} Redis key 在删除后也被清。
func TestPrivacyDeleteVisitor_RedisCascade(t *testing.T) {
	pool := helperPGIfAvailable(t)
	defer pool.Close()
	rdb := helperRedisClient(t)
	defer rdb.Close()

	ctx := context.Background()
	tenantID := storage.DefaultTenantID
	fp := "1ac-redis-cascade-" + uuid.New().String()[:8]

	// 1. seed visitor + session
	vid := uuid.New()
	sid := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, vid, tenantID, fp); err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sid, tenantID, vid); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	// 2. seed Redis claim key
	claimK := "claim:session:" + sid.String()
	if err := rdb.Set(ctx, claimK, uuid.New().String(), 5*time.Minute).Err(); err != nil {
		t.Fatalf("seed redis claim: %v", err)
	}
	// 验证 seed
	if exists, _ := rdb.Exists(ctx, claimK).Result(); exists != 1 {
		t.Fatalf("seed redis claim key not set")
	}

	// 3. 调用 deleteVisitor
	adminEmail := "1ac-redis-admin@example.com"
	admin := seedTestUser(t, pool, adminEmail, "admin")
	defer deleteTestUserByEmail(t, pool, adminEmail)

	stores := &storage.Stores{
		PG:    &storage.Postgres{Pool: pool},
		Redis: &storage.Redis{Client: rdb},
	}
	h := &PrivacyHandler{stores: stores, logger: testLogger()}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/privacy/visitor/"+fp, nil).WithContext(ctx)
	c.Params = gin.Params{{Key: "fingerprint", Value: fp}}
	c.Set("user_id", admin.ID)

	h.deleteVisitor(c)

	if w.Code != http.StatusOK {
		t.Fatalf("deleteVisitor status=%d want 200, body=%s", w.Code, w.Body.String())
	}

	// 4. 验证 Redis claim key 已清
	exists, _ := rdb.Exists(ctx, claimK).Result()
	if exists != 0 {
		t.Errorf("Redis claim key %q still exists after GDPR delete", claimK)
	}
}

// 防 pgxpool 类型 unused 引用(辅助函数已用)。
var _ = (*pgxpool.Pool)(nil)
