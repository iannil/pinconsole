// 1ad 测试:recording 包 lifecycle / LogExternalCall 接线源码契约(审计 T1-1s-LIF-05/06 + EXT-02/03)。
// 1af G1 扩展:加行为级测试验证 GC.runOnce Lifecycle 真产日志。
//
// 验证 FlushSession + GC.runOnce 都有 Lifecycle 埋点,
// 且 stream.go 的 minIO PutObject + PG CreateEventBlob 有 LogExternalCall。
package recording

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iannil/marketing-monitor/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestObservability_Lifecycle_OnFlushSession — T1-1s-LIF-05:
// stream.go FlushSession 必须有 observability.Lifecycle 埋点。
func TestObservability_Lifecycle_OnFlushSession(t *testing.T) {
	assertLifecycleInFile(t, "stream.go", "FlushSession")
}

// TestObservability_Lifecycle_OnGCRunOnce — T1-1s-LIF-06:
// gc.go runOnce 必须有 observability.Lifecycle 埋点。
func TestObservability_Lifecycle_OnGCRunOnce(t *testing.T) {
	assertLifecycleInFile(t, "gc.go", "GC.runOnce")
}

// TestObservability_LogExternalCall_MinIOPutObject — T1-1s-EXT-02:
// stream.go minio.PutObject 必须有 ok + error 两条 LogExternalCall 路径。
func TestObservability_LogExternalCall_MinIOPutObject(t *testing.T) {
	src := mustReadFile(t, "stream.go")
	for _, status := range []string{"ok", "error"} {
		needle := `observability.LogExternalCall(ctx, f.logger, "minio.PutObject", "` + status + `"`
		if !strings.Contains(src, needle) {
			t.Errorf("stream.go 缺失 LogExternalCall minio.PutObject status=%q", status)
		}
	}
}

// TestObservability_LogExternalCall_CreateEventBlob — T1-1s-EXT-03:
// stream.go pg.CreateEventBlob 必须有 LogExternalCall 路径。
func TestObservability_LogExternalCall_CreateEventBlob(t *testing.T) {
	src := mustReadFile(t, "stream.go")
	if !strings.Contains(src, `observability.LogExternalCall(ctx, f.logger, "pg.CreateEventBlob"`) {
		t.Errorf("stream.go 缺失 LogExternalCall pg.CreateEventBlob")
	}
}

// TestObservability_Compensation_MinIO_RemoveObject_OnPGFail — T1-1d-2:
// stream.go PG INSERT 失败时必须调 MinIO RemoveObject 补偿(防孤儿对象)。
// 详见 1o P1-7 修复。
func TestObservability_Compensation_MinIO_RemoveObject_OnPGFail(t *testing.T) {
	src := mustReadFile(t, "stream.go")
	// PutBytes 是 MinIO 上传入口,RemoveObject 是补偿删除
	putIdx := strings.Index(src, "MinIO.PutBytes(")
	if putIdx < 0 {
		t.Fatal("stream.go 缺失 MinIO.PutBytes(前置依赖)")
	}
	tail := src[putIdx:]
	if !strings.Contains(tail, "RemoveObject") {
		t.Errorf("stream.go PutBytes 之后缺失 RemoveObject 补偿(防孤儿对象)")
	}
}

func assertLifecycleInFile(t *testing.T, file, name string) {
	t.Helper()
	src := mustReadFile(t, file)
	needle := `observability.Lifecycle(ctx, "` + name + `"`
	if !strings.Contains(src, needle) {
		t.Errorf("%s 缺失 %q Lifecycle 埋点", file, needle)
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

// TestLifecycle_Behavioral_GCRunOnce — 1af G1 (LIF-06):
// 真调 GC.runOnce,验证产出 "GC.runOnce" span 的 Function_Start + Function_End 日志。
//
// 此前 source-contract 测试只 grep 字符串,不能捕获:
// - Lifecycle 调用包进 dead code(仍 grep 命中)
// - logger 字段未传入(nil logger silent no-op)
//
// 行为级测试用真 Redis + 真 PG + buffer logger,直接调 runOnce。
func TestLifecycle_Behavioral_GCRunOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("需要 PG + Redis")
	}

	// 真 PG
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, "postgres://mm:mm_dev@localhost:5432/marketing_monitor?sslmode=disable")
	if err != nil {
		t.Skipf("PG 不可用(%v),跳过", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("PG ping 失败(%v),跳过", err)
	}

	// buffer logger 捕获日志
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// 构造 GC(production 结构)
	gc := &GC{
		stores: &storage.Stores{PG: &storage.Postgres{Pool: pool}},
		logger: logger,
	}

	// 调 runOnce(可能 0 行删除,但 Lifecycle 必须记录)
	gc.runOnce(ctx)

	// 解析日志,找 GC.runOnce span
	logs := parseLifecycleLogsRec(&buf)
	hasStart, hasEnd := false, false
	for _, l := range logs {
		if l["span"] == "GC.runOnce" {
			if l["event_type"] == "function_start" || l["event_type"] == "Function_Start" {
				hasStart = true
			}
			if l["event_type"] == "function_end" || l["event_type"] == "Function_End" {
				hasEnd = true
			}
		}
	}
	if !hasStart {
		t.Errorf("GC.runOnce 未产生 Function_Start 日志 — LIF-06 接线破坏;logs=%v", logs)
	}
	if !hasEnd {
		t.Errorf("GC.runOnce 未产生 Function_End 日志 — defer 漏调;logs=%v", logs)
	}
}

// parseLifecycleLogsRec 解析 buffer 中的 JSON 日志(每行一条)。
// 复用 1ae 模式,加 _Rec 后缀避免与 api 包同名 helper 冲突。
func parseLifecycleLogsRec(buf *bytes.Buffer) []map[string]any {
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
