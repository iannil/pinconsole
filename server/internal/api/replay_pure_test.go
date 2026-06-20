// 1ai-h 测试:replay.go 纯函数测试。
//
// 4 个 0% 函数补全(零依赖,无 PG/Redis/MinIO):
//   - decodeRRWebEventsFromBlob
//   - extractRRWebEventsFromPayload
//   - decodePayloadAsEvent
//   - eventPayloadToMap
package api

import (
	"testing"

	"github.com/iannil/pinconsole/internal/proto"
	"github.com/vmihailenco/msgpack/v5"
)

// TestEventPayloadToMap_NilRRWeb_ReturnsNil — RRWeb=nil 时返回 nil(非 panic)。
func TestEventPayloadToMap_NilRRWeb_ReturnsNil(t *testing.T) {
	got := eventPayloadToMap(proto.EventPayload{
		Type:  proto.EvRRWeb,
		RRWeb: nil,
	})
	if got != nil {
		t.Errorf("eventPayloadToMap(NIL RRWeb) = %v, want nil", got)
	}
}

// TestEventPayloadToMap_NonRRWebType_ReturnsNil — Type 非 EvRRWeb 时返 nil。
func TestEventPayloadToMap_NonRRWebType_ReturnsNil(t *testing.T) {
	got := eventPayloadToMap(proto.EventPayload{
		Type: proto.EvClick, // 不是 EvRRWeb
		RRWeb: &proto.RRWebEvent{
			Type:      2,
			Timestamp: 1000,
			Data:      map[string]any{"x": 1},
		},
	})
	if got != nil {
		t.Errorf("eventPayloadToMap(EvClick) = %v, want nil(类型过滤)", got)
	}
}

// TestEventPayloadToMap_ValidRRWeb_ReturnsMap — 合法 RRWeb 返回 map(type/timestamp/data)。
func TestEventPayloadToMap_ValidRRWeb_ReturnsMap(t *testing.T) {
	got := eventPayloadToMap(proto.EventPayload{
		Type: proto.EvRRWeb,
		RRWeb: &proto.RRWebEvent{
			Type:      2,
			Timestamp: 1700000000,
			Data:      map[string]any{"href": "https://example.com"},
		},
	})
	if got == nil {
		t.Fatal("eventPayloadToMap 返 nil")
	}
	if got["type"].(int) != 2 {
		t.Errorf("type = %v, want 2", got["type"])
	}
	if got["timestamp"].(int64) != 1700000000 {
		t.Errorf("timestamp = %v, want 1700000000", got["timestamp"])
	}
	dataMap, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("data 不是 map[string]any: %T", got["data"])
	}
	if dataMap["href"] != "https://example.com" {
		t.Errorf("data.href = %v, want https://example.com", dataMap["href"])
	}
}

// TestDecodePayloadAsEvent_NotEvent_ReturnsErr — 非 event 格式 payload → 返 error。
func TestDecodePayloadAsEvent_NotEvent_ReturnsErr(t *testing.T) {
	// payload 是数组,不是单个 event
	arrPayload := []proto.EventPayload{
		{Type: proto.EvRRWeb, RRWeb: &proto.RRWebEvent{Type: 2}},
	}
	encoded, _ := msgpack.Marshal(arrPayload)

	got, err := decodePayloadAsEvent(encoded)
	if err == nil && got != nil {
		t.Errorf("decodePayloadAsEvent(数组) 应返 nil + err, got map=%v err=nil", got)
	}
}

// TestDecodeRRWebEventsFromBlob_EmptyInput_ReturnsEmpty — 空 array 输入返空。
func TestDecodeRRWebEventsFromBlob_EmptyInput_ReturnsEmpty(t *testing.T) {
	// 空数组 []] 的 msgpack 编码
	emptyArray, _ := msgpack.Marshal([][]byte{})

	got, err := decodeRRWebEventsFromBlob(emptyArray)
	if err != nil {
		t.Fatalf("decodeRRWebEventsFromBlob(空 array): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("返 = %v, want empty slice", got)
	}
}

// TestDecodeRRWebEventsFromBlob_InvalidMsgpack_ReturnsErr — 非 msgpack 输入 → error。
func TestDecodeRRWebEventsFromBlob_InvalidMsgpack_ReturnsErr(t *testing.T) {
	_, err := decodeRRWebEventsFromBlob([]byte("not-msgpack"))
	if err == nil {
		t.Error("decodeRRWebEventsFromBlob(非 msgpack) 应返 error")
	}
}

// TestExtractRRWebEventsFromPayload_NilPayload_ReturnsEmpty — nil payload → 空 slice(非 panic)。
func TestExtractRRWebEventsFromPayload_NilPayload_ReturnsEmpty(t *testing.T) {
	got := extractRRWebEventsFromPayload(nil)
	if len(got) != 0 {
		t.Errorf("extractRRWebEventsFromPayload(nil) = %v, want empty", got)
	}
}
