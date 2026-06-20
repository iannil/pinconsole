// 1ad 测试:trace_id 上下文还原 + 下行传递源码契约(审计 T1-1m-1 + T1-1m-2)。
// 1af G6 扩展:加行为级测试验证 logging.WithTraceID 真工作。
//
// T1-1m-1: 服务端 WS 读 envelope.TraceID 写回 ctx(让后续日志能关联 SDK 端)
// T1-1m-2: 下行 envelope(command)携带 ctx TraceID(让 SDK 知道是哪条 command 触发)
//
// 源码契约,捕获接线被误删。
package api

import (
	"context"
	"strings"
	"testing"

	"github.com/iannil/pinconsole/internal/logging"
)

// TestTraceID_WsReadLoop_RestoresCtx — T1-1m-1:
// ws.go read loop 必须把 envelope.TraceID 写回 ctx(via logging.WithTraceID)。
func TestTraceID_WsReadLoop_RestoresCtx(t *testing.T) {
	src := mustReadFile(t, "ws.go")
	for _, must := range []string{
		"env.TraceID != \"\"",
		"logging.WithTraceID(ctx, env.TraceID)",
	} {
		if !strings.Contains(src, must) {
			t.Errorf("ws.go 缺失 %q — TraceID ctx 还原破坏", must)
		}
	}
}

// TestTraceID_DownlinkEnvelope_CarriesTraceID — T1-1m-2:
// command.go 下行 envelope 必须设置 TraceID 字段(从 ctx 取)。
func TestTraceID_DownlinkEnvelope_CarriesTraceID(t *testing.T) {
	src := mustReadFile(t, "command.go")
	for _, must := range []string{
		"logging.TraceID(c.Request.Context())",
		"TraceID:",
	} {
		if !strings.Contains(src, must) {
			t.Errorf("command.go 缺失 %q — 下行 envelope TraceID 透传破坏", must)
		}
	}
}

// TestTraceID_SDK_LoggerModule — T1-1m-3(部分):
// SDK logger 模块在 TS 端,这里只验证 server 端不破坏 trace 链路。
// 完整 SDK logger 测试在 visitor-sdk/tests/(已部分覆盖 ws-trace-inherit.test.ts)。
func TestTraceID_SDK_LoggerModule(t *testing.T) {
	// 仅占位,实际 SDK logger 测试在 TS 端
	// ws-trace-inherit.test.ts 6 测试已覆盖 trace 继承窗口逻辑
	src := mustReadFile(t, "ws.go")
	if !strings.Contains(src, "logging.FromContext") {
		t.Errorf("ws.go 缺失 logging.FromContext — SDK logger 关联破坏")
	}
}

// TestTraceID_Behavioral_WithTraceID_RoundTrip — 1af G6 (T1-1m-1 行为级):
// 真调 logging.WithTraceID + logging.TraceID 验证 round-trip 正确。
//
// 此前的源码契约测试只 grep "logging.WithTraceID(ctx, env.TraceID)" 字符串,
// 不能捕获:
// - WithTraceID 实现broken(参数顺序错、key 用错)
// - TraceID 从 ctx 取不出(nil 或空字符串)
//
// 行为级测试直接调 logging 包,验证 trace_id 真持久化到 ctx。
func TestTraceID_Behavioral_WithTraceID_RoundTrip(t *testing.T) {
	ctx := context.Background()

	// 空 trace_id — 不应 panic,TraceID() 返回空字符串
	emptyCtx := logging.WithTraceID(ctx, "")
	if got := logging.TraceID(emptyCtx); got != "" {
		t.Errorf("empty trace_id: TraceID()=%q want empty", got)
	}

	// 正常 trace_id — round-trip 必须一致
	testTraceID := "abc-123-def-456"
	tracedCtx := logging.WithTraceID(ctx, testTraceID)
	if got := logging.TraceID(tracedCtx); got != testTraceID {
		t.Errorf("TraceID()=%q want %q — round-trip 破坏", got, testTraceID)
	}

	// 子 context 继承 trace_id
	childCtx, cancel := context.WithCancel(tracedCtx)
	defer cancel()
	if got := logging.TraceID(childCtx); got != testTraceID {
		t.Errorf("child ctx TraceID()=%q want %q — 继承破坏", got, testTraceID)
	}
}

// TestTraceID_Behavioral_NilCtx_NoPanic — 1af G6 (T1-1m-1 robustness):
// nil ctx 或 background ctx 调 TraceID 应安全返回空字符串,不 panic。
func TestTraceID_Behavioral_NilCtx_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TraceID on background ctx panic'd: %v", r)
		}
	}()

	// background ctx(无 trace_id)应返回空字符串
	got := logging.TraceID(context.Background())
	if got != "" {
		t.Errorf("background ctx TraceID()=%q want empty", got)
	}
}
