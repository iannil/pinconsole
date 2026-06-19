// 1ac 测试:WS rate limit 强制断开 + FlagSession 接线(审计 T0-1y-1 + T0-1y-2)。
//
// checkWSRateLimit 函数级测试已存在(ws_ratelimit_test.go 5 测试覆盖 fail-open 等)。
// 本测试覆盖**接线层**:验证 ws.go 在 checkWSRateLimit 返回 allowed=false 时:
//   - 调用 antiscrape.FlagSession(reason)(T0-1y-2)
//   - 调用 conn.Close(StatusPolicyViolation, ...)(T0-1y-1)
//
// 源码契约测试,捕获接线被误删/重构。
package api

import (
	"os"
	"strings"
	"testing"
)

// TestWSRateLimit_CloseOnExceed — T0-1y-1 源码契约:
// ws.go 必须在 checkWSRateLimit 返回 allowed=false 时调 conn.Close(StatusPolicyViolation, ...)。
//
// 否则:超限仅计数不断开,客户端可继续滥用。
func TestWSRateLimit_CloseOnExceed(t *testing.T) {
	src, err := os.ReadFile("ws.go")
	if err != nil {
		t.Fatalf("read ws.go: %v", err)
	}
	body := string(src)

	// 1. 必须调 checkWSRateLimit
	if !strings.Contains(body, "checkWSRateLimit(") {
		t.Fatal("ws.go 缺失 checkWSRateLimit 调用")
	}

	// 2. 在 checkWSRateLimit 调用附近的代码块(后 1500 字符),必须含 conn.Close(StatusPolicyViolation)
	// 跳过函数定义,找调用点(ctx 类型签名只在定义里出现)
	idx := strings.Index(body, "checkWSRateLimit(ctx,")
	if idx < 0 {
		t.Fatal("找不到 checkWSRateLimit 调用点")
	}
	tail := body[idx:]
	if len(tail) > 1500 {
		tail = tail[:1500]
	}

	for _, must := range []string{
		"websocket.StatusPolicyViolation",
		`conn.Close(websocket.StatusPolicyViolation`,
	} {
		if !strings.Contains(tail, must) {
			t.Errorf("checkWSRateLimit 后缺失 %q — 强制断开接线破坏", must)
		}
	}
}

// TestWSRateLimit_FlagSessionOnExceed — T0-1y-2 源码契约:
// ws.go 必须在超限时调 antiscrape.FlagSession(ctx, ..., sessionID, reason)。
//
// 否则:admin 看不到滥用 session。
func TestWSRateLimit_FlagSessionOnExceed(t *testing.T) {
	src, err := os.ReadFile("ws.go")
	if err != nil {
		t.Fatalf("read ws.go: %v", err)
	}
	body := string(src)

	for _, must := range []string{
		"antiscrape.FlagSession",
		`FlagSession(ctx`,
	} {
		if !strings.Contains(body, must) {
			t.Errorf("ws.go 缺失 %q — FlagSession 接线破坏(admin 不可见滥用)", must)
		}
	}

	// FlagSession 必须在 close 之前(否则 conn close 后 ctx 取消,FlagSession 可能失败)
	idxFlag := strings.Index(body, "antiscrape.FlagSession")
	idxClose := strings.Index(body, "conn.Close(websocket.StatusPolicyViolation")
	if idxFlag < 0 || idxClose < 0 {
		t.Fatal("missing FlagSession or PolicyViolation close")
	}
	if idxFlag > idxClose {
		t.Errorf("FlagSession (idx=%d) 在 close (idx=%d) 之后 — 顺序错,可能 race", idxFlag, idxClose)
	}
}

// TestWSRateLimit_ReasonThreading — reason 字段必须从 checkWSRateLimit 透传到 FlagSession。
// 否则 admin 只看到 "rate limit exceeded" 但不知道是 msgs 还是 bytes 超限。
func TestWSRateLimit_ReasonThreading(t *testing.T) {
	src, err := os.ReadFile("ws.go")
	if err != nil {
		t.Fatalf("read ws.go: %v", err)
	}
	body := string(src)

	// 找 checkWSRateLimit 调用位置(非函数定义)。
	// 函数定义模式:"checkWSRateLimit(ctx context.Context" — 用 ctx 跟 ctx 类型
	// 调用模式:"checkWSRateLimit(ctx, " 或 "checkWSRateLimit(\n  "
	callMarker := "checkWSRateLimit(ctx,"
	idxCheck := strings.Index(body, callMarker)
	if idxCheck < 0 {
		t.Fatal("找不到 checkWSRateLimit 调用点(函数定义而非调用?)")
	}
	idxFlag := strings.Index(body, "antiscrape.FlagSession(")
	if idxFlag < 0 {
		t.Fatal("找不到 antiscrape.FlagSession 调用")
	}
	if idxFlag < idxCheck {
		t.Fatal("FlagSession 在 checkWSRateLimit 之前(顺序错)")
	}

	between := body[idxCheck:idxFlag]
	if !strings.Contains(between, "reason") {
		t.Errorf("reason 字段未从 checkWSRateLimit 透传到 FlagSession:\n%s", between)
	}
}
