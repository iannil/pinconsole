// 1k 测试：popup URL scheme 白名单 (P0-8)。
package api

import (
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
