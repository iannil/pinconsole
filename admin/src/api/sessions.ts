// REST API 客户端：sessions 列表与 replay
// 详见 docs/progress/2026-06-17-slice-1d-spec.md §协议扩展

export interface EndedSession {
  session_id: string;
  visitor_id: string;
  fingerprint: string;
  started_at: number; // 毫秒时间戳
  ended_at: number;
  duration_ms: number;
  event_count: number;
  ua: string;
}

export interface ListEndedSessionsResponse {
  sessions: EndedSession[];
  total: number;
}

export interface RRWebEvent {
  type: number; // rrweb 原生类型
  timestamp: number;
  data: unknown;
}

export interface ReplayEventsResponse {
  session_id: string;
  events: RRWebEvent[];
  total: number;
  offset: number;
  limit: number;
  has_more: boolean;
}

export type SinceRange = '24h' | '7d' | '30d';

export async function listEndedSessions(
  since: SinceRange = '24h',
  limit = 200,
): Promise<ListEndedSessionsResponse> {
  const resp = await fetch(`/api/sessions/ended?since=${since}&limit=${limit}`);
  if (!resp.ok) throw new Error(`listEndedSessions: HTTP ${resp.status}`);
  return resp.json();
}

export async function getSessionReplay(
  sessionId: string,
  offset = 0,
  limit = 10000,
): Promise<ReplayEventsResponse> {
  const resp = await fetch(
    `/api/sessions/${encodeURIComponent(sessionId)}/replay?offset=${offset}&limit=${limit}`,
  );
  if (!resp.ok) throw new Error(`getSessionReplay: HTTP ${resp.status}`);
  return resp.json();
}

// ===== 1e：co-browsing 命令 =====

export type CommandType =
  | 'cursor_highlight'
  | 'click'
  | 'scroll'
  | 'fill_input'
  | 'navigate'
  | 'release_control'
  | 'show_popup'
  | 'chat_message';

export async function sendCommand(
  sessionId: string,
  type: CommandType,
  payload: Record<string, unknown>,
): Promise<{ ok: boolean }> {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(sessionId)}/command`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ type, payload }),
  });
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({}));
    throw new Error(`sendCommand: HTTP ${resp.status} ${err.error ?? ''}`);
  }
  return resp.json();
}

// ===== 1g：聊天消息 =====

export interface ChatMessageItem {
  id: number;
  sender: 'operator' | 'visitor';
  content: string;
  created_at: number;
}

export async function listMessages(
  sessionId: string,
  sinceId = 0,
): Promise<{ messages: ChatMessageItem[] }> {
  const resp = await fetch(
    `/api/sessions/${encodeURIComponent(sessionId)}/messages?since_id=${sinceId}`,
  );
  if (!resp.ok) throw new Error(`listMessages: HTTP ${resp.status}`);
  return resp.json();
}

export async function sendMessage(
  sessionId: string,
  content: string,
  sender: 'operator' | 'visitor' = 'operator',
): Promise<ChatMessageItem> {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(sessionId)}/messages`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content, sender }),
  });
  if (!resp.ok) throw new Error(`sendMessage: HTTP ${resp.status}`);
  return resp.json();
}
