// 1ad 续集测试:1g chat WS 下行 + popup XSS 接线源码契约(审计 T1-1g-2/3/4/5)。
//
// T1-1g-2: chat handler 通过 hub.SendCommandToVisitor 下行 chat_message 到 visitor
// T1-1g-3: chat postMessage 必须 requireClaimOwnership(已在 1ac-1k-2 authz_test cover)
// T1-1g-4: chat listMessages 不要求 claim(读取 vs 写入区分)
// T1-1g-5: popup body 渲染用 textContent 防 XSS(在 visitor-sdk/src/ui/popup.ts)
//
// 服务端 chat handler 接线 + SDK popup XSS 都是源码契约。
package api

import (
	"os"
	"strings"
	"testing"
)

// TestChat_PostMessage_WiresCommandDownlink — T1-1g-2:
// chat.go postMessage 必须 SendCommandToVisitor 下行到 visitor。
func TestChat_PostMessage_WiresCommandDownlink(t *testing.T) {
	src := mustReadFile(t, "chat.go")
	idx := strings.Index(src, "func (h *ChatHandler) postMessage")
	if idx < 0 {
		t.Fatal("找不到 postMessage")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	if !strings.Contains(fnBody, "SendCommandToVisitor(") {
		t.Errorf("postMessage 缺失 SendCommandToVisitor 调用 — chat 下行破坏")
	}
}

// TestChat_PostMessage_RequiresClaimOwnership — T1-1g-3:
// chat.go postMessage 必须 requireClaimOwnership(只 owner 可发消息)。
func TestChat_PostMessage_RequiresClaimOwnership(t *testing.T) {
	src := mustReadFile(t, "chat.go")
	idx := strings.Index(src, "func (h *ChatHandler) postMessage")
	if idx < 0 {
		t.Fatal("找不到 postMessage")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	if !strings.Contains(fnBody, "requireClaimOwnership(") {
		t.Errorf("postMessage 缺失 requireClaimOwnership — 任意认证用户可发消息(横向越权)")
	}
}

// TestChat_ListMessages_NoClaimRequired — T1-1g-4:
// chat.go listMessages 必须 NOT 调 requireClaimOwnership(只读端点不应要 claim 锁)。
// (v1-followups fix1: 1k P0-3 实现错误地对 listMessages 也 requireClaimOwnership)
func TestChat_ListMessages_NoClaimRequired(t *testing.T) {
	src := mustReadFile(t, "chat.go")
	idx := strings.Index(src, "func (h *ChatHandler) listMessages")
	if idx < 0 {
		// 可能函数名不同,搜下 GET messages handler
		idx = strings.Index(src, "listMessages")
		if idx < 0 {
			t.Skip("找不到 listMessages handler(可能命名不同)")
		}
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	if strings.Contains(fnBody, "requireClaimOwnership(") {
		t.Errorf("listMessages 调了 requireClaimOwnership — v1-followups fix1 修复被回退(只读端点不应要 claim)")
	}
}

// TestSDK_PopupXSS_TextContent — T1-1g-5:
// visitor-sdk/src/ui/popup.ts 必须用 textContent(不是 innerHTML)渲染 popup 内容。
// 否则 operator 注入恶意 HTML 可执行 visitor 浏览器中。
func TestSDK_PopupXSS_TextContent(t *testing.T) {
	src, err := os.ReadFile("../../../visitor-sdk/src/ui/popup.ts")
	if err != nil {
		t.Skipf("read popup.ts: %v(可能在不同的相对路径)", err)
	}
	body := string(src)

	// 必须用 textContent 渲染 title/body/label
	textContentCount := strings.Count(body, "textContent")
	if textContentCount < 3 {
		t.Errorf("popup.ts textContent 调用次数=%d, want ≥3 (title/body/action_label/dismiss)", textContentCount)
	}

	// 反模式:innerHTML 直接拼用户输入
	if strings.Contains(body, "innerHTML = p.") {
		t.Errorf("popup.ts 用 innerHTML = p.* — XSS 风险,应改 textContent")
	}
}
