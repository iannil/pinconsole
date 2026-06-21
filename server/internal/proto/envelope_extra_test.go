// Go-1 切片补测:DecodePayload 边界路径,提升 proto 包覆盖率 88.9% → 100%。
package proto

import (
	"testing"
	"time"
)

// TestDecodePayload_RoundTrip 验证 payload 二次序列化/反序列化到目标类型。
func TestDecodePayload_RoundTrip(t *testing.T) {
	original := HelloPayload{
		VisitorID:  "v-rt",
		SessionID:  "s-rt",
		SDKVersion: "0.2.0",
		Capabilities: Capabilities{
			Events:     []string{"mousemove", "click"},
			CoBrowsing: true,
			Recording:  false,
		},
	}

	// 模拟 msgpack 解码后 payload 的中间形态:map[string]interface{}
	raw, err := Encode(Envelope{
		V:       ProtocolVersion,
		Type:    MsgHello,
		TS:      time.Now().UnixMilli(),
		Payload: original,
	})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	env, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	var got HelloPayload
	if err := DecodePayload(env.Payload, &got); err != nil {
		t.Fatalf("DecodePayload: %v", err)
	}
	if got.VisitorID != original.VisitorID {
		t.Errorf("VisitorID: got %q, want %q", got.VisitorID, original.VisitorID)
	}
	if got.SessionID != original.SessionID {
		t.Errorf("SessionID: got %q, want %q", got.SessionID, original.SessionID)
	}
	if got.SDKVersion != original.SDKVersion {
		t.Errorf("SDKVersion: got %q, want %q", got.SDKVersion, original.SDKVersion)
	}
	if !got.Capabilities.CoBrowsing {
		t.Errorf("Capabilities.CoBrowsing: got false, want true")
	}
	if len(got.Capabilities.Events) != 2 {
		t.Errorf("Capabilities.Events len: got %d, want 2", len(got.Capabilities.Events))
	}
}

// TestDecodePayload_PrimitivePayload 验证原始类型 payload(int/string)也能二次解码。
func TestDecodePayload_PrimitivePayload(t *testing.T) {
	cases := []struct {
		name    string
		payload any
		want    int64
	}{
		{"int", 42, 42},
		{"int8", int8(7), 7},
		{"int16", int16(123), 123},
		{"int32", int32(456), 456},
		{"int64", int64(789), 789},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got int64
			if err := DecodePayload(tc.payload, &got); err != nil {
				t.Fatalf("DecodePayload(%v): %v", tc.name, err)
			}
			if got != tc.want {
				t.Errorf("DecodePayload(%v): got %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}

// TestDecodePayload_StringPayload 验证 string payload 解码到 string 目标。
func TestDecodePayload_StringPayload(t *testing.T) {
	var got string
	if err := DecodePayload("hello-pinconsole", &got); err != nil {
		t.Fatalf("DecodePayload: %v", err)
	}
	if got != "hello-pinconsole" {
		t.Errorf("got %q, want %q", got, "hello-pinconsole")
	}
}

// TestDecodePayload_BoolPayload 验证 bool payload 解码。
func TestDecodePayload_BoolPayload(t *testing.T) {
	var got bool
	if err := DecodePayload(true, &got); err != nil {
		t.Fatalf("DecodePayload: %v", err)
	}
	if !got {
		t.Errorf("got false, want true")
	}
}

// TestDecodePayload_IncompatibleDestination 验证类型不兼容时返回错误。
// msgpack 无法把 int 编码到 chan/func 等不兼容目标。
func TestDecodePayload_IncompatibleDestination(t *testing.T) {
	// 用 nil dst 触发 Unmarshal 错误
	err := DecodePayload(42, nil)
	if err == nil {
		t.Errorf("expected error for nil dst, got nil")
	}
}

// TestDecodePayload_MarshalFailure 验证 payload 无法 marshal 时返回错误。
// msgpack 无法 marshal chan 类型,触发 Marshal error 路径(覆盖 75%→100% 的关键 gap)。
func TestDecodePayload_MarshalFailure(t *testing.T) {
	// chan 是 msgpack 无法 marshal 的类型
	ch := make(chan int)
	var dst int
	err := DecodePayload(ch, &dst)
	if err == nil {
		t.Errorf("expected marshal error for chan payload, got nil")
	}
}
