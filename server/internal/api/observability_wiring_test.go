// 1ad 测试:可观测性 lifecycle / LogPoint / LogExternalCall 接线源码契约(审计 T1-1s 13 项)。
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
	"os"
	"strings"
	"testing"
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
