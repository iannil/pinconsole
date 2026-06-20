// 1ai-c 测试:AuthHandler happy path 行为级测试(用 mock 接口注入)。
//
// 此前 1ag 测了拒绝路径(invalid JSON/locked/no user_id/logout),
// 本切片补 happy path + 错密码 + 用户不存在 3 路径,用接口注入 mock。
//
// 模式:手写 mock 实现 authUserRepo + authRedisStore,
// 构造 AuthHandler{userRepo: mock, redis: mock, ...} 直接注入。
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// ============ mock 实现 ============

// mockUserRepo 是 authUserRepo 的手写 mock。
type mockUserRepo struct {
	byEmail    map[string]*storage.User // email → user(模拟 PG 查询)
	byID       map[uuid.UUID]*storage.User
	byEmailErr error // 模拟 PG error
	byIDErr    error
	mu         sync.Mutex
	emailCalls int
	idCalls    int
}

func (m *mockUserRepo) GetUserByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*storage.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emailCalls++
	if m.byEmailErr != nil {
		return nil, m.byEmailErr
	}
	return m.byEmail[email], nil
}

func (m *mockUserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*storage.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idCalls++
	if m.byIDErr != nil {
		return nil, m.byIDErr
	}
	return m.byID[id], nil
}

// mockRedisStore 是 authRedisStore 的手写 mock。
// 记录调用以让测试断言 recordLoginFailure 被调等。
type mockRedisStore struct {
	mu           sync.Mutex
	data         map[string][]byte
	ttl          map[string]time.Duration
	setCalls     int
	delCalls     int
	evalLuaCalls int
	getCalls     int
	ttlCalls     int
	evalLuaErr   error
}

func newMockRedisStore() *mockRedisStore {
	return &mockRedisStore{
		data: map[string][]byte{},
		ttl:  map[string]time.Duration{},
	}
}

func (m *mockRedisStore) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalls++
	v, ok := m.data[key]
	if !ok {
		return nil, nil // 模拟 storage.Redis 的 nil, nil 行为
	}
	return v, nil
}

func (m *mockRedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setCalls++
	m.data[key] = value
	m.ttl[key] = ttl
	return nil
}

// SetNX 1ai-g:加此方法让 mockRedisStore 满足 claimRedisStore 接口。
// 简单实现:key 不存在则写入返 true,已存在返 false(模拟 Redis SET NX 语义)。
func (m *mockRedisStore) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.data[key]; exists {
		return false, nil
	}
	m.data[key] = value
	m.ttl[key] = ttl
	return true, nil
}

func (m *mockRedisStore) Del(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delCalls++
	delete(m.data, key)
	return nil
}

func (m *mockRedisStore) EvalLua(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evalLuaCalls++
	if m.evalLuaErr != nil {
		return nil, m.evalLuaErr
	}
	// 模拟 INCR(简单实现,不计返回值精度)
	key := keys[0]
	count := 0
	if v, ok := m.data[key]; ok {
		count = int(v[0]) - '0' // 简化:只支持个位数
	}
	count++
	m.data[key] = []byte{byte('0' + count)}
	return int64(count), nil
}

func (m *mockRedisStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ttlCalls++
	return m.ttl[key], nil
}

// ============ helper ============

// mustBcryptHash 用 bcrypt MinCost 生成 hash(测试加速)。
func mustBcryptHash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt hash: %v", err)
	}
	return string(h)
}

// newLoginRequest 构造 POST /api/auth/login 的 *http.Request + ResponseRecorder。
func newLoginRequest(email, password string) (*http.Request, *httptest.ResponseRecorder) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req, httptest.NewRecorder()
}

// ============ 测试 ============

