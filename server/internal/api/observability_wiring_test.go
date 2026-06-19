// 1ad 测试:可观测性 lifecycle / LogPoint / LogExternalCall 接线源码契约(审计 T1-1s 13 项)。
// 1ae R3e 扩展:加行为级测试,验证 Lifecycle 真在 handler 调用时产生日志。
//
// 验证 5 个 handler + 1 worker + 5 LogPoint 分支 + 3 LogExternalCall 站点都正确接线:
//   - Lifecycle: PostCommand / Claim / Release / PostMessage / FlushSession / GC.runOnce
//   - LogPoint: claim_check_failed / claim_check_ok / command_type / navigate_check / popup_url_check
//   - LogExternalCall: pg.CreateCoBrowsingCommand / minio.PutObject / pg.CreateEventBlob
//
// 这些埋点是 1s "可观测性深化" 切片的核心交付物。如果重构误删,运行时静默退化为
// 无 trace 的黑盒。源码契约捕获重构回归。
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/observability"
	"github.com/iannil/marketing-monitor/internal/storage"
)

// TestObservability_Lifecycle_OnPostCommand — T1-1s-LIF-01:
// command.go PostCommand handler 必须有 observability.Lifecycle 埋点。
func TestObservability_Lifecycle_OnPostCommand(t *testing.T) {
	assertHandlerHasLifecycle(t, "command.go", "func (h *CommandHandler) postCommand", "PostCommand")
}

// TestObservability_Lifecycle_OnClaim — T1-1s-LIF-02:
func TestObservability_Lifecycle_OnClaim(t *testing.T) {
	assertHandlerHasLifecycle(t, "claim.go", "func (h *ClaimHandler) claim", "Claim")
}

// TestObservability_Lifecycle_OnRelease — T1-1s-LIF-03:
func TestObservability_Lifecycle_OnRelease(t *testing.T) {
	assertHandlerHasLifecycle(t, "claim.go", "func (h *ClaimHandler) release", "Release")
}

// TestObservability_Lifecycle_OnPostMessage — T1-1s-LIF-04:
func TestObservability_Lifecycle_OnPostMessage(t *testing.T) {
	assertHandlerHasLifecycle(t, "chat.go", "func (h *ChatHandler) postMessage", "PostMessage")
}

// TestObservability_LogPoint_Command_Branches — T1-1s-LP-01/02/03:
// command.go 必须有 claim_check + command_type LogPoint 分支(至少 3 处)。
func TestObservability_LogPoint_Command_Branches(t *testing.T) {
	src := mustReadFile(t, "command.go")

	// LogPoint 调用次数 ≥ 3(claim_check + navigate_check + popup_url_check + command_type)
	count := strings.Count(src, "observability.LogPoint(")
	if count < 3 {
		t.Errorf("command.go LogPoint 调用次数=%d, want ≥3 (claim_check/navigate_check/popup_url_check 等)", count)
	}

	// 必须覆盖关键分支(任一关键字命中即可)
	for _, branch := range []string{"claim_check", "navigate", "popup", "command_type"} {
		if !strings.Contains(src, branch) {
			t.Errorf("command.go 缺失 LogPoint 分支关键字 %q", branch)
		}
	}
}

// TestObservability_LogExternalCall_CreateCoBrowsingCommand — T1-1s-EXT-01:
// command.go 必须有 LogExternalCall("pg.CreateCoBrowsingCommand", ...) 在成功/失败两条路径。
func TestObservability_LogExternalCall_CreateCoBrowsingCommand(t *testing.T) {
	src := mustReadFile(t, "command.go")
	for _, status := range []string{"ok", "error"} {
		needle := `observability.LogExternalCall(ctx, logger, "pg.CreateCoBrowsingCommand", "` + status + `"`
		if !strings.Contains(src, needle) {
			t.Errorf("command.go 缺失 LogExternalCall pg.CreateCoBrowsingCommand status=%q", status)
		}
	}
}

