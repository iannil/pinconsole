package proto

import (
	"bytes"
	"testing"
	"time"
)

// TestEnvelopeEncodeDecode 验证 envelope 的 msgpack round-trip。
func TestEnvelopeEncodeDecode(t *testing.T) {
	cases := []struct {
		name string
		env  Envelope
	}{
		{
			name: "hello",
			env: Envelope{
				V:    ProtocolVersion,
				Type: MsgHello,
				TS:   time.Now().UnixMilli(),
				Payload: HelloPayload{
					VisitorID:  "v-123",
					SessionID:  "s-456",
					SDKVersion: "0.2.0",
					Capabilities: Capabilities{
						Events:     []string{"mouse_move", "click"},
						CoBrowsing: false,
						Recording:  true,
					},
				},
			},
		},
		{
			name: "ack",
			env: Envelope{
				V:       ProtocolVersion,
				Type:    MsgAck,
				TS:      1700000000000,
				Payload: AckPayload{OK: true},
			},
		},
		{
			name: "presence_online",
			env: Envelope{
				V:         ProtocolVersion,
				Type:      MsgPresence,
				SessionID: "session-abc",
				TS:        1700000000000,
				Payload: PresencePayload{
					Event:       "online",
					SessionID:   "session-abc",
					VisitorID:   "v-123",
					Fingerprint: "v-123",
					StartedAt:   1700000000000,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := Encode(tc.env)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if len(data) == 0 {
				t.Fatal("empty encoded data")
			}

			got, err := Decode(data)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.V != tc.env.V {
				t.Errorf("V: got %d, want %d", got.V, tc.env.V)
			}
			if got.Type != tc.env.Type {
				t.Errorf("Type: got %q, want %q", got.Type, tc.env.Type)
			}
			if got.SessionID != tc.env.SessionID {
				t.Errorf("SessionID: got %q, want %q", got.SessionID, tc.env.SessionID)
			}
			if got.TS != tc.env.TS {
				t.Errorf("TS: got %d, want %d", got.TS, tc.env.TS)
			}
		})
	}
}

// TestDecodeInvalidBytes 验证垃圾输入返回错误。
func TestDecodeInvalidBytes(t *testing.T) {
	_, err := Decode([]byte{0x00, 0x01, 0x02})
	if err == nil {
		t.Error("expected decode error for garbage input")
	}
}

// TestEventPayloadRoundTrip 验证 EventPayload 编解码（含四类事件）。
func TestEventPayloadRoundTrip(t *testing.T) {
	cases := []EventPayload{
		{
			Type:      EvMouseMove,
			TS:        1700000000000,
			MouseMove: &MouseMoveData{X: 100, Y: 200},
		},
		{
			Type: EvClick,
			TS:   1700000000001,
			Click: &ClickData{
				X:              50,
				Y:              75,
				Button:         0,
				TargetSelector: "button#submit",
			},
		},
		{
			Type:   EvScroll,
			TS:     1700000000002,
			Scroll: &ScrollData{X: 0, Y: 1024},
		},
		{
			Type: EvFormSubmit,
			TS:   1700000000003,
			FormSubmit: &FormSubmitData{
				FormID: "contact",
				Fields: map[string]string{"name": "张三", "phone": "13800138000"},
			},
		},
	}

	for i, payload := range cases {
		// 模拟 envelope → msgpack → 解析
		env := Envelope{
			V:       ProtocolVersion,
			Type:    MsgEvent,
			TS:      payload.TS,
			Payload: payload,
		}
		encoded, err := Encode(env)
		if err != nil {
			t.Fatalf("case %d encode: %v", i, err)
		}
		if bytes.Contains(encoded, []byte("invalid")) {
			t.Errorf("case %d: encoded contains marker", i)
		}

		decoded, err := Decode(encoded)
		if err != nil {
			t.Fatalf("case %d decode: %v", i, err)
		}

		var got EventPayload
		if err := DecodePayload(decoded.Payload, &got); err != nil {
			t.Fatalf("case %d decode payload: %v", i, err)
		}
		if got.Type != payload.Type {
			t.Errorf("case %d Type: got %s, want %s", i, got.Type, payload.Type)
		}
	}
}
