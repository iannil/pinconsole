package recording

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/iannil/marketing-monitor/internal/proto"
)

// TestEncodeBlobChecksum 验证 blob 编码的 checksum 与手动计算一致。
func TestEncodeBlobChecksum(t *testing.T) {
	// 构造 3 条 entry，每条是 envelope bytes
	envs := []proto.Envelope{
		{V: 1, Type: proto.MsgEvent, TS: 1700000000000, Payload: proto.EventPayload{Type: proto.EvMouseMove, TS: 1700000000000, MouseMove: &proto.MouseMoveData{X: 1, Y: 1}}},
		{V: 1, Type: proto.MsgEvent, TS: 1700000000001, Payload: proto.EventPayload{Type: proto.EvMouseMove, TS: 1700000000001, MouseMove: &proto.MouseMoveData{X: 2, Y: 2}}},
		{V: 1, Type: proto.MsgEvent, TS: 1700000000002, Payload: proto.EventPayload{Type: proto.EvClick, TS: 1700000000002, Click: &proto.ClickData{X: 3, Y: 3, Button: 0}}},
	}
	entries := make([]StreamEntry, 0, len(envs))
	for _, e := range envs {
		data, err := proto.Encode(e)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		entries = append(entries, StreamEntry{ID: "0", Data: data})
	}

	data, startedAt, endedAt, checksum, err := encodeBlob(entries)
	if err != nil {
		t.Fatalf("encodeBlob: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty blob")
	}
	if !startedAt.Before(endedAt) && !startedAt.Equal(endedAt) {
		t.Errorf("startedAt %v > endedAt %v", startedAt, endedAt)
	}

	// 验证 checksum
	sum := sha256.Sum256(data)
	expected := hex.EncodeToString(sum[:])
	if checksum != expected {
		t.Errorf("checksum: got %s, want %s", checksum, expected)
	}

	// 时间戳应来自 envelope.TS
	wantStart := time.UnixMilli(1700000000000)
	wantEnd := time.UnixMilli(1700000000002)
	if !startedAt.Equal(wantStart) {
		t.Errorf("startedAt: got %v, want %v", startedAt, wantStart)
	}
	if !endedAt.Equal(wantEnd) {
		t.Errorf("endedAt: got %v, want %v", endedAt, wantEnd)
	}
}

// TestEncodeBlobEmpty 验证空 entry 返回错误。
func TestEncodeBlobEmpty(t *testing.T) {
	_, _, _, _, err := encodeBlob(nil)
	if err == nil {
		t.Error("expected error for empty entries")
	}
}
