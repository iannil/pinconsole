// Package privacy 处理 GDPR/CCPA 合规相关的数据最小化与同意管理。
//
// 1l-privacy-gdpr:
//   - IP 截断(IPv4 /24, IPv6 /64)使 IP 不再是 GDPR 意义上的个人数据
//   - Consent 状态读写(由 api/privacy.go 调用)
//   - 数据保留策略(GC 扩展)
package privacy

import (
	"net/netip"
)

// TruncateIP 把 IP 截断到 GDPR Recital 26 "不再能识别个人" 的粒度。
//   - IPv4: /24(保留前 3 字节,末字节填 0)
//   - IPv6: /64(保留前 8 字节,后 8 字节填 0)
//   - 支持 host:port 形式(IPv4:port 或 [IPv6]:port)
//   - 无效 IP 或 host-only 字符串:原样返回(不阻塞写入)
//
// 例:
//
//	TruncateIP("192.168.1.42")          → "192.168.1.0"
//	TruncateIP("192.168.1.42:8080")     → "192.168.1.0"
//	TruncateIP("2001:db8::1")           → "2001:db8::"
//	TruncateIP("[2001:db8::1]:8080")    → "2001:db8::"
//	TruncateIP("invalid")               → "invalid"
//	TruncateIP("")                      → ""
func TruncateIP(raw string) string {
	if raw == "" {
		return ""
	}
	addr := parseAddrLoose(raw)
	if !addr.IsValid() {
		return raw
	}
	if addr.Is4() {
		b := addr.As4()
		b[3] = 0
		return netip.AddrFrom4(b).String()
	}
	if addr.Is6() {
		b := addr.As16()
		for i := 8; i < 16; i++ {
			b[i] = 0
		}
		return netip.AddrFrom16(b).String()
	}
	return raw
}

// parseAddrLoose 尝试用多种方式解析 IP:
// 1. 纯 IP(支持 v4/v6)
// 2. host:port(IPv4:port 或 [IPv6]:port)
// 失败返回 invalid netip.Addr。
func parseAddrLoose(raw string) netip.Addr {
	// 尝试纯 IP
	if addr, err := netip.ParseAddr(raw); err == nil {
		return addr
	}
	// 尝试 host:port
	if ap, err := netip.ParseAddrPort(raw); err == nil {
		return ap.Addr()
	}
	return netip.Addr{}
}