// TestLogin_Success_Returns200_SetCookie_Body — happy path:
// mock user 存在 + 密码匹配 → 200 + Set-Cookie + meResponse body。
func TestLogin_Success_Returns200_SetCookie_Body(t *testing.T) {
	uid := uuid.New()
	correctHash := mustBcryptHash(t, "correct-password")
	user := &storage.User{
		ID:           uid,
		TenantID:     storage.DefaultTenantID,
		Email:        "1aic-success@example.com",
		PasswordHash: correctHash,
		DisplayName:  "Operator 1ai-c",
		Role:         "operator",
	}

	mockUsers := &mockUserRepo{
		byEmail: map[string]*storage.User{user.Email: user},
		byID:    map[uuid.UUID]*storage.User{uid: user},
	}
	mockRedis := newMockRedisStore()

	h := &AuthHandler{
		userRepo:     mockUsers,
		redis:        mockRedis,
		logger:       testLogger(),
		secureCookie: false,
	}
	r := newAuthTestEngine(h)

	req, w := newLoginRequest(user.Email, "correct-password")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	// Set-Cookie 必含 session id(mm_session=...)
	setCookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "mm_session=") || strings.Contains(setCookie, "mm_session=;") {
		t.Errorf("Set-Cookie 未设 session: %s", setCookie)
	}
	// body 必含 meResponse 字段
	body := w.Body.String()
	if !strings.Contains(body, user.ID.String()) {
		t.Errorf("body 缺 user.ID: %s", body)
	}
	if !strings.Contains(body, `"role":"operator"`) {
		t.Errorf("body 缺 role=operator: %s", body)
	}

	// 验证 mock 调用:GetUserByEmail 1 次,Set 1 次(写 session)
	if mockUsers.emailCalls != 1 {
		t.Errorf("GetUserByEmail calls = %d, want 1", mockUsers.emailCalls)
	}
	if mockRedis.setCalls != 1 {
		t.Errorf("Redis.Set calls = %d, want 1 (写 session)", mockRedis.setCalls)
	}
	// Del 1 次(清 throttle 计数,login 成功路径)
	if mockRedis.delCalls != 1 {
		t.Errorf("Redis.Del calls = %d, want 1 (清 throttle)", mockRedis.delCalls)
	}
}

// TestLogin_WrongPassword_Returns401_NoCookie — mock user 存在但密码不匹配 →
// 401 + 无 Set-Cookie + recordLoginFailure 被调。
func TestLogin_WrongPassword_Returns401_NoCookie(t *testing.T) {
	uid := uuid.New()
	user := &storage.User{
		ID:           uid,
		Email:        "1aic-wrong@example.com",
		PasswordHash: mustBcryptHash(t, "correct-password"),
	}

	mockUsers := &mockUserRepo{
		byEmail: map[string]*storage.User{user.Email: user},
	}
	mockRedis := newMockRedisStore()

	h := &AuthHandler{
		userRepo:     mockUsers,
		redis:        mockRedis,
		logger:       testLogger(),
		secureCookie: false,
	}
	r := newAuthTestEngine(h)

	req, w := newLoginRequest(user.Email, "wrong-password")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_credentials") {
		t.Errorf("body = %q, want contains 'invalid_credentials'", w.Body.String())
	}
	// 不应设 cookie(防止攻击者通过 cookie 判断 user 存在)
	if setCookie := w.Header().Get("Set-Cookie"); setCookie != "" {
		t.Errorf("错密码不应设 Set-Cookie: %s", setCookie)
	}

	// EvalLua 1 次(recordLoginFailure)
	if mockRedis.evalLuaCalls != 1 {
		t.Errorf("Redis.EvalLua calls = %d, want 1 (recordLoginFailure)", mockRedis.evalLuaCalls)
	}
	// 不应 Set session
	if mockRedis.setCalls != 0 {
		t.Errorf("Redis.Set calls = %d, want 0 (不应写 session)", mockRedis.setCalls)
	}
}

// TestLogin_UserNotFound_Returns401_RecordsFailure — mock 返回 nil user →
// 401 + recordLoginFailure 被调。
//
// 注:auth.go 实际逻辑是 GetUserByEmail 返回 error(因 pgx.ErrNoRows)时进 401 分支,
// 返回 nil + nil error 时不会触发(因 nil deref)。本测试模拟 error 路径(更接近真实行为)。
func TestLogin_UserNotFound_Returns401_RecordsFailure(t *testing.T) {
	mockUsers := &mockUserRepo{
		byEmail:    map[string]*storage.User{}, // 空,任何 email 都查不到
		byEmailErr: fmt.Errorf("pgx.ErrNoRows: no rows in result set"),
	}
	mockRedis := newMockRedisStore()

	h := &AuthHandler{
		userRepo:     mockUsers,
		redis:        mockRedis,
		logger:       testLogger(),
		secureCookie: false,
	}
	r := newAuthTestEngine(h)

	req, w := newLoginRequest("1aic-missing@example.com", "any-password")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_credentials") {
		t.Errorf("body = %q, want contains 'invalid_credentials'", w.Body.String())
	}
	// 不应 Set session
	if mockRedis.setCalls != 0 {
		t.Errorf("Redis.Set calls = %d, want 0 (用户不存在不应写 session)", mockRedis.setCalls)
	}
	// recordLoginFailure 被调(防字典攻击:对不存在用户也计数)
	if mockRedis.evalLuaCalls != 1 {
		t.Errorf("Redis.EvalLua calls = %d, want 1 (recordLoginFailure for missing user)",
			mockRedis.evalLuaCalls)
	}
}
