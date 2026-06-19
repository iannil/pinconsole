// 1ad 续集测试:1d Flusher + R2 上传接线源码契约(审计 T1-1d-1/3)。
//
// T1-1d-1: flushSession 必须调 MinIO.PutObject 上传 blob
// T1-1d-3: Flusher 必须 Register/Unregister 完整生命周期(ws.go 调用)
//
// T1-1d-2(MinIO RemoveObject 补偿)已在 observability_wiring_test.go cover。
// T1-1d-4/5 已在 1ac cover(GC + erasure MinIO)。
package recording

import (
	"strings"
	"testing"
)

// Test1d_Flusher_HasRegisterUnregister — T1-1d-3:
// Flusher 必须有 Register + Unregister 方法,且 Unregister 同步 flush 最后一批。
func Test1d_Flusher_HasRegisterUnregister(t *testing.T) {
	src := mustReadFile(t, "stream.go")

	for _, must := range []string{
		"func (f *Flusher) Register(",
		"func (f *Flusher) Unregister(",
		"func (f *Flusher) Start(",
		"func (f *Flusher) Stop(",
		"func (f *Flusher) tick(",
		"func (f *Flusher) flushSession(",
	} {
		if !strings.Contains(src, must) {
			t.Errorf("stream.go 缺失 %q — Flusher API 破坏", must)
		}
	}
}

// Test1d_FlushSession_WiresMinIOPut — T1-1d-1:
// flushSession 必须调 MinIO PutBytes(或等价 PutObject)上传 blob。
func Test1d_FlushSession_WiresMinIOPut(t *testing.T) {
	src := mustReadFile(t, "stream.go")
	idx := strings.Index(src, "func (f *Flusher) flushSession")
	if idx < 0 {
		t.Fatal("找不到 flushSession")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	if !strings.Contains(fnBody, "MinIO.PutBytes(") {
		t.Errorf("flushSession 缺失 MinIO.PutBytes 调用 — R2 上传破坏")
	}
}

// Test1d_FlushSession_WiresPGInsertAndRedisXTRIM — T1-1d-1 副验:
// flushSession 完整流程:MinIO Put → PG INSERT event_blobs → Redis XTRIM。
func Test1d_FlushSession_WiresPGInsertAndRedisXTRIM(t *testing.T) {
	src := mustReadFile(t, "stream.go")
	idx := strings.Index(src, "func (f *Flusher) flushSession")
	if idx < 0 {
		t.Fatal("找不到 flushSession")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	for _, must := range []string{
		"CreateEventBlob", // PG INSERT
		"XTRIM",           // Redis stream trim
	} {
		if !strings.Contains(fnBody, must) {
			t.Errorf("flushSession 缺失 %q — flush 流程破坏(MinIO Put → PG INSERT → Redis XTRIM)", must)
		}
	}
}

// Test1d_GC_WiresChatAndCommandsCleanup — T1-1d-4 副验(部分已在 1ac cover):
// gc.go runOnce 必须清 chat_messages + co_browsing_commands(不只 event_blobs)。
func Test1d_GC_WiresChatAndCommandsCleanup(t *testing.T) {
	src := mustReadFile(t, "gc.go")
	idx := strings.Index(src, "func (g *GC) runOnce")
	if idx < 0 {
		t.Fatal("找不到 runOnce")
	}
	end := strings.Index(src[idx+1:], "\nfunc ")
	if end < 0 {
		end = len(src) - idx - 1
	}
	fnBody := src[idx : idx+1+end]

	for _, must := range []string{
		"ListChatMessagesOlderThan",
		"DeleteChatMessagesByID",
		"DeleteCoBrowsingCommandsOlderThan",
		"DeleteSessionsEndedBefore",
		"DeleteVisitorsLastSeenBefore",
	} {
		if !strings.Contains(fnBody, must) {
			t.Errorf("runOnce 缺失 %q — GC 5 表清理破坏", must)
		}
	}
}
