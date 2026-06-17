// Package proto：访客事件类型（discriminated union）。
//
// 切片 1b 仅含：mouse_move / click / scroll / form_submit。
// 切片 1c 起加 rrweb 全量事件（IncrementalSnapshot / FullSnapshot 等）。
package proto

// EventType 是 visitor 事件类型。
type EventType string

const (
	EvMouseMove   EventType = "mouse_move"
	EvClick       EventType = "click"
	EvScroll      EventType = "scroll"
	EvFormSubmit  EventType = "form_submit"
	EvRRWeb       EventType = "rrweb" // 切片 1c 新增
)

// EventPayload 是 MsgEvent 的 payload（含事件类型与具体数据）。
// 1b 实现四类事件，1c 加 RRWeb 字段。
// 1c SDK 完全替换 4 类 collector 为 rrweb，1b 字段保留以便兼容历史 blob 回放。
type EventPayload struct {
	Type       EventType       `msgpack:"type"`
	TS         int64           `msgpack:"ts"` // 事件原始时间（毫秒）
	MouseMove  *MouseMoveData  `msgpack:"mouse_move,omitempty"`
	Click      *ClickData      `msgpack:"click,omitempty"`
	Scroll     *ScrollData     `msgpack:"scroll,omitempty"`
	FormSubmit *FormSubmitData `msgpack:"form_submit,omitempty"`
	RRWeb      *RRWebEvent     `msgpack:"rrweb,omitempty"` // 1c
}

// RRWebEvent 是单个 rrweb 事件（与 rrweb v2 SnapshotEvent 对应）。
// Data 字段类型复杂（rrweb-snapshot 内部类型），1c 用 raw bytes 透传不类型化。
type RRWebEvent struct {
	Type      int             `msgpack:"type"`      // rrweb 事件类型枚举（FullSnapshot=2, IncrementalSnapshot=3, Meta=4）
	Timestamp int64           `msgpack:"timestamp"` // rrweb 原始时间戳（毫秒）
	Data      map[string]any  `msgpack:"data"`      // rrweb data 字段（含 node tree、source、positions 等）
}

// MouseMoveData 是 mouse_move 事件的 data。
type MouseMoveData struct {
	X int `msgpack:"x"`
	Y int `msgpack:"y"`
}

// ClickData 是 click 事件的 data。
type ClickData struct {
	X              int    `msgpack:"x"`
	Y              int    `msgpack:"y"`
	Button         int    `msgpack:"button"` // 0=左, 1=中, 2=右
	TargetSelector string `msgpack:"target_selector,omitempty"`
}

// ScrollData 是 scroll 事件的 data。
type ScrollData struct {
	X int `msgpack:"x"`
	Y int `msgpack:"y"`
}

// FormSubmitData 是 form_submit 事件的 data。
type FormSubmitData struct {
	FormID string            `msgpack:"form_id,omitempty"`
	Fields map[string]string `msgpack:"fields"`
}
