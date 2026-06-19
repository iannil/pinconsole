// 1ad 测试:flagged session warn 日志 + is_flagged 字段接线(审计 T1-1w 4 项)。
//
// 验证:
//   - ws.go subscribe flagged session 时 warn(T1-1w-2)
//   - replay.go replay flagged session 时 warn(T1-1w-3)
//   - session.go listSessions 返回 is_flagged 字段(T1-1w-4)
//   - Redis 失败 warn 不阻断(T1-1w-1)
package api

import (
	"strings"
	"testing"
)

// TestFlagged_SubscribeWarns — T1-1w-2:
// ws.go 订阅 flagged session 时必须 warn(让运营可见可疑活动)。
func TestFlagged_SubscribeWarns(t *testing.T) {
	src := mustReadFile(t, "ws.go")
	if !strings.Contains(src, "subscribing to flagged session") {
		t.Errorf("ws.go 缺失 'subscribing to flagged session' warn 日志")
	}
	if !strings.Contains(src, "is_session_flagged check failed on subscribe") {
		t.Errorf("ws.go 缺失 Redis 失败 warn(应 fail-open 但 warn)")
	}
}

// TestFlagged_ReplayWarns — T1-1w-3:
// replay.go replay flagged session 时必须 warn。
func TestFlagged_ReplayWarns(t *testing.T) {
	src := mustReadFile(t, "replay.go")
	for _, must := range []string{
		"replay requested for flagged session",
		"is_session_flagged check failed on replay",
	} {
		if !strings.Contains(src, must) {
			t.Errorf("replay.go 缺失 %q warn", must)
		}
	}
}

// TestFlagged_ListSessionsReturnsIsFlagged — T1-1w-4:
// session.go listSessions 响应必须含 is_flagged 字段(让 admin store 能映射)。
func TestFlagged_ListSessionsReturnsIsFlagged(t *testing.T) {
	src := mustReadFile(t, "session.go")
	if !strings.Contains(src, "IsFlagged  bool   `json:\"is_flagged\"`") &&
		!strings.Contains(src, "IsFlagged bool `json:\"is_flagged\"`") {
		t.Errorf("session.go 缺失 IsFlagged JSON 字段 — admin store 无法映射")
	}
}

// TestFlagged_RedisFailureFailOpen — T1-1w-1:
// session.go Redis 查 flagged 失败时必须 warn 不阻断(fail-open)。
func TestFlagged_RedisFailureFailOpen(t *testing.T) {
	src := mustReadFile(t, "session.go")
	if !strings.Contains(src, "is_session_flagged failed") {
		t.Errorf("session.go 缺失 'is_session_flagged failed' warn — Redis 失败应 fail-open 不阻断 listSessions")
	}
}

// TestFlagged_IsSessionFlagged_Helper_Exists — 防 helper 函数被误删。
func TestFlagged_IsSessionFlagged_Helper_Exists(t *testing.T) {
	src := mustReadFile(t, "session.go")
	// antiscrape.IsSessionFlagged 应被调用(从 Redis 读 flagged:session:{id})
	if !strings.Contains(src, "IsSessionFlagged(") {
		t.Errorf("session.go 缺失 IsSessionFlagged 调用 — flagged 字段读取破坏")
	}
}
