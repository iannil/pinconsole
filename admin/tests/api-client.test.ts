// 1z P1-1:验证 admin/src/api/client.ts 的 trace_id 生成 + X-Trace-Id 注入。
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { generateTraceId, apiFetch, apiJson } from '../src/api/client';

describe('generateTraceId', () => {
  it('returns 32-char hex (matches server logging.newID format)', () => {
    const id = generateTraceId();
    expect(id).toMatch(/^[0-9a-f]{32}$/);
  });

  it('returns unique values across calls', () => {
    const ids = new Set<string>();
    for (let i = 0; i < 100; i++) {
      ids.add(generateTraceId());
    }
    expect(ids.size).toBe(100);
  });
});

describe('apiFetch', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('injects X-Trace-Id header by default', async () => {
    const mockResp = new Response('{"ok":true}', {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResp);

    await apiFetch('/api/test');

    expect(globalThis.fetch).toHaveBeenCalledTimes(1);
    const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(url).toBe('/api/test');
    const headers = (init as RequestInit).headers as Headers;
    expect(headers.get('X-Trace-Id')).toMatch(/^[0-9a-f]{32}$/);
  });

  it('skips X-Trace-Id when skipTraceId:true', async () => {
    const mockResp = new Response('ok', { status: 200 });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResp);

    await apiFetch('/healthz', { skipTraceId: true });

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    const headers = (init as RequestInit).headers as Headers;
    expect(headers.get('X-Trace-Id')).toBeNull();
  });

  it('preserves client-provided headers and merges trace_id', async () => {
    const mockResp = new Response('{}', { status: 200 });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResp);

    await apiFetch('/api/foo', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    const headers = (init as RequestInit).headers as Headers;
    expect(headers.get('Content-Type')).toBe('application/json');
    expect(headers.get('X-Trace-Id')).toMatch(/^[0-9a-f]{32}$/);
  });

  it('defaults credentials to "include" for admin cookie auth', async () => {
    const mockResp = new Response('{}', { status: 200 });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResp);

    await apiFetch('/api/foo');

    const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect((init as RequestInit).credentials).toBe('include');
  });

  it('reads X-Trace-Id back from response header', async () => {
    const mockResp = new Response('{"ok":true}', {
      status: 200,
      headers: { 'X-Trace-Id': 'server-provided-id' },
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResp);

    const result = await apiFetch('/api/test');
    expect(result.traceId).toBe('server-provided-id');
  });
});

describe('apiJson', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('parses JSON response on 2xx', async () => {
    const mockResp = new Response('{"foo":"bar"}', {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResp);

    const { data, traceId } = await apiJson<{ foo: string }>('/api/test');
    expect(data.foo).toBe('bar');
    expect(traceId).toMatch(/^[0-9a-f]{32}$/);
  });

  it('throws with trace_id in error message on non-2xx', async () => {
    const mockResp = new Response('{"error":"invalid_session"}', {
      status: 401,
      headers: { 'Content-Type': 'application/json', 'X-Trace-Id': 'trace-xyz' },
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResp);

    await expect(apiJson('/api/test')).rejects.toThrow(/trace_id=trace-xyz/);
  });
});
