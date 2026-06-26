// 1ac 测试:GDPR Art.17 删除 admin-only 校验(审计 T0-1l-5)。
//
// 1ac 测试发现代码 bug:deleteVisitor 此前无 role 校验,任意认证用户(operator 含)
// 可调用 GDPR 删除接口。这是审计 T0-1l-5 的根因 — 不仅是测试缺,代码也缺。
//
// 修复:privacy.go 增加 GetUserByID + role == "admin" 校验。
// 本测试验证修复 + 防回归。
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

// helperPGIfAvailable 返回真 PG pool,不可用时 skip。
func helperPGIfAvailable(t *testing.T) *pgxpool.Pool {
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

// TestPrivacyDeleteVisitor_NonAdmin_Returns403 — T0-1l-5 核心:
// 任意认证用户(operator 含)不能调用 GDPR 删除,必须 admin only。
//
// 此前代码 bug:deleteVisitor 完全没有 role 校验,AuthMiddleware 通过即可调用。
// 1ac 修复后:必须 403。
func TestPrivacyDeleteVisitor_NonAdmin_Returns403(t *testing.T) {
	pool := helperPGIfAvailable(t)
	defer pool.Close()

	// 创建 operator 测试用户
	operatorEmail := "1ac-test-operator@example.com"
	operator := seedTestUser(t, pool, operatorEmail, "operator")
	defer deleteTestUserByEmail(t, pool, operatorEmail)

	// 构造 PrivacyHandler 用真 PG
	stores := &storage.Stores{
		PG:    &storage.Postgres{Pool: pool},
		Redis: nil, // admin check 在 Redis 调用前
		MinIO: nil,
	}
	h := &PrivacyHandler{stores: stores, logger: testLogger()}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete,
		"/api/privacy/visitor/any-fingerprint", nil)
	c.Params = gin.Params{{Key: "fingerprint", Value: "any-fingerprint"}}
	c.Set("user_id", operator.ID)

	h.deleteVisitor(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("non-admin DELETE: status=%d want 403 (admin_required), body=%s",
			w.Code, w.Body.String())
	}
	if !contains(w.Body.String(), "admin_required") {
		t.Errorf("body should contain admin_required, got: %s", w.Body.String())
	}
}

// TestPrivacyDeleteVisitor_Admin_OK — admin 角色应通过 role check(后续 visitor lookup
// 失败因 fingerprint 不存在,返回 200 visitor_not_found,证明 role check 通过)。
func TestPrivacyDeleteVisitor_Admin_OK(t *testing.T) {
	pool := helperPGIfAvailable(t)
	defer pool.Close()

	adminEmail := "1ac-test-admin@example.com"
	admin := seedTestUser(t, pool, adminEmail, "admin")
	defer deleteTestUserByEmail(t, pool, adminEmail)

	stores := &storage.Stores{
		PG: &storage.Postgres{Pool: pool},
	}
	h := &PrivacyHandler{stores: stores, logger: testLogger()}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete,
		"/api/privacy/visitor/nonexistent-fingerprint", nil)
	c.Params = gin.Params{{Key: "fingerprint", Value: "nonexistent-fingerprint"}}
	c.Set("user_id", admin.ID)

	h.deleteVisitor(c)

	// admin 通过 role check,visitor 不存在 → 200 visitor_not_found(幂等)
	if w.Code != http.StatusOK {
		t.Errorf("admin DELETE nonexistent fp: status=%d want 200 (visitor_not_found idempotent), body=%s",
			w.Code, w.Body.String())
	}
	if !contains(w.Body.String(), "visitor_not_found") {
		t.Errorf("body should contain visitor_not_found, got: %s", w.Body.String())
	}
}

// TestPrivacyDeleteVisitor_NoUserID_Returns401 — 兜底:ctx 缺 user_id → 401。
func TestPrivacyDeleteVisitor_NoUserID_Returns401(t *testing.T) {
	pool := helperPGIfAvailable(t)
	defer pool.Close()

	stores := &storage.Stores{PG: &storage.Postgres{Pool: pool}}
	h := &PrivacyHandler{stores: stores, logger: testLogger()}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete,
		"/api/privacy/visitor/any", nil)
	c.Params = gin.Params{{Key: "fingerprint", Value: "any"}}
	// 不 Set user_id

	h.deleteVisitor(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("no user_id: status=%d want 401 (not_authenticated)", w.Code)
	}
}

// seedTestUser 在 users 表插入测试用户,返回 User。
func seedTestUser(t *testing.T, pool *pgxpool.Pool, email, role string) *storage.User {
	t.Helper()
	ctx := context.Background()
	tenantID := storage.DefaultTenantID
	var u storage.User
	err := pool.QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, display_name, role)
		VALUES ($1, $2, 'test-hash', 'Test User', $3)
		ON CONFLICT (tenant_id, email) DO UPDATE SET role = EXCLUDED.role
		RETURNING id, tenant_id, email, password_hash, display_name, role, created_at, updated_at
	`, tenantID, email, role).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash,
		&u.DisplayName, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("seed user %s: %v", email, err)
	}
	return &u
}

func deleteTestUserByEmail(t *testing.T, pool *pgxpool.Pool, email string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email); err != nil {
		t.Logf("cleanup user %s: %v", email, err)
	}
}

// 防 uuid 包被裁剪。
var _ = uuid.New
