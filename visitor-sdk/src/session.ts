// 访客身份与会话管理
// visitor_id（localStorage 持久化）+ session_id（server 签发）

const VISITOR_ID_KEY = 'mm:visitor_id';
const SESSION_ID_KEY = 'mm:session_id';

export interface SessionInfo {
  visitorId: string;
  sessionId: string;
  tenantId: string;
}

/** 获取或生成 visitor_id（localStorage 中持久化） */
export function getOrCreateVisitorId(): string {
  let id = localStorage.getItem(VISITOR_ID_KEY);
  if (!id) {
    id = generateUUID();
    localStorage.setItem(VISITOR_ID_KEY, id);
  }
  return id;
}

/**
 * 调用 POST /api/session/init 获取新的 session_id。
 * visitor_id 由 SDK 提供（来自 localStorage）。
 */
export async function initSession(
  visitorId: string,
  apiBase: string,
  ua: string,
): Promise<SessionInfo> {
  const url = `${apiBase}/api/session/init`;
  const resp = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ visitor_id: visitorId, ua }),
    credentials: 'include',
  });
  if (!resp.ok) {
    throw new Error(`session init failed: HTTP ${resp.status}`);
  }
  const data = await resp.json();
  const info: SessionInfo = {
    visitorId: data.visitor_id,
    sessionId: data.session_id,
    tenantId: data.tenant_id,
  };
  // 持久化 session_id（重连时复用）
  if (info.sessionId) {
    localStorage.setItem(SESSION_ID_KEY, info.sessionId);
  }
  return info;
}

/**
 * 从 localStorage 取上一个 session_id（用于重连）。
 * 调用方应验证此 session 仍有效（GET /api/sessions/:id 或 WS 握手 ack）。
 * 若 session 已失效，需要重新 initSession。
 */
export function getCachedSessionId(): string | null {
  return localStorage.getItem(SESSION_ID_KEY);
}

/** 清除本地缓存的 session_id（一般不需要调用） */
export function clearCachedSessionId(): void {
  localStorage.removeItem(SESSION_ID_KEY);
}

// crypto.randomUUID 在所有 modern evergreen 浏览器可用（与 PLAN.md 浏览器矩阵一致）
function generateUUID(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  // fallback（极旧环境，理论上 PLAN.md 浏览器矩阵下不会触发）
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}
