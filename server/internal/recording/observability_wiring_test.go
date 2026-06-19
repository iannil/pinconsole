// 1ad 测试:recording 包 lifecycle / LogExternalCall 接线源码契约(审计 T1-1s-LIF-05/06 + EXT-02/03)。
//
// 验证 FlushSession + GC.runOnce 都有 Lifecycle 埋点,
// 且 stream.go 的 minIO PutObject + PG CreateEventBlob 有 LogExternalCall。
package recording

import (
	"os"
	"strings"
	"testing"
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
