// 1ac 测试:cookie 安全属性(审计 T0-1k-6 prod Secure + T0-1h-1 HttpOnly)。
//
// 验证 auth.go 两处 SetCookie 调用的属性:
//  1. 第 6 参数(secure)= h.secureCookie(可变,prod 模式 router.go 传 true)
//  2. 第 7 参数(httpOnly)= 硬编码 true(防 XSS 偷 cookie)
//
// 这两个属性是 deep-audit P0/P1 修复点。源码契约测试捕获重构回归。
package api

import (
	"os"
	"strings"
	"testing"
)

// TestAuthCookie_HttpOnly_AlwaysTrue — T0-1h-1: cookie HttpOnly 必须为 true。
//
// gin SetCookie 签名:SetCookie(name, value string, maxAge int, path, domain string, secure bool, httpOnly bool)
// auth.go 调用必须以 ", true)" 结尾(httpOnly 硬编码)。
func TestAuthCookie_HttpOnly_AlwaysTrue(t *testing.T) {
	src, err := os.ReadFile("auth.go")
	if err != nil {
		t.Fatalf("read auth.go: %v", err)
	}
	body := string(src)

	// 找所有 c.SetCookie 调用
	lines := strings.Split(body, "\n")
	setCookieLines := []string{}
	for i, line := range lines {
		if strings.Contains(line, "c.SetCookie(") {
			// SetCookie 调用可能跨行,抓取完整语句
			start := i
			end := i
			for j := i; j < len(lines); j++ {
				if strings.Contains(lines[j], "sessionCookieName") {
					end = j
				}
				if strings.Contains(lines[j], ")") {
					end = j
					break
				}
			}
			setCookieLines = append(setCookieLines, strings.Join(lines[start:end+1], "\n"))
		}
	}

	if len(setCookieLines) < 2 {
		t.Fatalf("auth.go 应有 ≥2 处 SetCookie 调用(login + logout),找到 %d", len(setCookieLines))
	}

	for i, call := range setCookieLines {
		// HttpOnly 是最后一个参数,必须为 true
		if !strings.Contains(call, ", true)") {
			t.Errorf("SetCookie 调用 #%d 的 httpOnly 不是 true:\n%s", i+1, call)
		}
	}
}

// TestAuthCookie_SecureFlag_Threading — T0-1k-6: prod 模式 cookie Secure=true。
//
// auth.go 的 SetCookie 第 6 参数(secure)必须用 h.secureCookie,
// 不能硬编码 false(否则 prod 模式 Secure flag 失效,HTTP 泄露 cookie)。
func TestAuthCookie_SecureFlag_Threading(t *testing.T) {
	src, err := os.ReadFile("auth.go")
	if err != nil {
		t.Fatalf("read auth.go: %v", err)
	}
	body := string(src)

	// 每处 SetCookie 都应该用 h.secureCookie(不是 true/false 硬编码)
	if !strings.Contains(body, "h.secureCookie, true)") {
		t.Errorf("auth.go 缺失 h.secureCookie 模式 — Secure flag 可能被硬编码")
	}

	// 拒绝硬编码 false 的安全反模式
	if strings.Contains(body, "c.SetCookie(") &&
		(strings.Contains(body, ", false, true)") || strings.Contains(body, ", false,true)")) {
		t.Errorf("auth.go 存在硬编码 secure=false — prod 模式下 HTTP 泄露 cookie 风险")
	}
}

// TestAuthCookie_SecureCookieField — 验证 AuthHandler.secureCookie 字段存在,
// 且通过 NewAuthHandler 注入(防字段重命名破坏 prod gating)。
func TestAuthCookie_SecureCookieField(t *testing.T) {
	src, err := os.ReadFile("auth.go")
	if err != nil {
		t.Fatalf("read auth.go: %v", err)
	}
	body := string(src)

	for _, must := range []string{
		"secureCookie",          // 字段名
		"NewAuthHandler",        // 构造器
		"secureCookie bool",     // 字段类型
		"h.secureCookie",        // 使用方式
	} {
		if !strings.Contains(body, must) {
			t.Errorf("auth.go 缺失 %q — Secure flag threading 破坏", must)
		}
	}
}

// TestRouter_AuthHandlerEnvGating — 验证 router.go 用 Env == "prod" gating secureCookie。
//
// 关联 deep-audit P0-1:`SERVER_ENV=prod` 才允许走 prod 路径,此时 Secure 必须 true。
// 源码契约:router.go 必须有 `NewAuthHandler(opts.Stores, opts.Logger, opts.Env == "prod")`。
func TestRouter_AuthHandlerEnvGating(t *testing.T) {
	src, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}
	body := string(src)

	expected := `NewAuthHandler(opts.Stores, opts.Logger, opts.Env == "prod")`
	if !strings.Contains(body, expected) {
		t.Errorf("router.go 缺失 prod-mode gating:\n  want: %s\n  让 Secure cookie 仅 prod 模式启用(dev 模式 HTTPS 证书不常见,Secure=true 会让 dev 登录失败)", expected)
	}
}

// TestAuthCookie_SessionCookieName_Constant — 防 cookie 名变更破坏浏览器识别。
func TestAuthCookie_SessionCookieName_Constant(t *testing.T) {
	if sessionCookieName != "mm_session" {
		t.Errorf("sessionCookieName = %q, want %q", sessionCookieName, "mm_session")
	}
}

// TestAuthCookie_SessionTTL_Constant — 防 TTL 误改(deep-audit 24h 决策)。
func TestAuthCookie_SessionTTL_Constant(t *testing.T) {
	if sessionTTL != 24*60*60*1_000_000_000 { // 24h in nanoseconds
		t.Errorf("sessionTTL = %v, want 24h", sessionTTL)
	}
}
