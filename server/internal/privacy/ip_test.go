// 1l 测试:IP 截断边界覆盖。
package privacy

import "testing"

func TestTruncateIP_IPv4(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"192.168.1.42", "192.168.1.0"},
		{"10.0.0.1", "10.0.0.0"},
		{"172.16.5.100", "172.16.5.0"},
		{"8.8.8.8", "8.8.8.0"},
		{"127.0.0.1", "127.0.0.0"},
		{"0.0.0.0", "0.0.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := TruncateIP(tt.in)
			if got != tt.want {
				t.Errorf("TruncateIP(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTruncateIP_IPv4WithPort(t *testing.T) {
	got := TruncateIP("192.168.1.42:8080")
	want := "192.168.1.0"
	if got != want {
		t.Errorf("TruncateIP(%q) = %q, want %q", "192.168.1.42:8080", got, want)
	}
}

func TestTruncateIP_IPv6(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"2001:db8::1", "2001:db8::"},
		{"fe80::1", "fe80::"},
		{"fd00::abcd", "fd00::"},
		{"::1", "::"},         // loopback
		{"2001:db8:abcd:ef00:1234:5678:9abc:def0", "2001:db8:abcd:ef00::"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := TruncateIP(tt.in)
			if got != tt.want {
				t.Errorf("TruncateIP(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTruncateIP_IPv6WithPort(t *testing.T) {
	got := TruncateIP("[2001:db8::1]:8080")
	want := "2001:db8::"
	if got != want {
		t.Errorf("TruncateIP(%q) = %q, want %q", "[2001:db8::1]:8080", got, want)
	}
}

func TestTruncateIP_Invalid(t *testing.T) {
	tests := []string{
		"",
		"not-an-ip",
		"example.com",
		"999.999.999.999",
	}
	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			got := TruncateIP(in)
			// 失败时原样返回;不 panic 即可
			if in != "" && got == "" {
				t.Errorf("TruncateIP(%q) returned empty for non-empty input", in)
			}
		})
	}
}

func TestTruncateIP_Empty(t *testing.T) {
	if TruncateIP("") != "" {
		t.Errorf("TruncateIP(\"\") should return empty")
	}
}
