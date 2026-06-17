// 访客事件类型（discriminated union，与 admin/server 同源）
// 1b 含：mouse_move / click / scroll / form_submit
// 1c 加：rrweb（覆盖前面 4 类）

export type EventType = 'mouse_move' | 'click' | 'scroll' | 'form_submit' | 'rrweb';

export interface MouseMoveData {
  x: number;
  y: number;
}

export interface ClickData {
  x: number;
  y: number;
  button: number;
  target_selector?: string;
}

export interface ScrollData {
  x: number;
  y: number;
}

export interface FormSubmitData {
  form_id?: string;
  fields: Record<string, string>;
}

/**
 * RRWebEvent 与 rrweb v2 的 SnapshotEvent 对应。
 * data 字段类型复杂，保留为 unknown 透传。
 */
export interface RRWebEvent {
  type: number; // rrweb 事件类型枚举：FullSnapshot=2, IncrementalSnapshot=3, Meta=4
  timestamp: number;
  data: unknown;
}

// 1b 用组合字段（与 Go 端 EventPayload 对应）
// 1c 加 rrweb 字段
export interface EventPayload {
  type: EventType;
  ts: number;
  mouse_move?: MouseMoveData;
  click?: ClickData;
  scroll?: ScrollData;
  form_submit?: FormSubmitData;
  rrweb?: RRWebEvent;
}

// 工厂函数：避免每次构造完整对象
export const mouseMove = (x: number, y: number, ts: number): EventPayload => ({
  type: 'mouse_move',
  ts,
  mouse_move: { x, y },
});

export const click = (
  x: number,
  y: number,
  button: number,
  targetSelector: string | undefined,
  ts: number,
): EventPayload => ({
  type: 'click',
  ts,
  click: { x, y, button, target_selector: targetSelector },
});

export const scroll = (x: number, y: number, ts: number): EventPayload => ({
  type: 'scroll',
  ts,
  scroll: { x, y },
});

export const formSubmit = (
  formId: string | undefined,
  fields: Record<string, string>,
  ts: number,
): EventPayload => ({
  type: 'form_submit',
  ts,
  form_submit: { form_id: formId, fields },
});

export const rrwebEvent = (rrweb: RRWebEvent): EventPayload => ({
  type: 'rrweb',
  ts: rrweb.timestamp,
  rrweb,
});

/** 判断是否为 rrweb FullSnapshot 事件（用于服务端缓存识别） */
export const isFullSnapshot = (e: EventPayload): boolean =>
  e.type === 'rrweb' && e.rrweb?.type === 2;