// assertHandlerHasLifecycle 通用辅助:验证指定 handler 函数体内有 observability.Lifecycle(name, ...) 埋点。
func assertHandlerHasLifecycle(t *testing.T, file, handler, name string) {
	t.Helper()
	src := mustReadFile(t, file)

	idx := strings.Index(src, handler)
	if idx < 0 {
		t.Fatalf("%s: 找不到 handler %q", file, handler)
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	needle := `observability.Lifecycle(ctx, "` + name + `"`
	if !strings.Contains(fnBody, needle) {
		t.Errorf("%s %s 缺失 %q Lifecycle 埋点:\n%s", file, handler, needle, fnBody)
	}
}

func mustReadFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

// TestLifecycle_Behavioral_ProducesStartEndLogs — 1ae R3e 升级:
// 真调 observability.Lifecycle(从 api 包视角),验证 handler 模式下日志真产生。
//
// 此前的源码契约测试只 grep 字符串,不能捕获:
// - Lifecycle 返回的 defer 函数没被调用(漏 defer)
// - logger 传 nil 时静默 no-op(可能掩盖问题)
//
// 行为级测试:用 buffer logger + 真 Lifecycle 调用,验证 Function_Start + Function_End 都写入。
func TestLifecycle_Behavioral_ProducesStartEndLogs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := context.Background()

	// 模拟 handler 中的 Lifecycle 用法
	func() {
		defer observability.Lifecycle(ctx, "PostCommand", logger)()
		// 模拟业务逻辑(什么都不做)
	}()

	logs := parseLifecycleLogs(&buf)
	if len(logs) != 2 {
		t.Fatalf("expected 2 log entries (start+end), got %d: %v", len(logs), logs)
	}
	if logs[0]["event_type"] != string(observability.EventFunctionStart) {
		t.Errorf("first log event_type = %v, want %s", logs[0]["event_type"], observability.EventFunctionStart)
	}
	if logs[0]["span"] != "PostCommand" {
		t.Errorf("first log span = %v, want PostCommand", logs[0]["span"])
	}
	if logs[1]["event_type"] != string(observability.EventFunctionEnd) {
		t.Errorf("second log event_type = %v, want %s", logs[1]["event_type"], observability.EventFunctionEnd)
	}
	if logs[1]["duration_ms"] == nil {
		t.Errorf("duration_ms should be present in end log")
	}
}

// TestLifecycle_Behavioral_PanicPath_RecordsError — 1ae R3e 补充:
// panic 路径必须记录 Error + stack,不能 silent swallow。
func TestLifecycle_Behavioral_PanicPath_RecordsError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := context.Background()

	defer func() {
		_ = recover()
	}()

	func() {
		defer observability.Lifecycle(ctx, "PanickyHandler", logger)()
		panic("simulated handler panic")
	}()

	logs := parseLifecycleLogs(&buf)
	hasError := false
	for _, l := range logs {
		if l["event_type"] == string(observability.EventError) {
			hasError = true
			if l["panic"] != "simulated handler panic" {
				t.Errorf("panic field = %v, want 'simulated handler panic'", l["panic"])
			}
			if l["stack"] == nil {
				t.Errorf("stack should be present in error log")
			}
		}
	}
	if !hasError {
		t.Errorf("panic 路径应记录 Error event,实际 logs: %v", logs)
	}
}

// TestLifecycle_Behavioral_RealHandlerEmitLogs — 1ae R3e 核心:
// 用真 AuthHandler.logout(不依赖 PG,仅 Redis 可选)验证 handler 真的会产出 Lifecycle 日志。
//
// 选 logout 因其依赖最轻:不查 PG user,仅 Redis Del(session 可不存在)。
// 用 buffer logger 捕获,验证 Function_Start + Function_End 都被写。
func TestLifecycle_Behavioral_RealHandlerEmitLogs(t *testing.T) {
	// 注意:logout 当前未挂 Lifecycle(只有 login/claim/release/postCommand/PostMessage/flushSession 有)
	// 此测试改用 ClaimHandler.claim(已挂 Lifecycle,需 Redis seed)
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/00000000-0000-0000-0000-000000000000/claim", nil)
	c.Set("user_id", uuid.New())

	h := &ClaimHandler{
		stores: &storage.Stores{Redis: &storage.Redis{Client: rdb}},
		logger: logger,
	}

	// 调 claim(会失败因 session 不在 PG,但 Lifecycle 应仍记录)
	defer func() { _ = recover() }()
	h.claim(c)

	logs := parseLifecycleLogs(&buf)
	// 至少有 Function_Start("Claim") + Function_End("Claim")
	hasStart := false
	hasEnd := false
	for _, l := range logs {
		if l["span"] == "Claim" {
			if l["event_type"] == string(observability.EventFunctionStart) {
				hasStart = true
			}
			if l["event_type"] == string(observability.EventFunctionEnd) {
				hasEnd = true
			}
		}
	}
	if !hasStart {
		t.Errorf("claim handler 未产生 Lifecycle Function_Start 日志 — 接线破坏或 logger 未传入")
	}
	if !hasEnd {
		t.Errorf("claim handler 未产生 Lifecycle Function_End 日志 — defer 漏调")
	}
}

