// 1z P1-1:统一 API 客户端,自动生成 + 注入 X-Trace-Id 头,
// 实现 admin SPA → server 的 trace_id 端到端传播。
//
// 设计:
// - 每次调用生成新 trace_id(crypto.randomUUID 取前 32 字符,与服务端 16-byte hex 格式一致)
// - 发送 X-Trace-Id 请求头;服务端 TraceMiddleware 已实现"客户端发了就保留"
// - 响应头回传 X-Trace-Id 供调试(应与我们发的一致)
// - 调用方可通过返回的 traceId 字段关联日志

/**
 * 生成与 server logging.newID() 格式一致的 trace_id(32 字符 hex)。
 * crypto.randomUUID() 在所有 modern evergreen 浏览器可用(与 PLAN.md 浏览器矩阵一致)。
 */
export function generateTraceId(): string {
  // randomUUID 返回 "xxxxxxxx-xxxx-..." 共 36 字符;去 dash 后 32 字符 hex
  return crypto.randomUUID().replace(/-/g, '');
}

/**
 * 调用选项:透传给 fetch + 自定义 headers 合并。
 */
export interface ApiFetchOptions extends RequestInit {
  /** 跳过 trace_id 注入(默认 false)。健康检查等无业务语义的端点可用。 */
  skipTraceId?: boolean;
}

/**
 * API 响应包装:fetch 原始 Response + 本次调用使用的 trace_id。
 */
export interface ApiFetchResult<T> {
  data: T;
  response: Response;
  traceId: string;
}

/**
 * apiFetch 是统一 fetch 包装,自动注入 X-Trace-Id 头。
 *
 * 用法:
 *   const { data, traceId } = await apiFetch<MyType>('/api/foo', { method: 'POST', ... });
 *
 * 不像 fetch 那样直接返回 Response,因为我们要把 traceId 暴露给调用方写日志。
 */
export async function apiFetch<T>(url: string, opts: ApiFetchOptions = {}): Promise<ApiFetchResult<T>> {
  const { skipTraceId = false, headers: customHeaders, ...rest } = opts;

  const traceId = skipTraceId ? '' : generateTraceId();
  const headers = new Headers(customHeaders);
  if (traceId) {
    headers.set('X-Trace-Id', traceId);
  }

  const response = await fetch(url, {
    ...rest,
    headers,
    credentials: opts.credentials ?? 'include', // admin SPA 全部走 cookie 认证
  });

  // 响应头里的 trace_id 应与我们发的一致(或服务端新生成,若是跳过场景)
  const responseTraceId = response.headers.get('X-Trace-Id') || traceId;

  return {
    response,
    traceId: responseTraceId,
    data: undefined as unknown as T, // 调用方负责 .json() 或 .text()
  };
}

/**
 * apiJson 是 apiFetch 的便捷封装,自动解析 JSON 响应。
 * 非 2xx 抛 Error(含 traceId 方便排查)。
 */
export async function apiJson<T>(url: string, opts: ApiFetchOptions = {}): Promise<ApiFetchResult<T>> {
  const result = await apiFetch<T>(url, opts);
  if (!result.response.ok) {
    let detail = '';
    try {
      const errBody = await result.response.clone().json();
      detail = errBody.error ? `: ${errBody.error}` : '';
    } catch {
      detail = `: ${result.response.statusText}`;
    }
    throw new Error(`HTTP ${result.response.status}${detail} [trace_id=${result.traceId}]`);
  }
  const data = (await result.response.json()) as T;
  return { ...result, data };
}
