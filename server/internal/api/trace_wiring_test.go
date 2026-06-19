// 1ad 测试:trace_id 上下文还原 + 下行传递源码契约(审计 T1-1m-1 + T1-1m-2)。
//
// T1-1m-1: 服务端 WS 读 envelope.TraceID 写回 ctx(让后续日志能关联 SDK 端)
// T1-1m-2: 下行 envelope(command)携带 ctx TraceID(让 SDK 知道是哪条 command 触发)
//
// 源码契约,捕获接线被误删。
package api

import (
	"strings"
	"testing"
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
