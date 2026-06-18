// WebSocket 消息封装（visitor-sdk 端，与 admin/server 同源）
// 协议规格：docs/progress/2026-06-17-slice-1b-spec.md §MessagePack Envelope

import type { EventType } from './events';

export const PROTOCOL_VERSION = 1 as const;

export type MessageType =
  | 'hello'
  | 'ack'
  | 'error'
  | 'event'
  | 'subscribe'
  | 'unsubscribe'
  | 'presence'
  | 'command';

export interface Envelope<T = unknown> {
  v: typeof PROTOCOL_VERSION;
  type: MessageType;
  session_id?: string;
  trace_id?: string;
  ts: number;
  payload?: T;
}

export interface HelloPayload {
  visitor_id: string;
  session_id: string;
  sdk_version: string;
  capabilities: Capabilities;
}

export interface Capabilities {
  events: EventType[];
  co_browsing: boolean;
  recording: boolean;
}

export interface AckPayload {
  ok: boolean;
}

export interface ErrorPayload {
  code: string;
  message: string;
}

export interface SubscribePayload {
  session_id: string;
}

export interface PresencePayload {
  event: 'online' | 'offline' | 'navigated';
  session_id: string;
  visitor_id: string;
  fingerprint: string;
  started_at: number;
  // 1f：navigated 事件的关联 session IDs
  old_session_id?: string;
  new_session_id?: string;
}
