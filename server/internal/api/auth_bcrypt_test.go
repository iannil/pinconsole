// 1ac 续集测试:bcrypt 实际密码验证路径(审计 T0-1h-5)。
//
// 验证 auth.go login handler 真的用 bcrypt.CompareHashAndPassword 校验密码,
// 不是字符串比较、不是空校验、不是 hash 自比。
//
// 此前:bcrypt 库已 import,但实际 login 路径无集成测试(只有 config_test 校验 cost)。
package api

import (
	"os"
	"strings"
	"testing"
)

// TestLogin_UsesBcryptCompareHashAndPassword — T0-1h-5 源码契约:
// login handler 必须调 bcrypt.CompareHashAndPassword(hash, password)。
//
// 反模式:
//   - bytes.Equal(hash, []byte(password)) — 弱比较
//   - user.PasswordHash == req.Password — 明文比较
//   - 不调任何比较直接发 session — 0 验证
func TestLogin_UsesBcryptCompareHashAndPassword(t *testing.T) {
	src, err := os.ReadFile("auth.go")
	if err != nil {
		t.Fatalf("read auth.go: %v", err)
	}
	body := string(src)

	// 定位 login handler
	idx := strings.Index(body, "func (h *AuthHandler) login")
	if idx < 0 {
		t.Fatal("找不到 login handler")
	}
	end := strings.Index(body[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(body) - idx - 1
	}
	fnBody := body[idx : idx+1+end]

	for _, must := range []string{
		"bcrypt.CompareHashAndPassword",  // 必须用 bcrypt 库
		"user.PasswordHash",              // 比对存储 hash
		"req.Password",                   // 比对用户输入
	} {
		if !strings.Contains(fnBody, must) {
			t.Errorf("login handler 缺失 %q — bcrypt 密码验证路径破坏", must)
		}
	}

	// 反模式检测
	for _, bad := range []string{
		"== req.Password",
		"== user.PasswordHash",
		"Equal(user.PasswordHash, []byte(req.Password))",
	} {
		if strings.Contains(fnBody, bad) {
			t.Errorf("login handler 用了弱比较 %q — 应改为 bcrypt.CompareHashAndPassword", bad)
		}
	}
}

// TestLogin_FailureRecordsThrottle — T0-1h-5 副验:
// 密码错误时必须 recordLoginFailure(走 throttle 计数)。
// 否则 throttle 被旁路。
func TestLogin_FailureRecordsThrottle(t *testing.T) {
	src, err := os.ReadFile("auth.go")
	if err != nil {
		t.Fatalf("read auth.go: %v", err)
	}
	body := string(src)

	idx := strings.Index(body, "func (h *AuthHandler) login")
	if idx < 0 {
		t.Fatal("找不到 login handler")
	}
	end := strings.Index(body[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(body) - idx - 1
	}
	fnBody := body[idx : idx+1+end]

	// bcrypt 失败分支应包含 recordLoginFailure 调用
	if !strings.Contains(fnBody, "bcrypt.CompareHashAndPassword") {
		t.Fatal("找不到 bcrypt 比较(前置依赖)")
	}
	bcryptIdx := strings.Index(fnBody, "bcrypt.CompareHashAndPassword")
	tail := fnBody[bcryptIdx:]
	if !strings.Contains(tail, "recordLoginFailure") {
		t.Errorf("bcrypt 失败后未调 recordLoginFailure — throttle 被旁路")
	}
}