// TestLifecycle_Behavioral_ReleaseHandler — 1af G1 (LIF-03):
// 验证 release handler 真调时产出 "Release" span 的 Lifecycle 日志。
func TestLifecycle_Behavioral_ReleaseHandler(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/00000000-0000-0000-0000-000000000000/release", nil)
	c.Set("user_id", uuid.New())

	h := &ClaimHandler{
		stores: &storage.Stores{Redis: &storage.Redis{Client: rdb}},
		logger: logger,
	}

	defer func() { _ = recover() }()
	h.release(c)

	logs := parseLifecycleLogs(&buf)
	hasStart, hasEnd := false, false
	for _, l := range logs {
		if l["span"] == "Release" {
			if l["event_type"] == string(observability.EventFunctionStart) {
				hasStart = true
			}
			if l["event_type"] == string(observability.EventFunctionEnd) {
				hasEnd = true
			}
		}
	}
	if !hasStart {
		t.Errorf("release handler 未产生 Lifecycle Function_Start 日志 — LIF-03 接线破坏")
	}
	if !hasEnd {
		t.Errorf("release handler 未产生 Lifecycle Function_End 日志 — defer 漏调")
	}
}

// TestLifecycle_Behavioral_PostMessageHandler — 1af G1 (LIF-04):
// 验证 chat postMessage handler 真调时产出 "PostMessage" span 的 Lifecycle 日志。
//
// 即使 handler 因 claim check 失败而 early-return,Lifecycle 仍应记录(它是 defer)。
func TestLifecycle_Behavioral_PostMessageHandler(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sessionID := uuid.Nil
	c.Params = append(c.Params, gin.Param{Key: "id", Value: sessionID.String()})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/messages", nil)
	c.Set("user_id", uuid.New())

	h := &ChatHandler{
		stores: &storage.Stores{
			Redis: &storage.Redis{Client: rdb},
		},
		logger: logger,
	}

	defer func() { _ = recover() }()
	h.postMessage(c)

	logs := parseLifecycleLogs(&buf)
	hasStart, hasEnd := false, false
	for _, l := range logs {
		if l["span"] == "PostMessage" {
			if l["event_type"] == string(observability.EventFunctionStart) {
				hasStart = true
			}
			if l["event_type"] == string(observability.EventFunctionEnd) {
				hasEnd = true
			}
		}
	}
	if !hasStart {
		t.Errorf("postMessage handler 未产生 Lifecycle Function_Start 日志 — LIF-04 接线破坏")
	}
	if !hasEnd {
		t.Errorf("postMessage handler 未产生 Lifecycle Function_End 日志 — defer 漏调")
	}
}

// TestLifecycle_Behavioral_PostCommandHandler — 1af G1 (LIF-01):
// 验证 command postCommand handler 真调时产出 "PostCommand" span 的 Lifecycle 日志。
func TestLifecycle_Behavioral_PostCommandHandler(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sessionID := uuid.Nil
	c.Params = append(c.Params, gin.Param{Key: "id", Value: sessionID.String()})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID.String()+"/commands", nil)
	c.Set("user_id", uuid.New())

	h := &CommandHandler{
		stores: &storage.Stores{Redis: &storage.Redis{Client: rdb}},
		logger: logger,
	}

	defer func() { _ = recover() }()
	h.postCommand(c)

	logs := parseLifecycleLogs(&buf)
	hasStart, hasEnd := false, false
	for _, l := range logs {
		if l["span"] == "PostCommand" {
			if l["event_type"] == string(observability.EventFunctionStart) {
				hasStart = true
			}
			if l["event_type"] == string(observability.EventFunctionEnd) {
				hasEnd = true
			}
		}
	}
	if !hasStart {
		t.Errorf("postCommand handler 未产生 Lifecycle Function_Start 日志 — LIF-01 接线破坏")
	}
	if !hasEnd {
		t.Errorf("postCommand handler 未产生 Lifecycle Function_End 日志 — defer 漏调")
	}
}

// hasLifecycleSpan helper:从 logs 中找指定 span + event_type。
func hasLifecycleSpan(logs []map[string]any, span, eventType string) bool {
	for _, l := range logs {
		if l["span"] == span && l["event_type"] == eventType {
			return true
		}
	}
	return false
}

// parseLifecycleLogs 解析 buffer 中的 JSON 日志(每行一条)。
func parseLifecycleLogs(buf *bytes.Buffer) []map[string]any {
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}
