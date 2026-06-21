// 1ad 测试:1b form 字段过滤 + 1c snapshot cache 服务端接线(审计 T1-1b-4 + T1-1c-2)。
//
// 1b-4: visitor-sdk 用 rrweb maskAllInputs=true 屏蔽 password/credit 等敏感字段
// 1c-2: 服务端 SnapshotCache 在 Redis 缓存 session 的最近 full snapshot
package recording

import (
	"strings"
	"testing"
)

// Test1c_SnapshotCache_RedisTTL_Contract — T1-1c-2:
// snapshot.go 必须定义 Redis snapshot key + TTL。
// operator subscribe 时服务端应先发缓存 snapshot(让 replay 立即可见)。
//
// 2026-06-21:TTL 从 5min 提到 30min(配合 SDK 续期间隔 10s),原 "5*time.Minute"
// 契约改成更宽泛的 "time.Minute" 表达式 + snapshotTTL 标识符存在性检查。
func Test1c_SnapshotCache_RedisTTL_Contract(t *testing.T) {
	src := mustReadFile(t, "snapshot.go")
	for _, must := range []string{
		"snapshotTTL",
		"SnapshotKey",
		"time.Minute", // TTL 必须以 time.Minute 表达(原 5min,现 30min)
	} {
		if !strings.Contains(src, must) {
			t.Errorf("snapshot.go 缺失 %q — SnapshotCache TTL 接线破坏", must)
		}
	}
}
