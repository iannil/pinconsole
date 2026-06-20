// 1ab P1-5:TrustedProxies + XFF 行为 integration test
// 验证 SetTrustedProxies 配置下 c.ClientIP() 的实际行为(非 mock)。
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTrustedProxyTestRouter(trustedProxies []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		panic(err)
	}
	// 测试 endpoint 直接返回 ClientIP,验证 SetTrustedProxies 效果
	r.GET("/_test/clientip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})
	return r
}

func TestTrustedProxies_Empty_XFFIgnored(t *testing.T) {
	// 不信任任何反代 → XFF 头被忽略,ClientIP 用 RemoteAddr
	r := newTrustedProxyTestRouter([]string{})

	req := httptest.NewRequest(http.MethodGet, "/_test/clientip", nil)
	req.RemoteAddr = "203.0.113.5:12345"          // 模拟直接暴露下的客户端 IP
	req.Header.Set("X-Forwarded-For", "10.0.0.1") // 伪造 XFF
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// 应返回 RemoteAddr 的 IP,XFF 被忽略
	if got := w.Body.String(); got != "203.0.113.5" {
		t.Errorf("ClientIP = %q, want '203.0.113.5' (XFF should be ignored when no trusted proxies)", got)
	}
}

func TestTrustedProxies_TrustedProxy_XFFParsed(t *testing.T) {
	// 信任 127.0.0.1 → RemoteAddr 来自 127.0.0.1 时,XFF 被采用
	r := newTrustedProxyTestRouter([]string{"127.0.0.1"})

	req := httptest.NewRequest(http.MethodGet, "/_test/clientip", nil)
	req.RemoteAddr = "127.0.0.1:12345"                // 反代的源 IP
	req.Header.Set("X-Forwarded-For", "203.0.113.99") // 真实客户端 IP
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Body.String(); got != "203.0.113.99" {
		t.Errorf("ClientIP = %q, want '203.0.113.99' (XFF should be parsed when proxy is trusted)", got)
	}
}

func TestTrustedProxies_UntrustedProxy_XFFIgnored(t *testing.T) {
	// 信任 10.0.0.0/8,但请求来自 127.0.0.1(非信任范围)→ XFF 应被忽略
	r := newTrustedProxyTestRouter([]string{"10.0.0.0/8"})

	req := httptest.NewRequest(http.MethodGet, "/_test/clientip", nil)
	req.RemoteAddr = "127.0.0.1:12345" // 非信任范围
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// gin 行为:RemoteAddr 非信任时,ClientIP 返回 RemoteAddr 的 IP
	if got := w.Body.String(); got != "127.0.0.1" {
		t.Errorf("ClientIP = %q, want '127.0.0.1' (XFF should be ignored from untrusted proxy)", got)
	}
}

func TestTrustedProxies_MultiHop_XFFChainParsed(t *testing.T) {
	// gin ClientIP 从右向左走 XFF,遇到非信任 IP 就停。
	// XFF "203.0.113.99, 10.0.0.1, 10.0.0.2" + RemoteAddr=127.0.0.1 (信任):
	//   - 127.0.0.1 trusted → 进 XFF
	//   - 10.0.0.2 non-trusted → 返回 10.0.0.2(右起第一个非信任)
	// 这是 gin 的防伪造策略:即使 XFF 里有 client IP,也不轻易相信。
	r := newTrustedProxyTestRouter([]string{"127.0.0.1"})

	req := httptest.NewRequest(http.MethodGet, "/_test/clientip", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.99, 10.0.0.1, 10.0.0.2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Body.String(); got != "10.0.0.2" {
		t.Errorf("ClientIP = %q, want '10.0.0.2' (rightmost non-trusted in XFF chain)", got)
	}
}

func TestTrustedProxies_FullChainTrusted_LeftmostClientReturned(t *testing.T) {
	// 信任整条链(127.0.0.1 + 10.0.0.0/8)→ 从右向左全部跳过 → 返回最左端真实 client IP
	r := newTrustedProxyTestRouter([]string{"127.0.0.1", "10.0.0.0/8"})

	req := httptest.NewRequest(http.MethodGet, "/_test/clientip", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.99, 10.0.0.1, 10.0.0.2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Body.String(); got != "203.0.113.99" {
		t.Errorf("ClientIP = %q, want '203.0.113.99' (leftmost when full chain trusted)", got)
	}
}

func TestTrustedProxies_Nil_XFFIgnored(t *testing.T) {
	// nil 等同空 → XFF 被忽略
	r := newTrustedProxyTestRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/_test/clientip", nil)
	req.RemoteAddr = "203.0.113.5:12345"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Body.String(); got != "203.0.113.5" {
		t.Errorf("ClientIP = %q, want '203.0.113.5' (XFF should be ignored when TrustedProxies is nil)", got)
	}
}

func TestTrustedProxies_ConfigDrivenIntegration(t *testing.T) {
	// 端到端验证:从 config 字符串解析 → 调用 SetTrustedProxies → XFF 行为
	// 模拟 main.go 的 trustedProxies 解析逻辑 + 路由配置
	cfgTrustedProxies := "127.0.0.1,10.0.0.0/8"

	// 复刻 main.go:94-101 的解析逻辑
	var trustedProxies []string
	for _, p := range strings.Split(cfgTrustedProxies, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			trustedProxies = append(trustedProxies, p)
		}
	}

	r := newTrustedProxyTestRouter(trustedProxies)

	// 测试 1:127.0.0.1 是信任的 → XFF 被采用
	req1 := httptest.NewRequest(http.MethodGet, "/_test/clientip", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	req1.Header.Set("X-Forwarded-For", "198.51.100.7")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if got := w1.Body.String(); got != "198.51.100.7" {
		t.Errorf("127.0.0.1 trusted: ClientIP = %q, want '198.51.100.7'", got)
	}

	// 测试 2:10.0.0.5 在 10.0.0.0/8 内 → XFF 被采用
	req2 := httptest.NewRequest(http.MethodGet, "/_test/clientip", nil)
	req2.RemoteAddr = "10.0.0.5:12345"
	req2.Header.Set("X-Forwarded-For", "198.51.100.8")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if got := w2.Body.String(); got != "198.51.100.8" {
		t.Errorf("10.0.0.5 trusted: ClientIP = %q, want '198.51.100.8'", got)
	}

	// 测试 3:203.0.113.x 非信任 → XFF 被忽略
	req3 := httptest.NewRequest(http.MethodGet, "/_test/clientip", nil)
	req3.RemoteAddr = "203.0.113.99:12345"
	req3.Header.Set("X-Forwarded-For", "198.51.100.9")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if got := w3.Body.String(); got != "203.0.113.99" {
		t.Errorf("203.0.113.99 untrusted: ClientIP = %q, want '203.0.113.99'", got)
	}
}
