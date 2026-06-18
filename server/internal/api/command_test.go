// 1k 测试：popup URL scheme 白名单 (P0-8)。
package api

import "testing"

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
