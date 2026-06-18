// Package observability 提供 CLAUDE.md "可观测性开发" 章节要求的能力:
//   - LifecycleTracker(Go defer 闭包模式)
//   - event_type 字段(Function_Start/End/Branch/Error/External_Call)
//   - LogPoint helper(关键节点埋点)
package observability

// EventType 是 CLAUDE.md 要求的 5 类事件类型。
type EventType string

const (
	EventFunctionStart EventType = "Function_Start"
	EventFunctionEnd   EventType = "Function_End"
	EventBranch        EventType = "Branch"
	EventError         EventType = "Error"
	EventExternalCall  EventType = "External_Call"
)
