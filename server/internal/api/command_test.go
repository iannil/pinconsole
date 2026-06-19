// 1k 测试：popup URL scheme 白名单 (P0-8) + 1ac T0-1k-3 OperatorID 审计完整性。
package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsURLSchemeAllowed(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		// 允许
		{"http URL", "http://example.com/path", true},
		{"https URL", "https://example.com/path", true},
		{"uppercase HTTPS", "HTTPS://Example.com", true},
		{"mixed case Http", "Http://example.com", true},
		{"empty string", "", true},
		{"relative path", "/internal/path", true},
		{"relative no slash", "page.html", true},
		{"protocol-relative", "//example.com/path", true},

		// 拒绝
		{"javascript scheme", "javascript:alert(1)", false},
		{"javascript uppercase", "JAVASCRIPT:alert(1)", false},
		{"javascript mixed", "JavaScript:fetch('/evil')", false},
		{"data scheme", "data:text/html,<script>alert(1)</script>", false},
		{"vbscript scheme", "vbscript:msgbox(1)", false},
		{"file scheme", "file:///etc/passwd", false},
		{"about blank", "about:blank", false},
		{"mailto (unknown scheme)", "mailto:foo@bar.com", false}, // 严格白名单
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isURLSchemeAllowed(tt.url)
			if got != tt.want {
				t.Errorf("isURLSchemeAllowed(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

// 1s 安全修复测试:sanitize 函数
func TestSanitizeCommandType(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"cursor_highlight", "cursor_highlight"},
		{"click", "click"},
		{"navigate", "navigate"},
		{"unknown_type", "unknown_type"}, // 短未知类型原样
		{"", ""},
		{strings.Repeat("x", 100), strings.Repeat("x", 50) + "...(truncated)"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := sanitizeCommandType(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeCommandType(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeURLForLog(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain url", "https://example.com/path", "https://example.com/path"},
		{"url with query", "https://example.com/path?token=secret&user=foo", "https://example.com/path"},
		{"url with fragment", "https://example.com/path#section", "https://example.com/path"},
		{"url with both", "https://example.com/path?a=1#section", "https://example.com/path"},
		{"javascript scheme", "javascript:alert(1)", "javascript:alert(1)"}, // 不做 scheme 校验,仅去 query
		{"very long url", "https://example.com/" + strings.Repeat("a", 250), "https://example.com/" + strings.Repeat("a", 180) + "...(truncated)"},
		{"relative", "/path?a=1", "/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeURLForLog(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeURLForLog(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestOperatorID_UsesCallerUID_NotClientIP — 1ac T0-1k-3 回归测试。
//
// deep-audit P0-3 修复:command.go 写 PG 审计时,OperatorID 必须用 callerUID.String()
// (来自 AuthMiddleware 注入的 user_id),不能用 c.ClientIP() — 否则审计完整性失守
// (IP 不稳定、可伪造、不指向具体运营)。
//
// 此为源码契约测试:验证 command.go 的 OperatorID 赋值来源是 callerUID 而非 ClientIP。
// 不做完整 handler 集成测试(需 PG fixture),用源码检查捕获重构回归。
func TestOperatorID_UsesCallerUID_NotClientIP(t *testing.T) {
	// 找到 command.go 源文件(同目录)
	srcPath := filepath.Join(".", "command.go")
	if _, err := os.Stat(srcPath); err != nil {
		// 测试在不同 cwd 运行时,从 runtime.Caller 找源文件路径
		abs, _ := filepath.Abs(srcPath)
		t.Skipf("command.go not found at %s (cwd mismatch); abs=%s", srcPath, abs)
	}
	src, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read command.go: %v", err)
	}
	body := string(src)

	// 必须存在 OperatorID 赋值,且赋值源是 callerUID.String()
	if !strings.Contains(body, "OperatorID:") {
		t.Fatal("command.go 缺失 OperatorID 字段赋值(P0-3 审计要求)")
	}
	if !strings.Contains(body, "OperatorID:   callerUID.String()") &&
		!strings.Contains(body, "OperatorID: callerUID.String()") {
		t.Errorf("OperatorID 不是 callerUID.String() — P0-3 审计要求用 user_id 而非 ClientIP")
	}
	// 显式拒绝 ClientIP() 作为 OperatorID 来源
	if strings.Contains(body, "OperatorID:") && strings.Contains(body, "ClientIP()") {
		// 进一步定位:ClientIP() 不能在 OperatorID 行附近
		for _, line := range strings.Split(body, "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "OperatorID:") &&
				strings.Contains(trim, "ClientIP()") {
				t.Errorf("OperatorID 用了 ClientIP() — P0-3 修复被回退:\n  %s", trim)
			}
		}
	}
}
