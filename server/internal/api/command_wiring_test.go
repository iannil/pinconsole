// 1ad 测试:command handler 接线源码契约(审计 T1-1e-1/2/3)。
//
// T1-1e-1: postCommand 必须调 buildCommandPayload(8 种命令类型构造)
// T1-1e-2: postCommand 必须调 hub.SendCommandToVisitor 下行到 visitor
// T1-1e-3: postCommand 必须写 co_browsing_commands PG 审计行
//
// T1-1e-4(OperatorID UUID)已由 1ac-1k-3 cover,不重复。
package api

import (
	"strings"
	"testing"
)

// TestCommand_PostCommand_WiresBuildPayload — T1-1e-1:
// postCommand handler 必须调 buildCommandPayload 构造命令载荷。
func TestCommand_PostCommand_WiresBuildPayload(t *testing.T) {
	src := mustReadFile(t, "command.go")
	idx := strings.Index(src, "func (h *CommandHandler) postCommand")
	if idx < 0 {
		t.Fatal("找不到 postCommand handler")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	if !strings.Contains(fnBody, "buildCommandPayload(") {
		t.Errorf("postCommand 缺失 buildCommandPayload 调用 — 8 种命令类型构造破坏")
	}
}

// TestCommand_PostCommand_WiresSendCommandToVisitor — T1-1e-2:
// postCommand 必须调 hub.SendCommandToVisitor 下行命令到 visitor client。
func TestCommand_PostCommand_WiresSendCommandToVisitor(t *testing.T) {
	src := mustReadFile(t, "command.go")
	idx := strings.Index(src, "func (h *CommandHandler) postCommand")
	if idx < 0 {
		t.Fatal("找不到 postCommand handler")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	if !strings.Contains(fnBody, "SendCommandToVisitor(") {
		t.Errorf("postCommand 缺失 SendCommandToVisitor 调用 — 下行路由破坏")
	}
}

// TestCommand_PostCommand_WiresAuditWrite — T1-1e-3:
// postCommand 必须写 co_browsing_commands PG 审计行(CreateCoBrowsingCommand)。
func TestCommand_PostCommand_WiresAuditWrite(t *testing.T) {
	src := mustReadFile(t, "command.go")
	idx := strings.Index(src, "func (h *CommandHandler) postCommand")
	if idx < 0 {
		t.Fatal("找不到 postCommand handler")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	if !strings.Contains(fnBody, "CreateCoBrowsingCommand(") {
		t.Errorf("postCommand 缺失 CreateCoBrowsingCommand 调用 — 审计写入破坏")
	}
}

// TestCommandHub_InterfaceContract — 防止 CommandHub interface 字段变更破坏依赖注入。
func TestCommandHub_InterfaceContract(t *testing.T) {
	src := mustReadFile(t, "command.go")
	if !strings.Contains(src, "SendCommandToVisitor(sessionID uuid.UUID, msg []byte) bool") {
		t.Errorf("command.go CommandHub interface 缺失 SendCommandToVisitor 签名")
	}
}
