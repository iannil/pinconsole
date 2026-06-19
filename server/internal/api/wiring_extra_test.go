// 1ad 测试:1f navigated + 1o per-sub cancel 接线源码契约(审计 T1-1f + T1-1o)。
//
// 1f: presence.navigated 事件 + SDK beforeunload notifyNavigated + admin auto-resubscribe
// 1o: per-sub context.WithCancel 防 goroutine 泄漏(1o P1-8 修复)
package api

import (
	"strings"
	"testing"
)

// Test1f_PresenceNavigated_EventType — T1-1f-1:
// proto/envelope.go 必须支持 "navigated" 事件类型(关联 session IDs)。
func Test1f_PresenceNavigated_EventType(t *testing.T) {
	src := mustReadFile(t, "../proto/envelope.go")
	for _, must := range []string{
		"navigated",
		"OldSessionID", // navigated 关联字段
		"NewSessionID",
	} {
		if !strings.Contains(src, must) {
			t.Errorf("proto/envelope.go 缺失 %q — navigated 事件结构破坏", must)
		}
	}
}

// Test1o_PerSubCancel_PreventsGoroutineLeak — T1-1o-1:
// ws.go 必须为每个 subscribe 创建独立 cancel ctx,unsubscribe 时 cancel。
// 否则 goroutine 泄漏(1o P1-8 修复的核心)。
func Test1o_PerSubCancel_PreventsGoroutineLeak(t *testing.T) {
	src := mustReadFile(t, "ws.go")

	for _, must := range []string{
		"subCancels",
		"subCtx, subCancel := context.WithCancel(ctx)",
		"subCancels[sid] = subCancel",
	} {
		if !strings.Contains(src, must) {
			t.Errorf("ws.go 缺失 %q — per-sub cancel 接线破坏(goroutine 泄漏风险)", must)
		}
	}

	// unsubscribe 时必须 cancel + delete
	if !strings.Contains(src, "delete(subCancels, cmd.sessionID)") {
		t.Errorf("ws.go unsubscribe 缺失 delete(subCancels) — cancel ctx 泄漏")
	}

	// defer 退出时 cancel 所有 sub
	if !strings.Contains(src, "for _, cancel := range subCancels") {
		t.Errorf("ws.go defer 缺失 cancel all subCancels — 连接断开时 goroutine 泄漏")
	}
}

// Test1o_OperatorWS_AlsoHasPerSubCancel — 1o P1-8 一致性:
// operatorWS 也用同样的 per-sub cancel 模式(防 goroutine 泄漏)。
func Test1o_OperatorWS_AlsoHasPerSubCancel(t *testing.T) {
	src := mustReadFile(t, "ws.go")
	idx := strings.Index(src, "func (h *WSHandler) operatorWS")
	if idx < 0 {
		t.Fatal("找不到 operatorWS")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	for _, must := range []string{
		"subCancels",
		"WithCancel",
	} {
		if !strings.Contains(fnBody, must) {
			t.Errorf("operatorWS 缺失 %q — per-sub cancel 在 operator 侧也应启用", must)
		}
	}
}
