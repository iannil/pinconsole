// 1ac 续集测试:WS 同源 cookie + operatorWS auth gap 文档(审计 T0-1h-6 + T0-1h-2)。
//
// T0-1h-6:websocket.Accept 必须设 InsecureSkipVerify: false(同源校验),
//   防止跨域 WS 滥用(CSWSH)。
//
// T0-1h-2:**发现代码 bug**:operatorWS 完全无认证检查。
//   - 路由层:/ws/operator 不在 protected group 下
//   - handler 层:operatorWS 不读 cookie/session,直接 Accept
//   - 任意匿名客户端可连 WS 接收所有 visitor 事件流(隐私泄露)
//
//   此 bug 非 1ac 范围可修(需 API 设计决策:WS 鉴权 token 在 query/header/first message)。
//   本测试用 t.Skip 占位,留作 1ac-final 或 1ad 修复。
package api

import (
	"os"
	"strings"
	"testing"
)

// TestWS_VisitorOriginCheck_Enabled — T0-1h-6:
// visitorWS 的 websocket.Accept 必须设 InsecureSkipVerify: false。
// 否则跨域连接可读取 visitor 录像数据。
func TestWS_VisitorOriginCheck_Enabled(t *testing.T) {
	src, err := os.ReadFile("ws.go")
	if err != nil {
		t.Fatalf("read ws.go: %v", err)
	}
	body := string(src)

	// 找 visitorWS 函数体内的 Accept 调用
	idx := strings.Index(body, "func (h *WSHandler) visitorWS")
	if idx < 0 {
		t.Fatal("找不到 visitorWS 函数")
	}
	end := strings.Index(body[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(body) - idx - 1
	}
	fnBody := body[idx : idx+1+end]

	if !strings.Contains(fnBody, "InsecureSkipVerify: false") {
		t.Errorf("visitorWS 缺失 InsecureSkipVerify: false — 同源校验破坏(CSWSH 风险):\n%s", fnBody)
	}
	// 反模式:InsecureSkipVerify: true 等同禁用同源校验
	if strings.Contains(fnBody, "InsecureSkipVerify: true") {
		t.Errorf("visitorWS 设了 InsecureSkipVerify: true — 跨域 WS 滥用风险")
	}
}

// TestWS_OperatorOriginCheck_Enabled — T0-1h-6 operator 侧:
// operatorWS 也必须设 InsecureSkipVerify: false。
func TestWS_OperatorOriginCheck_Enabled(t *testing.T) {
	src, err := os.ReadFile("ws.go")
	if err != nil {
		t.Fatalf("read ws.go: %v", err)
	}
	body := string(src)

	idx := strings.Index(body, "func (h *WSHandler) operatorWS")
	if idx < 0 {
		t.Fatal("找不到 operatorWS 函数")
	}
	end := strings.Index(body[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(body) - idx - 1
	}
	fnBody := body[idx : idx+1+end]

	if !strings.Contains(fnBody, "InsecureSkipVerify: false") {
		t.Errorf("operatorWS 缺失 InsecureSkipVerify: false — 同源校验破坏")
	}
	if strings.Contains(fnBody, "InsecureSkipVerify: true") {
		t.Errorf("operatorWS 设了 InsecureSkipVerify: true — 跨域 WS 滥用风险")
	}
}

// TestWS_OperatorAuth_KnownGap — T0-1h-2 占位测试(known bug):
//
// operatorWS 当前无认证检查。此测试 SKIP,留作修复后启用。
//
// 期望行为(修复后):
//   - 路由层:/ws/operator 应在 protected group 下,或
//   - handler 层:operatorWS 应读 cookie 校验 session
//
// 当前行为:
//   - 路由层:wsH.Register(r) 直接挂在 public r 上
//   - handler 层:operatorWS 直接 Accept,无 cookie/session 检查
//   - 注释:"hello 不强制；如果没有，直接当作 operator 上线"
//
// 风险:任意匿名客户端可连 /ws/operator,接收 tenant room 内全部 visitor 事件流,
// 包括录像内容、co-browsing 命令、聊天消息广播。隐私 + 安全双重违规。
//
// 修复方向(任一):
//   A. router.go: 把 wsH.Register(protected) — 但 WS upgrade 在 middleware 之前,
//      需要确认 gin 支持在 group 下挂 WS route
//   B. operatorWS 内部: Accept 前读 c.Cookie,校验 Redis session,
//      校验失败拒绝 upgrade(返回 401)
//   C. 用 query token: /ws/operator?token=xxx,token 是登录时签发的短期 WS token
//
// 推荐方案 B(与 cookie session 一致,无需新 token)。
func TestWS_OperatorAuth_KnownGap(t *testing.T) {
	t.Skip("known bug: operatorWS 无认证检查 — 见审计 T0-1h-2,留 1ad 修复(需 API 设计决策)")
}
