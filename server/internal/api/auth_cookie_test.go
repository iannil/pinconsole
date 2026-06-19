// 1ac 测试:cookie 安全属性(审计 T0-1k-6 prod Secure + T0-1h-1 HttpOnly)。
//
// 验证 auth.go 两处 SetCookie 调用的属性:
//  1. 第 6 参数(secure)= h.secureCookie(可变,prod 模式 router.go 传 true)
//  2. 第 7 参数(httpOnly)= 硬编码 true(防 XSS 偷 cookie)
//
// 这两个属性是 deep-audit P0/P1 修复点。源码契约测试捕获重构回归。
package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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

// TestSetSessionCookie_Behavioral_ProdMode — 1ae R3b 升级:
// 真调 setSessionCookie + 断言 Set-Cookie header 属性(Secure/HttpOnly/SameSite/MaxAge)。
//
// 此前的源码契约测试只能 grep "h.secureCookie, true)" 字符串,不能捕获:
// - SetSameSite 被误删(CSRF 风险)
// - SetCookie 第 7 参数(httpOnly)被改为 false
// - maxAge 计算错误
//
// 行为级测试通过 gin 的真 cookie 序列化,验证 header 输出。
func TestSetSessionCookie_Behavioral_ProdMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	h := &AuthHandler{secureCookie: true}
	h.setSessionCookie(c, "test-session-id", int(sessionTTL.Seconds()))

	setCookie := w.Header().Get("Set-Cookie")
	if setCookie == "" {
		t.Fatal("Set-Cookie header missing")
	}

	// 必须包含 session id
	if !strings.Contains(setCookie, "mm_session=test-session-id") {
		t.Errorf("Set-Cookie 不含 session id: %s", setCookie)
	}

	// 必须 Secure(prod 模式)
	if !strings.Contains(setCookie, "; Secure") {
		t.Errorf("Set-Cookie 缺 Secure 属性(prod 模式必须): %s", setCookie)
	}

	// 必须 HttpOnly
	if !strings.Contains(setCookie, "; HttpOnly") {
		t.Errorf("Set-Cookie 缺 HttpOnly 属性(XSS 防护): %s", setCookie)
	}

	// 必须 SameSite=Lax
	if !strings.Contains(setCookie, "; SameSite=Lax") {
		t.Errorf("Set-Cookie 缺 SameSite=Lax(CSRF 防护): %s", setCookie)
	}

	// 必须 MaxAge=86400(24h)
	if !strings.Contains(setCookie, "; Max-Age=86400") {
		t.Errorf("Set-Cookie 缺 Max-Age=86400(24h): %s", setCookie)
	}
}

// TestSetSessionCookie_Behavioral_DevMode — 1ae R3b 补充:
// dev 模式 Secure=false(让 HTTP localhost 能用 cookie)。
func TestSetSessionCookie_Behavioral_DevMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	h := &AuthHandler{secureCookie: false}
	h.setSessionCookie(c, "dev-session", 3600)

	setCookie := w.Header().Get("Set-Cookie")

	// dev 模式不能设 Secure(否则 HTTP localhost 登录失败)
	if strings.Contains(setCookie, "; Secure") {
		t.Errorf("dev 模式 Set-Cookie 不应含 Secure: %s", setCookie)
	}

	// HttpOnly 必须仍 true(dev 模式也不能弱化)
	if !strings.Contains(setCookie, "; HttpOnly") {
		t.Errorf("dev 模式 Set-Cookie 仍必须有 HttpOnly: %s", setCookie)
	}
}
