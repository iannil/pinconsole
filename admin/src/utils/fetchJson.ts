// 1h-ui:全局 fetch 包装
// 401 → 触发未授权回调(由 auth store 注册),统一处理 session 过期。
// 业务代码应用 fetchJson 而非 fetch,确保 401 自动重定向。

export interface FetchJsonOptions extends RequestInit {
  // queryParams?: Record<string, string>;  // 未来可加 query string 序列化
}

export type UnauthorizedHandler = () => void;

let unauthorizedHandler: UnauthorizedHandler | null = null;

export function setUnauthorizedHandler(h: UnauthorizedHandler): void {
  unauthorizedHandler = h;
}

export async function fetchJson<T = unknown>(url: string, opts: FetchJsonOptions = {}): Promise<T> {
  const resp = await fetch(url, {
    ...opts,
    credentials: opts.credentials ?? 'include',
    headers: {
      ...(opts.headers ?? {}),
      ...(opts.body ? { 'Content-Type': 'application/json' } : {}),
    },
  });

  if (resp.status === 401) {
    if (unauthorizedHandler) unauthorizedHandler();
    throw new Error('UNAUTHORIZED');
  }

  if (!resp.ok) {
    let detail = '';
    try {
      const data = await resp.json();
      detail = data.error || data.detail || '';
    } catch {
      // ignore
    }
    throw new Error(detail || `HTTP ${resp.status}`);
  }

  if (resp.status === 204) return undefined as T;
  return resp.json() as Promise<T>;
}
