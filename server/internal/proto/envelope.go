// Package proto 定义 WebSocket 消息封装与事件类型。
//
// 协议规格详见 docs/standards/proto-spec.md（待补）与
// docs/progress/2026-06-17-slice-1b-spec.md §MessagePack Envelope。
//
// Go 与 TS 端从此处手写代码同步；当协议稳定后引入 codegen（quicktype/buf）。
package proto

import (
	"github.com/vmihailenco/msgpack/v5"
)

// ProtocolVersion 是 envelope 版本。
const ProtocolVersion uint8 = 1

// MessageType 是 envelope.type 取值。
type MessageType string

const (
	MsgHello       MessageType = "hello"       // SDK 握手
	MsgAck         MessageType = "ack"          // 后端 ack
	MsgError       MessageType = "error"        // 错误
	MsgEvent       MessageType = "event"        // SDK → 后端：事件
	MsgSubscribe   MessageType = "subscribe"    // admin → 后端：订阅 session
	MsgUnsubscribe MessageType = "unsubscribe"  // admin → 后端：退订 session
	MsgPresence    MessageType = "presence"     // 后端 → admin：访客上线/下线
	MsgCommand     MessageType = "command"      // 后端 → SDK（1e 反向下行命令）
)

// Envelope 是所有 WS 消息的统一外壳。
//
// 与 docs/progress/2026-06-17-slice-1b-spec.md §MessagePack Envelope 一致。
type Envelope struct {
	V         uint8   `msgpack:"v"`
	Type      MessageType `msgpack:"type"`
	SessionID string  `msgpack:"session_id,omitempty"`
	TraceID   string  `msgpack:"trace_id,omitempty"`
	TS        int64   `msgpack:"ts"`                 // 毫秒时间戳
	Payload   any     `msgpack:"payload,omitempty"`  // 按 Type 分支
}

// Encode 把 envelope 序列化为 MessagePack 字节流。
func Encode(e Envelope) ([]byte, error) {
	return msgpack.Marshal(e)
}

// Decode 从 MessagePack 字节流解析 envelope（不含 payload，需要按 Type 二次解码）。
func Decode(b []byte) (Envelope, error) {
	var e Envelope
	if err := msgpack.Unmarshal(b, &e); err != nil {
		return Envelope{}, err
	}
	return e, nil
}

// DecodePayload 把 envelope.Payload（通常是 msgpack map）解析到具体类型。
// 调用方需根据 envelope.Type 选择目标类型。
func DecodePayload(payload any, dst any) error {
	// payload 已被 msgpack 解析为 map[string]interface{} 或类似结构
	// 重新序列化再解析到目标类型
	raw, err := msgpack.Marshal(payload)
	if err != nil {
		return err
	}
	return msgpack.Unmarshal(raw, dst)
}

// HelloPayload 是 MsgHello 的 payload。
type HelloPayload struct {
	VisitorID   string         `msgpack:"visitor_id"`
	SessionID   string         `msgpack:"session_id"`
	SDKVersion  string         `msgpack:"sdk_version"`
	Capabilities Capabilities  `msgpack:"capabilities"`
}

// Capabilities 描述客户端能力（握手时声明）。
type Capabilities struct {
	Events      []string `msgpack:"events"`
	CoBrowsing  bool     `msgpack:"co_browsing"`
	Recording   bool     `msgpack:"recording"`
}

// AckPayload 是 MsgAck 的 payload。
type AckPayload struct {
	OK bool `msgpack:"ok"`
}

// ErrorPayload 是 MsgError 的 payload。
type ErrorPayload struct {
	Code    string `msgpack:"code"`
	Message string `msgpack:"message"`
}

// SubscribePayload 是 MsgSubscribe / MsgUnsubscribe 的 payload。
type SubscribePayload struct {
	SessionID string `msgpack:"session_id"`
}

// PresencePayload 是 MsgPresence 的 payload。
type PresencePayload struct {
	Event       string `msgpack:"event"` // online / offline / navigated (1f)
	SessionID   string `msgpack:"session_id"`
	VisitorID   string `msgpack:"visitor_id"`
	Fingerprint string `msgpack:"fingerprint"`
	StartedAt   int64  `msgpack:"started_at"` // 毫秒时间戳
	// 1f：navigated 事件的关联 session IDs
	OldSessionID string `msgpack:"old_session_id,omitempty"`
	NewSessionID string `msgpack:"new_session_id,omitempty"`
}

// CommandPayload 是 MsgCommand 的 payload（1e）。
// 5 类核心命令共用此结构，按 Type 分支使用对应字段。
type CommandPayload struct {
	Type     string                 `msgpack:"type"`     // cursor_highlight / click / scroll / fill_input / navigate / release_control
	TS        int64               `msgpack:"ts"`
	Cursor    *CommandCursor      `msgpack:"cursor,omitempty"`
	Click     *CommandClick       `msgpack:"click,omitempty"`
	Scroll    *CommandScroll      `msgpack:"scroll,omitempty"`
	FillInput *CommandFillInput   `msgpack:"fill_input,omitempty"`
	Navigate  *CommandNavigate    `msgpack:"navigate,omitempty"`
	Popup     *CommandPopup       `msgpack:"popup,omitempty"`     // 1g
	Chat      *CommandChatMessage `msgpack:"chat,omitempty"`       // 1g
}

// CommandPopup 是 show_popup 命令的 data（1g）。
type CommandPopup struct {
	Title       string `msgpack:"title"`
	Body        string `msgpack:"body"`
	ActionLabel string `msgpack:"action_label,omitempty"`
	ActionURL   string `msgpack:"action_url,omitempty"`
	Dismissible bool   `msgpack:"dismissible"`
}

// CommandChatMessage 是 chat_message 命令的 data（1g，admin→访客）。
type CommandChatMessage struct {
	MessageID int64  `msgpack:"message_id"`
	Content   string `msgpack:"content"`
}

// CommandCursor 是 cursor_highlight 命令的 data。
type CommandCursor struct {
	X    int    `msgpack:"x"`
	Y    int    `msgpack:"y"`
	Name string `msgpack:"name"` // 运营名字（admin 端识别）
}

// CommandClick 是 click 命令的 data。
type CommandClick struct {
	NodeID int `msgpack:"node_id"` // rrweb-snapshot nodeID
	X      int `msgpack:"x"`
	Y      int `msgpack:"y"`
}

// CommandScroll 是 scroll 命令的 data。
type CommandScroll struct {
	X int `msgpack:"x"`
	Y int `msgpack:"y"`
}

// CommandFillInput 是 fill_input 命令的 data。
type CommandFillInput struct {
	NodeID int    `msgpack:"node_id"`
	Value  string `msgpack:"value"`
}

// CommandNavigate 是 navigate 命令的 data。
type CommandNavigate struct {
	URL string `msgpack:"url"`
}
